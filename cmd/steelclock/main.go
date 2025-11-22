package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

var (
	comp           *compositor.Compositor
	client         *gamesense.Client
	configPath     string
	logFile        *os.File
	mu             sync.Mutex
	lastGoodConfig *config.Config // Backup of last working config
)

const (
	// GameSense API constants
	defaultGameName    = "SteelClock"
	defaultGameDisplay = "SteelClock"
	eventName          = "STEELCLOCK_DISPLAY"
	deviceType         = "screened-128x40"
	developerName      = "Pozitronik"
	errorGameName      = "STEELCLOCK_ERROR"
	errorGameDisplay   = "SteelClock Error"
)

// BackendUnavailableError indicates SteelSeries GG backend is not available
type BackendUnavailableError struct {
	Err error
}

func (e *BackendUnavailableError) Error() string {
	return fmt.Sprintf("SteelSeries backend unavailable: %v", e.Err)
}

func (e *BackendUnavailableError) Unwrap() error {
	return e.Err
}

// NoWidgetsError indicates that no widgets are enabled in the configuration
type NoWidgetsError struct{}

func (e *NoWidgetsError) Error() string {
	return "no widgets enabled in configuration"
}

func main() {
	// Parse command line flags
	configPathFlag := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	configPath = *configPathFlag

	// Setup logging to file
	setupLogging()
	defer closeLogging()

	log.Println("========================================")
	log.Println("SteelClock starting...")
	log.Printf("Config: %s", configPath)
	log.Println("========================================")

	// Start the application
	if err := startApp(); err != nil {
		log.Println("========================================")
		log.Println("FATAL ERROR DURING STARTUP")
		log.Printf("Error: %v", err)
		log.Println("========================================")
		log.Println("")
		log.Println("Application failed to start. Please check the error above and fix config.json")
		log.Println("")

		// Ensure log is flushed before exit
		if logFile != nil {
			_ = logFile.Sync() // Ignore error - we're exiting anyway
		}

		os.Exit(1)
	}

	// Create tray manager
	trayMgr := tray.NewManager(configPath, reloadConfig, stopApp)

	log.Println("System tray initialized. Use tray icon to control the application.")

	// Run system tray (blocks until Quit)
	trayMgr.Run()

	log.Println("SteelClock shutting down...")
	stopAppAndWait()
	log.Println("SteelClock stopped")
}

// setupLogging configures logging to file
func setupLogging() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to get executable path: %v\n", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	// Create log file with timestamp
	logFileName := filepath.Join(exeDir, "steelclock.log")

	// Open log file (append mode)
	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to open log file: %v\n", err)
		return
	}

	// Write to both file and stderr (for debugging)
	multiWriter := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// closeLogging closes the log file
func closeLogging() {
	if logFile != nil {
		_ = logFile.Close()
	}
}

// startApp initializes and starts all components
func startApp() error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("ERROR: Failed to load config: %v", err)
		return handleStartupError(err, nil)
	}

	// Start with loaded config
	if err := startAppWithConfig(cfg); err != nil {
		return handleStartupError(err, cfg)
	}

	return nil
}

// startAppWithConfig initializes and starts all components with a provided config
func startAppWithConfig(cfg *config.Config) error {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	// Set bundled font URL if configured
	if cfg.BundledFontURL != "" {
		bitmap.SetBundledFontURL(cfg.BundledFontURL)
		log.Printf("Using custom bundled font URL: %s", cfg.BundledFontURL)
	}

	// Set bundled WAD URL if configured
	if cfg.BundledWadURL != "" {
		widget.SetBundledWadURL(cfg.BundledWadURL)
		log.Printf("Using custom bundled WAD URL: %s", cfg.BundledWadURL)
	}

	// Check if we need to recreate the client due to GameName change
	if client != nil && client.GameName() != cfg.GameName {
		log.Printf("GameName changed from %s to %s, recreating client...", client.GameName(), cfg.GameName)
		client = nil // Force recreation with new game name
	}

	// Create or reuse GameSense client
	var err error
	if client == nil {
		// First time startup - create new client and register
		client, err = gamesense.NewClient(cfg.GameName, cfg.GameDisplayName)
		if err != nil {
			log.Printf("ERROR: Failed to create GameSense client: %v", err)
			log.Println("SteelSeries GG may not be running or GameSense API is unavailable")
			// Return special error for backend unavailable
			return &BackendUnavailableError{Err: err}
		}

		log.Println("GameSense client created")

		// Register game
		if err := client.RegisterGame(developerName); err != nil {
			log.Printf("ERROR: Failed to register game: %v", err)
			client = nil // Clear client on registration failure
			return &BackendUnavailableError{Err: err}
		}

		// Bind screen event
		if err := client.BindScreenEvent(eventName, deviceType); err != nil {
			log.Printf("ERROR: Failed to bind screen event: %v", err)
			client = nil // Clear client on bind failure
			return &BackendUnavailableError{Err: err}
		}
	} else {
		// Reload - client already exists and game is registered
		log.Println("Reusing existing GameSense client (already registered)")
	}

	// Create widgets
	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		return fmt.Errorf("failed to create widgets: %w", err)
	}

	// Check if we have any widgets
	if len(widgets) == 0 {
		log.Println("WARNING: No widgets enabled in configuration")
		return &NoWidgetsError{}
	}

	log.Printf("Created %d widgets", len(widgets))
	for i := range widgets {
		log.Printf("  Widget %d: %s (type: %s)", i+1, cfg.Widgets[i].ID, cfg.Widgets[i].Type)
	}

	// Create layout manager
	layoutMgr := layout.NewManager(cfg.Display, widgets)

	// Create compositor
	comp = compositor.NewCompositor(client, layoutMgr, widgets, cfg)

	// Start compositor
	if err := comp.Start(); err != nil {
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	// Save as last good config
	lastGoodConfig = cfg

	log.Println("SteelClock started successfully")
	return nil
}

// stopApp stops all components gracefully
// During reload, we don't unregister to avoid blocking
func stopApp() {
	stopAppInternal(false)
}

// stopAppAndWait stops all components and optionally unregisters based on config
// Used during final shutdown
func stopAppAndWait() {
	stopAppInternal(true)
}

// handleStartupError handles errors during startup/reload and shows error display if appropriate
// Returns the original error or a wrapped error if error display fails
func handleStartupError(err error, cfg *config.Config) error {
	// Check for backend unavailable error
	var backendErr *BackendUnavailableError
	if errors.As(err, &backendErr) {
		log.Println("========================================")
		log.Println("CRITICAL: SteelSeries GG is not running")
		log.Println("Please start SteelSeries GG and try again")
		log.Println("========================================")
		return backendErr
	}

	// Determine display dimensions
	displayWidth := 128
	displayHeight := 40
	if cfg != nil {
		displayWidth = cfg.Display.Width
		displayHeight = cfg.Display.Height
	} else {
		// Read lastGoodConfig under lock to avoid data race
		mu.Lock()
		if lastGoodConfig != nil {
			displayWidth = lastGoodConfig.Display.Width
			displayHeight = lastGoodConfig.Display.Height
		}
		mu.Unlock()
	}

	// Determine error message based on error type
	errorMsg := "CONFIG"
	var noWidgetsErr *NoWidgetsError
	if errors.As(err, &noWidgetsErr) {
		errorMsg = "NO WIDGETS"
	}

	// Show error display
	log.Println("Displaying error on OLED screen...")
	if dispErr := startWithErrorDisplay(errorMsg, displayWidth, displayHeight); dispErr != nil {
		log.Printf("ERROR: Failed to show error display: %v", dispErr)
		return fmt.Errorf("startup failed and error display failed: %w", dispErr)
	}

	return err
}

// stopAppInternal stops all components
func stopAppInternal(isFinalShutdown bool) {
	mu.Lock()
	defer mu.Unlock()

	// Stop compositor first
	if comp != nil {
		comp.Stop()
		comp = nil
	}

	// Clean up GameSense registration
	if client != nil {
		// Check if we should unregister
		shouldUnregister := lastGoodConfig != nil && lastGoodConfig.UnregisterOnExit

		if shouldUnregister && isFinalShutdown {
			// Final shutdown - try to unregister if configured
			log.Println("Unregistering from GameSense (unregister_on_exit=true)...")
			if err := client.RemoveGame(); err != nil {
				log.Printf("Warning: Failed to unregister game: %v", err)
			} else {
				log.Println("Successfully unregistered from GameSense")
			}
			client = nil // Clear client after unregistering
		} else if !isFinalShutdown {
			// During reload - keep client and registration alive
			log.Println("Stopping compositor (keeping GameSense client and registration)")
			// DON'T set client = nil here - we'll reuse it
		} else {
			// Final shutdown with unregister_on_exit=false
			log.Println("Shutting down (keeping GameSense registration, unregister_on_exit=false)")
			client = nil // Clear client reference but don't unregister
		}
	}
}

// reloadConfig reloads configuration and restarts components
func reloadConfig() error {
	log.Println("========================================")
	log.Println("Reloading configuration...")
	log.Printf("Config file: %s", configPath)

	// Get absolute path for clarity
	absPath, _ := filepath.Abs(configPath)
	log.Printf("Absolute path: %s", absPath)

	// Check if file exists and get modification time
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		log.Printf("ERROR: Cannot access config file: %v", err)
		log.Println("Keeping current configuration running")
		return fmt.Errorf("cannot access config file: %w", err)
	}
	log.Printf("Config file last modified: %s", fileInfo.ModTime().Format("2006-01-02 15:04:05"))

	// Validate new config first (before stopping anything)
	newCfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("ERROR: Config validation failed: %v", err)
		log.Println("Stopping current instance and showing error...")

		// Stop current app
		stopApp()
		time.Sleep(1 * time.Second) // Wait for cleanup

		// Show error display using common handler
		return handleStartupError(err, nil)
	}

	log.Println("New config validated successfully")
	log.Printf("Loaded config: %s (%s) with %d widgets", newCfg.GameName, newCfg.GameDisplayName, len(newCfg.Widgets))

	// Stop current app
	log.Println("Stopping current instance...")
	stopApp()

	// Wait for GameSense API to settle
	// The API needs a moment to process cleanup before accepting new registrations
	log.Println("Waiting for GameSense API to settle...")
	time.Sleep(2 * time.Second)

	// Try to start with new config
	log.Println("Starting with new config...")
	if err := startAppWithConfig(newCfg); err != nil {
		log.Printf("ERROR: Failed to start with new config: %v", err)
		time.Sleep(1 * time.Second)
		return handleStartupError(err, newCfg)
	}

	log.Println("Configuration reloaded successfully!")
	log.Printf("Running with: %s (%s)", newCfg.GameName, newCfg.GameDisplayName)
	log.Println("========================================")
	return nil
}

// startWithErrorDisplay creates and runs an error display widget
func startWithErrorDisplay(message string, width, height int) error {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Starting error display: %s", message)

	// Create temporary GameSense client for error display
	// DO NOT reuse the global client to avoid polluting it with error state
	errorClient, err := gamesense.NewClient(errorGameName, errorGameDisplay)
	if err != nil {
		log.Printf("ERROR: Failed to create GameSense client for error display: %v", err)
		return fmt.Errorf("failed to create GameSense client: %w", err)
	}

	// Register game
	if err := errorClient.RegisterGame(developerName); err != nil {
		log.Printf("ERROR: Failed to register game for error display: %v", err)
		return fmt.Errorf("failed to register game: %w", err)
	}

	// Bind screen event
	if err := errorClient.BindScreenEvent(eventName, deviceType); err != nil {
		log.Printf("ERROR: Failed to bind screen event for error display: %v", err)
		return fmt.Errorf("failed to bind screen event: %w", err)
	}

	// Create error widget
	errorWidget := widget.NewErrorWidget(width, height, message)
	widgets := []widget.Widget{errorWidget}

	// Create simple display config
	displayCfg := config.DisplayConfig{
		Width:           width,
		Height:          height,
		BackgroundColor: 0,
	}

	// Create default config for compositor
	errorCfg := &config.Config{
		GameName:        errorGameName,
		GameDisplayName: errorGameDisplay,
		RefreshRateMs:   500, // Flash at 500ms intervals
		Display:         displayCfg,
		Widgets:         []config.WidgetConfig{},
	}

	// Create layout manager
	layoutMgr := layout.NewManager(displayCfg, widgets)

	// Create compositor with temporary error client
	comp = compositor.NewCompositor(errorClient, layoutMgr, widgets, errorCfg)

	// Start compositor
	if err := comp.Start(); err != nil {
		log.Printf("ERROR: Failed to start compositor for error display: %v", err)
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	log.Println("Error display started - screen will flash error message")
	log.Println("Fix the configuration file and reload to continue")

	return nil
}
