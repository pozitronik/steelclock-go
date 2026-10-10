package app

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/backend/webclient"
	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// DeviceInstance manages the lifecycle of a single display device.
// Each device has its own compositor, backend client, and widget set.
//
// A device that is missing (unplugged, or not connected at startup) is never
// given up on: the instance waits for it and resumes once it is back. A
// supervisor goroutine checks every reconnect interval, or immediately when
// woken (Wake), and reconnects the existing client when the backend supports
// it (display.Reconnectable), otherwise recreates the backend it was using.
type DeviceInstance struct {
	id             string
	comp           *compositor.Compositor
	client         display.Backend
	currentBackend string
	displayWidth   int
	displayHeight  int
	widgetMgr      *WidgetManager
	retryCancel    chan struct{}
	mu             sync.Mutex

	// Device recovery state, guarded by mu.
	cfg           *config.Config // configuration of the last Start
	waiting       bool           // the device is missing; the supervisor is acquiring it
	wantBackend   string         // backend to recreate while waiting; "" = select per cfg
	waitingLogged bool           // a failed recreate attempt has been logged this episode
	supStop       chan struct{}  // closes to stop the supervisor; nil when not running
	supDone       chan struct{}  // closed when the supervisor has exited
	wake          chan struct{}  // requests an immediate check (buffered, 1)
}

// NewDeviceInstance creates a new device instance with the given ID.
// retryCancel is shared across all devices for coordinated shutdown.
func NewDeviceInstance(id string, retryCancel chan struct{}) *DeviceInstance {
	return &DeviceInstance{
		id:          id,
		widgetMgr:   NewWidgetManager(),
		retryCancel: retryCancel,
		wake:        make(chan struct{}, 1),
	}
}

// Start initializes and starts the device with the given per-device configuration.
// A missing device is not an error: the instance waits for it and starts
// rendering once it is connected.
func (d *DeviceInstance) Start(cfg *config.Config, showSplash bool) error {
	d.stopSupervisor() // a previous run's supervisor must not race this start

	d.mu.Lock()
	defer d.mu.Unlock()

	log.Printf("[%s] Starting device (%dx%d)", d.id, cfg.Display.Width, cfg.Display.Height)

	d.cfg = cfg
	d.waiting = false
	d.wantBackend = ""
	d.waitingLogged = false
	d.displayWidth = cfg.Display.Width
	d.displayHeight = cfg.Display.Height

	if err := d.ensureClient(cfg); err != nil {
		var backendErr *BackendUnavailableError
		if !errors.As(err, &backendErr) {
			return err
		}
		log.Printf("[%s] Display device not available (%v); waiting for it", d.id, err)
		d.waiting = true
		d.waitingLogged = true
		d.startSupervisor(cfg)
		return nil
	}

	// A reused client may have lost its device while the device was stopped.
	if r, ok := d.client.(display.Reconnectable); ok && !r.IsConnected() && r.Reconnect() != nil {
		log.Printf("[%s] Display device is not connected; waiting for it", d.id)
		d.wantBackend = d.currentBackend
		d.waiting = true
		d.startSupervisor(cfg)
		return nil
	}

	if err := d.startRendering(cfg, showSplash); err != nil {
		return err
	}

	d.startSupervisor(cfg)
	log.Printf("[%s] Device started successfully", d.id)
	return nil
}

// startRendering applies device settings, creates the widgets and starts the
// compositor on the current client. Callers must hold d.mu.
func (d *DeviceInstance) startRendering(cfg *config.Config, showSplash bool) error {
	// Apply brightness if configured and supported
	if cfg.DirectDriver != nil && cfg.DirectDriver.Brightness != nil {
		if bc, ok := d.client.(display.BrightnessControl); ok {
			if err := bc.SetBrightness(*cfg.DirectDriver.Brightness); err != nil {
				log.Printf("[%s] Warning: Failed to set brightness: %v", d.id, err)
			}
		}
	}

	if showSplash {
		splash := NewSplashRenderer(d.client, d.displayWidth, d.displayHeight)
		if err := splash.ShowStartupAnimation(); err != nil {
			log.Printf("[%s] Warning: Startup animation failed: %v", d.id, err)
		}
	}

	setup, err := d.widgetMgr.CreateFromConfig(d.client, cfg)
	if err != nil {
		var noWidgetsErr *NoWidgetsError
		if errors.As(err, &noWidgetsErr) {
			log.Printf("[%s] WARNING: No widgets enabled", d.id)
		}
		return err
	}

	log.Printf("[%s] Created %d widgets", d.id, len(setup.Widgets))
	for i := range setup.Widgets {
		if i < len(cfg.Widgets) {
			log.Printf("[%s]   Widget %d: %s (type: %s)", d.id, i+1, cfg.Widgets[i].ID, cfg.Widgets[i].Type)
		}
	}

	comp := setup.Compositor
	comp.OnBackendFailure = func() {
		d.handleDeviceLost(comp)
	}
	d.comp = comp

	if err := comp.Start(); err != nil {
		return fmt.Errorf("[%s] failed to start compositor: %w", d.id, err)
	}
	return nil
}

// Stop stops the compositor but keeps the client for reuse
func (d *DeviceInstance) Stop() {
	d.stopSupervisor()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.waiting = false
	if d.comp != nil {
		d.comp.Stop()
		d.comp = nil
		log.Printf("[%s] Stopping compositor (keeping client)", d.id)
	}
}

// Shutdown performs a full shutdown of the device
func (d *DeviceInstance) Shutdown(unregisterOnExit bool) {
	d.stopSupervisor()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.waiting = false

	if d.comp != nil {
		d.comp.Stop()
		d.comp = nil
	}

	if d.client != nil {
		// Return to device's native UI if supported
		if uc, ok := d.client.(display.UIControl); ok {
			if err := uc.ReturnToUI(); err != nil {
				log.Printf("[%s] Warning: Failed to return to UI: %v", d.id, err)
			}
		}

		// Show exit message
		w, h := d.displayWidth, d.displayHeight
		if w == 0 {
			w = config.DefaultDisplayWidth
		}
		if h == 0 {
			h = config.DefaultDisplayHeight
		}
		splash := NewSplashRenderer(d.client, w, h)
		if err := splash.ShowExitMessage(); err != nil {
			log.Printf("[%s] Warning: Exit message failed: %v", d.id, err)
		}

		if unregisterOnExit {
			log.Printf("[%s] Unregistering...", d.id)
			if err := d.client.RemoveGame(); err != nil {
				log.Printf("[%s] Warning: Failed to unregister: %v", d.id, err)
			} else {
				log.Printf("[%s] Successfully unregistered", d.id)
			}
		}
		d.closeClient(d.client)
		d.client = nil
	}
}

// releaseClient gives up a backend client: it unregisters it (RemoveGame)
// and releases its local resources.
func (d *DeviceInstance) releaseClient(client display.Backend) {
	_ = client.RemoveGame()
	d.closeClient(client)
}

// closeClient releases a client's local resources (e.g. the direct driver's
// HID handle), whether or not it was unregistered.
func (d *DeviceInstance) closeClient(client display.Backend) {
	if c, ok := client.(display.Closer); ok {
		if err := c.Close(); err != nil {
			log.Printf("[%s] Warning: Failed to close backend: %v", d.id, err)
		}
	}
}

// ShowTransitionBanner displays a profile transition banner on this device
func (d *DeviceInstance) ShowTransitionBanner(profileName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client != nil && d.displayWidth > 0 {
		splash := NewSplashRenderer(d.client, d.displayWidth, d.displayHeight)
		if err := splash.ShowTransitionBanner(profileName); err != nil {
			log.Printf("[%s] Warning: Transition banner failed: %v", d.id, err)
		}
	}
}

// ShowWebClientModeMessage displays "WEB CLIENT" on this device
func (d *DeviceInstance) ShowWebClientModeMessage() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client != nil && d.displayWidth > 0 {
		splash := NewSplashRenderer(d.client, d.displayWidth, d.displayHeight)
		if err := splash.ShowWebClientModeMessage(); err != nil {
			log.Printf("[%s] Warning: Failed to show webclient mode message: %v", d.id, err)
		}
	}
}

// GetWebClient returns the webclient if this device uses the webclient backend
func (d *DeviceInstance) GetWebClient() *webclient.Client {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.currentBackend == "webclient" {
		if webClient, ok := d.client.(*webclient.Client); ok {
			return webClient
		}
	}
	return nil
}

// GetCurrentBackend returns the name of this device's backend
func (d *DeviceInstance) GetCurrentBackend() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.currentBackend
}

// ensureClient ensures a valid backend client exists for this device
func (d *DeviceInstance) ensureClient(cfg *config.Config) error {
	needNewClient := d.client == nil

	if d.client != nil {
		if d.currentBackend != cfg.Backend {
			log.Printf("[%s] Backend changed from %s to %s, recreating client...", d.id, d.currentBackend, cfg.Backend)
			needNewClient = true
		}
	}

	if !needNewClient {
		log.Printf("[%s] Reusing existing %s client", d.id, d.currentBackend)
		return nil
	}

	// Clean up old client
	if d.client != nil {
		d.releaseClient(d.client)
		d.client = nil
	}

	// Create new client
	var err error
	var backendName string
	d.client, backendName, err = CreateBackendClient(cfg)
	if err != nil {
		return err
	}
	d.currentBackend = backendName

	// Bind screen event (no-op for direct driver)
	if err := d.bindEventWithRetry(10, GameSenseScreenDeviceType); err != nil {
		log.Printf("[%s] ERROR: Failed to bind screen event after retries: %v", d.id, err)
		d.closeClient(d.client)
		d.client = nil
		return err
	}

	return nil
}

// bindEventWithRetry attempts to bind the screen event with exponential backoff
func (d *DeviceInstance) bindEventWithRetry(maxAttempts int, deviceType string) error {
	return RetryWithBackoff(maxAttempts, d.retryCancel, func(attempt int) error {
		log.Printf("[%s] Attempting to bind screen event (attempt %d/%d)...", d.id, attempt, maxAttempts)
		if err := d.client.BindScreenEvent(EventName, deviceType); err != nil {
			log.Printf("[%s] ERROR: Failed to bind screen event: %v", d.id, err)
			return err
		}
		log.Printf("[%s] Screen event bound successfully", d.id)
		return nil
	})
}

// handleDeviceLost is called by the compositor after repeated heartbeat
// failures. It stops rendering and waits for the device: a Reconnectable client
// is kept and reconnected later; any other client is closed and the same
// backend is recreated once it is available again. It never switches to a
// different backend.
func (d *DeviceInstance) handleDeviceLost(failed *compositor.Compositor) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.comp != failed {
		return // a stale callback from a compositor that was already replaced
	}

	log.Printf("[%s] Display device lost (backend: %s); waiting for it to come back", d.id, d.currentBackend)

	d.comp.Stop()
	d.comp = nil

	if _, ok := d.client.(display.Reconnectable); !ok && d.client != nil {
		d.releaseClient(d.client)
		d.client = nil
	}
	d.wantBackend = d.currentBackend
	d.waiting = true
	d.waitingLogged = false
}

// Wake asks the supervisor to check the device now instead of at its next
// interval, e.g. when the OS reports a device arrival. It never blocks.
func (d *DeviceInstance) Wake() {
	select {
	case d.wake <- struct{}{}:
	default: // a check is already pending
	}
}

// startSupervisor starts the device supervisor if it is not running.
// Callers must hold d.mu.
func (d *DeviceInstance) startSupervisor(cfg *config.Config) {
	if d.supStop != nil {
		return
	}

	interval := time.Duration(cfg.ReconnectIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = time.Duration(config.DefaultReconnectIntervalMs) * time.Millisecond
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	d.supStop, d.supDone = stop, done

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-d.retryCancel:
				return
			case <-ticker.C:
			case <-d.wake:
			}
			d.supervise(stop)
		}
	}()
}

// stopSupervisor stops the device supervisor and waits for it to exit.
// Callers must not hold d.mu: the supervisor takes it while checking.
func (d *DeviceInstance) stopSupervisor() {
	d.mu.Lock()
	stop, done := d.supStop, d.supDone
	d.supStop, d.supDone = nil, nil
	d.mu.Unlock()

	if stop == nil {
		return
	}
	close(stop)
	<-done
}

// supervise runs one device check: it acquires a missing device, or reconnects
// a running device whose client reports it disconnected (so a quick replug
// recovers within one interval, before the compositor gives up on it).
func (d *DeviceInstance) supervise(stop chan struct{}) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.supStop != stop {
		return // stopping; Stop/Start/Shutdown own the state now
	}

	if !d.waiting {
		if r, ok := d.client.(display.Reconnectable); ok && !r.IsConnected() {
			_ = r.Reconnect()
		}
		return
	}

	if !d.acquireClient() {
		return
	}
	if err := d.startRendering(d.cfg, false); err != nil {
		log.Printf("[%s] ERROR: Failed to resume device: %v", d.id, err)
		return
	}
	d.waiting = false
	log.Printf("[%s] Display device is back; resumed on %s backend", d.id, d.currentBackend)
}

// acquireClient tries to get a connected client for a waiting device, quietly
// while the device is still missing. Callers must hold d.mu.
func (d *DeviceInstance) acquireClient() bool {
	if r, ok := d.client.(display.Reconnectable); ok {
		return r.Reconnect() == nil
	}

	// Only run (and log) a full backend creation once it is likely to succeed.
	if !BackendAvailable(d.cfg, d.wantBackend) {
		return false
	}

	var client display.Backend
	var name string
	var err error
	if d.wantBackend != "" {
		name = d.wantBackend
		client, err = CreateBackendByName(name, d.cfg)
	} else {
		client, name, err = CreateBackendClient(d.cfg)
	}
	if err == nil {
		if err = client.BindScreenEvent(EventName, GameSenseScreenDeviceType); err != nil {
			d.releaseClient(client)
		}
	}
	if err != nil {
		if !d.waitingLogged {
			log.Printf("[%s] Display device not ready yet: %v", d.id, err)
			d.waitingLogged = true
		}
		return false
	}

	d.client = client
	d.currentBackend = name
	return true
}
