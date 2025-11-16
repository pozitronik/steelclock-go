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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:     "text",
			FontSize:        10,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
			FillColor:       255,
			HistoryLength:   30,
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_vertical",
			PerCore:     true,
			FillColor:   255,
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			// Intentionally leave fields empty to test defaults
		},
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
		},
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_vertical",
			PerCore:     true,
		},
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
			FontSize:    10,
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
			FillColor:   255,
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_vertical",
			PerCore:     true, // Test with per-core mode
			FillColor:   255,
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:   "graph",
			FillColor:     255,
			HistoryLength: 30,
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
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:   "graph",
			FillColor:     255,
			HistoryLength: 30,
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

// TestCPUWidget_ConcurrentAccess tests thread safety
func TestCPUWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_concurrent",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
		},
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
