package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pozitronik/steelclock/internal/compositor"
	"github.com/pozitronik/steelclock/internal/config"
	"github.com/pozitronik/steelclock/internal/gamesense"
	"github.com/pozitronik/steelclock/internal/layout"
	"github.com/pozitronik/steelclock/internal/widget"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	log.Println("SteelClock starting...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded: %s (%s)", cfg.GameName, cfg.GameDisplayName)

	// Create GameSense client
	client, err := gamesense.NewClient(cfg.GameName, cfg.GameDisplayName)
	if err != nil {
		log.Fatalf("Failed to create GameSense client: %v", err)
	}

	log.Println("GameSense client created")

	// Register game
	if err := client.RegisterGame("Custom"); err != nil {
		log.Fatalf("Failed to register game: %v", err)
	}

	// Bind screen event
	if err := client.BindScreenEvent("STEELCLOCK_DISPLAY", "screened-128x40"); err != nil {
		log.Fatalf("Failed to bind screen event: %v", err)
	}

	// Create widgets
	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		log.Fatalf("Failed to create widgets: %v", err)
	}

	log.Printf("Created %d widgets", len(widgets))

	// Create layout manager
	layoutMgr := layout.NewManager(cfg.Display, widgets)

	// Create compositor
	comp := compositor.NewCompositor(client, layoutMgr, widgets, cfg)

	// Start compositor
	if err := comp.Start(); err != nil {
		log.Fatalf("Failed to start compositor: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("SteelClock running. Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigChan

	log.Println("Shutting down...")

	// Stop compositor
	comp.Stop()

	// Unregister game
	if err := client.RemoveGame(); err != nil {
		log.Printf("Failed to remove game: %v", err)
	}

	log.Println("SteelClock stopped")
}
