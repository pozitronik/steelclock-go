package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestCreateWidget_AllTypes tests widget creation for all supported widget types
func TestCreateWidget_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		widgetType string
		wantErr    bool
	}{
		{
			name:       "create clock widget",
			widgetType: "clock",
			wantErr:    false,
		},
		{
			name:       "create cpu widget",
			widgetType: "cpu",
			wantErr:    false,
		},
		{
			name:       "create memory widget",
			widgetType: "memory",
			wantErr:    false,
		},
		{
			name:       "create network widget",
			widgetType: "network",
			wantErr:    false,
		},
		{
			name:       "create disk widget",
			widgetType: "disk",
			wantErr:    false,
		},
		{
			name:       "create keyboard widget",
			widgetType: "keyboard",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createDefaultConfig(tt.widgetType)
			widget, err := CreateWidget(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWidget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && widget == nil {
				t.Error("CreateWidget() returned nil widget")
			}
		})
	}
}

// TestCreateWidget_InvalidType tests error handling for invalid widget types
func TestCreateWidget_InvalidType(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "invalid_type",
		ID:      "test",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	widget, err := CreateWidget(cfg)
	if err == nil {
		t.Error("CreateWidget() should return error for invalid type")
	}

	if widget != nil {
		t.Error("CreateWidget() should return nil widget for invalid type")
	}
}

// TestCreateWidgets_MultipleWidgets tests creating multiple widgets from configs
func TestCreateWidgets_MultipleWidgets(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("clock"),
		createDefaultConfig("memory"),
		createDefaultConfig("cpu"),
	}

	widgets, err := CreateWidgets(configs)
	if err != nil {
		t.Fatalf("CreateWidgets() error = %v", err)
	}

	if len(widgets) != 3 {
		t.Errorf("CreateWidgets() returned %d widgets, want 3", len(widgets))
	}
}

// TestCreateWidgets_Empty tests creating widgets from empty config list
func TestCreateWidgets_Empty(t *testing.T) {
	widgets, err := CreateWidgets([]config.WidgetConfig{})
	if err != nil {
		t.Errorf("CreateWidgets() with empty config should not error, got %v", err)
	}

	if len(widgets) != 0 {
		t.Errorf("CreateWidgets() with empty config should return empty list, got %d widgets", len(widgets))
	}
}

// TestCreateWidgets_DisabledWidget tests that disabled widgets are skipped
func TestCreateWidgets_DisabledWidget(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("clock"),
		{
			Type:    "memory",
			ID:      "disabled_widget",
			Enabled: config.BoolPtr(false), // This widget is disabled
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("cpu"),
	}

	widgets, err := CreateWidgets(configs)
	if err != nil {
		t.Fatalf("CreateWidgets() error = %v", err)
	}

	// Should only create 2 widgets (disabled one should be skipped)
	if len(widgets) != 2 {
		t.Errorf("CreateWidgets() returned %d widgets, want 2 (one disabled should be skipped)", len(widgets))
	}
}

// TestCreateWidgets_PartialFailure tests graceful handling when some widgets fail
func TestCreateWidgets_PartialFailure(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("clock"),
		{
			Type:    "invalid_type",
			ID:      "bad_widget",
			Enabled: config.BoolPtr(true),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("cpu"),
	}

	widgets, err := CreateWidgets(configs)

	// Should succeed with partial widgets (skip the failed one)
	if err != nil {
		t.Errorf("CreateWidgets() should not error when some widgets succeed, got: %v", err)
	}

	// Should create 2 widgets (clock and cpu), skipping the invalid one
	if len(widgets) != 2 {
		t.Errorf("CreateWidgets() returned %d widgets, want 2 (skipping 1 failed widget)", len(widgets))
	}
}

// TestCreateWidgets_AllFailed tests error when ALL widgets fail to initialize
func TestCreateWidgets_AllFailed(t *testing.T) {
	configs := []config.WidgetConfig{
		{
			Type:    "invalid_type_1",
			ID:      "bad_widget_1",
			Enabled: config.BoolPtr(true),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		{
			Type:    "invalid_type_2",
			ID:      "bad_widget_2",
			Enabled: config.BoolPtr(true),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
	}

	widgets, err := CreateWidgets(configs)

	// Should error when ALL widgets fail
	if err == nil {
		t.Error("CreateWidgets() should return error when all widgets fail")
	}

	// Should return nil widgets list on complete failure
	if widgets != nil {
		t.Errorf("CreateWidgets() should return nil widgets list when all fail, got %d widgets", len(widgets))
	}
}

// createDefaultConfig creates a default widget configuration for testing
func createDefaultConfig(widgetType string) config.WidgetConfig {
	cfg := config.WidgetConfig{
		Type:    widgetType,
		ID:      "test_" + widgetType,
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
			FontSize:          10,
			HorizontalAlign:   "center",
			VerticalAlign:     "center",
			UpdateInterval:    1.0,
			DisplayMode:       "text",
			FillColor:         255,
			HistoryLength:     30,
			BarBorder:         false,
			MaxSpeedMbps:      -1,
			RxColor:           255,
			TxColor:           255,
			ReadColor:         255,
			WriteColor:        255,
			IndicatorColorOn:  255,
			IndicatorColorOff: 100,
		},
	}

	// Type-specific configurations
	switch widgetType {
	case "clock":
		cfg.Properties.Format = "%H:%M:%S"
	case "network":
		iface := "eth0"
		cfg.Properties.Interface = &iface
	case "disk":
		disk := "sda"
		cfg.Properties.DiskName = &disk
	}

	return cfg
}

// TestCreateWidget_Disabled tests that explicitly disabled widgets return error
func TestCreateWidget_Disabled(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_disabled",
		Enabled: config.BoolPtr(false),
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
			Format:   "15:04",
			FontSize: 12,
		},
	}

	// Disabled widget should return error
	widget, err := CreateWidget(cfg)
	if err == nil {
		t.Error("CreateWidget() should return error for disabled widget")
	}

	if widget != nil {
		t.Error("CreateWidget() should return nil widget when disabled")
	}
}

// TestCreateWidget_EnabledByDefault tests that widgets are enabled by default when field is omitted
func TestCreateWidget_EnabledByDefault(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "clock",
		ID:   "test_default_enabled",
		// Enabled field not set - should default to true
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
			Format:   "15:04",
			FontSize: 12,
		},
	}

	// Widget without explicit enabled field should be enabled by default
	widget, err := CreateWidget(cfg)
	if err != nil {
		t.Errorf("CreateWidget() error = %v, widget should be enabled by default", err)
	}

	if widget == nil {
		t.Error("CreateWidget() should return widget when enabled by default")
	}
}

// TestCreateWidget_ExplicitlyEnabled tests that explicitly enabled widgets work
func TestCreateWidget_ExplicitlyEnabled(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "clock",
		ID:      "test_explicitly_enabled",
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
			Format:   "15:04",
			FontSize: 12,
		},
	}

	// Explicitly enabled widget should work
	widget, err := CreateWidget(cfg)
	if err != nil {
		t.Errorf("CreateWidget() error = %v, widget should be enabled", err)
	}

	if widget == nil {
		t.Error("CreateWidget() should return widget when explicitly enabled")
	}
}
