package app

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
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

	// Wire up with tray manager
	a.trayMgr.SetWebEditor(a.webEditor)

	log.Println("Web editor: Configured (will start on first 'Edit Config' click)")
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

	return nil
}

// Stop stops all components gracefully (used during reload)
func (a *App) Stop() {
	a.lifecycle.Stop()
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
