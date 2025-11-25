package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestNewCPUWidget tests successful CPU widget creation
func TestNewCPUWidget(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewCPUWidget() returned nil")
	}

	if widget.Name() != "test_cpu" {
		t.Errorf("Name() = %s, want test_cpu", widget.Name())
	}
}

// TestNewCPUWidget_PerCore tests CPU widget with per-core mode
func TestNewCPUWidget_PerCore(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
	}

	if !widget.perCore {
		t.Error("NewCPUWidget() did not set perCore mode")
	}
}

// TestNewCPUWidget_Defaults tests that defaults are properly applied
func TestNewCPUWidget_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		// Intentionally leave fields empty to test defaults
	}

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
	}

	// Verify defaults
	if widget.displayMode != "text" {
		t.Errorf("default displayMode = %s, want text", widget.displayMode)
	}

	if widget.fontSize != 10 {
		t.Errorf("default fontSize = %d, want 10", widget.fontSize)
	}

	if widget.fillColor != 255 {
		t.Errorf("default fillColor = %d, want 255", widget.fillColor)
	}

	if widget.historyLen != 30 {
		t.Errorf("default historyLen = %d, want 30", widget.historyLen)
	}
}

// TestCPUWidget_Update tests CPU widget update functionality
func TestCPUWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
	}

	// First update
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify currentUsage was set
	widget.mu.RLock()
	hasUsage := widget.currentUsage != nil
	widget.mu.RUnlock()

	if !hasUsage {
		t.Error("Update() did not set currentUsage")
	}
}

// TestCPUWidget_Update_PerCore tests per-core CPU update
func TestCPUWidget_Update_PerCore(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
	}

	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// For per-core mode, currentUsage should be []float64
	widget.mu.RLock()
	_, ok := widget.currentUsage.([]float64)
	widget.mu.RUnlock()

	if !ok {
		t.Error("Update() with perCore=true should set currentUsage as []float64")
	}
}

// TestCPUWidget_RenderText tests text mode rendering
func TestCPUWidget_RenderText(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderBarHorizontal tests horizontal bar rendering
func TestCPUWidget_RenderBarHorizontal(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderBarVertical tests vertical bar rendering
func TestCPUWidget_RenderBarVertical(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderGraph tests graph mode rendering
func TestCPUWidget_RenderGraph(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderGraph_InsufficientHistory tests graph with insufficient data
func TestCPUWidget_RenderGraph_InsufficientHistory(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderGauge tests gauge mode rendering
func TestCPUWidget_RenderGauge(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderGauge_PerCore tests gauge mode with per-core
func TestCPUWidget_RenderGauge_PerCore(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_GaugeDefaults tests gauge mode with default colors
func TestCPUWidget_GaugeDefaults(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
	}

	// Verify defaults
	if widget.gaugeColor != 200 {
		t.Errorf("default gaugeColor = %d, want 200", widget.gaugeColor)
	}

	if widget.gaugeNeedleColor != 255 {
		t.Errorf("default gaugeNeedleColor = %d, want 255", widget.gaugeNeedleColor)
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

// TestCPUWidget_ConcurrentAccess tests thread safety
func TestCPUWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_concurrent",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderTextGrid tests text mode with per-core (grid layout)
func TestCPUWidget_RenderTextGrid(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderBarHorizontal_PerCore tests horizontal bar with per-core
func TestCPUWidget_RenderBarHorizontal_PerCore(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderGraph_PerCore tests graph mode with per-core
func TestCPUWidget_RenderGraph_PerCore(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_RenderGraph_EmptyHistory tests graph with no history
func TestCPUWidget_RenderGraph_EmptyHistory(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_SmallSize tests rendering with very small dimensions
func TestCPUWidget_SmallSize(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_ZeroSize tests rendering with zero dimensions
func TestCPUWidget_ZeroSize(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_zero",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 0, H: 0,
		},
		Mode: "text",
	}

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_InvalidDisplayMode tests handling of invalid display mode
func TestCPUWidget_InvalidDisplayMode(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_invalid_mode",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "invalid_mode",
	}

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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

// TestCPUWidget_BeforeFirstUpdate tests rendering before any update
func TestCPUWidget_BeforeFirstUpdate(t *testing.T) {
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

	widget, err := NewCPUWidget(cfg)
	if err != nil {
		t.Fatalf("NewCPUWidget() error = %v", err)
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
