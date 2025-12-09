package widget_test

import (
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"

	// Import all widget subpackages to trigger their init() registration
	_ "github.com/pozitronik/steelclock-go/internal/widget/audiovisualizer"
	_ "github.com/pozitronik/steelclock-go/internal/widget/battery"
	_ "github.com/pozitronik/steelclock-go/internal/widget/clock"
	_ "github.com/pozitronik/steelclock-go/internal/widget/cpu"
	_ "github.com/pozitronik/steelclock-go/internal/widget/disk"
	_ "github.com/pozitronik/steelclock-go/internal/widget/doom"
	_ "github.com/pozitronik/steelclock-go/internal/widget/gameoflife"
	_ "github.com/pozitronik/steelclock-go/internal/widget/hyperspace"
	_ "github.com/pozitronik/steelclock-go/internal/widget/keyboard"
	_ "github.com/pozitronik/steelclock-go/internal/widget/keyboardlayout"
	_ "github.com/pozitronik/steelclock-go/internal/widget/matrix"
	_ "github.com/pozitronik/steelclock-go/internal/widget/memory"
	_ "github.com/pozitronik/steelclock-go/internal/widget/network"
	_ "github.com/pozitronik/steelclock-go/internal/widget/starwarsintro"
	_ "github.com/pozitronik/steelclock-go/internal/widget/telegramcounter"
	_ "github.com/pozitronik/steelclock-go/internal/widget/telegramwidget"
	_ "github.com/pozitronik/steelclock-go/internal/widget/volume"
	_ "github.com/pozitronik/steelclock-go/internal/widget/volumemeter"
	_ "github.com/pozitronik/steelclock-go/internal/widget/weather"
	_ "github.com/pozitronik/steelclock-go/internal/widget/winampwidget"
)

// TestCreateWidget_AllTypes tests widget creation for all supported widget types
func TestCreateWidget_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		widgetType string
		wantErr    bool
		skipShort  bool // Skip in short mode (e.g., gore library has checkptr issues with race detection)
	}{
		{
			name:       "create keyboard widget",
			widgetType: "keyboard",
			wantErr:    false,
		},
		{
			name:       "create keyboard_layout widget",
			widgetType: "keyboard_layout",
			wantErr:    false,
		},
		{
			name:       "create winamp widget",
			widgetType: "winamp",
			wantErr:    false,
		},
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
			name:       "create battery widget",
			widgetType: "battery",
			wantErr:    false,
		},
		{
			name:       "create disk widget",
			widgetType: "disk",
			wantErr:    false,
		},
		{
			name:       "create network widget",
			widgetType: "network",
			wantErr:    false,
		},
		{
			name:       "create matrix widget",
			widgetType: "matrix",
			wantErr:    false,
		},
		{
			name:       "create hyperspace widget",
			widgetType: "hyperspace",
			wantErr:    false,
		},
		{
			name:       "create game_of_life widget",
			widgetType: "game_of_life",
			wantErr:    false,
		},
		{
			name:       "create starwars_intro widget",
			widgetType: "starwars_intro",
			wantErr:    false,
		},
		{
			name:       "create doom widget",
			widgetType: "doom",
			wantErr:    false,
			skipShort:  true, // gore library has checkptr issues with race detection
		},
		{
			name:       "create volume widget",
			widgetType: "volume",
			wantErr:    false,
		},
		{
			name:       "create volume_meter widget",
			widgetType: "volume_meter",
			wantErr:    false,
		},
		{
			name:       "create audio_visualizer widget",
			widgetType: "audio_visualizer",
			wantErr:    false,
		},
		{
			name:       "create telegram widget (requires auth)",
			widgetType: "telegram",
			wantErr:    true, // telegram requires auth configuration
		},
		{
			name:       "create telegram_counter widget (requires auth)",
			widgetType: "telegram_counter",
			wantErr:    true, // telegram requires auth configuration
		},
		{
			name:       "create weather widget (requires api_key)",
			widgetType: "weather",
			wantErr:    true, // weather requires api_key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipShort && testing.Short() {
				t.Skip("Skipping in short mode (checkptr issues with race detection)")
			}

			cfg := createDefaultConfig(tt.widgetType)
			w, err := widget.CreateWidget(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWidget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && w == nil {
				t.Error("CreateWidget() returned nil widget")
			}
		})
	}
}

// TestCreateWidgets_MultipleWidgets tests creating multiple widgets from configs
func TestCreateWidgets_MultipleWidgets(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("keyboard"),
		createDefaultConfig("keyboard_layout"),
		createDefaultConfig("winamp"),
	}

	widgets, err := widget.CreateWidgets(configs)
	if err != nil {
		t.Fatalf("CreateWidgets() error = %v", err)
	}

	if len(widgets) != 3 {
		t.Errorf("CreateWidgets() returned %d widgets, want 3", len(widgets))
	}
}

// TestCreateWidgets_DisabledWidget tests that disabled widgets are skipped
func TestCreateWidgets_DisabledWidget(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("keyboard"),
		{
			Type:    "keyboard_layout",
			ID:      "disabled_widget",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("winamp"),
	}

	widgets, err := widget.CreateWidgets(configs)
	if err != nil {
		t.Fatalf("CreateWidgets() error = %v", err)
	}

	if len(widgets) != 2 {
		t.Errorf("CreateWidgets() returned %d widgets, want 2 (one disabled should be skipped)", len(widgets))
	}
}

// TestCreateWidgets_PartialFailure tests graceful handling when some widgets fail
func TestCreateWidgets_PartialFailure(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("keyboard"),
		{
			Type:    "invalid_type",
			ID:      "bad_widget",
			Enabled: config.BoolPtr(true),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("winamp"),
	}

	widgets, err := widget.CreateWidgets(configs)

	if err != nil {
		t.Errorf("CreateWidgets() should not error when some widgets succeed, got: %v", err)
	}

	// Should create 3 widgets (keyboard, winamp, and error proxy for the failed one)
	if len(widgets) != 3 {
		t.Errorf("CreateWidgets() returned %d widgets, want 3 (2 good + 1 error proxy)", len(widgets))
	}
}

// TestCreateWidget_Disabled tests that explicitly disabled widgets return error
func TestCreateWidget_Disabled(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "winamp",
		ID:      "test_disabled",
		Enabled: config.BoolPtr(false),
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
	}

	w, err := widget.CreateWidget(cfg)
	if err == nil {
		t.Error("CreateWidget() should return error for disabled widget")
	}

	if w != nil {
		t.Error("CreateWidget() should return nil widget when disabled")
	}
}

// TestCreateWidget_EnabledByDefault tests that widgets are enabled by default when field is omitted
func TestCreateWidget_EnabledByDefault(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		ID:   "test_default_enabled",
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
	}

	w, err := widget.CreateWidget(cfg)
	if err != nil {
		t.Errorf("CreateWidget() error = %v, widget should be enabled by default", err)
	}

	if w == nil {
		t.Error("CreateWidget() should return widget when enabled by default")
	}
}

// TestCreateWidget_ExplicitlyEnabled tests that explicitly enabled widgets work
func TestCreateWidget_ExplicitlyEnabled(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "winamp",
		ID:      "test_explicitly_enabled",
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
	}

	w, err := widget.CreateWidget(cfg)
	if err != nil {
		t.Errorf("CreateWidget() error = %v, widget should be enabled", err)
	}

	if w == nil {
		t.Error("CreateWidget() should return widget when explicitly enabled")
	}
}

// TestCreateWidgets_MixedEnabledDisabled tests mix of enabled and disabled widgets
func TestCreateWidgets_MixedEnabledDisabled(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("keyboard"),
		{
			Type:    "keyboard_layout",
			ID:      "disabled1",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("winamp"),
		{
			Type:    "telegram",
			ID:      "disabled2",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("volume"),
	}

	widgets, err := widget.CreateWidgets(configs)
	if err != nil {
		t.Fatalf("CreateWidgets() error = %v", err)
	}

	if len(widgets) != 3 {
		t.Errorf("CreateWidgets() returned %d widgets, want 3", len(widgets))
	}
}

// TestCreateWidget_ErrorMessage tests error message format
func TestCreateWidget_ErrorMessage(t *testing.T) {
	t.Run("unknown type error message", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Type:    "nonexistent",
			ID:      "test",
			Enabled: config.BoolPtr(true),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		}

		_, err := widget.CreateWidget(cfg)
		if err == nil {
			t.Fatal("CreateWidget() should return error for unknown type")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "unknown widget type: nonexistent") {
			t.Errorf("Error message should contain 'unknown widget type: nonexistent', got %q", errMsg)
		}
		if !strings.Contains(errMsg, "valid:") {
			t.Errorf("Error message should contain list of valid types, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "winamp") {
			t.Errorf("Error message should list 'winamp' as valid type, got %q", errMsg)
		}
	})

	t.Run("disabled widget error message", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Type:    "winamp",
			ID:      "my_winamp",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		}

		_, err := widget.CreateWidget(cfg)
		if err == nil {
			t.Fatal("CreateWidget() should return error for disabled widget")
		}

		expectedMsg := "widget my_winamp is disabled"
		if err.Error() != expectedMsg {
			t.Errorf("Error message = %q, want %q", err.Error(), expectedMsg)
		}
	})
}

// TestRegisteredTypes tests that all expected widget types are registered
func TestRegisteredTypes(t *testing.T) {
	types := widget.RegisteredTypes()

	// Should return a sorted list
	for i := 0; i < len(types)-1; i++ {
		if types[i] > types[i+1] {
			t.Error("RegisteredTypes should return sorted list")
			break
		}
	}

	// All widget types should be registered via subpackage init()
	expectedTypes := []string{
		"audio_visualizer",
		"battery",
		"clock",
		"cpu",
		"disk",
		"doom",
		"game_of_life",
		"hyperspace",
		"keyboard",
		"keyboard_layout",
		"matrix",
		"memory",
		"network",
		"starwars_intro",
		"telegram",
		"telegram_counter",
		"volume",
		"volume_meter",
		"weather",
		"winamp",
	}

	for _, expected := range expectedTypes {
		found := false
		for _, t := range types {
			if t == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected widget type %q to be registered", expected)
		}
	}

	t.Logf("Found %d registered types: %v", len(types), types)
}

// TestRegisteredTypesList tests the RegisteredTypesList function
func TestRegisteredTypesList(t *testing.T) {
	list := widget.RegisteredTypesList()

	if !strings.Contains(list, ", ") {
		t.Error("RegisteredTypesList should be comma-separated")
	}

	if !strings.Contains(list, "winamp") {
		t.Error("RegisteredTypesList should contain 'winamp'")
	}

	if !strings.Contains(list, "clock") {
		t.Error("RegisteredTypesList should contain 'clock'")
	}

	t.Logf("Registered types list: %s", list)
}

// createDefaultConfig creates a default widget configuration for testing
func createDefaultConfig(widgetType string) config.WidgetConfig {
	cfg := config.WidgetConfig{
		Type:           widgetType,
		ID:             "test_" + widgetType,
		Enabled:        config.BoolPtr(true),
		UpdateInterval: 1.0,
		Mode:           "text",
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
		Text: &config.TextConfig{
			Size: 10,
			Align: &config.AlignConfig{
				H: "center",
				V: "center",
			},
		},
		Colors: &config.ColorsConfig{
			Fill:  config.IntPtr(255),
			Rx:    config.IntPtr(255),
			Tx:    config.IntPtr(255),
			Read:  config.IntPtr(255),
			Write: config.IntPtr(255),
			On:    config.IntPtr(255),
			Off:   config.IntPtr(100),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
		Bar: &config.BarConfig{
			Border: false,
		},
	}

	// Type-specific configurations
	switch widgetType {
	case "network":
		iface := "eth0"
		cfg.Interface = &iface
		cfg.MaxSpeedMbps = -1
	case "disk":
		disk := "sda"
		cfg.Disk = &disk
	}

	return cfg
}
