//go:build !windows

package widget

import (
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewAudioVisualizerWidget_Unix(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_unix",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     false,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
	}

	// On Unix, NewAudioVisualizerWidget should succeed (stub widget created)
	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() on Unix error = %v (should create stub widget)", err)
	}

	if widget == nil {
		t.Fatal("NewAudioVisualizerWidget() returned nil on Unix")
	}

	if widget.Name() != "test_unix" {
		t.Errorf("Name() = %s, want test_unix", widget.Name())
	}
}

func TestAudioVisualizerWidget_Unix_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_unix_render",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     false,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Render should return an error on Unix
	img, err := widget.Render()
	if err == nil {
		t.Error("Render() on Unix should return error, got nil")
	}

	// Error message should indicate platform limitation
	if err != nil && !strings.Contains(err.Error(), "Windows") {
		t.Errorf("Render() error message should mention Windows, got: %v", err)
	}

	// But it should still return an image with error message
	if img == nil {
		t.Fatal("Render() returned nil image on Unix (should return error message image)")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}

func TestAudioVisualizerWidget_Unix_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_unix_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Update should succeed (no-op on Unix)
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() on Unix error = %v (should be no-op)", err)
	}
}

func TestAudioVisualizerWidget_Unix_GetMethods(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_unix_getters",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 10,
			Y: 20,
			W: 128,
			H: 40,
			Z: 5,
		},
		Style: &config.StyleConfig{
			Background:  100,
			Border:      true,
			BorderColor: 200,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.05,
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Test Name()
	if widget.Name() != "test_unix_getters" {
		t.Errorf("Name() = %s, want test_unix_getters", widget.Name())
	}

	// Test GetPosition()
	pos := widget.GetPosition()
	if pos.X != 10 || pos.Y != 20 || pos.W != 128 || pos.H != 40 || pos.Z != 5 {
		t.Errorf("GetPosition() = %+v, want {X:10 Y:20 W:128 H:40 Z:5}", pos)
	}

	// Test GetStyle()
	style := widget.GetStyle()
	if style.Background != 100 || !style.Border || style.BorderColor != 200 {
		t.Errorf("GetStyle() = %+v, want {Background:100 Border:true BorderColor:200}", style)
	}

	// Test GetUpdateInterval()
	interval := widget.GetUpdateInterval()
	expectedInterval := int64(0.05 * 1e9) // 50ms in nanoseconds
	if interval.Nanoseconds() != expectedInterval {
		t.Errorf("GetUpdateInterval() = %v, want 50ms", interval)
	}
}

func TestGetSharedAudioCapture_Unix(t *testing.T) {
	// On Unix, GetSharedAudioCapture should return an error
	capture, err := GetSharedAudioCapture()
	if err == nil {
		t.Error("GetSharedAudioCapture() on Unix should return error, got nil")
	}

	if capture != nil {
		t.Error("GetSharedAudioCapture() on Unix should return nil capture, got non-nil")
	}

	// Error should mention platform limitation
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "platform") {
		t.Errorf("GetSharedAudioCapture() error should mention platform, got: %v", err)
	}
}

func TestAudioVisualizerWidget_Unix_BorderRendering(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_unix_border",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background:  0,
			Border:      true,
			BorderColor: 255,
		},
		Mode: "spectrum",
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Render should return error but still provide image
	img, err := widget.Render()
	if err == nil {
		t.Error("Render() should return error on Unix")
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	// Image should have correct dimensions
	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 40 {
		t.Errorf("image dimensions = %dx%d, want 128x40", img.Bounds().Dx(), img.Bounds().Dy())
	}
}
