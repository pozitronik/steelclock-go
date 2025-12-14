package app

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/tray"
	"github.com/pozitronik/steelclock-go/internal/webeditor"
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

// App encapsulates all application state and lifecycle management.
// It acts as the main orchestrator, delegating to specialized managers.
type App struct {
	lifecycle *LifecycleManager
	configMgr *ConfigManager
	trayMgr   *tray.Manager
	webEditor *webeditor.Server

	// configMu serializes config reload and profile switch operations.
	// This prevents race conditions when multiple sources (tray, web editor)
	// trigger config changes concurrently.
	configMu sync.Mutex

	// webclientBrowserOpened tracks if we've already opened the webclient browser
	// to avoid opening multiple times during session
	webclientBrowserOpened bool

	// WebClient override state - for temporary webclient backend when using config editor
	webclientOverrideActive   bool
	webclientOverrideOriginal string // Original backend name to restore
}

// NewApp creates a new application instance (legacy single-config mode)
func NewApp(configPath string) *App {
	return &App{
		lifecycle: NewLifecycleManager(),
		configMgr: NewConfigManager(configPath),
	}
}

// NewAppWithProfiles creates a new application instance with profile support
func NewAppWithProfiles(profileMgr *config.ProfileManager) *App {
	return &App{
		lifecycle: NewLifecycleManager(),
		configMgr: NewConfigManagerWithProfiles(profileMgr),
	}
}

// Run starts the application with system tray
func (a *App) Run() {
	log.Println("========================================")
	log.Println("SteelClock starting...")

	// Log configuration info
	a.configMgr.LogStartupInfo()

	// Create tray manager based on mode
	if a.configMgr.HasProfiles() {
		a.trayMgr = tray.NewManagerWithProfiles(a.configMgr.GetProfileManager(), a.ReloadConfig, a.SwitchProfile, a.Stop)
	} else {
		a.trayMgr = tray.NewManager(a.configMgr.GetConfigPath(), a.ReloadConfig, a.Stop)
	}

	// Create web editor server
	a.createWebEditor()

	log.Println("========================================")

	// Set callback to run when tray is ready
	a.trayMgr.OnReady(func() {
		if err := a.Start(); err != nil {
			a.handleStartupFailure(err)
		}

		// Auto-start web editor
		if a.webEditor != nil {
			if err := a.webEditor.Start(); err != nil {
				log.Printf("Failed to auto-start web editor: %v", err)
			} else {
				log.Printf("Web editor started at %s", a.webEditor.GetURL())
				// Try to open webclient browser now that web editor is running
				a.openWebClientBrowser()
			}
		}
	})

	log.Println("System tray initializing. Use tray icon to control the application.")

	// Run system tray (blocks until Quit)
	a.trayMgr.Run()

	log.Println("SteelClock shutting down...")

	// Stop web editor if running
	if a.webEditor != nil {
		if err := a.webEditor.Stop(); err != nil {
			log.Printf("Failed to stop web editor: %v", err)
		}
	}

	a.lifecycle.Shutdown()
	log.Println("SteelClock stopped")
}

// createWebEditor creates and configures the web editor server
func (a *App) createWebEditor() {
	// Find schema path relative to config file location
	configPath := a.configMgr.GetConfigPath()
	if configPath == "" {
		log.Println("Web editor: No config path available, skipping web editor setup")
		return
	}

	// Schema is in profiles/schema/ relative to config file's directory
	configDir := filepath.Dir(configPath)
	schemaPath := filepath.Join(configDir, "profiles", "schema", "config.schema.json")

	// If config is in profiles/ directory, adjust path
	if filepath.Base(configDir) == "profiles" {
		schemaPath = filepath.Join(configDir, "schema", "config.schema.json")
	}

	// Create providers
	configProvider := NewConfigProviderAdapter(a.configMgr)
	var profileProvider webeditor.ProfileProvider
	var onProfileSwitch func(path string) error
	if a.configMgr.HasProfiles() {
		profileProvider = NewProfileProviderAdapter(a.configMgr.GetProfileManager())
		onProfileSwitch = a.switchProfileAndUpdateTray
	}

	// Create web editor server
	a.webEditor = webeditor.NewServer(configProvider, profileProvider, schemaPath, a.ReloadConfig, onProfileSwitch)

	// Set webclient override callback
	a.webEditor.SetPreviewOverrideCallback(a.SetWebClientOverride)

	// Wire up with tray manager
	a.trayMgr.SetWebEditor(a.webEditor)

	log.Println("Web editor: Configured")
}

// Start initializes and starts all components
func (a *App) Start() error {
	cfg, err := a.configMgr.Load()
	if err != nil {
		log.Printf("ERROR: Failed to load config: %v", err)
		return a.handleStartupError(err, nil)
	}

	if err := a.lifecycle.Start(cfg); err != nil {
		return a.handleStartupError(err, cfg)
	}

	// Update webclient provider if webclient backend is active
	a.updateWebClientProvider()

	return nil
}

// Stop stops all components gracefully (used during reload)
func (a *App) Stop() {
	a.lifecycle.Stop()
}

// updateWebClientProvider updates the web editor with the webclient provider if webclient backend is active
func (a *App) updateWebClientProvider() {
	if a.webEditor == nil {
		return
	}

	webClient := a.lifecycle.GetWebClient()
	if webClient != nil {
		adapter := NewWebClientProviderAdapter(webClient)
		a.webEditor.SetPreviewProvider(adapter)
		log.Println("WebClient provider connected to web editor")

		// Auto-open browser for webclient if web editor is running
		a.openWebClientBrowser()
	} else {
		a.webEditor.SetPreviewProvider(nil)
	}
}

// openWebClientBrowser opens the webclient page in browser if conditions are met
func (a *App) openWebClientBrowser() {
	// Only open once per session
	if a.webclientBrowserOpened {
		return
	}

	// Check if web editor is running
	if !a.webEditor.IsRunning() {
		log.Println("WebClient: web editor not running yet, will open browser later")
		return
	}

	// Check if webclient backend is active
	if a.lifecycle.GetWebClient() == nil {
		return
	}

	a.webclientBrowserOpened = true
	webclientURL := a.webEditor.GetURL() + "/preview"
	log.Printf("Opening webclient in browser: %s", webclientURL)
	if err := openBrowser(webclientURL); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// SetWebClientOverride enables or disables temporary webclient backend override.
// When enabled, switches to webclient backend regardless of config setting.
// When disabled, restores the original backend from config.
func (a *App) SetWebClientOverride(enable bool) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	if enable {
		return a.enableWebClientOverride()
	}
	return a.disableWebClientOverride()
}

// enableWebClientOverride switches to webclient backend temporarily
func (a *App) enableWebClientOverride() error {
	if a.webclientOverrideActive {
		log.Println("WebClient override already active")
		return nil
	}

	// Get current backend name
	currentBackend := a.lifecycle.GetCurrentBackend()
	if currentBackend == "webclient" {
		log.Println("WebClient override: already using webclient backend")
		return nil
	}

	log.Println("========================================")
	log.Printf("Enabling webclient override (current backend: %s)", currentBackend)

	// Store original backend name
	a.webclientOverrideOriginal = currentBackend

	// Get current config
	cfg := a.lifecycle.GetLastGoodConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Stop current compositor first (so it stops sending frames)
	log.Println("Stopping current compositor...")
	a.lifecycle.Stop()

	// Show "WEB CLIENT" message on hardware (now nothing will overwrite it)
	a.lifecycle.ShowWebClientModeMessage()

	// Create a modified config with webclient backend
	webclientCfg := *cfg
	webclientCfg.Backend = "webclient"

	// Start with webclient backend
	log.Println("Starting with webclient backend...")
	if err := a.lifecycle.Start(&webclientCfg); err != nil {
		log.Printf("ERROR: Failed to start webclient backend: %v", err)
		// Try to restore original backend
		log.Println("Attempting to restore original backend...")
		if restoreErr := a.lifecycle.Start(cfg); restoreErr != nil {
			log.Printf("ERROR: Failed to restore original backend: %v", restoreErr)
		}
		return fmt.Errorf("failed to enable webclient override: %w", err)
	}

	a.webclientOverrideActive = true

	// Update webclient provider
	a.updateWebClientProviderUnlocked()

	log.Println("WebClient override enabled")
	log.Println("========================================")
	return nil
}

// disableWebClientOverride restores the original backend
func (a *App) disableWebClientOverride() error {
	if !a.webclientOverrideActive {
		log.Println("WebClient override not active")
		return nil
	}

	log.Println("========================================")
	log.Printf("Disabling webclient override (restoring backend: %s)", a.webclientOverrideOriginal)

	// Get current config (with webclient backend)
	cfg := a.lifecycle.GetLastGoodConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Create config with original backend
	originalCfg := *cfg
	originalCfg.Backend = a.webclientOverrideOriginal

	// Stop webclient compositor
	log.Println("Stopping webclient compositor...")
	a.lifecycle.Stop()

	// Start with original backend
	log.Printf("Starting with original backend: %s", a.webclientOverrideOriginal)
	if err := a.lifecycle.Start(&originalCfg); err != nil {
		log.Printf("ERROR: Failed to restore original backend: %v", err)
		return fmt.Errorf("failed to disable webclient override: %w", err)
	}

	a.webclientOverrideActive = false
	a.webclientOverrideOriginal = ""

	// Clear webclient provider since we're no longer using webclient backend
	if a.webEditor != nil {
		a.webEditor.SetPreviewProvider(nil)
	}

	log.Println("WebClient override disabled, original backend restored")
	log.Println("========================================")
	return nil
}

// updateWebClientProviderUnlocked updates webclient provider without acquiring configMu
// (caller must hold configMu)
func (a *App) updateWebClientProviderUnlocked() {
	if a.webEditor == nil {
		return
	}

	webClient := a.lifecycle.GetWebClient()
	if webClient != nil {
		adapter := NewWebClientProviderAdapter(webClient)
		a.webEditor.SetPreviewProvider(adapter)
		log.Println("WebClient provider connected to web editor")
	} else {
		a.webEditor.SetPreviewProvider(nil)
	}
}

// ReloadConfig reloads configuration and restarts components.
// This operation is serialized with other config operations via configMu.
func (a *App) ReloadConfig() error {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	log.Println("========================================")
	log.Println("Reloading configuration...")

	newCfg, fileInfo, err := a.configMgr.Reload()
	if fileInfo != nil {
		log.Printf("Config file: %s", fileInfo.Path)
		log.Printf("Absolute path: %s", fileInfo.AbsolutePath)
		log.Printf("Config file last modified: %s", fileInfo.ModTime)
	}

	if err != nil {
		if fileInfo == nil {
			// Could not access file at all
			log.Printf("ERROR: Cannot access config file: %v", err)
			log.Println("Keeping current configuration running")
			return err
		}

		// File accessible but config invalid
		log.Printf("ERROR: Config validation failed: %v", err)
		log.Println("Stopping current instance and showing error...")

		a.lifecycle.Stop()
		time.Sleep(1 * time.Second)

		return a.handleStartupError(err, nil)
	}

	log.Println("New config validated successfully")
	log.Printf("Loaded config: %s (%s) with %d widgets", newCfg.GameName, newCfg.GameDisplayName, len(newCfg.Widgets))

	log.Println("Stopping current instance...")
	a.lifecycle.Stop()

	log.Println("Waiting for GameSense API to settle...")
	time.Sleep(2 * time.Second)

	log.Println("Starting with new config...")
	if err := a.lifecycle.Start(newCfg); err != nil {
		log.Printf("ERROR: Failed to start with new config: %v", err)
		time.Sleep(1 * time.Second)
		return a.handleStartupError(err, newCfg)
	}

	// Update webclient provider if webclient backend is active
	a.updateWebClientProvider()

	log.Println("Configuration reloaded successfully!")
	log.Printf("Running with: %s (%s)", newCfg.GameName, newCfg.GameDisplayName)
	log.Println("========================================")
	return nil
}

// SwitchProfile switches to a different configuration profile.
// This operation is serialized with other config operations via configMu.
func (a *App) SwitchProfile(path string) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	if !a.configMgr.HasProfiles() {
		return fmt.Errorf("profile manager not available")
	}

	log.Println("========================================")
	log.Printf("Switching to profile: %s", path)

	// Load new config via ConfigManager
	newCfg, err := a.configMgr.SwitchProfile(path)
	if err != nil {
		log.Printf("ERROR: Failed to switch profile: %v", err)
		log.Println("Stopping current instance and showing error...")
		a.lifecycle.Stop()
		return a.handleStartupError(err, nil)
	}

	log.Printf("Loaded config: %s (%s) with %d widgets", newCfg.GameName, newCfg.GameDisplayName, len(newCfg.Widgets))

	// Get profile name for transition banner
	profileName := a.configMgr.GetActiveProfileName()
	if profileName == "" {
		profileName = "Unknown"
	}

	// Stop compositor first to free the display
	log.Println("Stopping current instance...")
	a.lifecycle.Stop()

	// Show transition banner
	a.lifecycle.ShowTransitionBanner(profileName)

	log.Println("Waiting for GameSense API to settle...")
	time.Sleep(500 * time.Millisecond)

	// Start with new config
	log.Println("Starting with new profile...")
	if err := a.lifecycle.Start(newCfg); err != nil {
		log.Printf("ERROR: Failed to start with new profile: %v", err)
		time.Sleep(1 * time.Second)
		return a.handleStartupError(err, newCfg)
	}

	// Update webclient provider if webclient backend is active
	a.updateWebClientProvider()

	log.Printf("Profile switched successfully to: %s", profileName)
	log.Println("========================================")
	return nil
}

// switchProfileAndUpdateTray switches profile and updates the tray menu
// This is used by the web editor to ensure UI consistency
func (a *App) switchProfileAndUpdateTray(path string) error {
	if err := a.SwitchProfile(path); err != nil {
		return err
	}

	// Update tray menu to reflect the new active profile
	if a.trayMgr != nil {
		a.trayMgr.UpdateActiveProfile()
	}

	return nil
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

	// Determine display dimensions
	width, height := a.lifecycle.GetDisplayDimensions()
	if cfg != nil {
		width = cfg.Display.Width
		height = cfg.Display.Height
	}

	// Determine error message
	errorMsg := "CONFIG"
	var noWidgetsErr *NoWidgetsError
	if errors.As(err, &noWidgetsErr) {
		errorMsg = "NO WIDGETS"
	}

	log.Println("Displaying error on OLED screen...")
	if dispErr := a.lifecycle.StartErrorDisplay(errorMsg, width, height); dispErr != nil {
		log.Printf("ERROR: Failed to show error display: %v", dispErr)
		return fmt.Errorf("startup failed and error display failed: %w", dispErr)
	}

	return err
}
