package widget

import (
	"testing"

	"github.com/pozitronik/steelclock/internal/config"
)

func TestNewClockWidget(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewClockWidget() returned nil")
	}

	if widget.Name() != "test_clock" {
		t.Errorf("Name() = %s, want test_clock", widget.Name())
	}
}

func TestClockWidgetUpdate(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04:05",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
	}

	// Update should populate currentTime
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	if widget.currentTime == "" {
		t.Error("Update() did not set currentTime")
	}
}

func TestClockWidgetRender(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_clock",
		Enabled: true,
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        12,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	widget, err := NewClockWidget(cfg)
	if err != nil {
		t.Fatalf("NewClockWidget() error = %v", err)
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

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}
