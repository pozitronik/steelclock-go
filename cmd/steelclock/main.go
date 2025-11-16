package main

import (
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
		fmt.Fprintf(os.Stderr, "Warning: Failed to get executable path: %v\n", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	// Create log file with timestamp
	logFileName := filepath.Join(exeDir, "steelclock.log")

	// Open log file (append mode)
	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to open log file: %v\n", err)
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
		logFile.Close()
	}
}

// startApp initializes and starts all components
func startApp() error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return startAppWithConfig(cfg)
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

	// Create or reuse GameSense client
	var err error
	if client == nil {
		// First time startup - create new client and register
		client, err = gamesense.NewClient(cfg.GameName, cfg.GameDisplayName)
		if err != nil {
			return fmt.Errorf("failed to create GameSense client: %w", err)
		}

		log.Println("GameSense client created")

		// Register game
		if err := client.RegisterGame("Custom"); err != nil {
			return fmt.Errorf("failed to register game: %w", err)
		}

		// Bind screen event
		if err := client.BindScreenEvent("STEELCLOCK_DISPLAY", "screened-128x40"); err != nil {
			return fmt.Errorf("failed to bind screen event: %w", err)
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
		log.Println("Keeping current configuration running")
		return fmt.Errorf("config validation failed: %w", err)
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

		// Try to recover with last good config
		if lastGoodConfig != nil {
			log.Println("Attempting recovery with last good config...")
			log.Println("Waiting before recovery attempt...")
			time.Sleep(2 * time.Second)

			// Try to restart with old config (without loading from file)
			if recoverErr := startAppWithConfig(lastGoodConfig); recoverErr != nil {
				log.Printf("CRITICAL: Recovery failed: %v", recoverErr)
				log.Println("Application is stopped. Please check config and restart manually.")
				return fmt.Errorf("reload failed and recovery failed: %w", recoverErr)
			}

			log.Println("Successfully recovered with previous config")
			log.Println("Please fix config.json and try reload again")
			return fmt.Errorf("new config failed, reverted to previous: %w", err)
		}

		log.Println("CRITICAL: No backup config available for recovery")
		log.Println("Application is stopped. Please check config and restart manually.")
		return fmt.Errorf("reload failed and no backup available: %w", err)
	}

	log.Println("Configuration reloaded successfully!")
	log.Printf("Running with: %s (%s)", newCfg.GameName, newCfg.GameDisplayName)
	log.Println("========================================")
	return nil
}
