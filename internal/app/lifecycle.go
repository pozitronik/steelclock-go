package app

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/backend"
	"github.com/pozitronik/steelclock-go/internal/backend/preview"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// ErrorDisplayRefreshRateMs is the refresh rate for error display (flash interval)
const ErrorDisplayRefreshRateMs = 500

// LifecycleManager handles the lifecycle of the display system.
// It manages the compositor, backend client, and transitions between states.
type LifecycleManager struct {
	comp           *compositor.Compositor
	client         display.Backend
	currentBackend string
	lastGoodConfig *config.Config
	isFirstStart   bool
	retryCancel    chan struct{}
	widgetMgr      *WidgetManager
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

// Start initializes and starts the compositor with the given configuration.
// Returns NoWidgetsError if no widgets are enabled.
// Returns BackendUnavailableError if the backend cannot be reached.
func (m *LifecycleManager) Start(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	// Apply custom font URL if configured
	if cfg.BundledFontURL != nil && *cfg.BundledFontURL != "" {
		bitmap.SetBundledFontURL(*cfg.BundledFontURL)
		log.Printf("Using custom bundled font URL: %s", *cfg.BundledFontURL)
	}

	// Determine if we need a new client
	if err := m.ensureClient(cfg); err != nil {
		return err
	}

	// Show startup animation on first start
	if m.isFirstStart {
		splash := NewSplashRenderer(m.client, cfg.Display.Width, cfg.Display.Height)
		if err := splash.ShowStartupAnimation(); err != nil {
			log.Printf("Warning: Startup animation failed: %v", err)
		}
		m.isFirstStart = false
	}

	// Create widgets and compositor using WidgetManager
	setup, err := m.widgetMgr.CreateFromConfig(m.client, cfg)
	if err != nil {
		var noWidgetsErr *NoWidgetsError
		if errors.As(err, &noWidgetsErr) {
			log.Println("WARNING: No widgets enabled in configuration")
		}
		return err
	}

	log.Printf("Created %d widgets", len(setup.Widgets))
	for i := range setup.Widgets {
		log.Printf("  Widget %d: %s (type: %s)", i+1, cfg.Widgets[i].ID, cfg.Widgets[i].Type)
	}

	m.comp = setup.Compositor

	// Set up backend failover callback for auto-select mode
	if cfg.Backend == "" {
		m.comp.OnBackendFailure = func() {
			m.handleBackendFailure(cfg)
		}
	}

	if err := m.comp.Start(); err != nil {
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	m.lastGoodConfig = cfg
	log.Println("SteelClock started successfully")
	return nil
}

// Stop stops the compositor but keeps the client for reuse
func (m *LifecycleManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.comp != nil {
		m.comp.Stop()
		m.comp = nil
		log.Println("Stopping compositor (keeping client and registration)")
	}
}

// Shutdown performs a full shutdown including client cleanup
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

	if m.comp != nil {
		m.comp.Stop()
		m.comp = nil
	}

	if m.client != nil {
		// Show exit message
		displayWidth := config.DefaultDisplayWidth
		displayHeight := config.DefaultDisplayHeight
		if m.lastGoodConfig != nil {
			displayWidth = m.lastGoodConfig.Display.Width
			displayHeight = m.lastGoodConfig.Display.Height
		}
		splash := NewSplashRenderer(m.client, displayWidth, displayHeight)
		if err := splash.ShowExitMessage(); err != nil {
			log.Printf("Warning: Exit message failed: %v", err)
		}

		// Unregister if configured
		shouldUnregister := m.lastGoodConfig != nil && m.lastGoodConfig.UnregisterOnExit
		if shouldUnregister {
			log.Println("Unregistering from GameSense (unregister_on_exit=true)...")
			if err := m.client.RemoveGame(); err != nil {
				log.Printf("Warning: Failed to unregister game: %v", err)
			} else {
				log.Println("Successfully unregistered from GameSense")
			}
		} else {
			log.Println("Shutting down (keeping GameSense registration, unregister_on_exit=false)")
		}
		m.client = nil
	}
}

// ShowTransitionBanner displays a profile transition banner
func (m *LifecycleManager) ShowTransitionBanner(profileName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		displayWidth := config.DefaultDisplayWidth
		displayHeight := config.DefaultDisplayHeight
		if m.lastGoodConfig != nil {
			displayWidth = m.lastGoodConfig.Display.Width
			displayHeight = m.lastGoodConfig.Display.Height
		}
		splash := NewSplashRenderer(m.client, displayWidth, displayHeight)
		if err := splash.ShowTransitionBanner(profileName); err != nil {
			log.Printf("Warning: Transition banner failed: %v", err)
		}
	}
}

// StartErrorDisplay starts a compositor showing an error message
func (m *LifecycleManager) StartErrorDisplay(message string, width, height int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("Starting error display: %s", message)

	errorClient := m.client
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

		// Bind screen event (no-op for direct driver)
		if err := errorClient.BindScreenEvent(EventName, DeviceType); err != nil {
			log.Printf("ERROR: Failed to bind screen event for error display: %v", err)
			return fmt.Errorf("failed to bind screen event: %w", err)
		}
	}

	setup := m.widgetMgr.CreateErrorDisplay(errorClient, message, width, height)
	m.comp = setup.Compositor

	if err := m.comp.Start(); err != nil {
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

// GetDisplayDimensions returns the display dimensions from config or defaults
func (m *LifecycleManager) GetDisplayDimensions() (width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.lastGoodConfig != nil {
		return m.lastGoodConfig.Display.Width, m.lastGoodConfig.Display.Height
	}
	return config.DefaultDisplayWidth, config.DefaultDisplayHeight
}

// ensureClient ensures a valid backend client exists, creating one if needed
func (m *LifecycleManager) ensureClient(cfg *config.Config) error {
	needNewClient := m.client == nil

	if m.client != nil {
		// Check if backend changed
		if m.currentBackend != cfg.Backend {
			log.Printf("Backend changed from %s to %s, recreating client...", m.currentBackend, cfg.Backend)
			needNewClient = true
		}
	}

	if !needNewClient {
		log.Printf("Reusing existing %s client", m.currentBackend)
		return nil
	}

	// Clean up old client
	if m.client != nil {
		_ = m.client.RemoveGame()
		m.client = nil
	}

	// Create new client
	var err error
	var backendName string
	m.client, backendName, err = CreateBackendClient(cfg)
	if err != nil {
		return err
	}
	m.currentBackend = backendName

	// Bind screen event (no-op for direct driver)
	if err := m.bindEventWithRetry(10); err != nil {
		log.Printf("ERROR: Failed to bind screen event after retries: %v", err)
		m.client = nil
		return err
	}

	return nil
}

// bindEventWithRetry attempts to bind the screen event with exponential backoff
func (m *LifecycleManager) bindEventWithRetry(maxAttempts int) error {
	return RetryWithBackoff(maxAttempts, m.retryCancel, func(attempt int) error {
		log.Printf("Attempting to bind screen event (attempt %d/%d)...", attempt, maxAttempts)
		if err := m.client.BindScreenEvent(EventName, DeviceType); err != nil {
			log.Printf("ERROR: Failed to bind screen event: %v", err)
			return err
		}
		log.Println("Screen event bound successfully")
		return nil
	})
}

// handleBackendFailure attempts to switch to alternative backend when current backend fails
func (m *LifecycleManager) handleBackendFailure(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("========================================")
	log.Printf("Backend failure detected (current: %s)", m.currentBackend)
	log.Println("Attempting to switch to alternative backend...")

	if m.comp != nil {
		m.comp.Stop()
		m.comp = nil
	}

	// Try to create a backend, excluding the current one
	newClient, newBackend, err := CreateBackendExcluding(cfg, m.currentBackend)
	if err != nil {
		log.Printf("ERROR: Failed to switch to alternative backend: %v", err)
		log.Println("Will retry on next heartbeat cycle...")
		return
	}

	m.client = newClient
	m.currentBackend = newBackend
	log.Printf("Successfully switched to %s backend", m.currentBackend)

	// Bind screen event (no-op for direct driver)
	if err := m.client.BindScreenEvent(EventName, DeviceType); err != nil {
		log.Printf("ERROR: Failed to bind screen event: %v", err)
		return
	}

	setup, err := m.widgetMgr.CreateFromConfig(m.client, cfg)
	if err != nil {
		log.Printf("ERROR: Failed to recreate widgets: %v", err)
		return
	}

	m.comp = setup.Compositor
	m.comp.OnBackendFailure = func() {
		m.handleBackendFailure(cfg)
	}

	if err := m.comp.Start(); err != nil {
		log.Printf("ERROR: Failed to start compositor with new backend: %v", err)
		return
	}

	log.Println("Successfully recovered with alternative backend")
	log.Println("========================================")
}

// GetPreviewClient returns the preview client if the preview backend is active
func (m *LifecycleManager) GetPreviewClient() *preview.Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentBackend == "preview" {
		if previewClient, ok := m.client.(*preview.Client); ok {
			return previewClient
		}
	}
	return nil
}

// GetCurrentBackend returns the name of the current backend
func (m *LifecycleManager) GetCurrentBackend() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentBackend
}

// ShowPreviewModeMessage displays "PREVIEW MODE" on the current backend
// This is called before switching to preview backend
func (m *LifecycleManager) ShowPreviewModeMessage() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return
	}

	displayWidth := config.DefaultDisplayWidth
	displayHeight := config.DefaultDisplayHeight
	if m.lastGoodConfig != nil {
		displayWidth = m.lastGoodConfig.Display.Width
		displayHeight = m.lastGoodConfig.Display.Height
	}

	splash := NewSplashRenderer(m.client, displayWidth, displayHeight)
	if err := splash.ShowPreviewModeMessage(); err != nil {
		log.Printf("Warning: Failed to show preview mode message: %v", err)
	}
}
