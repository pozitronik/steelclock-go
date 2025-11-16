package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and parses a configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file (invalid JSON): %w", err)
	}

	// Apply defaults for missing fields
	applyDefaults(&cfg)

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validateConfig checks that required fields are present and valid
func validateConfig(cfg *Config) error {
	// Check required game info
	if cfg.GameName == "" {
		return fmt.Errorf("game_name is required")
	}
	if cfg.GameDisplayName == "" {
		return fmt.Errorf("game_display_name is required")
	}

	// Check display dimensions are positive
	if cfg.Display.Width <= 0 {
		return fmt.Errorf("display width must be positive (got %d)", cfg.Display.Width)
	}
	if cfg.Display.Height <= 0 {
		return fmt.Errorf("display height must be positive (got %d)", cfg.Display.Height)
	}

	// Check refresh rate is positive
	if cfg.RefreshRateMs <= 0 {
		return fmt.Errorf("refresh_rate_ms must be positive (got %d)", cfg.RefreshRateMs)
	}

	// Check widgets
	if len(cfg.Widgets) == 0 {
		return fmt.Errorf("at least one widget must be configured")
	}

	enabledCount := 0
	validTypes := map[string]bool{
		"clock": true, "cpu": true, "memory": true,
		"network": true, "disk": true, "keyboard": true,
	}

	for i, w := range cfg.Widgets {
		// Check widget ID
		if w.ID == "" {
			return fmt.Errorf("widget[%d]: id is required", i)
		}

		// Check widget type
		if w.Type == "" {
			return fmt.Errorf("widget[%d] (%s): type is required", i, w.ID)
		}
		if !validTypes[w.Type] {
			return fmt.Errorf("widget[%d] (%s): invalid type '%s' (valid: clock, cpu, memory, network, disk, keyboard)", i, w.ID, w.Type)
		}

		if w.Enabled {
			enabledCount++

			// Type-specific validation (only required properties)
			if err := validateWidgetProperties(i, &w); err != nil {
				return err
			}
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("at least one widget must be enabled")
	}

	return nil
}

// validateWidgetProperties validates type-specific widget properties
func validateWidgetProperties(index int, w *WidgetConfig) error {
	switch w.Type {
	case "clock":
		if w.Properties.Format == "" {
			return fmt.Errorf("widget[%d] (%s): clock format is required", index, w.ID)
		}

	case "network":
		if w.Properties.Interface == nil || *w.Properties.Interface == "" {
			return fmt.Errorf("widget[%d] (%s): network interface is required", index, w.ID)
		}

	case "disk":
		if w.Properties.DiskName == nil || *w.Properties.DiskName == "" {
			return fmt.Errorf("widget[%d] (%s): disk_name is required", index, w.ID)
		}
	}

	return nil
}

// applyDefaults fills in default values for optional fields
func applyDefaults(cfg *Config) {
	if cfg.RefreshRateMs == 0 {
		cfg.RefreshRateMs = 100
	}

	if cfg.Display.Width == 0 {
		cfg.Display.Width = 128
	}

	if cfg.Display.Height == 0 {
		cfg.Display.Height = 40
	}

	// Apply widget defaults
	for i := range cfg.Widgets {
		w := &cfg.Widgets[i]

		// Default enabled to true if not specified
		if !w.Enabled && w.Type != "" {
			w.Enabled = true
		}

		// Default update interval
		if w.Properties.UpdateInterval == 0 {
			w.Properties.UpdateInterval = 1.0
		}

		// Default font size
		if w.Properties.FontSize == 0 {
			w.Properties.FontSize = 10
		}

		// Default alignments
		if w.Properties.HorizontalAlign == "" {
			w.Properties.HorizontalAlign = "center"
		}

		if w.Properties.VerticalAlign == "" {
			w.Properties.VerticalAlign = "center"
		}

		// Default background opacity
		if w.Style.BackgroundOpacity == 0 {
			w.Style.BackgroundOpacity = 255
		}

		// Widget-specific defaults
		switch w.Type {
		case "clock":
			if w.Properties.Format == "" {
				w.Properties.Format = "%H:%M:%S"
			}

		case "cpu", "memory":
			if w.Properties.DisplayMode == "" {
				w.Properties.DisplayMode = "text"
			}
			if w.Properties.FillColor == 0 {
				w.Properties.FillColor = 255
			}
			if w.Properties.HistoryLength == 0 {
				w.Properties.HistoryLength = 30
			}

		case "network":
			if w.Properties.DisplayMode == "" {
				w.Properties.DisplayMode = "text"
			}
			if w.Properties.RxColor == 0 {
				w.Properties.RxColor = 255
			}
			if w.Properties.TxColor == 0 {
				w.Properties.TxColor = 255
			}
			if w.Properties.MaxSpeedMbps == 0 {
				w.Properties.MaxSpeedMbps = -1
			}
			if w.Properties.HistoryLength == 0 {
				w.Properties.HistoryLength = 30
			}

		case "disk":
			if w.Properties.DisplayMode == "" {
				w.Properties.DisplayMode = "text"
			}
			if w.Properties.ReadColor == 0 {
				w.Properties.ReadColor = 255
			}
			if w.Properties.WriteColor == 0 {
				w.Properties.WriteColor = 255
			}
			if w.Properties.MaxSpeedMbps == 0 {
				w.Properties.MaxSpeedMbps = -1
			}
			if w.Properties.HistoryLength == 0 {
				w.Properties.HistoryLength = 30
			}

		case "keyboard":
			if w.Properties.IndicatorColorOn == 0 {
				w.Properties.IndicatorColorOn = 255
			}
			if w.Properties.IndicatorColorOff == 0 {
				w.Properties.IndicatorColorOff = 100
			}
		}
	}
}
