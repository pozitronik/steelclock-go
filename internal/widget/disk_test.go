package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestNewDiskWidget tests successful disk widget creation
func TestNewDiskWidget(t *testing.T) {
	diskName := "sda"
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DiskName:      &diskName,
			DisplayMode:   "text",
			FontSize:      10,
			ReadColor:     255,
			WriteColor:    200,
			MaxSpeedMbps:  100,
			HistoryLength: 30,
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewDiskWidget() returned nil")
	}

	if widget.Name() != "test_disk" {
		t.Errorf("Name() = %s, want test_disk", widget.Name())
	}
}

// TestNewDiskWidget_AllDisks tests monitoring all disks
func TestNewDiskWidget_AllDisks(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_all",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			// DiskName intentionally nil to monitor all disks
			DisplayMode: "text",
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	if widget.diskName != nil {
		t.Error("NewDiskWidget() should accept nil diskName for all disks mode")
	}
}

// TestNewDiskWidget_Defaults tests default values
func TestNewDiskWidget_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_defaults",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	// Check defaults
	if widget.displayMode != "text" {
		t.Errorf("default displayMode = %s, want text", widget.displayMode)
	}

	if widget.readColor != 255 {
		t.Errorf("default readColor = %d, want 255", widget.readColor)
	}

	if widget.writeColor != 255 {
		t.Errorf("default writeColor = %d, want 255", widget.writeColor)
	}

	if widget.maxSpeedMbps != -1 {
		t.Errorf("default maxSpeedMbps = %f, want -1 (auto)", widget.maxSpeedMbps)
	}
}

// TestDiskWidget_Update tests disk I/O stat collection
func TestDiskWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_update",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	// First update initializes baseline
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Second update calculates deltas
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify stats were collected
	widget.mu.RLock()
	hasStats := widget.currentReadMbps >= 0 && widget.currentWriteMbps >= 0
	widget.mu.RUnlock()

	if !hasStats {
		t.Error("Update() did not collect disk stats")
	}
}

// TestDiskWidget_RenderText tests text mode rendering
func TestDiskWidget_RenderText(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_text",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
			FontSize:    10,
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	_ = widget.Update()

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

// TestDiskWidget_RenderBarHorizontal tests horizontal bar rendering
func TestDiskWidget_RenderBarHorizontal(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_bar_h",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:  "bar_horizontal",
			ReadColor:    255,
			WriteColor:   200,
			MaxSpeedMbps: 100,
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestDiskWidget_RenderBarVertical tests vertical bar rendering
func TestDiskWidget_RenderBarVertical(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_bar_v",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_vertical",
			ReadColor:   255,
			WriteColor:  200,
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestDiskWidget_RenderGraph tests graph mode rendering
func TestDiskWidget_RenderGraph(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_graph",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:   "graph",
			ReadColor:     255,
			WriteColor:    200,
			HistoryLength: 30,
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	// Build history
	for i := 0; i < 5; i++ {
		_ = widget.Update()
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestDiskWidget_RenderGraph_InsufficientHistory tests graph with insufficient data
func TestDiskWidget_RenderGraph_InsufficientHistory(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_graph_nodata",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:   "graph",
			HistoryLength: 30,
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	// Update only once (not enough for graph)
	_ = widget.Update()

	// Should still render without error
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with insufficient data error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestDiskWidget_AutoScale tests auto-scaling
func TestDiskWidget_AutoScale(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_autoscale",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:  "bar_horizontal",
			MaxSpeedMbps: -1, // Auto-scale
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with auto-scale error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestDiskWidget_SpecificDisk tests monitoring specific disk
func TestDiskWidget_SpecificDisk(t *testing.T) {
	// Use a likely non-existent disk
	diskName := "nonexistent_disk_xyz123"
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_specific",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DiskName:    &diskName,
			DisplayMode: "text",
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	// Update should work even if disk doesn't exist (returns 0 stats)
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() with nonexistent disk should not error, got %v", err)
	}
}

// TestDiskWidget_ConcurrentAccess tests thread safety
func TestDiskWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_concurrent",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
		},
	}

	widget, err := NewDiskWidget(cfg)
	if err != nil {
		t.Fatalf("NewDiskWidget() error = %v", err)
	}

	done := make(chan bool)

	// Concurrent updates
	go func() {
		for i := 0; i < 10; i++ {
			_ = widget.Update()
		}
		done <- true
	}()

	// Concurrent renders
	go func() {
		for i := 0; i < 10; i++ {
			_, _ = widget.Render()
		}
		done <- true
	}()

	<-done
	<-done
	// Should not panic or race
}
