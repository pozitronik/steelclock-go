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
		Enabled: true,
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
			Enabled: false, // This widget is disabled
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

// TestCreateWidgets_ErrorPropagation tests that errors are properly propagated
func TestCreateWidgets_ErrorPropagation(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("clock"),
		{
			Type:    "invalid_type",
			ID:      "bad_widget",
			Enabled: true,
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
	}

	widgets, err := CreateWidgets(configs)
	if err == nil {
		t.Error("CreateWidgets() should return error when one widget fails")
	}

	if widgets != nil {
		t.Error("CreateWidgets() should return nil widgets list on error")
	}
}

// createDefaultConfig creates a default widget configuration for testing
func createDefaultConfig(widgetType string) config.WidgetConfig {
	cfg := config.WidgetConfig{
		Type:    widgetType,
		ID:      "test_" + widgetType,
		Enabled: true,
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: config.StyleConfig{
			BackgroundColor:   0,
			BackgroundOpacity: 255,
			Border:            false,
			BorderColor:       255,
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
