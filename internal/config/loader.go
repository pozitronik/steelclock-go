package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultGameName Default game registration values
	// Note: game_name and game_display_name MUST be different or GameSense API returns 400 error
	DefaultGameName    = "STEELCLOCK"
	DefaultGameDisplay = "SteelClock"
)

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to an int value
func IntPtr(i int) *int {
	return &i
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
		GameName:        DefaultGameName,
		GameDisplayName: DefaultGameDisplay,
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
					Z: 0,
				},
				Style: &StyleConfig{
					Background: 0,
					Border:     -1, // disabled
				},
				Text: &TextConfig{
					Format: "%H:%M:%S",
					Size:   10,
					Align: &AlignConfig{
						H: "center",
						V: "center",
					},
				},
				UpdateInterval: 1.0,
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

// ValidBackends contains valid backend values
var ValidBackends = map[string]bool{
	"":          true, // Empty = default (gamesense)
	"gamesense": true,
	"direct":    true,
	"any":       true,
}

// validateConfig checks that required fields are present and valid
func validateConfig(cfg *Config) error {
	// Note: game_name and game_display_name are optional - defaults applied in applyDefaults()

	// Validate backend
	if !ValidBackends[cfg.Backend] {
		return fmt.Errorf("invalid backend '%s' (valid: gamesense, direct, any)", cfg.Backend)
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

	// Check deinitialize_timer_ms if specified
	if cfg.DeinitializeTimerMs != 0 {
		if cfg.DeinitializeTimerMs < 1000 || cfg.DeinitializeTimerMs > 60000 {
			return fmt.Errorf("deinitialize_timer_ms must be between 1000 and 60000 (got %d)", cfg.DeinitializeTimerMs)
		}
	}

	// Check event_batch_size if specified
	if cfg.EventBatchSize != 0 {
		if cfg.EventBatchSize < 1 || cfg.EventBatchSize > 100 {
			return fmt.Errorf("event_batch_size must be between 1 and 100 (got %d)", cfg.EventBatchSize)
		}
	}

	// Check supported_resolutions if specified
	for i, res := range cfg.SupportedResolutions {
		if res.Width <= 0 {
			return fmt.Errorf("supported_resolutions[%d]: width must be positive (got %d)", i, res.Width)
		}
		if res.Height <= 0 {
			return fmt.Errorf("supported_resolutions[%d]: height must be positive (got %d)", i, res.Height)
		}
	}

	// Check widgets
	if len(cfg.Widgets) == 0 {
		return fmt.Errorf("at least one widget must be configured")
	}

	validTypes := map[string]bool{
		"clock": true, "cpu": true, "memory": true,
		"network": true, "disk": true, "keyboard": true, "keyboard_layout": true,
		"volume": true, "volume_meter": true, "audio_visualizer": true, "doom": true,
		"winamp": true,
	}

	// Track widget counts for auto-generating IDs
	typeCounts := make(map[string]int)

	for i := range cfg.Widgets {
		w := &cfg.Widgets[i]

		// Check widget type
		if w.Type == "" {
			return fmt.Errorf("widget[%d]: type is required", i)
		}
		if !validTypes[w.Type] {
			return fmt.Errorf("widget[%d]: invalid type '%s' (valid: clock, cpu, memory, network, disk, keyboard, keyboard_layout, volume, volume_meter, audio_visualizer, doom, winamp)", i, w.Type)
		}

		// Auto-generate ID based on type and index
		w.ID = fmt.Sprintf("%s_%d", w.Type, typeCounts[w.Type])
		typeCounts[w.Type]++

		// Only validate properties for enabled widgets
		if w.IsEnabled() {
			// Type-specific validation (only required properties)
			if err := validateWidgetProperties(i, w); err != nil {
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
		// Format can be in Text.Format or as a fallback we apply defaults
		// No strict validation needed - defaults will be applied

	case "network":
		if w.Interface == nil || *w.Interface == "" {
			return fmt.Errorf("widget[%d] (%s): interface is required", index, w.ID)
		}

	case "disk":
		if w.Disk == nil || *w.Disk == "" {
			return fmt.Errorf("widget[%d] (%s): disk is required", index, w.ID)
		}
	}

	return nil
}

// applyDefaults fills in default values for optional fields
func applyDefaults(cfg *Config) {
	// Apply default game name if not specified
	if cfg.GameName == "" {
		cfg.GameName = DefaultGameName
	}
	if cfg.GameDisplayName == "" {
		cfg.GameDisplayName = DefaultGameDisplay
	}

	// Apply default backend
	if cfg.Backend == "" {
		cfg.Backend = "gamesense"
	}

	// Apply direct driver defaults
	applyDirectDriverDefaults(cfg)

	applyDisplayDefaults(cfg)

	for i := range cfg.Widgets {
		applyWidgetDefaults(&cfg.Widgets[i])
	}
}

// applyDirectDriverDefaults sets default values for direct driver configuration
func applyDirectDriverDefaults(cfg *Config) {
	if cfg.DirectDriver == nil {
		cfg.DirectDriver = &DirectDriverConfig{}
	}

	if cfg.DirectDriver.Interface == "" {
		cfg.DirectDriver.Interface = "mi_01"
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

	// Apply default for event_batch_size if not specified
	if cfg.EventBatchingEnabled && cfg.EventBatchSize == 0 {
		cfg.EventBatchSize = 10
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
	if w.UpdateInterval == 0 {
		w.UpdateInterval = 1.0
	}

	// Initialize Style config if nil
	if w.Style == nil {
		w.Style = &StyleConfig{}
	}

	// Initialize Text config if nil and apply defaults
	if w.Text == nil {
		w.Text = &TextConfig{}
	}

	if w.Text.Size == 0 {
		w.Text.Size = 10
	}

	if w.Text.Align == nil {
		w.Text.Align = &AlignConfig{}
	}

	if w.Text.Align.H == "" {
		w.Text.Align.H = "center"
	}

	if w.Text.Align.V == "" {
		w.Text.Align.V = "center"
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
		// Color defaults are now handled in the widget constructor
		// to properly distinguish between nil (not set) and 0 (explicitly set to black)
		return
	case "audio_visualizer":
		applyAudioVisualizerDefaults(w)
	case "volume_meter":
		applyVolumeMeterDefaults(w)
	}
}

// applyClockDefaults sets default values for clock widgets
func applyClockDefaults(w *WidgetConfig) {
	if w.Text == nil {
		w.Text = &TextConfig{}
	}
	if w.Text.Format == "" {
		w.Text.Format = "%H:%M:%S"
	}
}

// applyMetricWidgetDefaults sets default values for CPU and Memory widgets
func applyMetricWidgetDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "text"
	}

	// Initialize Colors if nil
	if w.Colors == nil {
		w.Colors = &ColorsConfig{}
	}
	if w.Colors.Fill == nil {
		w.Colors.Fill = IntPtr(255)
	}

	// Initialize Graph config with defaults
	if w.Graph == nil {
		w.Graph = &GraphConfig{}
	}
	if w.Graph.History == 0 {
		w.Graph.History = 30
	}
}

// applyNetworkDefaults sets default values for network widgets
func applyNetworkDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "text"
	}

	// Initialize Colors if nil
	if w.Colors == nil {
		w.Colors = &ColorsConfig{}
	}
	if w.Colors.Rx == nil {
		w.Colors.Rx = IntPtr(255)
	}
	if w.Colors.Tx == nil {
		w.Colors.Tx = IntPtr(255)
	}

	if w.MaxSpeedMbps == 0 {
		w.MaxSpeedMbps = -1
	}

	// Initialize Graph config with defaults
	if w.Graph == nil {
		w.Graph = &GraphConfig{}
	}
	if w.Graph.History == 0 {
		w.Graph.History = 30
	}
}

// applyDiskDefaults sets default values for disk widgets
func applyDiskDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "text"
	}

	// Initialize Colors if nil
	if w.Colors == nil {
		w.Colors = &ColorsConfig{}
	}
	if w.Colors.Read == nil {
		w.Colors.Read = IntPtr(255)
	}
	if w.Colors.Write == nil {
		w.Colors.Write = IntPtr(255)
	}

	if w.MaxSpeedMbps == 0 {
		w.MaxSpeedMbps = -1
	}

	// Initialize Graph config with defaults
	if w.Graph == nil {
		w.Graph = &GraphConfig{}
	}
	if w.Graph.History == 0 {
		w.Graph.History = 30
	}
}

// applyAudioVisualizerDefaults sets default values for audio visualizer widgets
func applyAudioVisualizerDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "spectrum"
	}

	// Initialize Colors if nil
	if w.Colors == nil {
		w.Colors = &ColorsConfig{}
	}
	if w.Colors.Fill == nil {
		w.Colors.Fill = IntPtr(255)
	}
	if w.Colors.Left == nil {
		w.Colors.Left = IntPtr(255)
	}
	if w.Colors.Right == nil {
		w.Colors.Right = IntPtr(200)
	}

	// Channel mode default
	if w.Channel == "" {
		w.Channel = "stereo_combined"
	}

	// Spectrum defaults
	if w.Spectrum == nil {
		w.Spectrum = &SpectrumConfig{}
	}
	if w.Spectrum.Bars == 0 {
		w.Spectrum.Bars = 32
	}
	if w.Spectrum.Scale == "" {
		w.Spectrum.Scale = "logarithmic"
	}
	if w.Spectrum.Style == "" {
		w.Spectrum.Style = "bars"
	}
	if w.Spectrum.Smoothing == 0 {
		w.Spectrum.Smoothing = 0.7
	}

	// Peak defaults
	if w.Peak == nil {
		w.Peak = &PeakConfig{}
	}
	if w.Peak.HoldTime == 0 {
		w.Peak.HoldTime = 1.0
	}

	// Oscilloscope defaults
	if w.Oscilloscope == nil {
		w.Oscilloscope = &OscilloscopeConfig{}
	}
	if w.Oscilloscope.Style == "" {
		w.Oscilloscope.Style = "line"
	}
	if w.Oscilloscope.Samples == 0 {
		w.Oscilloscope.Samples = 128
	}
}

// applyVolumeMeterDefaults sets default values for volume meter widgets
func applyVolumeMeterDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "bar"
	}

	// Initialize Bar config with defaults
	if w.Bar == nil {
		w.Bar = &BarConfig{}
	}
	if w.Bar.Direction == "" {
		w.Bar.Direction = "horizontal"
	}

	// Initialize Stereo config with defaults
	if w.Stereo == nil {
		w.Stereo = &StereoConfig{}
	}

	// Initialize Metering config with defaults
	if w.Metering == nil {
		w.Metering = &MeteringConfig{}
	}
	if w.Metering.DecayRate == 0 {
		w.Metering.DecayRate = 2.0
	}
	if w.Metering.SilenceThreshold == 0 {
		w.Metering.SilenceThreshold = 0.01
	}

	// Initialize Peak config with defaults
	if w.Peak == nil {
		w.Peak = &PeakConfig{}
	}
	if w.Peak.HoldTime == 0 {
		w.Peak.HoldTime = 1.0
	}

	// Initialize Clipping config with defaults
	if w.Clipping == nil {
		w.Clipping = &ClippingConfig{}
	}
	if w.Clipping.Threshold == 0 {
		w.Clipping.Threshold = 0.99
	}
}
