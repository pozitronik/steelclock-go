//go:build !windows && !linux

package widget

import (
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewAudioVisualizerWidget_Stub(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_stub",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
	}

	// On unsupported platforms, NewAudioVisualizerWidget should succeed (stub widget created)
	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v (should create stub widget)", err)
	}

	if widget == nil {
		t.Fatal("NewAudioVisualizerWidget() returned nil")
	}

	if widget.Name() != "test_stub" {
		t.Errorf("Name() = %s, want test_stub", widget.Name())
	}
}

func TestAudioVisualizerWidget_Stub_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_stub_render",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.033,
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Render should succeed (stub renders via ErrorWidget)
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v (stub should render without error)", err)
	}

	// Should return an image
	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}

func TestAudioVisualizerWidget_Stub_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_stub_update",
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

	// Update should succeed (delegates to ErrorWidget)
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v (should succeed)", err)
	}
}

func TestAudioVisualizerWidget_Stub_GetMethods(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_stub_getters",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 10,
			Y: 20,
			W: 128,
			H: 40,
			Z: 5,
		},
		Style: &config.StyleConfig{
			Background: 100,
			Border:     200,
		},
		Mode:           "spectrum",
		UpdateInterval: 0.05,
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Test Name()
	if widget.Name() != "test_stub_getters" {
		t.Errorf("Name() = %s, want test_stub_getters", widget.Name())
	}

	// Test GetPosition()
	pos := widget.GetPosition()
	if pos.X != 10 || pos.Y != 20 || pos.W != 128 || pos.H != 40 || pos.Z != 5 {
		t.Errorf("GetPosition() = %+v, want {X:10 Y:20 W:128 H:40 Z:5}", pos)
	}

	// Test GetStyle()
	style := widget.GetStyle()
	if style.Background != 100 || style.Border != 200 {
		t.Errorf("GetStyle() = %+v, want {Background:100 Border:200}", style)
	}

	// Test GetUpdateInterval()
	interval := widget.GetUpdateInterval()
	expectedInterval := int64(0.05 * 1e9) // 50ms in nanoseconds
	if interval.Nanoseconds() != expectedInterval {
		t.Errorf("GetUpdateInterval() = %v, want 50ms", interval)
	}
}

func TestGetSharedAudioCapture_Stub(t *testing.T) {
	// On unsupported platforms, GetSharedAudioCapture should return an error
	capture, err := GetSharedAudioCapture()
	if err == nil {
		t.Error("GetSharedAudioCapture() should return error on unsupported platform, got nil")
	}

	if capture != nil {
		t.Error("GetSharedAudioCapture() should return nil capture on unsupported platform, got non-nil")
	}

	// Error should mention platform limitation
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "platform") {
		t.Errorf("GetSharedAudioCapture() error should mention platform, got: %v", err)
	}
}

func TestAudioVisualizerWidget_Stub_BorderRendering(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "audio_visualizer",
		ID:      "test_stub_border",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     255,
		},
		Mode: "spectrum",
	}

	widget, err := NewAudioVisualizerWidget(cfg)
	if err != nil {
		t.Fatalf("NewAudioVisualizerWidget() error = %v", err)
	}

	// Render should succeed (stub renders via ErrorWidget)
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v (stub should render without error)", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	// Image should have correct dimensions
	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 40 {
		t.Errorf("image dimensions = %dx%d, want 128x40", img.Bounds().Dx(), img.Bounds().Dy())
	}
}
