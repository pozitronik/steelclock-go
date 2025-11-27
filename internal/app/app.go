package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/tray"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// GameSense API constants
const (
	EventName     = "STEELCLOCK_DISPLAY"
	DeviceType    = "screened-128x40"
	DeveloperName = "Pozitronik"
)

// BackendUnavailableError indicates display backend is not available
type BackendUnavailableError struct {
	Err error
}

func (e *BackendUnavailableError) Error() string {
	return fmt.Sprintf("backend unavailable: %v", e.Err)
}

func (e *BackendUnavailableError) Unwrap() error {
	return e.Err
}

// NoWidgetsError indicates that no widgets are enabled in the configuration
type NoWidgetsError struct{}

func (e *NoWidgetsError) Error() string {
	return "no widgets enabled in configuration"
}

// App encapsulates all application state and lifecycle management
type App struct {
	comp           *compositor.Compositor
	client         gamesense.API
	trayMgr        *tray.Manager
	configPath     string
	profileMgr     *config.ProfileManager
	mu             sync.Mutex
	lastGoodConfig *config.Config
	retryCancel    chan struct{}
	currentBackend string
}

// NewApp creates a new application instance (legacy single-config mode)
func NewApp(configPath string) *App {
	return &App{
		configPath:  configPath,
		retryCancel: make(chan struct{}),
	}
}

// NewAppWithProfiles creates a new application instance with profile support
func NewAppWithProfiles(profileMgr *config.ProfileManager) *App {
	return &App{
		profileMgr:  profileMgr,
		retryCancel: make(chan struct{}),
	}
}

// Run starts the application with system tray
func (a *App) Run() {
	log.Println("========================================")
	log.Println("SteelClock starting...")

	// Create tray manager based on mode
	if a.profileMgr != nil {
		activeProfile := a.profileMgr.GetActiveProfile()
		if activeProfile != nil {
			log.Printf("Active profile: %s (%s)", activeProfile.Name, activeProfile.Path)
		}
		log.Printf("Available profiles: %d", len(a.profileMgr.GetProfiles()))
		a.trayMgr = tray.NewManagerWithProfiles(a.profileMgr, a.ReloadConfig, a.SwitchProfile, a.Stop)
	} else {
		log.Printf("Config: %s", a.configPath)
		a.trayMgr = tray.NewManager(a.configPath, a.ReloadConfig, a.Stop)
	}

	log.Println("========================================")

	// Set callback to run when tray is ready
	a.trayMgr.OnReady(func() {
		if err := a.Start(); err != nil {
			a.handleStartupFailure(err)
		}
	})

	log.Println("System tray initializing. Use tray icon to control the application.")

	// Run system tray (blocks until Quit)
	a.trayMgr.Run()

	log.Println("SteelClock shutting down...")

	// Cancel any ongoing retry
	close(a.retryCancel)

	a.stopAndWait()
	log.Println("SteelClock stopped")
}

// handleStartupFailure logs and notifies user about startup errors
func (a *App) handleStartupFailure(err error) {
	log.Println("========================================")
	log.Println("STARTUP ERROR")
	log.Printf("Error: %v", err)
	log.Println("========================================")
	log.Println("")

	var backendErr *BackendUnavailableError
	var noWidgetsErr *NoWidgetsError

	if errors.As(err, &backendErr) {
		log.Println("Cannot connect to display backend.")
		log.Println("This usually happens when:")
		log.Println("  - Device is not connected")
		log.Println("  - SteelSeries GG is not running (for gamesense backend)")
		log.Println("  - Backend is still cleaning up from previous instance")
		log.Println("")
		log.Println("The application will continue running. Use 'Reload Config' to retry.")
		tray.ShowNotification("SteelClock Connection Error", "Cannot connect to display. Check device connection and try 'Reload Config'.")
	} else if errors.As(err, &noWidgetsErr) {
		log.Println("Application failed to start. Please check the error above and fix config.json")
		log.Println("Use 'Reload Config' to retry after fixing the issue.")
		tray.ShowNotification("SteelClock Error", "No widgets enabled in configuration. Please check config.json")
	} else {
		log.Println("Application failed to start. Please check the error above and fix config.json")
		log.Println("Use 'Reload Config' to retry after fixing the issue.")
		tray.ShowNotification("SteelClock Configuration Error", "Failed to load configuration. Please check config.json for errors.")
	}
	log.Println("")
}

// Start initializes and starts all components
func (a *App) Start() error {
	var cfg *config.Config
	var err error

	if a.profileMgr != nil {
		cfg, err = a.profileMgr.GetActiveConfig()
	} else {
		cfg, err = config.Load(a.configPath)
	}

	if err != nil {
		log.Printf("ERROR: Failed to load config: %v", err)
		return a.handleStartupError(err, nil)
	}

	if err := a.startWithConfig(cfg); err != nil {
		return a.handleStartupError(err, cfg)
	}

	return nil
}

// Stop stops all components gracefully (used during reload)
func (a *App) Stop() {
	a.stopInternal(false)
}

// stopAndWait stops all components and optionally unregisters based on config
func (a *App) stopAndWait() {
	a.stopInternal(true)
}

// stopInternal stops all components
func (a *App) stopInternal(isFinalShutdown bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.comp != nil {
		a.comp.Stop()
		a.comp = nil
	}

	if a.client != nil {
		shouldUnregister := a.lastGoodConfig != nil && a.lastGoodConfig.UnregisterOnExit

		if shouldUnregister && isFinalShutdown {
			log.Println("Unregistering from GameSense (unregister_on_exit=true)...")
			if err := a.client.RemoveGame(); err != nil {
				log.Printf("Warning: Failed to unregister game: %v", err)
			} else {
				log.Println("Successfully unregistered from GameSense")
			}
			a.client = nil
		} else if !isFinalShutdown {
			log.Println("Stopping compositor (keeping GameSense client and registration)")
		} else {
			log.Println("Shutting down (keeping GameSense registration, unregister_on_exit=false)")
			a.client = nil
		}
	}
}

// ReloadConfig reloads configuration and restarts components
func (a *App) ReloadConfig() error {
	log.Println("========================================")
	log.Println("Reloading configuration...")

	var configPath string
	if a.profileMgr != nil {
		activeProfile := a.profileMgr.GetActiveProfile()
		if activeProfile != nil {
			configPath = activeProfile.Path
			log.Printf("Active profile: %s", activeProfile.Name)
		} else {
			return fmt.Errorf("no active profile")
		}
	} else {
		configPath = a.configPath
	}

	log.Printf("Config file: %s", configPath)

	absPath, _ := filepath.Abs(configPath)
	log.Printf("Absolute path: %s", absPath)

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		log.Printf("ERROR: Cannot access config file: %v", err)
		log.Println("Keeping current configuration running")
		return fmt.Errorf("cannot access config file: %w", err)
	}
	log.Printf("Config file last modified: %s", fileInfo.ModTime().Format("2006-01-02 15:04:05"))

	newCfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("ERROR: Config validation failed: %v", err)
		log.Println("Stopping current instance and showing error...")

		a.Stop()
		time.Sleep(1 * time.Second)

		return a.handleStartupError(err, nil)
	}

	log.Println("New config validated successfully")
	log.Printf("Loaded config: %s (%s) with %d widgets", newCfg.GameName, newCfg.GameDisplayName, len(newCfg.Widgets))

	log.Println("Stopping current instance...")
	a.Stop()

	log.Println("Waiting for GameSense API to settle...")
	time.Sleep(2 * time.Second)

	log.Println("Starting with new config...")
	if err := a.startWithConfig(newCfg); err != nil {
		log.Printf("ERROR: Failed to start with new config: %v", err)
		time.Sleep(1 * time.Second)
		return a.handleStartupError(err, newCfg)
	}

	log.Println("Configuration reloaded successfully!")
	log.Printf("Running with: %s (%s)", newCfg.GameName, newCfg.GameDisplayName)
	log.Println("========================================")
	return nil
}

// SwitchProfile switches to a different configuration profile
func (a *App) SwitchProfile(path string) error {
	if a.profileMgr == nil {
		return fmt.Errorf("profile manager not available")
	}

	log.Println("========================================")
	log.Printf("Switching to profile: %s", path)

	// Update active profile in profile manager
	if err := a.profileMgr.SetActiveProfile(path); err != nil {
		log.Printf("ERROR: Failed to set active profile: %v", err)
		return err
	}

	// Load new config
	newCfg, err := a.profileMgr.GetActiveConfig()
	if err != nil {
		log.Printf("ERROR: Failed to load profile config: %v", err)
		return a.handleStartupError(err, nil)
	}

	log.Printf("Loaded config: %s (%s) with %d widgets", newCfg.GameName, newCfg.GameDisplayName, len(newCfg.Widgets))

	// Stop current instance
	log.Println("Stopping current instance...")
	a.Stop()

	log.Println("Waiting for GameSense API to settle...")
	time.Sleep(2 * time.Second)

	// Start with new config
	log.Println("Starting with new profile...")
	if err := a.startWithConfig(newCfg); err != nil {
		log.Printf("ERROR: Failed to start with new profile: %v", err)
		time.Sleep(1 * time.Second)
		return a.handleStartupError(err, newCfg)
	}

	activeProfile := a.profileMgr.GetActiveProfile()
	log.Printf("Profile switched successfully to: %s", activeProfile.Name)
	log.Println("========================================")
	return nil
}

// startWithConfig initializes and starts all components with a provided config
func (a *App) startWithConfig(cfg *config.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	if cfg.BundledFontURL != nil && *cfg.BundledFontURL != "" {
		bitmap.SetBundledFontURL(*cfg.BundledFontURL)
		log.Printf("Using custom bundled font URL: %s", *cfg.BundledFontURL)
	}

	needNewClient := a.client == nil
	if a.client != nil {
		if gsClient, ok := a.client.(*gamesense.Client); ok {
			if gsClient.GameName() != cfg.GameName {
				log.Printf("GameName changed from %s to %s, recreating client...", gsClient.GameName(), cfg.GameName)
				needNewClient = true
			}
		}
		if a.currentBackend != cfg.Backend {
			log.Printf("Backend changed from %s to %s, recreating client...", a.currentBackend, cfg.Backend)
			needNewClient = true
		}
	}

	var err error
	if needNewClient {
		if a.client != nil {
			_ = a.client.RemoveGame()
			a.client = nil
		}

		a.client, err = CreateBackendClient(cfg)
		if err != nil {
			return err
		}

		if _, ok := a.client.(*gamesense.Client); ok {
			a.currentBackend = "gamesense"
		} else {
			a.currentBackend = "direct"
		}

		if a.currentBackend == "gamesense" {
			if err := a.bindEventWithRetry(10); err != nil {
				log.Printf("ERROR: Failed to bind screen event after retries: %v", err)
				a.client = nil
				return err
			}
		}
	} else {
		log.Printf("Reusing existing %s client", a.currentBackend)
	}

	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		return fmt.Errorf("failed to create widgets: %w", err)
	}

	if len(widgets) == 0 {
		log.Println("WARNING: No widgets enabled in configuration")
		return &NoWidgetsError{}
	}

	log.Printf("Created %d widgets", len(widgets))
	for i := range widgets {
		log.Printf("  Widget %d: %s (type: %s)", i+1, cfg.Widgets[i].ID, cfg.Widgets[i].Type)
	}

	layoutMgr := layout.NewManager(cfg.Display, widgets)
	a.comp = compositor.NewCompositor(a.client, layoutMgr, widgets, cfg)

	if cfg.Backend == "any" {
		a.comp.OnBackendFailure = func() {
			a.handleBackendFailure(cfg)
		}
	}

	if err := a.comp.Start(); err != nil {
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	a.lastGoodConfig = cfg

	log.Println("SteelClock started successfully")
	return nil
}

// bindEventWithRetry attempts to bind the screen event with exponential backoff
func (a *App) bindEventWithRetry(maxAttempts int) error {
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			delay := time.Duration(float64(baseDelay) * float64(uint(1)<<uint(attempt-2)))
			if delay > maxDelay {
				delay = maxDelay
			}

			log.Printf("Retrying bind in %v... (attempt %d/%d)", delay, attempt, maxAttempts)

			select {
			case <-time.After(delay):
			case <-a.retryCancel:
				log.Println("Bind retry cancelled")
				return fmt.Errorf("bind retry cancelled")
			}
		}

		log.Printf("Attempting to bind screen event (attempt %d/%d)...", attempt, maxAttempts)
		if err := a.client.BindScreenEvent(EventName, DeviceType); err != nil {
			log.Printf("ERROR: Failed to bind screen event: %v", err)
			if attempt == maxAttempts {
				return &BackendUnavailableError{Err: err}
			}
			continue
		}

		log.Println("Screen event bound successfully")
		return nil
	}

	return &BackendUnavailableError{Err: fmt.Errorf("failed to bind after %d attempts", maxAttempts)}
}

// handleBackendFailure attempts to switch to alternative backend when current backend fails
func (a *App) handleBackendFailure(cfg *config.Config) {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Println("========================================")
	log.Printf("Backend failure detected (current: %s)", a.currentBackend)
	log.Println("Attempting to switch to alternative backend...")

	if a.comp != nil {
		a.comp.Stop()
		a.comp = nil
	}

	var newClient gamesense.API
	var newBackend string
	var err error

	if a.currentBackend == "gamesense" {
		log.Println("Trying direct driver backend...")
		newClient, err = CreateDirectClient(cfg)
		newBackend = "direct"
	} else {
		log.Println("Trying GameSense backend...")
		newClient, err = CreateGameSenseClient(cfg)
		newBackend = "gamesense"
	}

	if err != nil {
		log.Printf("ERROR: Failed to switch to %s backend: %v", newBackend, err)
		log.Println("Will retry on next heartbeat cycle...")
		return
	}

	a.client = newClient
	a.currentBackend = newBackend
	log.Printf("Successfully switched to %s backend", a.currentBackend)

	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		log.Printf("ERROR: Failed to recreate widgets: %v", err)
		return
	}

	layoutMgr := layout.NewManager(cfg.Display, widgets)
	a.comp = compositor.NewCompositor(a.client, layoutMgr, widgets, cfg)

	a.comp.OnBackendFailure = func() {
		a.handleBackendFailure(cfg)
	}

	if err := a.comp.Start(); err != nil {
		log.Printf("ERROR: Failed to start compositor with new backend: %v", err)
		return
	}

	log.Println("Successfully recovered with alternative backend")
	log.Println("========================================")
}

// handleStartupError handles errors during startup/reload and shows error display if appropriate
func (a *App) handleStartupError(err error, cfg *config.Config) error {
	var backendErr *BackendUnavailableError
	if errors.As(err, &backendErr) {
		log.Println("========================================")
		log.Println("CRITICAL: Cannot connect to display backend")
		log.Println("This may indicate:")
		log.Println("  - Device is not connected")
		log.Println("  - SteelSeries GG is not running (for gamesense backend)")
		log.Println("  - Backend is still cleaning up from previous instance")
		log.Println("========================================")
		return backendErr
	}

	displayWidth := config.DefaultDisplayWidth
	displayHeight := config.DefaultDisplayHeight
	if cfg != nil {
		displayWidth = cfg.Display.Width
		displayHeight = cfg.Display.Height
	} else {
		if a.lastGoodConfig != nil {
			displayWidth = a.lastGoodConfig.Display.Width
			displayHeight = a.lastGoodConfig.Display.Height
		}
	}

	errorMsg := "CONFIG"
	var noWidgetsErr *NoWidgetsError
	if errors.As(err, &noWidgetsErr) {
		errorMsg = "NO WIDGETS"
	}

	log.Println("Displaying error on OLED screen...")
	if dispErr := a.startWithErrorDisplay(errorMsg, displayWidth, displayHeight); dispErr != nil {
		log.Printf("ERROR: Failed to show error display: %v", dispErr)
		return fmt.Errorf("startup failed and error display failed: %w", dispErr)
	}

	return err
}
