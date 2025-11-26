package config

import (
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"DefaultGameName", DefaultGameName, "STEELCLOCK"},
		{"DefaultGameDisplay", DefaultGameDisplay, "SteelClock"},
		{"DefaultDisplayWidth", DefaultDisplayWidth, 128},
		{"DefaultDisplayHeight", DefaultDisplayHeight, 40},
		{"DefaultRefreshRateMs", DefaultRefreshRateMs, 100},
		{"BorderDisabled", BorderDisabled, -1},
		{"DefaultUpdateInterval", DefaultUpdateInterval, 1.0},
		{"DefaultFontSize", DefaultFontSize, 10},
		{"DefaultGraphHistory", DefaultGraphHistory, 30},
		{"DefaultEventBatchSize", DefaultEventBatchSize, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	truePtr := BoolPtr(true)
	if truePtr == nil {
		t.Fatal("BoolPtr(true) returned nil")
	}
	if *truePtr != true {
		t.Errorf("*BoolPtr(true) = %v, want true", *truePtr)
	}

	falsePtr := BoolPtr(false)
	if falsePtr == nil {
		t.Fatal("BoolPtr(false) returned nil")
	}
	if *falsePtr != false {
		t.Errorf("*BoolPtr(false) = %v, want false", *falsePtr)
	}

	// Verify they are different pointers
	if truePtr == falsePtr {
		t.Error("BoolPtr should return different pointers for different calls")
	}
}

func TestIntPtr(t *testing.T) {
	tests := []int{0, 1, -1, 255, 1000, -1000}

	for _, val := range tests {
		ptr := IntPtr(val)
		if ptr == nil {
			t.Fatalf("IntPtr(%d) returned nil", val)
		}
		if *ptr != val {
			t.Errorf("*IntPtr(%d) = %d, want %d", val, *ptr, val)
		}
	}

	// Verify different calls return different pointers
	ptr1 := IntPtr(42)
	ptr2 := IntPtr(42)
	if ptr1 == ptr2 {
		t.Error("IntPtr should return different pointers for different calls")
	}
}

func TestCreateDefault(t *testing.T) {
	cfg := CreateDefault()

	if cfg == nil {
		t.Fatal("CreateDefault returned nil")
	}

	// Check global defaults
	if cfg.GameName != DefaultGameName {
		t.Errorf("GameName = %q, want %q", cfg.GameName, DefaultGameName)
	}
	if cfg.GameDisplayName != DefaultGameDisplay {
		t.Errorf("GameDisplayName = %q, want %q", cfg.GameDisplayName, DefaultGameDisplay)
	}
	if cfg.RefreshRateMs != DefaultRefreshRateMs {
		t.Errorf("RefreshRateMs = %d, want %d", cfg.RefreshRateMs, DefaultRefreshRateMs)
	}

	// Check display defaults
	if cfg.Display.Width != DefaultDisplayWidth {
		t.Errorf("Display.Width = %d, want %d", cfg.Display.Width, DefaultDisplayWidth)
	}
	if cfg.Display.Height != DefaultDisplayHeight {
		t.Errorf("Display.Height = %d, want %d", cfg.Display.Height, DefaultDisplayHeight)
	}

	// Check default widget
	if len(cfg.Widgets) != 1 {
		t.Fatalf("len(Widgets) = %d, want 1", len(cfg.Widgets))
	}

	widget := cfg.Widgets[0]
	if widget.Type != "clock" {
		t.Errorf("Widget.Type = %q, want %q", widget.Type, "clock")
	}
	if widget.Enabled == nil || !*widget.Enabled {
		t.Error("Widget should be enabled by default")
	}
	if widget.Position.W != DefaultDisplayWidth {
		t.Errorf("Widget.Position.W = %d, want %d", widget.Position.W, DefaultDisplayWidth)
	}
	if widget.Position.H != DefaultDisplayHeight {
		t.Errorf("Widget.Position.H = %d, want %d", widget.Position.H, DefaultDisplayHeight)
	}
	if widget.Style == nil || widget.Style.Border != BorderDisabled {
		t.Error("Widget border should be disabled by default")
	}
	if widget.Text == nil || widget.Text.Format != "%H:%M:%S" {
		t.Error("Widget should have default time format")
	}
}

func TestApplyGlobalDefaults(t *testing.T) {
	tests := []struct {
		name            string
		input           Config
		expectedGame    string
		expectedDisplay string
		expectedBackend string
	}{
		{
			name:            "all empty",
			input:           Config{},
			expectedGame:    DefaultGameName,
			expectedDisplay: DefaultGameDisplay,
			expectedBackend: "gamesense",
		},
		{
			name: "custom game name",
			input: Config{
				GameName: "CUSTOM",
			},
			expectedGame:    "CUSTOM",
			expectedDisplay: DefaultGameDisplay,
			expectedBackend: "gamesense",
		},
		{
			name: "custom backend",
			input: Config{
				Backend: "direct",
			},
			expectedGame:    DefaultGameName,
			expectedDisplay: DefaultGameDisplay,
			expectedBackend: "direct",
		},
		{
			name: "all custom",
			input: Config{
				GameName:        "MY_GAME",
				GameDisplayName: "My Game",
				Backend:         "any",
			},
			expectedGame:    "MY_GAME",
			expectedDisplay: "My Game",
			expectedBackend: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			applyGlobalDefaults(&cfg)

			if cfg.GameName != tt.expectedGame {
				t.Errorf("GameName = %q, want %q", cfg.GameName, tt.expectedGame)
			}
			if cfg.GameDisplayName != tt.expectedDisplay {
				t.Errorf("GameDisplayName = %q, want %q", cfg.GameDisplayName, tt.expectedDisplay)
			}
			if cfg.Backend != tt.expectedBackend {
				t.Errorf("Backend = %q, want %q", cfg.Backend, tt.expectedBackend)
			}
		})
	}
}

func TestApplyDirectDriverDefaults(t *testing.T) {
	// Test nil DirectDriver
	cfg := &Config{}
	applyDirectDriverDefaults(cfg)

	if cfg.DirectDriver == nil {
		t.Fatal("DirectDriver should be initialized")
	}
	if cfg.DirectDriver.Interface != "mi_01" {
		t.Errorf("Interface = %q, want %q", cfg.DirectDriver.Interface, "mi_01")
	}

	// Test with custom interface
	cfg2 := &Config{
		DirectDriver: &DirectDriverConfig{
			Interface: "mi_02",
		},
	}
	applyDirectDriverDefaults(cfg2)

	if cfg2.DirectDriver.Interface != "mi_02" {
		t.Errorf("Custom interface should be preserved, got %q", cfg2.DirectDriver.Interface)
	}
}

func TestApplyDisplayDefaults(t *testing.T) {
	tests := []struct {
		name           string
		input          Config
		expectedWidth  int
		expectedHeight int
		expectedRate   int
	}{
		{
			name:           "all zero",
			input:          Config{},
			expectedWidth:  DefaultDisplayWidth,
			expectedHeight: DefaultDisplayHeight,
			expectedRate:   DefaultRefreshRateMs,
		},
		{
			name: "custom width",
			input: Config{
				Display: DisplayConfig{Width: 256},
			},
			expectedWidth:  256,
			expectedHeight: DefaultDisplayHeight,
			expectedRate:   DefaultRefreshRateMs,
		},
		{
			name: "custom refresh rate",
			input: Config{
				RefreshRateMs: 50,
			},
			expectedWidth:  DefaultDisplayWidth,
			expectedHeight: DefaultDisplayHeight,
			expectedRate:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			applyDisplayDefaults(&cfg)

			if cfg.Display.Width != tt.expectedWidth {
				t.Errorf("Display.Width = %d, want %d", cfg.Display.Width, tt.expectedWidth)
			}
			if cfg.Display.Height != tt.expectedHeight {
				t.Errorf("Display.Height = %d, want %d", cfg.Display.Height, tt.expectedHeight)
			}
			if cfg.RefreshRateMs != tt.expectedRate {
				t.Errorf("RefreshRateMs = %d, want %d", cfg.RefreshRateMs, tt.expectedRate)
			}
		})
	}
}

func TestApplyDisplayDefaultsEventBatching(t *testing.T) {
	// With batching enabled but no size
	cfg := &Config{
		EventBatchingEnabled: true,
		EventBatchSize:       0,
	}
	applyDisplayDefaults(cfg)

	if cfg.EventBatchSize != DefaultEventBatchSize {
		t.Errorf("EventBatchSize = %d, want %d", cfg.EventBatchSize, DefaultEventBatchSize)
	}

	// With batching disabled
	cfg2 := &Config{
		EventBatchingEnabled: false,
		EventBatchSize:       0,
	}
	applyDisplayDefaults(cfg2)

	if cfg2.EventBatchSize != 0 {
		t.Errorf("EventBatchSize should remain 0 when batching disabled, got %d", cfg2.EventBatchSize)
	}

	// With custom batch size
	cfg3 := &Config{
		EventBatchingEnabled: true,
		EventBatchSize:       5,
	}
	applyDisplayDefaults(cfg3)

	if cfg3.EventBatchSize != 5 {
		t.Errorf("Custom EventBatchSize should be preserved, got %d", cfg3.EventBatchSize)
	}
}

func TestApplyCommonWidgetDefaults(t *testing.T) {
	w := &WidgetConfig{}
	applyCommonWidgetDefaults(w)

	if w.UpdateInterval != DefaultUpdateInterval {
		t.Errorf("UpdateInterval = %f, want %f", w.UpdateInterval, DefaultUpdateInterval)
	}
	if w.Style == nil {
		t.Error("Style should be initialized")
	}
	if w.Text == nil {
		t.Error("Text should be initialized")
	}
	if w.Text.Size != DefaultFontSize {
		t.Errorf("Text.Size = %d, want %d", w.Text.Size, DefaultFontSize)
	}
	if w.Text.Align == nil {
		t.Error("Text.Align should be initialized")
	}
	if w.Text.Align.H != "center" {
		t.Errorf("Text.Align.H = %q, want %q", w.Text.Align.H, "center")
	}
	if w.Text.Align.V != "center" {
		t.Errorf("Text.Align.V = %q, want %q", w.Text.Align.V, "center")
	}
}

func TestApplyClockDefaults(t *testing.T) {
	w := &WidgetConfig{Type: "clock"}
	applyClockDefaults(w)

	if w.Text == nil {
		t.Fatal("Text should be initialized")
	}
	if w.Text.Format != "%H:%M:%S" {
		t.Errorf("Text.Format = %q, want %q", w.Text.Format, "%H:%M:%S")
	}

	// Test with custom format
	w2 := &WidgetConfig{
		Type: "clock",
		Text: &TextConfig{Format: "%H:%M"},
	}
	applyClockDefaults(w2)

	if w2.Text.Format != "%H:%M" {
		t.Errorf("Custom format should be preserved, got %q", w2.Text.Format)
	}
}

func TestApplyMetricWidgetDefaults(t *testing.T) {
	for _, widgetType := range []string{"cpu", "memory"} {
		t.Run(widgetType, func(t *testing.T) {
			w := &WidgetConfig{Type: widgetType}
			applyMetricWidgetDefaults(w)

			if w.Mode != "text" {
				t.Errorf("Mode = %q, want %q", w.Mode, "text")
			}
			if w.Colors == nil {
				t.Fatal("Colors should be initialized")
			}
			if w.Colors.Fill == nil || *w.Colors.Fill != 255 {
				t.Error("Colors.Fill should be 255")
			}
			if w.Graph == nil {
				t.Fatal("Graph should be initialized")
			}
			if w.Graph.History != DefaultGraphHistory {
				t.Errorf("Graph.History = %d, want %d", w.Graph.History, DefaultGraphHistory)
			}
		})
	}
}

func TestApplyNetworkDefaults(t *testing.T) {
	w := &WidgetConfig{Type: "network"}
	applyNetworkDefaults(w)

	if w.Mode != "text" {
		t.Errorf("Mode = %q, want %q", w.Mode, "text")
	}
	if w.Colors == nil {
		t.Fatal("Colors should be initialized")
	}
	if w.Colors.Rx == nil || *w.Colors.Rx != 255 {
		t.Error("Colors.Rx should be 255")
	}
	if w.Colors.Tx == nil || *w.Colors.Tx != 255 {
		t.Error("Colors.Tx should be 255")
	}
	if w.MaxSpeedMbps != -1 {
		t.Errorf("MaxSpeedMbps = %f, want -1", w.MaxSpeedMbps)
	}
	if w.Graph == nil || w.Graph.History != DefaultGraphHistory {
		t.Errorf("Graph.History should be %d", DefaultGraphHistory)
	}
}

func TestApplyDiskDefaults(t *testing.T) {
	w := &WidgetConfig{Type: "disk"}
	applyDiskDefaults(w)

	if w.Mode != "text" {
		t.Errorf("Mode = %q, want %q", w.Mode, "text")
	}
	if w.Colors == nil {
		t.Fatal("Colors should be initialized")
	}
	if w.Colors.Read == nil || *w.Colors.Read != 255 {
		t.Error("Colors.Read should be 255")
	}
	if w.Colors.Write == nil || *w.Colors.Write != 255 {
		t.Error("Colors.Write should be 255")
	}
	if w.MaxSpeedMbps != -1 {
		t.Errorf("MaxSpeedMbps = %f, want -1", w.MaxSpeedMbps)
	}
}

func TestApplyAudioVisualizerDefaults(t *testing.T) {
	w := &WidgetConfig{Type: "audio_visualizer"}
	applyAudioVisualizerDefaults(w)

	if w.Mode != "spectrum" {
		t.Errorf("Mode = %q, want %q", w.Mode, "spectrum")
	}
	if w.Channel != "stereo_combined" {
		t.Errorf("Channel = %q, want %q", w.Channel, "stereo_combined")
	}
	if w.Colors == nil {
		t.Fatal("Colors should be initialized")
	}
	if w.Colors.Fill == nil || *w.Colors.Fill != 255 {
		t.Error("Colors.Fill should be 255")
	}
	if w.Colors.Left == nil || *w.Colors.Left != 255 {
		t.Error("Colors.Left should be 255")
	}
	if w.Colors.Right == nil || *w.Colors.Right != 200 {
		t.Error("Colors.Right should be 200")
	}

	// Spectrum defaults
	if w.Spectrum == nil {
		t.Fatal("Spectrum should be initialized")
	}
	if w.Spectrum.Bars != 32 {
		t.Errorf("Spectrum.Bars = %d, want 32", w.Spectrum.Bars)
	}
	if w.Spectrum.Scale != "logarithmic" {
		t.Errorf("Spectrum.Scale = %q, want %q", w.Spectrum.Scale, "logarithmic")
	}
	if w.Spectrum.Style != "bars" {
		t.Errorf("Spectrum.Style = %q, want %q", w.Spectrum.Style, "bars")
	}
	if w.Spectrum.Smoothing != 0.7 {
		t.Errorf("Spectrum.Smoothing = %f, want 0.7", w.Spectrum.Smoothing)
	}

	// Peak defaults
	if w.Peak == nil || w.Peak.HoldTime != 1.0 {
		t.Error("Peak.HoldTime should be 1.0")
	}

	// Oscilloscope defaults
	if w.Oscilloscope == nil {
		t.Fatal("Oscilloscope should be initialized")
	}
	if w.Oscilloscope.Style != "line" {
		t.Errorf("Oscilloscope.Style = %q, want %q", w.Oscilloscope.Style, "line")
	}
	if w.Oscilloscope.Samples != DefaultDisplayWidth {
		t.Errorf("Oscilloscope.Samples = %d, want %d", w.Oscilloscope.Samples, DefaultDisplayWidth)
	}
}

func TestApplyVolumeMeterDefaults(t *testing.T) {
	w := &WidgetConfig{Type: "volume_meter"}
	applyVolumeMeterDefaults(w)

	if w.Mode != "bar" {
		t.Errorf("Mode = %q, want %q", w.Mode, "bar")
	}

	// Bar defaults
	if w.Bar == nil {
		t.Fatal("Bar should be initialized")
	}
	if w.Bar.Direction != "horizontal" {
		t.Errorf("Bar.Direction = %q, want %q", w.Bar.Direction, "horizontal")
	}

	// Stereo defaults
	if w.Stereo == nil {
		t.Error("Stereo should be initialized")
	}

	// Metering defaults
	if w.Metering == nil {
		t.Fatal("Metering should be initialized")
	}
	if w.Metering.DecayRate != 2.0 {
		t.Errorf("Metering.DecayRate = %f, want 2.0", w.Metering.DecayRate)
	}
	if w.Metering.SilenceThreshold != 0.01 {
		t.Errorf("Metering.SilenceThreshold = %f, want 0.01", w.Metering.SilenceThreshold)
	}

	// Peak defaults
	if w.Peak == nil || w.Peak.HoldTime != 1.0 {
		t.Error("Peak.HoldTime should be 1.0")
	}

	// Clipping defaults
	if w.Clipping == nil {
		t.Fatal("Clipping should be initialized")
	}
	if w.Clipping.Threshold != 0.99 {
		t.Errorf("Clipping.Threshold = %f, want 0.99", w.Clipping.Threshold)
	}
}

func TestGenerateWidgetIDs(t *testing.T) {
	widgets := []WidgetConfig{
		{Type: "clock"},
		{Type: "cpu"},
		{Type: "clock"},
		{Type: "memory"},
		{Type: "cpu"},
	}

	generateWidgetIDs(widgets)

	expected := []string{"clock_0", "cpu_0", "clock_1", "memory_0", "cpu_1"}
	for i, w := range widgets {
		if w.ID != expected[i] {
			t.Errorf("widgets[%d].ID = %q, want %q", i, w.ID, expected[i])
		}
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{
		Widgets: []WidgetConfig{
			{Type: "clock"},
			{Type: "cpu"},
		},
	}

	applyDefaults(cfg)

	// Check global defaults applied
	if cfg.GameName != DefaultGameName {
		t.Errorf("GameName not set to default")
	}
	if cfg.Backend != "gamesense" {
		t.Errorf("Backend not set to default")
	}

	// Check display defaults applied
	if cfg.Display.Width != DefaultDisplayWidth {
		t.Errorf("Display.Width not set to default")
	}

	// Check widget defaults applied
	for _, w := range cfg.Widgets {
		if w.UpdateInterval != DefaultUpdateInterval {
			t.Errorf("Widget UpdateInterval not set to default")
		}
		if w.Text == nil || w.Text.Size != DefaultFontSize {
			t.Errorf("Widget Text.Size not set to default")
		}
	}
}
