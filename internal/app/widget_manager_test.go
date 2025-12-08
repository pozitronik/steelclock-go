package app

import (
	"errors"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/testutil"
)

func TestNewWidgetManager(t *testing.T) {
	mgr := NewWidgetManager()
	if mgr == nil {
		t.Fatal("NewWidgetManager returned nil")
	}
}

func TestWidgetManager_CreateFromConfig_NoWidgets(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	cfg := &config.Config{
		GameName:        "test",
		GameDisplayName: "Test",
		RefreshRateMs:   100,
		Display: config.DisplayConfig{
			Width:      128,
			Height:     40,
			Background: 0,
		},
		Widgets: []config.WidgetConfig{},
	}

	setup, err := mgr.CreateFromConfig(client, cfg)
	if setup != nil {
		t.Error("CreateFromConfig should return nil setup when no widgets")
	}

	var noWidgetsErr *NoWidgetsError
	if err == nil {
		t.Fatal("CreateFromConfig should return NoWidgetsError")
	}
	if !errors.As(err, &noWidgetsErr) {
		t.Errorf("expected NoWidgetsError, got %T", err)
	}
}

func TestWidgetManager_CreateFromConfig_WithWidgets(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	cfg := &config.Config{
		GameName:        "test",
		GameDisplayName: "Test",
		RefreshRateMs:   100,
		Display: config.DisplayConfig{
			Width:      128,
			Height:     40,
			Background: 0,
		},
		Widgets: []config.WidgetConfig{
			{
				ID:   "clock1",
				Type: "clock",
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
			},
		},
	}

	setup, err := mgr.CreateFromConfig(client, cfg)
	if err != nil {
		t.Fatalf("CreateFromConfig error: %v", err)
	}
	if setup == nil {
		t.Fatal("CreateFromConfig returned nil setup")
	}
	if setup.Compositor == nil {
		t.Error("Compositor is nil")
	}
	if setup.Layout == nil {
		t.Error("Layout is nil")
	}
	if len(setup.Widgets) != 1 {
		t.Errorf("Widgets count = %d, want 1", len(setup.Widgets))
	}
}

func TestWidgetManager_CreateErrorDisplay(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	setup := mgr.CreateErrorDisplay(client, "TEST ERROR", 128, 40)
	if setup == nil {
		t.Fatal("CreateErrorDisplay returned nil")
	}
	if setup.Compositor == nil {
		t.Error("Compositor is nil")
	}
	if setup.Layout == nil {
		t.Error("Layout is nil")
	}
	if len(setup.Widgets) != 1 {
		t.Errorf("Widgets count = %d, want 1", len(setup.Widgets))
	}
}
