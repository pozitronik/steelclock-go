package widget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestNewVolumeWidget tests volume widget creation
func TestNewVolumeWidget(t *testing.T) {
	tests := []struct {
		name        string
		displayMode string
	}{
		{"Text mode", "text"},
		{"Bar horizontal", "bar_horizontal"},
		{"Bar vertical", "bar_vertical"},
		{"Gauge mode", "gauge"},
		{"Triangle mode", "triangle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type: "volume",
				ID:   "test_volume",
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
				Properties: config.WidgetProperties{
					DisplayMode: tt.displayMode,
				},
			}

			widget, err := NewVolumeWidget(cfg)
			if err != nil {
				t.Fatalf("NewVolumeWidget() error = %v", err)
			}

			if widget.Name() != "test_volume" {
				t.Errorf("Name() = %v, want %v", widget.Name(), "test_volume")
			}

			if widget.displayMode != tt.displayMode {
				t.Errorf("displayMode = %v, want %v", widget.displayMode, tt.displayMode)
			}
		})
	}
}

// TestNewVolumeWidget_Defaults tests default values
func TestNewVolumeWidget_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Properties: config.WidgetProperties{},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	if widget.displayMode != "bar_horizontal" {
		t.Errorf("default displayMode = %v, want bar_horizontal", widget.displayMode)
	}

	if widget.fillColor != 255 {
		t.Errorf("default fillColor = %v, want 255", widget.fillColor)
	}

	if widget.autoHideTimeout != 2.0 {
		t.Errorf("default autoHideTimeout = %v, want 2.0", widget.autoHideTimeout)
	}

	if widget.gaugeColor != 200 {
		t.Errorf("default gaugeColor = %v, want 200", widget.gaugeColor)
	}

	if widget.gaugeNeedleColor != 255 {
		t.Errorf("default gaugeNeedleColor = %v, want 255", widget.gaugeNeedleColor)
	}

	if widget.triangleFillColor != 255 {
		t.Errorf("default triangleFillColor = %v, want 255", widget.triangleFillColor)
	}
}

// TestVolumeWidget_Update tests volume updates
func TestVolumeWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	// Update volume
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Volume should be set (mock returns 75.0)
	widget.mu.RLock()
	volume := widget.volume
	widget.mu.RUnlock()

	if volume < 0 || volume > 100 {
		t.Errorf("volume = %v, want 0-100", volume)
	}
}

// TestVolumeWidget_RenderText tests text mode rendering
func TestVolumeWidget_RenderText(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "text",
			FontSize:    10,
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	// Update first
	_ = widget.Update()

	// Render
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}

	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 40 {
		t.Errorf("Image size = %v, want 128x40", img.Bounds())
	}
}

// TestVolumeWidget_RenderBarHorizontal tests horizontal bar rendering
func TestVolumeWidget_RenderBarHorizontal(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 20,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
			FillColor:   255,
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestVolumeWidget_RenderBarVertical tests vertical bar rendering
func TestVolumeWidget_RenderBarVertical(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 20,
			H: 128,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_vertical",
			FillColor:   255,
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestVolumeWidget_RenderGauge tests gauge rendering
func TestVolumeWidget_RenderGauge(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
			H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:      "gauge",
			GaugeColor:       200,
			GaugeNeedleColor: 255,
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestVolumeWidget_RenderTriangle tests triangle rendering
func TestVolumeWidget_RenderTriangle(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 40,
			H: 60,
		},
		Properties: config.WidgetProperties{
			DisplayMode:       "triangle",
			TriangleFillColor: 255,
			TriangleBorder:    true,
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestVolumeWidget_AutoHide tests auto-hide functionality
func TestVolumeWidget_AutoHide(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:     "bar_horizontal",
			AutoHide:        true,
			AutoHideTimeout: 0.1, // 100ms for testing
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	// Update to set initial volume
	_ = widget.Update()

	// Widget should be visible initially (within timeout)
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	// Wait for auto-hide timeout
	time.Sleep(150 * time.Millisecond)

	// Widget should still render but might be empty/transparent
	// (depends on implementation - just verify it doesn't crash)
	img, err = widget.Render()
	if err != nil {
		t.Errorf("Render() after timeout error = %v", err)
	}
	if img == nil {
		t.Error("Render() after timeout returned nil image")
	}
}

// TestVolumeWidget_AllModes tests all display modes
func TestVolumeWidget_AllModes(t *testing.T) {
	modes := []string{"text", "bar_horizontal", "bar_vertical", "gauge", "triangle"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type: "volume",
				ID:   "test_volume",
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 64,
					H: 64,
				},
				Properties: config.WidgetProperties{
					DisplayMode: mode,
					FontSize:    10,
				},
			}

			widget, err := NewVolumeWidget(cfg)
			if err != nil {
				t.Fatalf("NewVolumeWidget() error = %v", err)
			}

			_ = widget.Update()

			img, err := widget.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}
			if img == nil {
				t.Error("Render() returned nil image")
			}
		})
	}
}

// TestVolumeWidget_SmallSize tests rendering with very small size
func TestVolumeWidget_SmallSize(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 10,
			H: 10,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	_ = widget.Update()

	// Should not crash with small size
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with small size error = %v", err)
	}
	if img == nil {
		t.Error("Render() with small size returned nil image")
	}
}

// TestVolumeWidget_WithBorder tests rendering with border
func TestVolumeWidget_WithBorder(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			Border:      true,
			BorderColor: 255,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
			BarBorder:   true,
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestVolumeWidget_ConcurrentAccess tests concurrent access safety
func TestVolumeWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode: "bar_horizontal",
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}

	// Run concurrent updates and renders
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			_ = widget.Update()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = widget.Render()
		}
		done <- true
	}()

	<-done
	<-done

	// Should not crash or race
}
