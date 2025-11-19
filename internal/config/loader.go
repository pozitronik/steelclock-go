package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// Load reads and parses a configuration file
// If the file doesn't exist, returns a default configuration
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			cfg := CreateDefault()
			return cfg, nil
		}
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

// CreateDefault creates a configuration with sensible defaults
func CreateDefault() *Config {
	cfg := &Config{
		GameName:        "STEELCLOCK",
		GameDisplayName: "SteelClock",
		RefreshRateMs:   100,
		Display: DisplayConfig{
			Width:  128,
			Height: 40,
		},
		Widgets: []WidgetConfig{
			{
				ID:      "clock",
				Type:    "clock",
				Enabled: BoolPtr(true),
				Position: PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Properties: WidgetProperties{
					Format:          "%H:%M:%S",
					FontSize:        10,
					HorizontalAlign: "center",
					VerticalAlign:   "center",
					UpdateInterval:  1.0,
				},
				Style: StyleConfig{
					BackgroundColor: 0,
					Border:          false,
				},
			},
		},
	}

	return cfg
}

// SaveDefault creates and saves a default configuration file
func SaveDefault(path string) error {
	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create default config
	cfg := CreateDefault()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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

	validTypes := map[string]bool{
		"clock": true, "cpu": true, "memory": true,
		"network": true, "disk": true, "keyboard": true, "volume": true, "volume_meter": true, "doom": true,
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
			return fmt.Errorf("widget[%d] (%s): invalid type '%s' (valid: clock, cpu, memory, network, disk, keyboard, volume, volume_meter, doom)", i, w.ID, w.Type)
		}

		// Only validate properties for enabled widgets
		if w.IsEnabled() {
			// Type-specific validation (only required properties)
			if err := validateWidgetProperties(i, &w); err != nil {
				return err
			}
		}
	}

	// Note: We don't validate that at least one widget is enabled here.
	// A config with all widgets disabled is valid - it will be handled at runtime
	// by showing the "NO WIDGETS" error display on the OLED screen.

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
	applyDisplayDefaults(cfg)

	for i := range cfg.Widgets {
		applyWidgetDefaults(&cfg.Widgets[i])
	}
}

// applyDisplayDefaults sets default values for display configuration
func applyDisplayDefaults(cfg *Config) {
	if cfg.RefreshRateMs == 0 {
		cfg.RefreshRateMs = 100
	}

	if cfg.Display.Width == 0 {
		cfg.Display.Width = 128
	}

	if cfg.Display.Height == 0 {
		cfg.Display.Height = 40
	}
}

// applyWidgetDefaults sets default values for a widget
func applyWidgetDefaults(w *WidgetConfig) {
	// Enabled defaults to true via IsEnabled() method - no need to set it here

	applyCommonWidgetDefaults(w)
	applyTypeSpecificDefaults(w)
}

// applyCommonWidgetDefaults sets default values common to all widgets
func applyCommonWidgetDefaults(w *WidgetConfig) {
	if w.Properties.UpdateInterval == 0 {
		w.Properties.UpdateInterval = 1.0
	}

	if w.Properties.FontSize == 0 {
		w.Properties.FontSize = 10
	}

	if w.Properties.HorizontalAlign == "" {
		w.Properties.HorizontalAlign = "center"
	}

	if w.Properties.VerticalAlign == "" {
		w.Properties.VerticalAlign = "center"
	}
}

// applyTypeSpecificDefaults sets default values specific to widget types
func applyTypeSpecificDefaults(w *WidgetConfig) {
	switch w.Type {
	case "clock":
		applyClockDefaults(w)
	case "cpu", "memory":
		applyMetricWidgetDefaults(w)
	case "network":
		applyNetworkDefaults(w)
	case "disk":
		applyDiskDefaults(w)
	case "keyboard":
		applyKeyboardDefaults(w)
	}
}

// applyClockDefaults sets default values for clock widgets
func applyClockDefaults(w *WidgetConfig) {
	if w.Properties.Format == "" {
		w.Properties.Format = "%H:%M:%S"
	}
}

// applyMetricWidgetDefaults sets default values for CPU and Memory widgets
func applyMetricWidgetDefaults(w *WidgetConfig) {
	if w.Properties.DisplayMode == "" {
		w.Properties.DisplayMode = "text"
	}
	if w.Properties.FillColor == 0 {
		w.Properties.FillColor = 255
	}
	if w.Properties.HistoryLength == 0 {
		w.Properties.HistoryLength = 30
	}
}

// applyNetworkDefaults sets default values for network widgets
func applyNetworkDefaults(w *WidgetConfig) {
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
}

// applyDiskDefaults sets default values for disk widgets
func applyDiskDefaults(w *WidgetConfig) {
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
}

// applyKeyboardDefaults sets default values for keyboard widgets
func applyKeyboardDefaults(w *WidgetConfig) {
	if w.Properties.IndicatorColorOn == 0 {
		w.Properties.IndicatorColorOn = 255
	}
	if w.Properties.IndicatorColorOff == 0 {
		w.Properties.IndicatorColorOff = 100
	}
}
