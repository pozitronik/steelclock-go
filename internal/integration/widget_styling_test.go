// Package integration contains integration tests that span multiple packages
package integration

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/testutil"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// =============================================================================
// Background Color Tests
// =============================================================================

// TestStyle_BackgroundBlack tests widget with black background (default)
func TestStyle_BackgroundBlack(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
		Style:    &config.StyleConfig{Background: 0},
	}

	memWidget, _ := widget.NewMemoryWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Black background: %d set pixels", pixelCount)

	// Background is black, so only bar pixels should be set
	totalPixels := 128 * 40
	if pixelCount > totalPixels/2 {
		t.Logf("More than half pixels set - might be using white background instead")
	}
}

// TestStyle_BackgroundWhite tests widget with white background
func TestStyle_BackgroundWhite(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Text:     &config.TextConfig{Format: "HI"},
		Style:    &config.StyleConfig{Background: 255},
	}

	clockWidget, _ := widget.NewClockWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("White background: %d set pixels", pixelCount)

	// With white background, most pixels should be set
	totalPixels := 128 * 40
	if pixelCount < totalPixels/2 {
		t.Logf("Less than half pixels set - background might not be rendering")
	}
}

// TestStyle_BackgroundTransparent tests transparent background
func TestStyle_BackgroundTransparent(t *testing.T) {
	// Background widget
	bgCfg := config.WidgetConfig{
		ID:       "bg",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	// Foreground with transparent background
	fgCfg := config.WidgetConfig{
		ID:       "fg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 1},
		Text:     &config.TextConfig{Format: "OVERLAY"},
		Style:    &config.StyleConfig{Background: -1}, // Transparent
	}

	bgWidget, _ := widget.NewMemoryWidget(bgCfg)
	fgWidget, _ := widget.NewClockWidget(fgCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{bgWidget, fgWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Transparent overlay: %d pixels (should have both widget content)", pixelCount)
}

// TestStyle_BackgroundGray tests widget with gray background
func TestStyle_BackgroundGray(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Text:     &config.TextConfig{Format: "GRAY"},
		Style:    &config.StyleConfig{Background: 128},
	}

	clockWidget, _ := widget.NewClockWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Check if frame is not blank - gray background should still produce non-blank
	if testutil.IsBlankFrame(frame.Data) {
		t.Error("Gray background should produce non-blank frame")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Gray background (128): %d set pixels", pixelCount)
}

// =============================================================================
// Border Tests
// =============================================================================

// TestStyle_BorderEnabled tests widget with border
func TestStyle_BorderEnabled(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 10, Y: 5, W: 108, H: 30},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Border:    true,
		},
	}

	memWidget, _ := widget.NewMemoryWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Bar with border: %d pixels", pixelCount)

	// Border should add perimeter pixels
	// Perimeter = 2*(w+h) - 4 corners = 2*(108+30) - 4 = 272
	// But border is internal to bar, so this is approximate
}

// TestStyle_BorderDisabled tests widget without border
func TestStyle_BorderDisabled(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 10, Y: 5, W: 108, H: 30},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Border:    false,
		},
	}

	memWidget, _ := widget.NewMemoryWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Bar without border: %d pixels", pixelCount)
}

// TestStyle_BorderComparison compares bar with and without border
func TestStyle_BorderComparison(t *testing.T) {
	// Widget with border
	withBorderCfg := config.WidgetConfig{
		ID:       "with-border",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Border:    true,
		},
	}

	withBorderWidget, _ := widget.NewMemoryWidget(withBorderCfg)
	cfg := createTestConfig()
	client1, comp1 := createTestSetup([]widget.Widget{withBorderWidget}, cfg)

	_ = comp1.Start()
	_ = client1.WaitForFrames(3, 2*time.Second)
	comp1.Stop()

	frame1 := client1.LastFrame()
	if frame1 == nil {
		t.Fatal("No frame captured for bordered widget")
	}
	pixelsWithBorder := testutil.CountSetPixels(frame1.Data)

	// Widget without border
	withoutBorderCfg := config.WidgetConfig{
		ID:       "without-border",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Border:    false,
		},
	}

	withoutBorderWidget, _ := widget.NewMemoryWidget(withoutBorderCfg)
	client2, comp2 := createTestSetup([]widget.Widget{withoutBorderWidget}, cfg)

	_ = comp2.Start()
	_ = client2.WaitForFrames(3, 2*time.Second)
	comp2.Stop()

	frame2 := client2.LastFrame()
	if frame2 == nil {
		t.Fatal("No frame captured for borderless widget")
	}
	pixelsWithoutBorder := testutil.CountSetPixels(frame2.Data)

	t.Logf("With border: %d pixels, Without border: %d pixels", pixelsWithBorder, pixelsWithoutBorder)

	// Bordered should have more pixels (the border itself)
	if pixelsWithBorder <= pixelsWithoutBorder {
		t.Log("Border should add pixels - may depend on memory level")
	}
}

// =============================================================================
// Color Configuration Tests
// =============================================================================

// TestStyle_BarFillColor tests bar with custom fill color
func TestStyle_BarFillColor(t *testing.T) {
	fillColor := 200
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Colors: &config.ModeColorsConfig{
				Fill: &fillColor,
			},
		},
	}

	memWidget, _ := widget.NewMemoryWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Custom fill color (200): %d pixels", pixelCount)
}

// TestStyle_GraphColors tests graph with custom colors
func TestStyle_GraphColors(t *testing.T) {
	fillColor := 180
	lineColor := 255
	filled := true
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "graph",
		Graph: &config.GraphConfig{
			Filled:  &filled,
			History: 128,
			Colors: &config.ModeColorsConfig{
				Fill: &fillColor,
				Line: &lineColor,
			},
		},
	}

	memWidget, _ := widget.NewMemoryWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(10, 3*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Graph with custom colors: %d pixels", pixelCount)
}

// TestStyle_GaugeColors tests gauge with custom colors
func TestStyle_GaugeColors(t *testing.T) {
	arcColor := 100
	needleColor := 255
	showTicks := true
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "cpu",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 40, H: 40},
		Mode:     "gauge",
		Gauge: &config.GaugeConfig{
			ShowTicks: &showTicks,
			Colors: &config.ModeColorsConfig{
				Arc:    &arcColor,
				Needle: &needleColor,
			},
		},
	}

	cpuWidget, _ := widget.NewCPUWidget(widgetCfg)

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
	t.Logf("Gauge with custom colors: %d pixels", pixelCount)
}

// =============================================================================
// Text Styling Tests
// =============================================================================

// TestStyle_TextAlignment tests text alignment options
func TestStyle_TextAlignment(t *testing.T) {
	alignments := []struct {
		name string
		h    string
		v    string
	}{
		{"TopLeft", "left", "top"},
		{"TopCenter", "center", "top"},
		{"TopRight", "right", "top"},
		{"CenterLeft", "left", "center"},
		{"Center", "center", "center"},
		{"CenterRight", "right", "center"},
		{"BottomLeft", "left", "bottom"},
		{"BottomCenter", "center", "bottom"},
		{"BottomRight", "right", "bottom"},
	}

	for _, a := range alignments {
		t.Run(a.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test",
				Type:     "clock",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Text: &config.TextConfig{
					Format: "X",
					Align: &config.AlignConfig{
						H: a.h,
						V: a.v,
					},
				},
			}

			clockWidget, _ := widget.NewClockWidget(widgetCfg)

			cfg := createTestConfig()
			client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

			_ = comp.Start()
			_ = client.WaitForFrames(2, 2*time.Second)
			comp.Stop()

			frame := client.LastFrame()
			if frame == nil {
				t.Fatal("No frame captured")
			}

			if testutil.IsBlankFrame(frame.Data) {
				t.Error("Text should render")
			}

			pixelCount := testutil.CountSetPixels(frame.Data)
			t.Logf("Alignment %s: %d pixels", a.name, pixelCount)
		})
	}
}

// =============================================================================
// Display Background Tests
// =============================================================================

// TestStyle_DisplayBackgroundBlack tests display with black background
func TestStyle_DisplayBackgroundBlack(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 32, Y: 10, W: 64, H: 20},
		Text:     &config.TextConfig{Format: "HI"},
	}

	clockWidget, _ := widget.NewClockWidget(widgetCfg)

	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:      128,
			Height:     40,
			Background: 0, // Black display background
		},
	}
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Black display background: %d pixels", pixelCount)

	// Only widget content should be visible
	totalPixels := 128 * 40
	if pixelCount > totalPixels/4 {
		t.Log("More pixels than expected for small centered widget")
	}
}

// TestStyle_DisplayBackgroundWhite tests display with white background
func TestStyle_DisplayBackgroundWhite(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 32, Y: 10, W: 64, H: 20},
		Text:     &config.TextConfig{Format: "HI"},
	}

	clockWidget, _ := widget.NewClockWidget(widgetCfg)

	cfg := &config.Config{
		RefreshRateMs: 50,
		Display: config.DisplayConfig{
			Width:      128,
			Height:     40,
			Background: 255, // White display background
		},
	}
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("White display background: %d pixels", pixelCount)

	// Most of display should be white
	totalPixels := 128 * 40
	if pixelCount < totalPixels/2 {
		t.Log("Less pixels than expected for white background")
	}
}

// =============================================================================
// Combined Styling Tests
// =============================================================================

// TestStyle_ComplexStyling tests widget with multiple style options
func TestStyle_ComplexStyling(t *testing.T) {
	fillColor := 200
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 10, Y: 5, W: 108, H: 30},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
			Border:    true,
			Colors: &config.ModeColorsConfig{
				Fill: &fillColor,
			},
		},
		Style: &config.StyleConfig{
			Background: 50,
			Padding:    2,
		},
	}

	memWidget, _ := widget.NewMemoryWidget(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Complex styling: %d pixels", pixelCount)
}

// TestStyle_MultipleStyledWidgets tests multiple widgets with different styles
func TestStyle_MultipleStyledWidgets(t *testing.T) {
	// Widget 1: White on black
	cfg1 := config.WidgetConfig{
		ID:       "w1",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 64, H: 20, Z: 0},
		Text:     &config.TextConfig{Format: "A"},
		Style:    &config.StyleConfig{Background: 0},
	}

	// Widget 2: Black on white (inverted)
	cfg2 := config.WidgetConfig{
		ID:       "w2",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 64, Y: 0, W: 64, H: 20, Z: 0},
		Text:     &config.TextConfig{Format: "B"},
		Style:    &config.StyleConfig{Background: 255},
	}

	// Widget 3: Gray background
	cfg3 := config.WidgetConfig{
		ID:       "w3",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 20, W: 64, H: 20, Z: 0},
		Text:     &config.TextConfig{Format: "C"},
		Style:    &config.StyleConfig{Background: 128},
	}

	// Widget 4: Transparent overlay
	cfg4 := config.WidgetConfig{
		ID:       "w4",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 64, Y: 20, W: 64, H: 20, Z: 0},
		Text:     &config.TextConfig{Format: "D"},
		Style:    &config.StyleConfig{Background: -1},
	}

	w1, _ := widget.NewClockWidget(cfg1)
	w2, _ := widget.NewClockWidget(cfg2)
	w3, _ := widget.NewClockWidget(cfg3)
	w4, _ := widget.NewClockWidget(cfg4)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{w1, w2, w3, w4}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Check each quadrant
	blank := make([]int, 640)
	quadrants := []struct {
		name string
		x, y int
	}{
		{"TopLeft-Black", 0, 0},
		{"TopRight-White", 64, 0},
		{"BottomLeft-Gray", 0, 20},
		{"BottomRight-Transparent", 64, 20},
	}

	for _, q := range quadrants {
		diff := testutil.CompareRegion(frame.Data, blank, q.x, q.y, 64, 20)
		t.Logf("%s: %d pixels", q.name, diff.DifferentPixels)
	}
}
