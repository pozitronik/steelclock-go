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

func TestWidgetManager_CreateErrorDisplay_DifferentMessages(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	messages := []string{
		"CONFIG",
		"NO WIDGETS",
		"ERROR",
		"",
		"Very long error message that exceeds normal display width",
	}

	for _, msg := range messages {
		t.Run(msg, func(t *testing.T) {
			setup := mgr.CreateErrorDisplay(client, msg, 128, 40)
			if setup == nil {
				t.Fatal("CreateErrorDisplay returned nil")
			}
			if len(setup.Widgets) != 1 {
				t.Errorf("Widgets count = %d, want 1", len(setup.Widgets))
			}
		})
	}
}

func TestWidgetManager_CreateErrorDisplay_DifferentDimensions(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	dimensions := []struct {
		width  int
		height int
	}{
		{128, 40},
		{256, 64},
		{64, 20},
		{0, 0},
		{128, 0},
		{0, 40},
	}

	for _, dim := range dimensions {
		t.Run("", func(t *testing.T) {
			setup := mgr.CreateErrorDisplay(client, "ERROR", dim.width, dim.height)
			if setup == nil {
				t.Fatal("CreateErrorDisplay returned nil")
			}
		})
	}
}

func TestWidgetManager_CreateFromConfig_MultipleWidgets(t *testing.T) {
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
					W: 64,
					H: 40,
				},
			},
			{
				ID:   "clock2",
				Type: "clock",
				Position: config.PositionConfig{
					X: 64,
					Y: 0,
					W: 64,
					H: 40,
				},
			},
		},
	}

	setup, err := mgr.CreateFromConfig(client, cfg)
	if err != nil {
		t.Fatalf("CreateFromConfig error: %v", err)
	}
	if len(setup.Widgets) != 2 {
		t.Errorf("Widgets count = %d, want 2", len(setup.Widgets))
	}
}

func TestWidgetManager_CreateFromConfig_DisabledWidget(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	disabled := false // false means disabled (Enabled is the field name)
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
				ID:      "clock1",
				Type:    "clock",
				Enabled: &disabled,
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 128,
					H: 40,
				},
			},
		},
	}

	// All widgets disabled - should return NoWidgetsError
	setup, err := mgr.CreateFromConfig(client, cfg)
	if setup != nil {
		t.Error("setup should be nil when all widgets disabled")
	}

	var noWidgetsErr *NoWidgetsError
	if err == nil || !errors.As(err, &noWidgetsErr) {
		t.Error("expected NoWidgetsError when all widgets disabled")
	}
}

func TestWidgetManager_CreateFromConfig_MixedDisabled(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	disabled := false // Enabled=false means disabled
	enabled := true   // Enabled=true means enabled
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
				ID:      "clock1",
				Type:    "clock",
				Enabled: &disabled,
				Position: config.PositionConfig{
					X: 0,
					Y: 0,
					W: 64,
					H: 40,
				},
			},
			{
				ID:      "clock2",
				Type:    "clock",
				Enabled: &enabled,
				Position: config.PositionConfig{
					X: 64,
					Y: 0,
					W: 64,
					H: 40,
				},
			},
		},
	}

	setup, err := mgr.CreateFromConfig(client, cfg)
	if err != nil {
		t.Fatalf("CreateFromConfig error: %v", err)
	}
	// Only one widget enabled
	if len(setup.Widgets) != 1 {
		t.Errorf("Widgets count = %d, want 1 (one disabled)", len(setup.Widgets))
	}
}

func TestWidgetManager_CreateFromConfig_ZeroRefreshRate(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	cfg := &config.Config{
		GameName:        "test",
		GameDisplayName: "Test",
		RefreshRateMs:   0, // Zero refresh rate
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
		t.Fatal("setup should not be nil")
	}
}

func TestWidgetManager_CreateFromConfig_SupportedResolutions(t *testing.T) {
	mgr := NewWidgetManager()
	client := testutil.NewTestClient()

	cfg := &config.Config{
		GameName:        "test",
		GameDisplayName: "Test",
		RefreshRateMs:   100,
		SupportedResolutions: []config.ResolutionConfig{
			{Width: 128, Height: 40},
			{Width: 256, Height: 64},
		},
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
		t.Fatal("setup should not be nil")
	}
}
