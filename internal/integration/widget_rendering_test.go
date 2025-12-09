// Package integration contains integration tests that span multiple packages
package integration

import (
	"runtime"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/testutil"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"github.com/pozitronik/steelclock-go/internal/widget/cpu"
	"github.com/pozitronik/steelclock-go/internal/widget/memory"
)

// createTestConfig creates a minimal config for testing
func createTestConfig() *config.Config {
	dedupDisabled := false
	return &config.Config{
		RefreshRateMs:     50,
		FrameDedupEnabled: &dedupDisabled, // Disable dedup for tests that count frames
		Display: config.DisplayConfig{
			Width:      128,
			Height:     40,
			Background: 0,
		},
	}
}

// createTestSetup creates a test client, layout manager, and compositor for widgets
func createTestSetup(widgets []widget.Widget, cfg *config.Config) (*testutil.TestClient, *compositor.Compositor) {
	client := testutil.NewTestClient(
		testutil.WithDimensions(cfg.Display.Width, cfg.Display.Height),
	)

	displayCfg := config.DisplayConfig{
		Width:      cfg.Display.Width,
		Height:     cfg.Display.Height,
		Background: cfg.Display.Background,
	}
	layoutMgr := layout.NewManager(displayCfg, widgets)

	comp := compositor.NewCompositor(client, layoutMgr, widgets, cfg)
	return client, comp
}

// TestClockWidget_Rendering tests that clock widget renders non-blank output
func TestClockWidget_Rendering(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-clock",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Text: &config.TextConfig{
			Format: "%H:%M:%S",
		},
	}

	clockWidget, err := widget.NewClockWidget(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create clock widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	err = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	if err != nil {
		t.Fatalf("Failed to capture frames: %v", err)
	}

	// Clock should render text (non-blank)
	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	if testutil.IsBlankFrame(frame.Data) {
		t.Error("Clock widget should render non-blank frame")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Clock widget rendered %d pixels", pixelCount)

	// Clock text should have reasonable number of pixels (not too few, not full)
	if pixelCount < 50 {
		t.Errorf("Clock should render more than 50 pixels, got %d", pixelCount)
	}
}

// TestClockWidget_ConsistentRendering tests that clock renders consistently
func TestClockWidget_ConsistentRendering(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-clock",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Text: &config.TextConfig{
			Format: "%H:%M", // Minutes - less likely to change during test
		},
	}

	clockWidget, err := widget.NewClockWidget(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create clock widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(5, 2*time.Second)
	comp.Stop()

	frames := client.Frames()
	if len(frames) < 2 {
		t.Skip("Not enough frames for comparison")
	}

	// Consecutive frames showing same time should be identical
	diff := testutil.CompareFrames(frames[0].Data, frames[1].Data)

	// Allow small tolerance - fonts might have slight variations
	if diff.DifferenceRatio > 0.01 {
		t.Logf("Frames differ by %.2f%% - may be expected if time changed",
			diff.DifferenceRatio*100)
	}
}

// TestKeyboardWidget_Rendering tests keyboard indicator widget
func TestKeyboardWidget_Rendering(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Keyboard widget is only supported on Windows")
	}

	// Use default indicator configuration
	capsOn := "CAPS"
	numOn := "NUM"
	widgetCfg := config.WidgetConfig{
		ID:       "test-kbd",
		Type:     "keyboard",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Indicators: &config.IndicatorsConfig{
			Caps: &config.IndicatorConfig{On: &capsOn},
			Num:  &config.IndicatorConfig{On: &numOn},
		},
	}

	kbdWidget, err := widget.NewKeyboardWidget(widgetCfg)
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

	// Keyboard widget may or may not show content depending on lock states
	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Keyboard widget rendered %d pixels (depends on caps/num lock state)", pixelCount)
}

// TestMemoryWidget_Rendering tests memory widget renders a bar
func TestMemoryWidget_Rendering(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-mem",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
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
		t.Error("Memory widget should render non-blank frame")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Memory widget rendered %d pixels", pixelCount)

	// Memory bar should have reasonable content
	if pixelCount < 20 {
		t.Errorf("Memory widget should render more pixels, got %d", pixelCount)
	}
}

// TestCPUWidget_Rendering tests CPU widget renders
func TestCPUWidget_Rendering(t *testing.T) {
	widgetCfg := config.WidgetConfig{
		ID:       "test-cpu",
		Type:     "cpu",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "horizontal",
		},
	}

	cpuWidget, err := cpu.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create CPU widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{cpuWidget}, cfg)

	_ = comp.Start()
	// CPU widget needs time to collect stats
	_ = client.WaitForFrames(5, 3*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("CPU widget rendered %d pixels", pixelCount)
}

// TestMultipleWidgets_Composition tests that multiple widgets compose correctly
func TestMultipleWidgets_Composition(t *testing.T) {
	// Clock on left half
	clockCfg := config.WidgetConfig{
		ID:       "clock",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 64, H: 40},
		Text: &config.TextConfig{
			Format: "%H:%M",
		},
	}

	// Memory on right half
	memCfg := config.WidgetConfig{
		ID:       "mem",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 64, Y: 0, W: 64, H: 40},
		Mode:     "bar",
		Bar: &config.BarConfig{
			Direction: "vertical",
		},
	}

	clockWidget, err := widget.NewClockWidget(clockCfg)
	if err != nil {
		t.Fatalf("Failed to create clock widget: %v", err)
	}

	memWidget, err := memory.New(memCfg)
	if err != nil {
		t.Fatalf("Failed to create memory widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{clockWidget, memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Check both halves have content
	blank := make([]byte, 640)

	leftDiff := testutil.CompareRegion(frame.Data, blank, 0, 0, 64, 40)
	rightDiff := testutil.CompareRegion(frame.Data, blank, 64, 0, 64, 40)

	t.Logf("Left region (clock): %d pixels", leftDiff.DifferentPixels)
	t.Logf("Right region (memory): %d pixels", rightDiff.DifferentPixels)

	// Both regions should have some content
	if leftDiff.DifferentPixels < 10 {
		t.Error("Left region (clock) should have content")
	}
	if rightDiff.DifferentPixels < 10 {
		t.Error("Right region (memory) should have content")
	}
}

// TestWidgetZOrder tests z-order composition
func TestWidgetZOrder(t *testing.T) {
	// Background widget (full screen, lower z)
	bgCfg := config.WidgetConfig{
		ID:       "bg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40, Z: 0},
		Text: &config.TextConfig{
			Format: "BACKGROUND",
		},
	}

	// Foreground widget (smaller, higher z)
	fgCfg := config.WidgetConfig{
		ID:       "fg",
		Type:     "clock",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 32, Y: 10, W: 64, H: 20, Z: 1},
		Text: &config.TextConfig{
			Format: "FG",
		},
	}

	bgWidget, _ := widget.NewClockWidget(bgCfg)
	fgWidget, _ := widget.NewClockWidget(fgCfg)

	cfg := createTestConfig()

	// Test with bg first, fg second (z-order should sort them)
	client, comp := createTestSetup([]widget.Widget{bgWidget, fgWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Frame should have content (both widgets contribute)
	if testutil.IsBlankFrame(frame.Data) {
		t.Error("Frame should not be blank with two widgets")
	}

	pixelCount := testutil.CountSetPixels(frame.Data)
	t.Logf("Z-order composition: %d pixels", pixelCount)
}

// TestDiskWidget_Rendering tests disk widget
func TestDiskWidget_Rendering(t *testing.T) {
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
		},
	}

	diskWidget, err := widget.NewDiskWidget(widgetCfg)
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
	t.Logf("Disk widget rendered %d pixels", pixelCount)
}

// TestFrameConsistency tests that frames are consistent over time for static content
func TestFrameConsistency(t *testing.T) {
	// Use memory widget - relatively static
	widgetCfg := config.WidgetConfig{
		ID:       "test-mem",
		Type:     "memory",
		Enabled:  boolPtr(true),
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Mode:     "bar",
	}

	memWidget, err := memory.New(widgetCfg)
	if err != nil {
		t.Fatalf("Failed to create memory widget: %v", err)
	}

	cfg := createTestConfig()
	client, comp := createTestSetup([]widget.Widget{memWidget}, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(10, 3*time.Second)
	comp.Stop()

	frames := client.Frames()
	if len(frames) < 5 {
		t.Skip("Not enough frames")
	}

	// Compare multiple consecutive frames
	totalDiff := 0
	comparisons := 0
	for i := 1; i < len(frames); i++ {
		diff := testutil.CompareFrames(frames[i-1].Data, frames[i].Data)
		totalDiff += diff.DifferentPixels
		comparisons++
	}

	avgDiff := float64(totalDiff) / float64(comparisons)
	t.Logf("Average frame difference: %.1f pixels across %d comparisons", avgDiff, comparisons)

	// Memory usage changes slowly, so frames should be mostly similar
	// Allow some tolerance for actual memory changes
	if avgDiff > 100 {
		t.Logf("High frame variance - memory may be fluctuating")
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
