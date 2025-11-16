package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/pozitronik/steelclock/internal/compositor"
	"github.com/pozitronik/steelclock/internal/config"
	"github.com/pozitronik/steelclock/internal/gamesense"
	"github.com/pozitronik/steelclock/internal/layout"
	"github.com/pozitronik/steelclock/internal/tray"
	"github.com/pozitronik/steelclock/internal/widget"
)

var (
	comp       *compositor.Compositor
	client     *gamesense.Client
	configPath string
	logFile    *os.File
	mu         sync.Mutex
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
	log.Printf("Version: 1.0.0")
	log.Printf("Config: %s", configPath)
	log.Println("========================================")

	// Start the application
	if err := startApp(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	// Create tray manager
	trayMgr := tray.NewManager(configPath, reloadConfig, stopApp)

	log.Println("System tray initialized. Use tray icon to control the application.")

	// Run system tray (blocks until Quit)
	trayMgr.Run()

	log.Println("SteelClock shutting down...")
	stopApp()
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
	mu.Lock()
	defer mu.Unlock()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	// Create GameSense client
	client, err = gamesense.NewClient(cfg.GameName, cfg.GameDisplayName)
	if err != nil {
		return err
	}

	log.Println("GameSense client created")

	// Register game
	if err := client.RegisterGame("Custom"); err != nil {
		return err
	}

	// Bind screen event
	if err := client.BindScreenEvent("STEELCLOCK_DISPLAY", "screened-128x40"); err != nil {
		return err
	}

	// Create widgets
	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		return err
	}

	log.Printf("Created %d widgets", len(widgets))

	// Create layout manager
	layoutMgr := layout.NewManager(cfg.Display, widgets)

	// Create compositor
	comp = compositor.NewCompositor(client, layoutMgr, widgets, cfg)

	// Start compositor
	if err := comp.Start(); err != nil {
		return err
	}

	log.Println("SteelClock started successfully")
	return nil
}

// stopApp stops all components gracefully
func stopApp() {
	mu.Lock()
	defer mu.Unlock()

	if comp != nil {
		comp.Stop()
		comp = nil
	}

	if client != nil {
		if err := client.RemoveGame(); err != nil {
			log.Printf("Failed to remove game: %v", err)
		}
		client = nil
	}
}

// reloadConfig reloads configuration and restarts components
func reloadConfig() error {
	log.Println("Reloading configuration...")

	// Validate config first
	if err := tray.ValidateConfig(configPath); err != nil {
		log.Printf("Config validation failed: %v", err)
		return err
	}

	// Stop current app
	stopApp()

	// Start with new config
	if err := startApp(); err != nil {
		log.Printf("Failed to restart with new config: %v", err)
		return err
	}

	log.Println("Configuration reloaded successfully")
	return nil
}
