// Package direct provides a direct USB HID driver backend implementation.
package direct

import (
	"fmt"
	"log"
	"strconv"

	"github.com/pozitronik/steelclock-go/internal/backend"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
	"github.com/pozitronik/steelclock-go/internal/driver"
)

// Priority for auto-selection (lower = tried first)
// Direct driver has lower priority than GameSense (tried as fallback)
const Priority = 20

func init() {
	backend.Register("direct", newBackend, Priority)
	backend.RegisterProber("direct", probe)
}

// probe reports whether the configured (or any known) device is connected,
// without opening it.
func probe(cfg *config.Config) bool {
	driverCfg, err := driverConfig(cfg)
	if err != nil {
		return false
	}
	return driver.DevicePresent(driverCfg)
}

// driverConfig builds the HID driver configuration from the app configuration.
func driverConfig(cfg *config.Config) (driver.Config, error) {
	var vid, pid uint16
	if cfg.DirectDriver != nil {
		if cfg.DirectDriver.VID != "" {
			v, err := strconv.ParseUint(cfg.DirectDriver.VID, 16, 16)
			if err != nil {
				return driver.Config{}, fmt.Errorf("invalid VID '%s': %w", cfg.DirectDriver.VID, err)
			}
			vid = uint16(v)
		}
		if cfg.DirectDriver.PID != "" {
			p, err := strconv.ParseUint(cfg.DirectDriver.PID, 16, 16)
			if err != nil {
				return driver.Config{}, fmt.Errorf("invalid PID '%s': %w", cfg.DirectDriver.PID, err)
			}
			pid = uint16(p)
		}
	}

	iface := ""
	if cfg.DirectDriver != nil && cfg.DirectDriver.Interface != "" {
		iface = cfg.DirectDriver.Interface
	}

	return driver.Config{
		VID:       vid,
		PID:       pid,
		Interface: iface,
		Width:     cfg.Display.Width,
		Height:    cfg.Display.Height,
	}, nil
}

// newBackend creates a direct USB HID driver backend from configuration
func newBackend(cfg *config.Config) (display.Backend, error) {
	driverCfg, err := driverConfig(cfg)
	if err != nil {
		return nil, err
	}

	client, err := driver.NewClient(driverCfg)
	if err != nil {
		log.Printf("ERROR: Failed to create direct driver client: %v", err)
		return nil, err
	}

	log.Printf("Direct driver client created (backend: direct, VID: %04X, PID: %04X)",
		driverCfg.VID, driverCfg.PID)

	return client, nil
}
