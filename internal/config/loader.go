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
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for missing fields
	applyDefaults(&cfg)

	return &cfg, nil
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
