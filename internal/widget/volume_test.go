package widget

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// skipIfNoAudioDevice skips the test if no audio device is available
// This is common in CI environments
func skipIfNoAudioDevice(t *testing.T) {
	t.Helper()

	// Volume widget is Windows-only
	if runtime.GOOS != "windows" {
		t.Skip("Volume widget is Windows-only (requires Windows Core Audio API)")
		return
	}

	// Try to create a volume reader to see if audio devices are available
	reader, err := newVolumeReader()
	if err != nil {
		// Check if error is "Element not found" (no audio device)
		if strings.Contains(err.Error(), "Element not found") {
			t.Skip("No audio device available (common in CI environments)")
		}
		// For other errors, skip as well but with different message
		t.Skipf("Cannot initialize audio: %v", err)
	}

	// Clean up the test reader
	if reader != nil {
		reader.Close()
	}
}

// TestNewVolumeWidget tests volume widget creation
func TestNewVolumeWidget(t *testing.T) {
	skipIfNoAudioDevice(t)

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
	skipIfNoAudioDevice(t)

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

	if widget.GetAutoHideTimeout() != 2*time.Second {
		t.Errorf("default autoHideTimeout = %v, want 2s", widget.GetAutoHideTimeout())
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
	skipIfNoAudioDevice(t)

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
	skipIfNoAudioDevice(t)

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
	skipIfNoAudioDevice(t)

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
			FillColor:   config.IntPtr(255),
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
	skipIfNoAudioDevice(t)

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
			FillColor:   config.IntPtr(255),
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
	skipIfNoAudioDevice(t)

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
			GaugeColor:       config.IntPtr(200),
			GaugeNeedleColor: config.IntPtr(255),
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
	skipIfNoAudioDevice(t)

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
			TriangleFillColor: config.IntPtr(255),
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
	// Skip on non-Windows platforms - volume reading is Windows-only
	if runtime.GOOS != "windows" {
		t.Skip("Volume widget auto-hide test requires Windows (volume reading not supported on this platform)")
	}
	skipIfNoAudioDevice(t)

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
	defer widget.Stop()

	// Widget should start hidden (auto-hide lastTriggerTime = zero time)
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() initial error = %v", err)
	}
	if img != nil {
		t.Error("Widget should start hidden with auto-hide enabled, but got image")
	}

	// Wait for background polling to detect volume (triggers volume change)
	time.Sleep(200 * time.Millisecond)

	// After volume is detected, widget should be visible
	img, err = widget.Render()
	if err != nil {
		t.Errorf("Render() after volume change error = %v", err)
	}
	if img == nil {
		t.Error("Widget should be visible after volume change, but got nil")
	}

	// Wait for auto-hide timeout
	time.Sleep(150 * time.Millisecond)

	// Widget should be hidden again
	img, err = widget.Render()
	if err != nil {
		t.Errorf("Render() after timeout error = %v", err)
	}
	if img != nil {
		t.Error("Widget should be hidden after timeout, but got image")
	}
}

// TestVolumeWidget_AllModes tests all display modes
func TestVolumeWidget_AllModes(t *testing.T) {
	skipIfNoAudioDevice(t)

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
	skipIfNoAudioDevice(t)

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
	skipIfNoAudioDevice(t)

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
	skipIfNoAudioDevice(t)

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

// TestVolumeWidget_Stop tests proper Stop() functionality
func TestVolumeWidget_Stop(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume_stop",
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

	// Let it run briefly to accumulate some metrics
	time.Sleep(200 * time.Millisecond)

	// Verify widget is functioning
	widget.mu.RLock()
	totalCallsBefore := widget.totalCalls
	widget.mu.RUnlock()

	if totalCallsBefore == 0 {
		t.Error("Widget should have made some calls before Stop()")
	}

	// Stop the widget
	widget.Stop()

	// Verify cleanup
	if widget.reader != nil {
		t.Error("Reader should be cleaned up after Stop()")
	}

	// Give time for background goroutine to actually stop
	time.Sleep(100 * time.Millisecond)

	// Verify no more calls are being made
	widget.mu.RLock()
	totalCallsAfter := widget.totalCalls
	widget.mu.RUnlock()

	if totalCallsAfter != totalCallsBefore {
		t.Errorf("Calls should stop after Stop(), before=%d, after=%d", totalCallsBefore, totalCallsAfter)
	}
}

// TestVolumeWidget_RenderMuted tests rendering with muted state
func TestVolumeWidget_RenderMuted(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume_muted",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
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
	defer widget.Stop()

	// Simulate muted state
	widget.mu.Lock()
	widget.isMuted = true
	widget.volume = 50.0
	widget.mu.Unlock()

	// Render should include mute indicator
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with muted state error = %v", err)
	}

	if img == nil {
		t.Error("Render() with muted state returned nil image")
	}

	// Verify image was rendered (mute indicator should be drawn)
	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 40 {
		t.Errorf("Image size = %v, want 64x40", bounds)
	}
}

// TestVolumeWidget_UpdateInterval tests GetUpdateInterval
func TestVolumeWidget_UpdateInterval(t *testing.T) {
	skipIfNoAudioDevice(t)

	cfg := config.WidgetConfig{
		Type: "volume",
		ID:   "test_volume_interval",
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Properties: config.WidgetProperties{
			DisplayMode:    "bar_horizontal",
			UpdateInterval: 0.5, // 500ms
		},
	}

	widget, err := NewVolumeWidget(cfg)
	if err != nil {
		t.Fatalf("NewVolumeWidget() error = %v", err)
	}
	defer widget.Stop()

	interval := widget.GetUpdateInterval()
	expected := 500 * time.Millisecond

	if interval != expected {
		t.Errorf("GetUpdateInterval() = %v, want %v", interval, expected)
	}
}
