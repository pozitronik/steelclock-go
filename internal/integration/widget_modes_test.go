// Package integration contains integration tests that span multiple packages
package integration

import (
	"runtime"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/testutil"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"github.com/pozitronik/steelclock-go/internal/widget/clock"
	"github.com/pozitronik/steelclock-go/internal/widget/cpu"
	"github.com/pozitronik/steelclock-go/internal/widget/disk"
	"github.com/pozitronik/steelclock-go/internal/widget/keyboard"
	"github.com/pozitronik/steelclock-go/internal/widget/memory"
	"github.com/pozitronik/steelclock-go/internal/widget/network"
	"github.com/pozitronik/steelclock-go/internal/widget/volume"
)

// =============================================================================
// Clock Widget Mode Tests
// =============================================================================

// TestClockWidget_TextMode tests clock widget with text format
func TestClockWidget_TextMode(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		minPixels  int
		shouldRend bool
	}{
		{"HoursMinutesSeconds", "%H:%M:%S", 80, true},
		{"HoursMinutes", "%H:%M", 50, true},
		{"DateOnly", "%Y-%m-%d", 60, true},
		{"FullDateTime", "%Y-%m-%d %H:%M", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-clock",
				Type:     "clock",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Text: &config.TextConfig{
					Format: tt.format,
				},
			}

			clockWidget, err := clock.New(widgetCfg)
			if err != nil {
				t.Fatalf("Failed to create clock widget: %v", err)
			}

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

			_ = comp.Start()
			_ = client.WaitForFrames(3, 2*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			if tt.shouldRend && testutil.IsBlankFrame(frame.Data) {
				t.Error("Clock widget should render non-blank frame")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("Format '%s' rendered %d pixels", tt.format, pixelCount)

			if pixelCount < tt.minPixels {
				t.Errorf("Expected at least %d pixels, got %d", tt.minPixels, pixelCount)
			}
		})
	}
}

// TestClockWidget_AnalogMode tests clock widget in analog mode
func TestClockWidget_AnalogMode(t *testing.T) {
	tests := []struct {
		name        string
		showSeconds bool
		showTicks   bool
	}{
		{"BasicAnalog", false, false},
		{"WithSeconds", true, false},
		{"WithTicks", false, true},
		{"FullAnalog", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-clock",
				Type:     "clock",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 40, H: 40},
				Mode:     "analog",
				Analog: &config.AnalogConfig{
					ShowSeconds: tt.showSeconds,
					ShowTicks:   tt.showTicks,
				},
			}

			clockWidget, err := clock.New(widgetCfg)
			if err != nil {
				t.Fatalf("Failed to create clock widget: %v", err)
			}

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

			_ = comp.Start()
			_ = client.WaitForFrames(3, 2*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			if testutil.IsBlankFrame(frame.Data) {
				t.Error("Analog clock should render non-blank frame")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("Analog clock (%s) rendered %d pixels", tt.name, pixelCount)

			// Analog clock should have clock hands at minimum
			if pixelCount < 20 {
				t.Errorf("Analog clock should render more than 20 pixels, got %d", pixelCount)
			}
		})
	}
}

// =============================================================================
// Memory Widget Mode Tests
// =============================================================================

// TestMemoryWidget_BarMode tests memory widget in bar mode with different directions
func TestMemoryWidget_BarMode(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		border    bool
	}{
		{"HorizontalNoBorder", "horizontal", false},
		{"HorizontalWithBorder", "horizontal", true},
		{"VerticalNoBorder", "vertical", false},
		{"VerticalWithBorder", "vertical", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-mem",
				Type:     "memory",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Mode:     "bar",
				Bar: &config.BarConfig{
					Direction: tt.direction,
					Border:    tt.border,
				},
			}

			memWidget, err := memory.New(widgetCfg)
			if err != nil {
				t.Fatalf("Failed to create memory widget: %v", err)
			}

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

			_ = comp.Start()
			_ = client.WaitForFrames(3, 2*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			if testutil.IsBlankFrame(frame.Data) {
				t.Error("Memory bar should render non-blank frame")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("Memory bar %s rendered %d pixels", tt.name, pixelCount)

			// If border is enabled, we expect more pixels
			if tt.border && pixelCount < 100 {
				t.Errorf("Memory bar with border should render more pixels, got %d", pixelCount)
			}
		})
	}
}

// TestMemoryWidget_GraphMode tests memory widget in graph mode
func TestMemoryWidget_GraphMode(t *testing.T) {
	tests := []struct {
		name      string
		fillColor int // -1 = line only, 0-255 = fill color
		history   int
	}{
		{"FilledDefault", 255, 0},
		{"LineOnly", -1, 0},
		{"FilledHistory50", 255, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-mem",
				Type:     "memory",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Mode:     "graph",
				Graph: &config.GraphConfig{
					History: tt.history,
					Colors: &config.ModeColorsConfig{
						Fill: &tt.fillColor,
					},
				},
			}

			memWidget, err := memory.New(widgetCfg)
			if err != nil {
				t.Fatalf("Failed to create memory widget: %v", err)
			}

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

			_ = comp.Start()
			// Graph needs more frames to show history
			_ = client.WaitForFrames(10, 3*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("Memory graph %s rendered %d pixels", tt.name, pixelCount)

			// Graph may render 0 pixels if history hasn't built up yet
			// This is expected behavior for graphs with short run times
			if pixelCount == 0 {
				t.Logf("Graph rendered 0 pixels - may need more time to build history")
			}
		})
	}
}

// TestMemoryWidget_TextMode tests memory widget in text mode
func TestMemoryWidget_TextMode(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-mem",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "text",
		Text: &config.TextConfig{
			Format: "%.1f%%",
		},
	}

	memWidget, err := memory.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create memory widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	if testutil.IsBlankFrame(frame.Data) {
		t.Error("Memory text should render non-blank frame")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Memory text rendered %d pixels", pixelCount)
}

// =============================================================================
// CPU Widget Mode Tests
// =============================================================================

// TestCPUWidget_BarMode tests CPU widget in bar mode
func TestCPUWidget_BarMode(t *testing.T) {
	tests := []struct {
		name      string
		direction string
	}{
		{"Horizontal", "horizontal"},
		{"Vertical", "vertical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-cpu",
				Type:     "cpu",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Mode:     "bar",
				Bar: &config.BarConfig{
					Direction: tt.direction,
				},
			}

			cpuWidget, err := cpu.New(widgetCfg)
			if err != nil {
				t.Fatalf("Failed to create CPU widget: %v", err)
			}

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{cpuWidget}, cfg)

			_ = comp.Start()
			// CPU needs time to collect stats
			_ = client.WaitForFrames(5, 3*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("CPU bar %s rendered %d pixels", tt.name, pixelCount)
		})
	}
}

// TestCPUWidget_GaugeMode tests CPU widget in gauge mode
func TestCPUWidget_GaugeMode(t *testing.T) {
	showTicks := true
	widgetCfg := config.WidgetConfig{
		ID:       "test-cpu",
		Type:     "cpu",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 40, H: 40},
		Mode:     "gauge",
		Gauge: &config.GaugeConfig{
			ShowTicks: &showTicks,
		},
	}

	cpuWidget, err := cpu.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create CPU widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{cpuWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(5, 3*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("CPU gauge rendered %d pixels", pixelCount)

	// Gauge should render arc at minimum
	if pixelCount < 30 {
		t.Errorf("CPU gauge should render arc, got only %d pixels", pixelCount)
	}
}

// TestCPUWidget_PerCore tests CPU widget with per-core display
func TestCPUWidget_PerCore(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-cpu",
		Type:     "cpu",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "vertical",
		},
		PerCore: &config.PerCoreConfig{
			Enabled: true,
			Margin:  1,
			Border:  true,
		},
	}

	cpuWidget, err := cpu.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create CPU widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{cpuWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(5, 3*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("CPU per-core rendered %d pixels", pixelCount)
}

// =============================================================================
// Network Widget Mode Tests
// =============================================================================

// TestNetworkWidget_GraphMode tests network widget with dual graph
func TestNetworkWidget_GraphMode(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-net",
		Type:     "network",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "graph",
		Graph: &config.GraphConfig{
			History: 128,
		},
	}

	netWidget, err := network.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create network widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{netWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(5, 3*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Network graph rendered %d pixels", pixelCount)
}

// TestNetworkWidget_GaugeMode tests network widget in gauge mode
func TestNetworkWidget_GaugeMode(t *testing.T) {
	showTicks := true
	widgetCfg := config.WidgetConfig{
		ID:           "test-net",
		Type:         "network",
		Enabled:      boolPtr(true),
		Position:     config.PositionConfig{X: 0, Y: 0, W: 80, H: 40},
		Mode:         "gauge",
		MaxSpeedMbps: 100.0,
		Gauge: &config.GaugeConfig{
			ShowTicks: &showTicks,
		},
	}

	netWidget, err := network.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create network widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{netWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(5, 3*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Network gauge rendered %d pixels", pixelCount)
}

// =============================================================================
// Volume Widget Mode Tests
// =============================================================================

// TestVolumeWidget_BarMode tests volume widget in bar mode
func TestVolumeWidget_BarMode(t *testing.T) {
	tests := []struct {
		name      string
		direction string
	}{
		{"Horizontal", "horizontal"},
		{"Vertical", "vertical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-vol",
				Type:     "volume",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Mode:     "bar",
				Bar: &config.BarConfig{
					Direction: tt.direction,
				},
			}

			volWidget, err := volume.New(widgetCfg)
			if err != nil {
				t.Skipf("Volume widget not available: %v", err)
			}

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{volWidget}, cfg)

			_ = comp.Start()
			_ = client.WaitForFrames(3, 2*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("Volume bar %s rendered %d pixels", tt.name, pixelCount)
		})
	}
}

// TestVolumeWidget_GaugeMode tests volume widget in gauge mode
func TestVolumeWidget_GaugeMode(t *testing.T) {
	showTicks := true
	widgetCfg := config.WidgetConfig{
		ID:       "test-vol",
		Type:     "volume",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 40, H: 40},
		Mode:     "gauge",
		Gauge: &config.GaugeConfig{
			ShowTicks: &showTicks,
		},
	}

	volWidget, err := volume.New(widgetCfg)
	if err != nil {
		t.Skipf("Volume widget not available: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{volWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Volume gauge rendered %d pixels", pixelCount)
}

// =============================================================================
// Keyboard Widget Tests
// =============================================================================

// TestKeyboardWidget_WithIndicators tests keyboard widget with custom indicators
func TestKeyboardWidget_WithIndicators(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Keyboard widget is only supported on Windows")
	}

	capsOn := "CAPS"
	capsOff := ""
	numOn := "NUM"
	numOff := ""
	scrollOn := "SCROLL"
	scrollOff := ""

	widgetCfg := config.WidgetConfig{
		ID:       "test-kbd",
		Type:     "keyboard",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Indicators: &config.IndicatorsConfig{
			Caps:   &config.IndicatorConfig{On: &capsOn, Off: &capsOff},
			Num:    &config.IndicatorConfig{On: &numOn, Off: &numOff},
			Scroll: &config.IndicatorConfig{On: &scrollOn, Off: &scrollOff},
		},
	}

	kbdWidget, err := keyboard.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create keyboard widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{kbdWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Keyboard widget rendered %d pixels (depends on lock states)", pixelCount)
}

// TestKeyboardWidget_WithLayout tests keyboard widget with layout settings
func TestKeyboardWidget_WithLayout(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Keyboard widget is only supported on Windows")
	}

	capsOn := "C"
	numOn := "N"

	widgetCfg := config.WidgetConfig{
		ID:       "test-kbd",
		Type:     "keyboard",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Indicators: &config.IndicatorsConfig{
			Caps: &config.IndicatorConfig{On: &capsOn},
			Num:  &config.IndicatorConfig{On: &numOn},
		},
		Layout: &config.KeyboardLayout{
			Spacing:   5,
			Separator: " | ",
		},
	}

	kbdWidget, err := keyboard.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create keyboard widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{kbdWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Keyboard with layout rendered %d pixels", pixelCount)
}

// =============================================================================
// Disk Widget Mode Tests
// =============================================================================

// TestDiskWidget_BarMode tests disk widget in bar mode
func TestDiskWidget_BarMode(t *testing.T) {
	diskPath := "/"
	widgetCfg := config.WidgetConfig{
		ID:       "test-disk",
		Type:     "disk",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Disk:     &diskPath,
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Border:    true,
		},
	}

	diskWidget, err := disk.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create disk widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{diskWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Disk bar rendered %d pixels", pixelCount)
}

// TestDiskWidget_TextMode tests disk widget in text mode
func TestDiskWidget_TextMode(t *testing.T) {
	diskPath := "/"
	widgetCfg := config.WidgetConfig{
		ID:       "test-disk",
		Type:     "disk",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "text",
		Disk:     &diskPath,
		Text: &config.TextConfig{
			Format: "%.1f%%",
		},
	}

	diskWidget, err := disk.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create disk widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{diskWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Disk text rendered %d pixels", pixelCount)
}
