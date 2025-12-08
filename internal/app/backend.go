package app

import (
	"fmt"
	"log"
	"strconv"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
	"github.com/pozitronik/steelclock-go/internal/driver"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
)

// CreateBackendClient creates the appropriate client based on backend configuration
func CreateBackendClient(cfg *config.Config) (display.Backend, error) {
	switch cfg.Backend {
	case "gamesense":
		return CreateGameSenseClient(cfg)

	case "direct":
		return CreateDirectClient(cfg)

	case "any":
		log.Println("Backend 'any': trying GameSense first...")
		gsClient, err := CreateGameSenseClient(cfg)
		if err == nil {
			return gsClient, nil
		}
		log.Printf("GameSense backend failed: %v", err)
		log.Println("Backend 'any': falling back to direct driver...")
		return CreateDirectClient(cfg)

	default:
		return CreateGameSenseClient(cfg)
	}
}

// CreateGameSenseClient creates a GameSense API client
func CreateGameSenseClient(cfg *config.Config) (display.Backend, error) {
	client, err := gamesense.NewClient(cfg.GameName, cfg.GameDisplayName)
	if err != nil {
		log.Printf("ERROR: Failed to create GameSense client: %v", err)
		log.Println("SteelSeries GG may not be running or GameSense API is unavailable")
		return nil, &BackendUnavailableError{Err: err}
	}

	log.Println("GameSense client created")

	if err := client.RegisterGame(DeveloperName, cfg.DeinitializeTimerMs); err != nil {
		log.Printf("ERROR: Failed to register game: %v", err)
		return nil, &BackendUnavailableError{Err: err}
	}

	return client, nil
}

// CreateDirectClient creates a direct USB HID driver client
func CreateDirectClient(cfg *config.Config) (display.Backend, error) {
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

	iface := "mi_01"
	if cfg.DirectDriver != nil && cfg.DirectDriver.Interface != "" {
		iface = cfg.DirectDriver.Interface
	}

	driverCfg := driver.Config{
		VID:       vid,
		PID:       pid,
		Interface: iface,
		Width:     cfg.Display.Width,
		Height:    cfg.Display.Height,
	}

	client, err := driver.NewClient(driverCfg)
	if err != nil {
		log.Printf("ERROR: Failed to create direct driver client: %v", err)
		return nil, &BackendUnavailableError{Err: err}
	}

	log.Printf("Direct driver client created (backend: direct, VID: %04X, PID: %04X)",
		driverCfg.VID, driverCfg.PID)

	return client, nil
}
