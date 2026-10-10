package disk

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
)

// TestNew tests successful disk widget creation
func TestNew(t *testing.T) {
	diskName := "sda"
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "text",
		Disk:         &diskName,
		MaxSpeedMbps: 100,
		Text: &config.TextConfig{
			Size: 10,
		},
		Colors: &config.ColorsConfig{
			Read:  config.IntPtr(255),
			Write: config.IntPtr(200),
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

	if widget.Name() != "test_disk" {
		t.Errorf("Name() = %s, want test_disk", widget.Name())
	}
}

// TestNew_AllDisks tests monitoring all disks
func TestNew_AllDisks(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_all",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		// Disk intentionally nil to monitor all disks
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget.diskName != nil {
		t.Error("New() should accept nil diskName for all disks mode")
	}
}

// TestNew_Defaults tests default values
func TestNew_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Check defaults
	if widget.DisplayMode != render.DisplayModeText {
		t.Errorf("default DisplayMode = %s, want text", widget.DisplayMode)
	}

	if widget.Renderer.Bar.PrimaryColor != 255 {
		t.Errorf("default Bar.PrimaryColor = %d, want 255", widget.Renderer.Bar.PrimaryColor)
	}

	if widget.Renderer.Bar.SecondaryColor != 255 {
		t.Errorf("default Bar.SecondaryColor = %d, want 255", widget.Renderer.Bar.SecondaryColor)
	}

	if widget.MaxSpeedBps != -1 {
		t.Errorf("default MaxSpeedBps = %f, want -1 (auto)", widget.MaxSpeedBps)
	}
}

// TestWidget_Update tests disk I/O stat collection
func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_update",
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
	widget.Mu.RLock()
	hasStats := widget.PrimaryValue >= 0 && widget.SecondaryValue >= 0
	widget.Mu.RUnlock()

	if !hasStats {
		t.Error("Update() did not collect disk stats")
	}
}

// TestWidget_RenderText tests text mode rendering
func TestWidget_RenderText(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_text",
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

// TestWidget_RenderBarHorizontal tests horizontal bar rendering
func TestWidget_RenderBarHorizontal(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_bar_h",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "bar_horizontal",
		MaxSpeedMbps: 100,
		Colors: &config.ColorsConfig{
			Read:  config.IntPtr(255),
			Write: config.IntPtr(200),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

// TestWidget_RenderBarVertical tests vertical bar rendering
func TestWidget_RenderBarVertical(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_bar_v",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_vertical",
		Colors: &config.ColorsConfig{
			Read:  config.IntPtr(255),
			Write: config.IntPtr(200),
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

// TestWidget_RenderGraph tests graph mode rendering
func TestWidget_RenderGraph(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_graph",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "graph",
		Colors: &config.ColorsConfig{
			Read:  config.IntPtr(255),
			Write: config.IntPtr(200),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

// TestWidget_RenderGraph_InsufficientHistory tests graph with insufficient data
func TestWidget_RenderGraph_InsufficientHistory(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_graph_nodata",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "graph",
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

// TestWidget_AutoScale tests auto-scaling
func TestWidget_AutoScale(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_autoscale",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "bar_horizontal",
		MaxSpeedMbps: -1, // Auto-scale
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
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

// TestWidget_SpecificDisk tests monitoring specific disk
func TestWidget_SpecificDisk(t *testing.T) {
	// Use a likely non-existent disk
	diskName := "nonexistent_disk_xyz123"
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_specific",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		Disk: &diskName,
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update should work even if disk doesn't exist (returns 0 stats)
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() with nonexistent disk should not error, got %v", err)
	}
}

// TestWidget_ConcurrentAccess tests thread safety
func TestWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "disk",
		ID:      "test_disk_concurrent",
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

// TestUpdate_CounterRates checks the rates computed from two counter samples,
// including counters that went down (disk reset or removed), which must
// not wrap into huge values.
func TestUpdate_CounterRates(t *testing.T) {
	tests := []struct {
		name      string
		lastRead  uint64
		lastWrite uint64
		read      uint64
		write     uint64
		wantRead  float64
		wantWrite float64
	}{
		{"counters grow", 1000, 2000, 3000, 2500, 2000, 500},
		{"counters reset", 100, 100, 50, 25, 0, 0},
		{"disk gone", 5000, 5000, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(config.WidgetConfig{Type: "disk", Position: config.PositionConfig{W: 128, H: 40}, Mode: "bar"})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			w.diskProvider = &metrics.MockDisk{IOCountersFunc: func() (map[string]metrics.DiskStat, error) {
				return map[string]metrics.DiskStat{"sda": {Name: "sda", ReadBytes: tt.read, WriteBytes: tt.write}}, nil
			}}
			w.lastRead, w.lastWrite = tt.lastRead, tt.lastWrite
			w.lastTime = time.Now().Add(-time.Second)

			if err := w.Update(); err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			// Elapsed time is slightly over one second, so allow a small margin.
			if w.PrimaryValue > tt.wantRead || w.PrimaryValue < tt.wantRead*0.9 {
				t.Errorf("read rate = %g, want about %g", w.PrimaryValue, tt.wantRead)
			}
			if w.SecondaryValue > tt.wantWrite || w.SecondaryValue < tt.wantWrite*0.9 {
				t.Errorf("write rate = %g, want about %g", w.SecondaryValue, tt.wantWrite)
			}
			if w.lastRead != tt.read || w.lastWrite != tt.write {
				t.Errorf("baseline = %d/%d, want %d/%d", w.lastRead, w.lastWrite, tt.read, tt.write)
			}
		})
	}
}
