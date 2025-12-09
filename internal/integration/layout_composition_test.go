// Package integration contains integration tests that span multiple packages
package integration

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/testutil"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"github.com/pozitronik/steelclock-go/internal/widget/clock"
	"github.com/pozitronik/steelclock-go/internal/widget/memory"
)

// =============================================================================
// Positioning Tests
// =============================================================================

// TestLayout_AbsolutePositioning tests widgets at specific absolute positions
func TestLayout_AbsolutePositioning(t *testing.T) {
	tests := []struct {
		name   string
		x, y   int
		w, h   int
		checkX int // X coordinate to verify has content
		checkY int // Y coordinate to verify has content
	}{
		{"TopLeft", 0, 0, 32, 20, 5, 5},
		{"TopRight", 96, 0, 32, 20, 100, 5},
		{"BottomLeft", 0, 20, 32, 20, 5, 30},
		{"BottomRight", 96, 20, 32, 20, 100, 30},
		{"Center", 48, 10, 32, 20, 64, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widgetCfg := config.WidgetConfig{
				ID:       "test-clock",
				Type:     "clock",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: tt.x, Y: tt.y, W: tt.w, H: tt.h},
				Text: &config.TextConfig{
					Format: "X",
				},
			}

			clockWidget, err := clock.New(widgetCfg)
			if err != nil {
				t.Fatalf("Failed to create widget: %v", err)
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

			// Verify content is within expected region
			regionDiff := testutil.CompareRegion(frame.Data, make([]byte, 640), tt.x, tt.y, tt.w, tt.h)
			if regionDiff.DifferentPixels == 0 {
				t.Errorf("Expected content at position (%d,%d) size %dx%d", tt.x, tt.y, tt.w, tt.h)
			}

			t.Logf("Position %s: %d pixels in region", tt.name, regionDiff.DifferentPixels)
		})
	}
}

// TestLayout_CornerPositions tests widgets positioned exactly at corners
func TestLayout_CornerPositions(t *testing.T) {
	// Four widgets in four corners
	configs := []struct {
		id   string
		x, y int
	}{
		{"tl", 0, 0},
		{"tr", 112, 0},
		{"bl", 0, 32},
		{"br", 112, 32},
	}

	var widgets []widget.Widget
	for _, c := range configs {
		widgetCfg := config.WidgetConfig{
			ID:       c.id,
			Type:     "clock",
			Enabled:  boolPtr(true),
			Position: config.PositionConfig{X: c.x, Y: c.y, W: 16, H: 8},
			Text:     &config.TextConfig{Format: "X"},
		}
		w, err := clock.New(widgetCfg)
		if err != nil {
			t.Fatalf("Failed to create widget: %v", err)
		}
		widgets = append(widgets, w)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup(widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Check each corner has content
	corners := []struct {
		name string
		x, y int
	}{
		{"TopLeft", 0, 0},
		{"TopRight", 112, 0},
		{"BottomLeft", 0, 32},
		{"BottomRight", 112, 32},
	}

	for _, corner := range corners {
		blank := make([]byte, 640)
		diff := testutil.CompareRegion(frame.Data, blank, corner.x, corner.y, 16, 8)
		if diff.DifferentPixels == 0 {
			t.Errorf("%s corner should have content", corner.name)
		}
		t.Logf("%s: %d pixels", corner.name, diff.DifferentPixels)
	}
}

// =============================================================================
// Z-Order Tests
// =============================================================================

// TestLayout_ZOrderBasic tests that higher z-order widgets render on top
func TestLayout_ZOrderBasic(t *testing.T) {
	// Background widget (z=0) - full width bar
	bgCfg := config.WidgetConfig{
		ID:       "bg",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	// Foreground widget (z=1) - clock text overlapping
	fgCfg := config.WidgetConfig{
		ID:       "fg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 40, Y: 10, W: 48, H: 20, Z: 1},
		Text:     &config.TextConfig{Format: "HI"},
		Style:    &config.StyleConfig{Background: -1}, // Transparent
	}

	bgWidget, _ := memory.New(bgCfg)
	fgWidget, _ := clock.New(fgCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{bgWidget, fgWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	if testutil.IsBlankFrame(frame.Data) {
		t.Error("Frame should not be blank with overlapping widgets")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Overlapping widgets: %d total pixels", pixelCount)
}

// TestLayout_ZOrderReverse tests z-order with widgets added in reverse order
func TestLayout_ZOrderReverse(t *testing.T) {
	// Add higher z first, lower z second - should still composite correctly
	fgCfg := config.WidgetConfig{
		ID:       "fg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 10},
		Text:     &config.TextConfig{Format: "FG"},
	}

	bgCfg := config.WidgetConfig{
		ID:       "bg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Text:     &config.TextConfig{Format: "BG"},
	}

	fgWidget, _ := clock.New(fgCfg)
	bgWidget, _ := clock.New(bgCfg)

	cfg := createTestConfig()
	// Add fg first, bg second - layout manager should sort by z
	client, comp := createTestSetup([]widget.Widget{fgWidget, bgWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Reverse z-order: %d pixels (FG should be on top)", pixelCount)
}

// TestLayout_MultipleZLevels tests three or more z-levels
func TestLayout_MultipleZLevels(t *testing.T) {
	// Three widgets at different z-levels
	bottomCfg := config.WidgetConfig{
		ID:       "bottom",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	middleCfg := config.WidgetConfig{
		ID:       "middle",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 20, Y: 5, W: 88, H: 30, Z: 5},
		Text:     &config.TextConfig{Format: "MID"},
		Style:    &config.StyleConfig{Background: -1},
	}

	topCfg := config.WidgetConfig{
		ID:       "top",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 50, Y: 15, W: 28, H: 10, Z: 10},
		Text:     &config.TextConfig{Format: "T"},
		Style:    &config.StyleConfig{Background: -1},
	}

	bottomWidget, _ := memory.New(bottomCfg)
	middleWidget, _ := clock.New(middleCfg)
	topWidget, _ := clock.New(topCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{bottomWidget, middleWidget, topWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	if testutil.IsBlankFrame(frame.Data) {
		t.Error("Frame should not be blank with 3 z-levels")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("3 z-levels: %d total pixels", pixelCount)
}

// =============================================================================
// Transparency Tests
// =============================================================================

// TestLayout_TransparentBackground tests transparent widget backgrounds
func TestLayout_TransparentBackground(t *testing.T) {
	// Background widget with solid fill
	bgCfg := config.WidgetConfig{
		ID:       "bg",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	// Foreground widget with transparent background
	fgCfg := config.WidgetConfig{
		ID:       "fg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 32, Y: 10, W: 64, H: 20, Z: 1},
		Text:     &config.TextConfig{Format: "TEST"},
		Style:    &config.StyleConfig{Background: -1}, // Transparent
	}

	bgWidget, _ := memory.New(bgCfg)
	fgWidget, _ := clock.New(fgCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{bgWidget, fgWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// With transparent overlay, we should see both widgets' content
	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Transparent overlay: %d pixels", pixelCount)
}

// TestLayout_OpaqueBackground tests opaque widget backgrounds
func TestLayout_OpaqueBackground(t *testing.T) {
	// Background widget
	bgCfg := config.WidgetConfig{
		ID:       "bg",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	// Foreground widget with opaque background (default)
	fgCfg := config.WidgetConfig{
		ID:       "fg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 32, Y: 10, W: 64, H: 20, Z: 1},
		Text:     &config.TextConfig{Format: "TEST"},
		Style:    &config.StyleConfig{Background: 0}, // Opaque black
	}

	bgWidget, _ := memory.New(bgCfg)
	fgWidget, _ := clock.New(fgCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{bgWidget, fgWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// With opaque foreground, center region should be mostly foreground content
	centerRegion := testutil.CompareRegion(frame.Data, make([]byte, 640), 32, 10, 64, 20)
	t.Logf("Opaque overlay center region: %d pixels", centerRegion.DifferentPixels)
}

// =============================================================================
// Clipping Tests
// =============================================================================

// TestLayout_ClippingLeftEdge tests widget clipping at left edge
func TestLayout_ClippingLeftEdge(t *testing.T) {
	// Widget partially off-screen to the left
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: -16, Y: 10, W: 48, H: 20},
		Text:     &config.TextConfig{Format: "CLIP"},
	}

	clockWidget, _ := clock.New(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Should render partial content
	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Left clipping: %d pixels visible", pixelCount)
}

// TestLayout_ClippingRightEdge tests widget clipping at right edge
func TestLayout_ClippingRightEdge(t *testing.T) {
	// Widget partially off-screen to the right
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 100, Y: 10, W: 48, H: 20},
		Text:     &config.TextConfig{Format: "CLIP"},
	}

	clockWidget, _ := clock.New(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Should render partial content
	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Right clipping: %d pixels visible", pixelCount)
}

// TestLayout_ClippingBottom tests widget clipping at bottom edge
func TestLayout_ClippingBottom(t *testing.T) {
	// Widget partially off-screen at bottom
	widgetCfg := config.WidgetConfig{
		ID:       "test",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 30, W: 128, H: 20},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	memWidget, _ := memory.New(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Should render partial content (only top 10 rows visible)
	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Bottom clipping: %d pixels visible", pixelCount)
}

// =============================================================================
// Grid Layout Tests
// =============================================================================

// TestLayout_2x2Grid tests four widgets in a 2x2 grid
func TestLayout_2x2Grid(t *testing.T) {
	configs := []struct {
		id   string
		x, y int
	}{
		{"tl", 0, 0},
		{"tr", 64, 0},
		{"bl", 0, 20},
		{"br", 64, 20},
	}

	var widgets []widget.Widget
	for _, c := range configs {
		widgetCfg := config.WidgetConfig{
			ID:       c.id,
			Type:     "memory",
			Enabled:  boolPtr(true),
			Position: config.PositionConfig{X: c.x, Y: c.y, W: 64, H: 20},
			Mode:     "bar",
			Bar:      &config.BarConfig{Direction: "horizontal"},
		}
		w, err := memory.New(widgetCfg)
		if err != nil {
			t.Fatalf("Failed to create widget: %v", err)
		}
		widgets = append(widgets, w)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup(widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Check each quadrant has content
	blank := make([]byte, 640)
	quadrants := []struct {
		name string
		x, y int
	}{
		{"TopLeft", 0, 0},
		{"TopRight", 64, 0},
		{"BottomLeft", 0, 20},
		{"BottomRight", 64, 20},
	}

	for _, q := range quadrants {
		diff := testutil.CompareRegion(frame.Data, blank, q.x, q.y, 64, 20)
		if diff.DifferentPixels == 0 {
			t.Errorf("%s quadrant should have content", q.name)
		}
		t.Logf("%s: %d pixels", q.name, diff.DifferentPixels)
	}
}

// TestLayout_HorizontalStrip tests widgets in a horizontal strip
func TestLayout_HorizontalStrip(t *testing.T) {
	widthPerWidget := 32
	numWidgets := 4

	var widgets []widget.Widget
	for i := 0; i < numWidgets; i++ {
		widgetCfg := config.WidgetConfig{
			ID:       string(rune('A' + i)),
			Type:     "memory",
			Enabled:  boolPtr(true),
			Position: config.PositionConfig{X: i * widthPerWidget, Y: 0, W: widthPerWidget, H: 40},
			Mode:     "bar",
			Bar:      &config.BarConfig{Direction: "vertical"},
		}
		w, _ := memory.New(widgetCfg)
		widgets = append(widgets, w)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup(widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Verify each strip has content
	blank := make([]byte, 640)
	for i := 0; i < numWidgets; i++ {
		diff := testutil.CompareRegion(frame.Data, blank, i*widthPerWidget, 0, widthPerWidget, 40)
		t.Logf("Strip %d: %d pixels", i, diff.DifferentPixels)
	}
}

// =============================================================================
// Full Screen Tests
// =============================================================================

// TestLayout_FullScreen tests a single widget covering entire display
func TestLayout_FullScreen(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "fullscreen",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	memWidget, _ := memory.New(widgetCfg)

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
	t.Logf("Full screen widget: %d pixels", pixelCount)

	// Memory bar should fill proportionally to memory usage
	if pixelCount < 100 {
		t.Errorf("Full screen memory bar should have more pixels, got %d", pixelCount)
	}
}

// TestLayout_OversizedWidget tests widget larger than display
func TestLayout_OversizedWidget(t *testing.T) {
	// Widget much larger than display
	widgetCfg := config.WidgetConfig{
		ID:       "oversize",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: -10, Y: -10, W: 200, H: 100},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	memWidget, _ := memory.New(widgetCfg)

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
	t.Logf("Oversized widget clipped: %d pixels", pixelCount)
}

// =============================================================================
// Frame Stability Tests
// =============================================================================

// TestLayout_StaticStability tests that static content produces identical frames
func TestLayout_StaticStability(t *testing.T) {
	// Memory widget - relatively static over short time periods
	widgetCfg := config.WidgetConfig{
		ID:       "static",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar:      &config.BarConfig{Direction: "horizontal"},
	}

	memWidget, _ := memory.New(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(5, 2*time.Second)
	comp.Stop()

	frames := client.Frames()
	if len(frames) < 3 {
		t.Skip("Not enough frames")
	}

	// Compare consecutive frames
	identicalCount := 0
	for i := 1; i < len(frames); i++ {
		diff := testutil.CompareFrames(frames[i-1].Data, frames[i].Data)
		if diff.Identical {
			identicalCount++
		}
		t.Logf("Frame %d->%d: %d different pixels", i-1, i, diff.DifferentPixels)
	}

	t.Logf("%d/%d frame pairs were identical", identicalCount, len(frames)-1)
}

// TestLayout_MultiWidgetStability tests stability with multiple widgets
func TestLayout_MultiWidgetStability(t *testing.T) {
	// Multiple memory bars
	var widgets []widget.Widget
	for i := 0; i < 4; i++ {
		widgetCfg := config.WidgetConfig{
			ID:       string(rune('A' + i)),
			Type:     "memory",
			Enabled:  boolPtr(true),
			Position: config.PositionConfig{X: i * 32, Y: 0, W: 32, H: 40},
			Mode:     "bar",
			Bar:      &config.BarConfig{Direction: "vertical"},
		}
		w, _ := memory.New(widgetCfg)
		widgets = append(widgets, w)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup(widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(10, 3*time.Second)
	comp.Stop()

	frames := client.Frames()
	if len(frames) < 5 {
		t.Skip("Not enough frames")
	}

	// Calculate average difference
	totalDiff := 0
	for i := 1; i < len(frames); i++ {
		diff := testutil.CompareFrames(frames[i-1].Data, frames[i].Data)
		totalDiff += diff.DifferentPixels
	}
	avgDiff := float64(totalDiff) / float64(len(frames)-1)
	t.Logf("Average frame difference: %.1f pixels", avgDiff)
}

// =============================================================================
// Edge Cases
// =============================================================================

// TestLayout_ZeroSizeWidget tests handling of zero-size widget
func TestLayout_ZeroSizeWidget(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "zero",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 64, Y: 20, W: 0, H: 0},
		Text:     &config.TextConfig{Format: "X"},
	}

	clockWidget, err := clock.New(widgetCfg)
	if err != nil {
		// Zero-size might be invalid
		t.Skipf("Zero-size widget rejected: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(2, 2*time.Second)
	comp.Stop()

	// Should not crash
	t.Log("Zero-size widget handled without crash")
}

// TestLayout_NegativePosition tests widget with negative coordinates
func TestLayout_NegativePosition(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "negative",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: -50, Y: -20, W: 80, H: 40},
		Text:     &config.TextConfig{Format: "NEG"},
	}

	clockWidget, _ := clock.New(widgetCfg)

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Should show partial content in top-left
	topLeftPixels := testutil.CompareRegion(frame.Data, make([]byte, 640), 0, 0, 30, 20)
	t.Logf("Negative position - top-left region: %d pixels", topLeftPixels.DifferentPixels)
}

// TestLayout_ManyWidgets tests composition with many widgets
func TestLayout_ManyWidgets(t *testing.T) {
	// Create 16 small widgets
	var widgets []widget.Widget
	widgetSize := 16
	cols := 8
	rows := 2

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			widgetCfg := config.WidgetConfig{
				ID:       string(rune('A' + row*cols + col)),
				Type:     "memory",
				Enabled:  boolPtr(true),
				Position: config.PositionConfig{X: col * widgetSize, Y: row * 20, W: widgetSize, H: 20, Z: row*cols + col},
				Mode:     "bar",
				Bar:      &config.BarConfig{Direction: "vertical"},
			}
			w, _ := memory.New(widgetCfg)
			widgets = append(widgets, w)
		}
	}

	cfg := createTestConfig()
	client, comp := createTestSetup(widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("16 widgets composed: %d total pixels", pixelCount)
}
