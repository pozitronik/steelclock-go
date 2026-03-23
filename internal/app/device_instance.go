package app

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/backend/webclient"
	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// DeviceInstance manages the lifecycle of a single display device.
// Each device has its own compositor, backend client, and widget set.
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
}

// NewDeviceInstance creates a new device instance with the given ID.
// retryCancel is shared across all devices for coordinated shutdown.
func NewDeviceInstance(id string, retryCancel chan struct{}) *DeviceInstance {
	return &DeviceInstance{
		id:          id,
		widgetMgr:   NewWidgetManager(),
		retryCancel: retryCancel,
	}
}

// Start initializes and starts the device with the given per-device configuration.
func (d *DeviceInstance) Start(cfg *config.Config, showSplash bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	log.Printf("[%s] Starting device (%dx%d)", d.id, cfg.Display.Width, cfg.Display.Height)

	if err := d.ensureClient(cfg); err != nil {
		return err
	}

	d.displayWidth = cfg.Display.Width
	d.displayHeight = cfg.Display.Height

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

	d.comp = setup.Compositor

	// Set up backend failover callback for auto-select mode
	if cfg.Backend == "" {
		d.comp.OnBackendFailure = func() {
			d.handleBackendFailure(cfg)
		}
	}

	if err := d.comp.Start(); err != nil {
		return fmt.Errorf("[%s] failed to start compositor: %w", d.id, err)
	}

	log.Printf("[%s] Device started successfully", d.id)
	return nil
}

// Stop stops the compositor but keeps the client for reuse
func (d *DeviceInstance) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.comp != nil {
		d.comp.Stop()
		d.comp = nil
		log.Printf("[%s] Stopping compositor (keeping client)", d.id)
	}
}

// Shutdown performs a full shutdown of the device
func (d *DeviceInstance) Shutdown(unregisterOnExit bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.comp != nil {
		d.comp.Stop()
		d.comp = nil
	}

	if d.client != nil {
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
		d.client = nil
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
		_ = d.client.RemoveGame()
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
	deviceType := DeviceTypeForDisplay(cfg.Display.Width, cfg.Display.Height)
	if err := d.bindEventWithRetry(10, deviceType); err != nil {
		log.Printf("[%s] ERROR: Failed to bind screen event after retries: %v", d.id, err)
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

// handleBackendFailure attempts to switch to alternative backend
func (d *DeviceInstance) handleBackendFailure(cfg *config.Config) {
	d.mu.Lock()
	defer d.mu.Unlock()

	log.Println("========================================")
	log.Printf("[%s] Backend failure detected (current: %s)", d.id, d.currentBackend)
	log.Printf("[%s] Attempting to switch to alternative backend...", d.id)

	if d.comp != nil {
		d.comp.Stop()
		d.comp = nil
	}

	newClient, newBackend, err := CreateBackendExcluding(cfg, d.currentBackend)
	if err != nil {
		log.Printf("[%s] ERROR: Failed to switch to alternative backend: %v", d.id, err)
		log.Printf("[%s] Will retry on next heartbeat cycle...", d.id)
		return
	}

	d.client = newClient
	d.currentBackend = newBackend
	log.Printf("[%s] Successfully switched to %s backend", d.id, d.currentBackend)

	deviceType := DeviceTypeForDisplay(cfg.Display.Width, cfg.Display.Height)
	if err := d.client.BindScreenEvent(EventName, deviceType); err != nil {
		log.Printf("[%s] ERROR: Failed to bind screen event: %v", d.id, err)
		return
	}

	setup, err := d.widgetMgr.CreateFromConfig(d.client, cfg)
	if err != nil {
		log.Printf("[%s] ERROR: Failed to recreate widgets: %v", d.id, err)
		return
	}

	d.comp = setup.Compositor
	d.comp.OnBackendFailure = func() {
		d.handleBackendFailure(cfg)
	}

	if err := d.comp.Start(); err != nil {
		log.Printf("[%s] ERROR: Failed to start compositor with new backend: %v", d.id, err)
		return
	}

	log.Printf("[%s] Successfully recovered with alternative backend", d.id)
	log.Println("========================================")
}
