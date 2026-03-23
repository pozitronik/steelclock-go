package app

import (
	"fmt"
	"log"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/backend"
	"github.com/pozitronik/steelclock-go/internal/backend/webclient"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// ErrorDisplayRefreshRateMs is the refresh rate for error display (flash interval)
const ErrorDisplayRefreshRateMs = 500

// LifecycleManager handles the lifecycle of the display system.
// It orchestrates multiple DeviceInstances, one per configured device.
type LifecycleManager struct {
	devices        []*DeviceInstance
	errorComp      *compositor.Compositor // Used only for error display mode
	errorClient    display.Backend        // Used only for error display mode
	lastGoodConfig *config.Config
	isFirstStart   bool
	retryCancel    chan struct{}
	widgetMgr      *WidgetManager // For error display only
	mu             sync.Mutex
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		isFirstStart: true,
		retryCancel:  make(chan struct{}),
		widgetMgr:    NewWidgetManager(),
	}
}

// Start initializes and starts all devices from the given configuration.
// In single-device mode (top-level widgets), a single DeviceInstance is created.
// In multi-device mode (devices array), one DeviceInstance per device is created.
// Returns the first error encountered; other devices may still start successfully.
func (m *LifecycleManager) Start(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	// Apply custom font URL if configured
	if cfg.BundledFontURL != nil && *cfg.BundledFontURL != "" {
		bitmap.SetBundledFontURL(*cfg.BundledFontURL)
		log.Printf("Using custom bundled font URL: %s", *cfg.BundledFontURL)
	}

	// Stop any active error display
	m.stopErrorDisplay()

	// Get per-device configurations
	deviceConfigs := cfg.GetDevices()
	showSplash := m.isFirstStart
	m.isFirstStart = false

	var firstErr error
	var startedDevices []*DeviceInstance

	for _, devCfg := range deviceConfigs {
		deviceID := devCfg.ID
		if deviceID == "" {
			deviceID = fmt.Sprintf("device_%d", len(startedDevices))
		}

		// Build per-device config by merging global + device-specific settings
		perDeviceCfg := cfg.ConfigForDevice(devCfg)

		// Try to reuse existing DeviceInstance with same ID
		instance := m.findDevice(deviceID)
		if instance == nil {
			instance = NewDeviceInstance(deviceID, m.retryCancel)
		}

		if err := instance.Start(perDeviceCfg, showSplash); err != nil {
			log.Printf("[%s] ERROR: Failed to start device: %v", deviceID, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		startedDevices = append(startedDevices, instance)
	}

	// Shut down old devices that are no longer in the config
	m.shutdownOldDevices(startedDevices)
	m.devices = startedDevices

	if len(startedDevices) == 0 && firstErr != nil {
		return firstErr
	}

	m.lastGoodConfig = cfg
	log.Printf("SteelClock started successfully (%d device(s))", len(startedDevices))
	return nil
}

// Stop stops all device compositors but keeps clients for reuse
func (m *LifecycleManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopErrorDisplay()

	for _, dev := range m.devices {
		dev.Stop()
	}
}

// Shutdown performs a full shutdown including client cleanup for all devices
func (m *LifecycleManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel any ongoing retry
	select {
	case <-m.retryCancel:
		// Already closed
	default:
		close(m.retryCancel)
	}

	m.stopErrorDisplay()

	shouldUnregister := m.lastGoodConfig != nil && m.lastGoodConfig.UnregisterOnExit

	for _, dev := range m.devices {
		dev.Shutdown(shouldUnregister)
	}
	m.devices = nil

	if shouldUnregister {
		log.Println("All devices unregistered (unregister_on_exit=true)")
	} else {
		log.Println("Shutting down (keeping registrations, unregister_on_exit=false)")
	}
}

// ShowTransitionBanner displays a profile transition banner on all devices
func (m *LifecycleManager) ShowTransitionBanner(profileName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, dev := range m.devices {
		dev.ShowTransitionBanner(profileName)
	}
}

// StartErrorDisplay starts a compositor showing an error message
func (m *LifecycleManager) StartErrorDisplay(message string, width, height int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("Starting error display: %s", message)

	// Try to get an existing client from a device
	errorClient := m.findAnyClient()
	if errorClient == nil {
		log.Println("No existing client, creating one for error display...")

		var err error
		if m.lastGoodConfig != nil {
			errorClient, _, err = CreateBackendClient(m.lastGoodConfig)
		} else {
			// Use default config for error display
			defaultCfg := config.CreateDefault()
			result, createErr := backend.Create(defaultCfg)
			if createErr != nil {
				err = createErr
			} else {
				errorClient = result.Backend
			}
		}
		if err != nil {
			log.Printf("ERROR: Failed to create client for error display: %v", err)
			return fmt.Errorf("failed to create client: %w", err)
		}
		m.errorClient = errorClient

		// Bind screen event (no-op for direct driver)
		deviceType := DeviceTypeForDisplay(width, height)
		if err := errorClient.BindScreenEvent(EventName, deviceType); err != nil {
			log.Printf("ERROR: Failed to bind screen event for error display: %v", err)
			return fmt.Errorf("failed to bind screen event: %w", err)
		}
	}

	setup := m.widgetMgr.CreateErrorDisplay(errorClient, message, width, height)
	m.errorComp = setup.Compositor

	if err := m.errorComp.Start(); err != nil {
		log.Printf("ERROR: Failed to start compositor for error display: %v", err)
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	log.Println("Error display started - screen will flash error message")
	log.Println("Fix the configuration file and reload to continue")

	return nil
}

// GetLastGoodConfig returns the last successfully loaded configuration
func (m *LifecycleManager) GetLastGoodConfig() *config.Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastGoodConfig
}

// GetDisplayDimensions returns the display dimensions from the first device or defaults
func (m *LifecycleManager) GetDisplayDimensions() (width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.lastGoodConfig != nil {
		devices := m.lastGoodConfig.GetDevices()
		if len(devices) > 0 {
			return devices[0].Display.Width, devices[0].Display.Height
		}
		return m.lastGoodConfig.Display.Width, m.lastGoodConfig.Display.Height
	}
	return config.DefaultDisplayWidth, config.DefaultDisplayHeight
}

// GetWebClient returns the webclient if any device uses the webclient backend
func (m *LifecycleManager) GetWebClient() *webclient.Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, dev := range m.devices {
		if wc := dev.GetWebClient(); wc != nil {
			return wc
		}
	}
	return nil
}

// GetWebClients returns all webclient backends keyed by device ID.
// Used for multi-device preview in the web editor.
func (m *LifecycleManager) GetWebClients() map[string]*webclient.Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	clients := make(map[string]*webclient.Client)
	for _, dev := range m.devices {
		if wc := dev.GetWebClient(); wc != nil {
			clients[dev.id] = wc
		}
	}
	return clients
}

// GetCurrentBackend returns the name of the first device's backend
func (m *LifecycleManager) GetCurrentBackend() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.devices) > 0 {
		return m.devices[0].GetCurrentBackend()
	}
	return ""
}

// ShowWebClientModeMessage displays "WEB CLIENT" on all devices
func (m *LifecycleManager) ShowWebClientModeMessage() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, dev := range m.devices {
		dev.ShowWebClientModeMessage()
	}
}

// findDevice returns an existing DeviceInstance by ID, or nil
func (m *LifecycleManager) findDevice(id string) *DeviceInstance {
	for _, dev := range m.devices {
		if dev.id == id {
			return dev
		}
	}
	return nil
}

// findAnyClient returns a backend client from any active device
func (m *LifecycleManager) findAnyClient() display.Backend {
	for _, dev := range m.devices {
		dev.mu.Lock()
		client := dev.client
		dev.mu.Unlock()
		if client != nil {
			return client
		}
	}
	return nil
}

// shutdownOldDevices shuts down devices that are not in the new set
func (m *LifecycleManager) shutdownOldDevices(newDevices []*DeviceInstance) {
	newSet := make(map[*DeviceInstance]bool, len(newDevices))
	for _, dev := range newDevices {
		newSet[dev] = true
	}

	for _, dev := range m.devices {
		if !newSet[dev] {
			dev.Shutdown(false) // Don't unregister on config reload
		}
	}
}

// stopErrorDisplay stops the error display compositor if active
func (m *LifecycleManager) stopErrorDisplay() {
	if m.errorComp != nil {
		m.errorComp.Stop()
		m.errorComp = nil
	}
	if m.errorClient != nil {
		_ = m.errorClient.RemoveGame()
		m.errorClient = nil
	}
}
