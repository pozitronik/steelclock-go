package cpu

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
)

// TestNew tests successful CPU widget creation
func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Size:  10,
			Align: &config.AlignConfig{H: "center", V: "center"},
		},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget == nil {
		t.Fatal("New() returned nil")
	}

	if widget.Name() != "test_cpu" {
		t.Errorf("Name() = %s, want test_cpu", widget.Name())
	}
}

// TestNew_PerCore tests CPU widget with per-core mode
func TestNew_PerCore(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_per_core",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:    "bar_vertical",
		PerCore: &config.PerCoreConfig{Enabled: true},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !widget.perCore {
		t.Error("New() did not set perCore mode")
	}
}

// TestNew_Defaults tests that defaults are properly applied
func TestNew_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		// Intentionally leave fields empty to test defaults
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify defaults
	if widget.displayMode != shared.DisplayModeText {
		t.Errorf("default displayMode = %s, want text", widget.displayMode)
	}

	if widget.fillColor != 255 {
		t.Errorf("default fillColor = %d, want 255", widget.fillColor)
	}

	if widget.historyLen != 30 {
		t.Errorf("default historyLen = %d, want 30", widget.historyLen)
	}
}

// TestWidget_Update tests CPU widget update functionality
func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// First update
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify currentUsage was set
	widget.mu.RLock()
	hasUsage := widget.hasData
	widget.mu.RUnlock()

	if !hasUsage {
		t.Error("Update() did not set currentUsage")
	}
}

// TestWidget_Update_PerCore tests per-core CPU update
func TestWidget_Update_PerCore(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_per_core",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:    "bar_vertical",
		PerCore: &config.PerCoreConfig{Enabled: true},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// For per-core mode, currentUsagePerCore should be set
	widget.mu.RLock()
	hasData := widget.hasData
	hasPerCoreData := len(widget.currentUsagePerCore) > 0
	widget.mu.RUnlock()

	if !hasData {
		t.Error("Update() with perCore=true should set hasData")
	}
	if !hasPerCoreData {
		t.Error("Update() with perCore=true should set currentUsagePerCore")
	}
}

// TestWidget_RenderText tests text mode rendering
func TestWidget_RenderText(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_render_text",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Size: 10,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update before render
	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Render
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

// TestWidget_RenderBarHorizontal tests horizontal bar rendering
func TestWidget_RenderBarHorizontal(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_bar_h",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_horizontal",
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderBarVertical tests vertical bar rendering
func TestWidget_RenderBarVertical(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_bar_v",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:    "bar_vertical",
		PerCore: &config.PerCoreConfig{Enabled: true},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderGraph tests graph mode rendering
func TestWidget_RenderGraph(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_graph",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "graph",
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Need multiple updates to build history for graph
	for i := 0; i < 5; i++ {
		err = widget.Update()
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderGraph_InsufficientHistory tests graph with insufficient data
func TestWidget_RenderGraph_InsufficientHistory(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_graph_nodata",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "graph",
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update only once (not enough for graph)
	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Render should still work, just won't show graph
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderGauge tests gauge mode rendering
func TestWidget_RenderGauge(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_gauge",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		Mode: "gauge",
		Colors: &config.ColorsConfig{
			Arc:    config.IntPtr(200),
			Needle: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderGauge_PerCore tests gauge mode with per-core
func TestWidget_RenderGauge_PerCore(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_gauge_percore",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		Mode:    "gauge",
		PerCore: &config.PerCoreConfig{Enabled: true},
		Colors: &config.ColorsConfig{
			Arc:    config.IntPtr(200),
			Needle: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_GaugeDefaults tests gauge mode with default colors
func TestWidget_GaugeDefaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_gauge_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		Mode: "gauge",
		// Don't specify colors to test defaults
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify defaults via renderer config
	if widget.Renderer.Gauge.ArcColor != 200 {
		t.Errorf("default Gauge.ArcColor = %d, want 200", widget.Renderer.Gauge.ArcColor)
	}

	if widget.Renderer.Gauge.NeedleColor != 255 {
		t.Errorf("default Gauge.NeedleColor = %d, want 255", widget.Renderer.Gauge.NeedleColor)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_ConcurrentAccess tests thread safety
func TestWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_concurrent",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Initial update
	if err := widget.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Concurrent updates and renders
	done := make(chan bool)

	// Update goroutine
	go func() {
		for i := 0; i < 10; i++ {
			_ = widget.Update()
		}
		done <- true
	}()

	// Render goroutine
	go func() {
		for i := 0; i < 10; i++ {
			_, _ = widget.Render()
		}
		done <- true
	}()

	// Wait for completion
	<-done
	<-done

	// Should not panic or race
}

// TestWidget_RenderTextGrid tests text mode with per-core (grid layout)
func TestWidget_RenderTextGrid(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_text_grid",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 64,
		},
		Mode:    "text",
		PerCore: &config.PerCoreConfig{Enabled: true},
		Text: &config.TextConfig{
			Size: 8,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderBarHorizontal_PerCore tests horizontal bar with per-core
func TestWidget_RenderBarHorizontal_PerCore(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_bar_h_percore",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:    "bar_horizontal",
		PerCore: &config.PerCoreConfig{Enabled: true},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderGraph_PerCore tests graph mode with per-core
func TestWidget_RenderGraph_PerCore(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_graph_percore",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:    "graph",
		PerCore: &config.PerCoreConfig{Enabled: true},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Need multiple updates to build history for graph
	for i := 0; i < 10; i++ {
		err = widget.Update()
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_RenderGraph_EmptyHistory tests graph with no history
func TestWidget_RenderGraph_EmptyHistory(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_graph_empty",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "graph",
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Don't update - render with no data
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with no data should not error, got: %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestWidget_SmallSize tests rendering with very small dimensions
func TestWidget_SmallSize(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_small",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		Mode: "bar_horizontal",
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Should not crash with small size
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with small size error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() with small size returned nil image")
	}
}

// TestWidget_ZeroSize tests rendering with zero dimensions
func TestWidget_ZeroSize(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_zero",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 0, H: 0,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Should handle zero size gracefully
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with zero size error = %v", err)
	}

	// Image should be created even if size is zero
	if img == nil {
		t.Error("Render() with zero size returned nil image")
	}
}

// TestWidget_InvalidDisplayMode tests handling of invalid display mode
func TestWidget_InvalidDisplayMode(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_invalid_mode",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "invalid_mode",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Render should handle invalid mode gracefully (likely defaults to text)
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with invalid mode error = %v", err)
	}

	if img == nil {
		t.Error("Render() with invalid mode returned nil image")
	}
}

// TestWidget_BeforeFirstUpdate tests rendering before any update
func TestWidget_BeforeFirstUpdate(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_no_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Size: 10,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render without calling Update first
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() before Update() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() before Update() returned nil image")
	}
}
