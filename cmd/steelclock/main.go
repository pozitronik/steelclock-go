package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	mu         sync.Mutex
)

func main() {
	// Parse command line flags
	configPathFlag := flag.String("config", "config.json", "Path to configuration file")
	consoleMode := flag.Bool("console", false, "Run in console mode (no system tray)")
	flag.Parse()

	configPath = *configPathFlag

	if *consoleMode {
		runConsoleMode()
	} else {
		runHeadlessMode()
	}
}

// runConsoleMode runs the application in console mode
func runConsoleMode() {
	log.Println("SteelClock starting (console mode)...")

	if err := startApp(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("SteelClock running. Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigChan

	log.Println("Shutting down...")
	stopApp()
	log.Println("SteelClock stopped")
}

// runHeadlessMode runs the application with system tray
func runHeadlessMode() {
	log.Println("SteelClock starting (headless mode)...")

	// Start the application
	if err := startApp(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	// Create tray manager
	trayMgr := tray.NewManager(configPath, reloadConfig, stopApp)

	log.Println("System tray initialized. Use tray icon to control the application.")

	// Run system tray (blocks until Quit)
	trayMgr.Run()
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
