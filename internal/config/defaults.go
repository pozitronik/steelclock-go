package config

import "fmt"

const (
	// DefaultGameName Default game registration values
	// Note: game_name and game_display_name MUST be different or GameSense API returns 400 error
	DefaultGameName    = "STEELCLOCK"
	DefaultGameDisplay = "SteelClock"

	// DefaultDisplayWidth is the common OLED display width for SteelSeries devices
	DefaultDisplayWidth = 128

	// DefaultDisplayHeight is the common OLED display height for SteelSeries devices
	DefaultDisplayHeight = 40

	// DefaultRefreshRateMs is the default frame rate (10 FPS)
	DefaultRefreshRateMs = 100

	// BorderDisabled represents disabled border value
	BorderDisabled = -1

	// DefaultUpdateInterval is the default widget update interval in seconds
	DefaultUpdateInterval = 1.0

	// DefaultPollInterval is the default internal polling interval for volume widgets in seconds
	// Fast polling (100ms) ensures volume changes are detected quickly for responsive UI
	DefaultPollInterval = 0.1

	// DefaultFontSize is the default text font size
	DefaultFontSize = 10

	// DefaultGraphHistory is the default number of history points for graphs
	DefaultGraphHistory = 30

	// DefaultEventBatchSize is the default batch size for event batching
	DefaultEventBatchSize = 10
)

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to an int value
func IntPtr(i int) *int {
	return &i
}

// CreateDefault creates a configuration with sensible defaults
func CreateDefault() *Config {
	cfg := &Config{
		GameName:        DefaultGameName,
		GameDisplayName: DefaultGameDisplay,
		RefreshRateMs:   DefaultRefreshRateMs,
		Display: DisplayConfig{
			Width:  DefaultDisplayWidth,
			Height: DefaultDisplayHeight,
		},
		Widgets: []WidgetConfig{
			{
				ID:      "clock",
				Type:    "clock",
				Enabled: BoolPtr(true),
				Position: PositionConfig{
					X: 0,
					Y: 0,
					W: DefaultDisplayWidth,
					H: DefaultDisplayHeight,
					Z: 0,
				},
				Style: &StyleConfig{
					Background: 0,
					Border:     BorderDisabled,
				},
				Text: &TextConfig{
					Format: "%H:%M:%S",
					Size:   DefaultFontSize,
					Align: &AlignConfig{
						H: "center",
						V: "center",
					},
				},
				UpdateInterval: DefaultUpdateInterval,
			},
		},
	}

	return cfg
}

// applyDefaults fills in default values for optional fields
func applyDefaults(cfg *Config) {
	applyGlobalDefaults(cfg)
	applyDirectDriverDefaults(cfg)
	applyDisplayDefaults(cfg)

	for i := range cfg.Widgets {
		applyWidgetDefaults(&cfg.Widgets[i])
	}
}

// applyGlobalDefaults sets default values for global configuration
func applyGlobalDefaults(cfg *Config) {
	if cfg.GameName == "" {
		cfg.GameName = DefaultGameName
	}
	if cfg.GameDisplayName == "" {
		cfg.GameDisplayName = DefaultGameDisplay
	}
	if cfg.Backend == "" {
		cfg.Backend = "gamesense"
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
		cfg.RefreshRateMs = DefaultRefreshRateMs
	}

	if cfg.Display.Width == 0 {
		cfg.Display.Width = DefaultDisplayWidth
	}

	if cfg.Display.Height == 0 {
		cfg.Display.Height = DefaultDisplayHeight
	}

	if cfg.EventBatchingEnabled && cfg.EventBatchSize == 0 {
		cfg.EventBatchSize = DefaultEventBatchSize
	}
}

// applyWidgetDefaults sets default values for a widget
func applyWidgetDefaults(w *WidgetConfig) {
	applyCommonWidgetDefaults(w)
	applyTypeSpecificDefaults(w)
}

// applyCommonWidgetDefaults sets default values common to all widgets
func applyCommonWidgetDefaults(w *WidgetConfig) {
	if w.UpdateInterval == 0 {
		w.UpdateInterval = DefaultUpdateInterval
	}

	if w.Style == nil {
		w.Style = &StyleConfig{}
	}

	if w.Text == nil {
		w.Text = &TextConfig{}
	}

	if w.Text.Size == 0 {
		w.Text.Size = DefaultFontSize
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
		// Color defaults handled in widget constructor
		return
	case "audio_visualizer":
		applyAudioVisualizerDefaults(w)
	case "volume":
		applyVolumeDefaults(w)
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

	if w.Colors == nil {
		w.Colors = &ColorsConfig{}
	}
	if w.Colors.Fill == nil {
		w.Colors.Fill = IntPtr(255)
	}

	if w.Graph == nil {
		w.Graph = &GraphConfig{}
	}
	if w.Graph.History == 0 {
		w.Graph.History = DefaultGraphHistory
	}
}

// applyNetworkDefaults sets default values for network widgets
func applyNetworkDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "text"
	}

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

	if w.Graph == nil {
		w.Graph = &GraphConfig{}
	}
	if w.Graph.History == 0 {
		w.Graph.History = DefaultGraphHistory
	}
}

// applyDiskDefaults sets default values for disk widgets
func applyDiskDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "text"
	}

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

	if w.Graph == nil {
		w.Graph = &GraphConfig{}
	}
	if w.Graph.History == 0 {
		w.Graph.History = DefaultGraphHistory
	}
}

// applyAudioVisualizerDefaults sets default values for audio visualizer widgets
func applyAudioVisualizerDefaults(w *WidgetConfig) {
	if w.Mode == "" {
		w.Mode = "spectrum"
	}

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

	if w.Channel == "" {
		w.Channel = "stereo_combined"
	}

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

	if w.Peak == nil {
		w.Peak = &PeakConfig{}
	}
	if w.Peak.HoldTime == 0 {
		w.Peak.HoldTime = 1.0
	}

	if w.Oscilloscope == nil {
		w.Oscilloscope = &OscilloscopeConfig{}
	}
	if w.Oscilloscope.Style == "" {
		w.Oscilloscope.Style = "line"
	}
	if w.Oscilloscope.Samples == 0 {
		w.Oscilloscope.Samples = DefaultDisplayWidth
	}
}

// applyVolumeDefaults sets default values for volume widgets
func applyVolumeDefaults(w *WidgetConfig) {
	if w.PollInterval == 0 {
		w.PollInterval = DefaultPollInterval
	}

	if w.Mode == "" {
		w.Mode = "bar"
	}

	if w.Colors == nil {
		w.Colors = &ColorsConfig{}
	}
	if w.Colors.Fill == nil {
		w.Colors.Fill = IntPtr(255)
	}

	if w.Bar == nil {
		w.Bar = &BarConfig{}
	}
	if w.Bar.Direction == "" {
		w.Bar.Direction = "horizontal"
	}
}

// applyVolumeMeterDefaults sets default values for volume meter widgets
func applyVolumeMeterDefaults(w *WidgetConfig) {
	if w.PollInterval == 0 {
		w.PollInterval = DefaultPollInterval
	}

	if w.Mode == "" {
		w.Mode = "bar"
	}

	if w.Bar == nil {
		w.Bar = &BarConfig{}
	}
	if w.Bar.Direction == "" {
		w.Bar.Direction = "horizontal"
	}

	if w.Stereo == nil {
		w.Stereo = &StereoConfig{}
	}

	if w.Metering == nil {
		w.Metering = &MeteringConfig{}
	}
	if w.Metering.DecayRate == 0 {
		w.Metering.DecayRate = 2.0
	}
	if w.Metering.SilenceThreshold == 0 {
		w.Metering.SilenceThreshold = 0.01
	}

	if w.Peak == nil {
		w.Peak = &PeakConfig{}
	}
	if w.Peak.HoldTime == 0 {
		w.Peak.HoldTime = 1.0
	}

	if w.Clipping == nil {
		w.Clipping = &ClippingConfig{}
	}
	if w.Clipping.Threshold == 0 {
		w.Clipping.Threshold = 0.99
	}
}

// generateWidgetIDs assigns unique IDs to widgets based on type
func generateWidgetIDs(widgets []WidgetConfig) {
	typeCounts := make(map[string]int)
	for i := range widgets {
		w := &widgets[i]
		w.ID = fmt.Sprintf("%s_%d", w.Type, typeCounts[w.Type])
		typeCounts[w.Type]++
	}
}
