package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/driver"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/tray"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// FIXME: Global mutable state makes testing difficult and creates tight coupling.
// Consider encapsulating this state in an App struct with methods like:
//   type App struct { comp, client, trayMgr, ... }
//   func NewApp() *App
//   func (a *App) Start() error
//   func (a *App) Stop()
//   func (a *App) ReloadConfig() error
// This would allow for better testability and dependency injection.
var (
	comp           *compositor.Compositor
	client         gamesense.API // Can be *gamesense.Client or *driver.Client
	trayMgr        *tray.Manager
	configPath     string
	logFile        *os.File
	mu             sync.Mutex
	lastGoodConfig *config.Config // Backup of last working config
	retryCancel    chan struct{}  // Channel to cancel retry goroutine
	currentBackend string         // Current backend type: "gamesense" or "direct"
)

const (
	// GameSense API constants
	eventName     = "STEELCLOCK_DISPLAY"
	deviceType    = "screened-128x40"
	developerName = "Pozitronik"
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

	// Initialize retry cancel channel
	retryCancel = make(chan struct{})

	// Create tray manager first
	trayMgr = tray.NewManager(configPath, reloadConfig, stopApp)

	// Set callback to run when tray is ready
	trayMgr.OnReady(func() {
		// Start the application after tray is initialized
		if err := startApp(); err != nil {
			log.Println("========================================")
			log.Println("STARTUP ERROR")
			log.Printf("Error: %v", err)
			log.Println("========================================")
			log.Println("")

			// Provide context-specific guidance
			var backendErr *BackendUnavailableError
			if errors.As(err, &backendErr) {
				log.Println("Cannot connect to display backend.")
				log.Println("This usually happens when:")
				log.Println("  - Device is not connected")
				log.Println("  - SteelSeries GG is not running (for gamesense backend)")
				log.Println("  - Backend is still cleaning up from previous instance")
				log.Println("")
				log.Println("The application will continue running. Use 'Reload Config' to retry.")
			} else {
				log.Println("Application failed to start. Please check the error above and fix config.json")
				log.Println("Use 'Reload Config' to retry after fixing the issue.")
			}
			log.Println("")

			// Show error notification
			var noWidgetsErr *NoWidgetsError
			if errors.As(err, &noWidgetsErr) {
				tray.ShowNotification("SteelClock Error", "No widgets enabled in configuration. Please check config.json")
			} else if errors.As(err, &backendErr) {
				tray.ShowNotification("SteelClock Connection Error", "Cannot connect to display. Check device connection and try 'Reload Config'.")
			} else {
				tray.ShowNotification("SteelClock Configuration Error", "Failed to load configuration. Please check config.json for errors.")
			}
		}
	})

	log.Println("System tray initializing. Use tray icon to control the application.")

	// Run system tray (blocks until Quit)
	trayMgr.Run()

	log.Println("SteelClock shutting down...")

	// Cancel any ongoing retry
	close(retryCancel)

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

// bindEventWithRetry attempts to bind the screen event with exponential backoff
// Returns nil on success, error if all retries failed
func bindEventWithRetry(maxAttempts int) error {
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Calculate exponential backoff delay
			delay := time.Duration(float64(baseDelay) * float64(uint(1)<<uint(attempt-2)))
			if delay > maxDelay {
				delay = maxDelay
			}

			log.Printf("Retrying bind in %v... (attempt %d/%d)", delay, attempt, maxAttempts)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-retryCancel:
				log.Println("Bind retry cancelled")
				return fmt.Errorf("bind retry cancelled")
			}
		}

		log.Printf("Attempting to bind screen event (attempt %d/%d)...", attempt, maxAttempts)
		if err := client.BindScreenEvent(eventName, deviceType); err != nil {
			log.Printf("ERROR: Failed to bind screen event: %v", err)
			if attempt == maxAttempts {
				return &BackendUnavailableError{Err: err}
			}
			// Continue to next attempt
			continue
		}

		// Success
		log.Println("Screen event bound successfully")
		return nil
	}

	return &BackendUnavailableError{Err: fmt.Errorf("failed to bind after %d attempts", maxAttempts)}
}

// startAppWithConfig initializes and starts all components with a provided config
func startAppWithConfig(cfg *config.Config) error {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	// Set bundled font URL if configured
	if cfg.BundledFontURL != nil && *cfg.BundledFontURL != "" {
		bitmap.SetBundledFontURL(*cfg.BundledFontURL)
		log.Printf("Using custom bundled font URL: %s", *cfg.BundledFontURL)
	}

	// Check if we need to recreate the client due to backend or GameName change
	needNewClient := client == nil
	if client != nil {
		// Check for GameName change (only relevant for gamesense backend)
		if gsClient, ok := client.(*gamesense.Client); ok {
			if gsClient.GameName() != cfg.GameName {
				log.Printf("GameName changed from %s to %s, recreating client...", gsClient.GameName(), cfg.GameName)
				needNewClient = true
			}
		}
		// Check for backend change
		if currentBackend != cfg.Backend {
			log.Printf("Backend changed from %s to %s, recreating client...", currentBackend, cfg.Backend)
			needNewClient = true
		}
	}

	// Create or reuse client based on backend setting
	var err error
	if needNewClient {
		// Close existing client if any
		if client != nil {
			_ = client.RemoveGame()
			client = nil
		}

		// Create client based on backend
		client, err = createBackendClient(cfg)
		if err != nil {
			return err
		}

		// Track actual backend used (important for "any" mode)
		if _, ok := client.(*gamesense.Client); ok {
			currentBackend = "gamesense"
		} else {
			currentBackend = "direct"
		}

		// For gamesense backend, we need to bind screen event
		if currentBackend == "gamesense" {
			// Bind screen event with retry
			if err := bindEventWithRetry(10); err != nil {
				log.Printf("ERROR: Failed to bind screen event after retries: %v", err)
				client = nil // Clear client on bind failure
				return err
			}
		}
	} else {
		// Reload - client already exists
		log.Printf("Reusing existing %s client", currentBackend)
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

	// Set backend failure callback for "any" mode
	if cfg.Backend == "any" {
		comp.OnBackendFailure = func() {
			handleBackendFailure(cfg)
		}
	}

	// Start compositor
	if err := comp.Start(); err != nil {
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	// Save as last good config
	lastGoodConfig = cfg

	log.Println("SteelClock started successfully")
	return nil
}

// createBackendClient creates the appropriate client based on backend configuration
func createBackendClient(cfg *config.Config) (gamesense.API, error) {
	switch cfg.Backend {
	case "gamesense":
		return createGameSenseClient(cfg)

	case "direct":
		return createDirectClient(cfg)

	case "any":
		// Try gamesense first, then fallback to direct
		log.Println("Backend 'any': trying GameSense first...")
		gsClient, err := createGameSenseClient(cfg)
		if err == nil {
			return gsClient, nil
		}
		log.Printf("GameSense backend failed: %v", err)
		log.Println("Backend 'any': falling back to direct driver...")
		return createDirectClient(cfg)

	default:
		// Should not happen due to validation, but fallback to gamesense
		return createGameSenseClient(cfg)
	}
}

// createGameSenseClient creates a GameSense API client
func createGameSenseClient(cfg *config.Config) (gamesense.API, error) {
	client, err := gamesense.NewClient(cfg.GameName, cfg.GameDisplayName)
	if err != nil {
		log.Printf("ERROR: Failed to create GameSense client: %v", err)
		log.Println("SteelSeries GG may not be running or GameSense API is unavailable")
		return nil, &BackendUnavailableError{Err: err}
	}

	log.Println("GameSense client created")

	// Register game
	if err := client.RegisterGame(developerName, cfg.DeinitializeTimerMs); err != nil {
		log.Printf("ERROR: Failed to register game: %v", err)
		return nil, &BackendUnavailableError{Err: err}
	}

	return client, nil
}

// createDirectClient creates a direct USB HID driver client
func createDirectClient(cfg *config.Config) (gamesense.API, error) {
	// Parse VID/PID from config if specified
	var vid, pid uint16
	if cfg.DirectDriver != nil {
		if cfg.DirectDriver.VID != "" {
			v, err := strconv.ParseUint(cfg.DirectDriver.VID, 16, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid VID '%s': %w", cfg.DirectDriver.VID, err)
			}
			vid = uint16(v)
		}
		if cfg.DirectDriver.PID != "" {
			p, err := strconv.ParseUint(cfg.DirectDriver.PID, 16, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid PID '%s': %w", cfg.DirectDriver.PID, err)
			}
			pid = uint16(p)
		}
	}

	// Get interface from config
	iface := "mi_01"
	if cfg.DirectDriver != nil && cfg.DirectDriver.Interface != "" {
		iface = cfg.DirectDriver.Interface
	}

	// Create driver config
	driverCfg := driver.Config{
		VID:       vid,
		PID:       pid,
		Interface: iface,
		Width:     cfg.Display.Width,
		Height:    cfg.Display.Height,
	}

	// Create driver client
	client, err := driver.NewClient(driverCfg)
	if err != nil {
		log.Printf("ERROR: Failed to create direct driver client: %v", err)
		return nil, &BackendUnavailableError{Err: err}
	}

	log.Printf("Direct driver client created (backend: direct, VID: %04X, PID: %04X)",
		driverCfg.VID, driverCfg.PID)

	return client, nil
}

// handleBackendFailure attempts to switch to alternative backend when current backend fails
// Only called for "any" backend mode
func handleBackendFailure(cfg *config.Config) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("========================================")
	log.Printf("Backend failure detected (current: %s)", currentBackend)
	log.Println("Attempting to switch to alternative backend...")

	// Stop current compositor
	if comp != nil {
		comp.Stop()
		comp = nil
	}

	// Determine alternative backend
	var newClient gamesense.API
	var newBackend string
	var err error

	if currentBackend == "gamesense" {
		// Try direct driver
		log.Println("Trying direct driver backend...")
		newClient, err = createDirectClient(cfg)
		newBackend = "direct"
	} else {
		// Try gamesense
		log.Println("Trying GameSense backend...")
		newClient, err = createGameSenseClient(cfg)
		newBackend = "gamesense"
	}

	if err != nil {
		log.Printf("ERROR: Failed to switch to %s backend: %v", newBackend, err)
		log.Println("Will retry on next heartbeat cycle...")
		return
	}

	// Update client and backend
	client = newClient
	currentBackend = newBackend
	log.Printf("Successfully switched to %s backend", currentBackend)

	// Recreate widgets and compositor
	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		log.Printf("ERROR: Failed to recreate widgets: %v", err)
		return
	}

	layoutMgr := layout.NewManager(cfg.Display, widgets)
	comp = compositor.NewCompositor(client, layoutMgr, widgets, cfg)

	// Set callback again for future failures
	comp.OnBackendFailure = func() {
		handleBackendFailure(cfg)
	}

	if err := comp.Start(); err != nil {
		log.Printf("ERROR: Failed to start compositor with new backend: %v", err)
		return
	}

	log.Println("Successfully recovered with alternative backend")
	log.Println("========================================")
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
		log.Println("CRITICAL: Cannot connect to display backend")
		log.Println("This may indicate:")
		log.Println("  - Device is not connected")
		log.Println("  - SteelSeries GG is not running (for gamesense backend)")
		log.Println("  - Backend is still cleaning up from previous instance")
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

	// Try to use existing client first (works with both gamesense and direct backends)
	errorClient := client
	if errorClient == nil {
		log.Println("No existing client, creating one for error display...")

		// Try to create a client based on last good config or defaults
		var err error
		if lastGoodConfig != nil {
			errorClient, err = createBackendClient(lastGoodConfig)
		} else {
			// Fallback: try GameSense with defaults
			errorClient, err = gamesense.NewClient(config.DefaultGameName, config.DefaultGameDisplay)
			if err == nil {
				err = errorClient.RegisterGame(developerName, 0)
			}
		}
		if err != nil {
			log.Printf("ERROR: Failed to create client for error display: %v", err)
			return fmt.Errorf("failed to create client: %w", err)
		}
	}

	// Register and bind (no-ops for direct driver, required for gamesense)
	if err := errorClient.RegisterGame(developerName, 0); err != nil {
		log.Printf("ERROR: Failed to register game for error display: %v", err)
		return fmt.Errorf("failed to register game: %w", err)
	}

	if err := errorClient.BindScreenEvent(eventName, deviceType); err != nil {
		log.Printf("ERROR: Failed to bind screen event for error display: %v", err)
		return fmt.Errorf("failed to bind screen event: %w", err)
	}

	// Create error widget
	errorWidget := widget.NewErrorWidget(width, height, message)
	widgets := []widget.Widget{errorWidget}

	// Create simple display config
	displayCfg := config.DisplayConfig{
		Width:      width,
		Height:     height,
		Background: 0,
	}

	// Create default config for compositor
	errorCfg := &config.Config{
		GameName:        config.DefaultGameName,
		GameDisplayName: config.DefaultGameDisplay,
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
