package layout

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func TestNewManager(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	clockCfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock1",
		Enabled: config.BoolPtr(true),
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

	clockWidget, err := widget.NewClockWidget(clockCfg)
	if err != nil {
		t.Fatalf("failed to create clock widget: %v", err)
	}

	widgets := []widget.Widget{clockWidget}

	mgr := NewManager(displayCfg, widgets)

	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestManagerComposite(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	clockCfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock1",
		Enabled: config.BoolPtr(true),
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

	clockWidget, err := widget.NewClockWidget(clockCfg)
	if err != nil {
		t.Fatalf("failed to create clock widget: %v", err)
	}

	// Update widget before compositing
	if err := clockWidget.Update(); err != nil {
		t.Fatalf("failed to update widget: %v", err)
	}

	widgets := []widget.Widget{clockWidget}
	mgr := NewManager(displayCfg, widgets)

	img, err := mgr.Composite()
	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	if img == nil {
		t.Fatal("Composite() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("composite width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("composite height = %d, want 40", img.Bounds().Dy())
	}
}

func TestManagerCompositeMultipleWidgets(t *testing.T) {
	displayCfg := config.DisplayConfig{
		Width:  128,
		Height: 40,
	}

	clock1Cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock1",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 64,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04",
			FontSize:        10,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	clock2Cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "clock2",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 64,
			Y: 0,
			W: 64,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          true,
			BorderColor:     255,
		},
		Properties: config.WidgetProperties{
			Format:          "15:04:05",
			FontSize:        10,
			HorizontalAlign: "center",
			VerticalAlign:   "center",
		},
	}

	clock1, err := widget.NewClockWidget(clock1Cfg)
	if err != nil {
		t.Fatalf("failed to create clock1: %v", err)
	}

	clock2, err := widget.NewClockWidget(clock2Cfg)
	if err != nil {
		t.Fatalf("failed to create clock2: %v", err)
	}

	if err := clock1.Update(); err != nil {
		t.Fatalf("failed to update clock1: %v", err)
	}
	if err := clock2.Update(); err != nil {
		t.Fatalf("failed to update clock2: %v", err)
	}

	widgets := []widget.Widget{clock1, clock2}
	mgr := NewManager(displayCfg, widgets)

	img, err := mgr.Composite()
	if err != nil {
		t.Fatalf("Composite() error = %v", err)
	}

	if img == nil {
		t.Fatal("Composite() returned nil image")
	}
}
