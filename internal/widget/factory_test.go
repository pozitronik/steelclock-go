package widget

import (
	"strings"
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
		{
			name:       "create keyboard_layout widget",
			widgetType: "keyboard_layout",
			wantErr:    false,
		},
		{
			name:       "create matrix widget",
			widgetType: "matrix",
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
		createDefaultConfig("battery"),
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
			Type:    "battery",
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

	// Should succeed (no error returned)
	if err != nil {
		t.Errorf("CreateWidgets() should not error when some widgets succeed, got: %v", err)
	}

	// Should create 3 widgets (clock, cpu, and error proxy for the failed one)
	if len(widgets) != 3 {
		t.Errorf("CreateWidgets() returned %d widgets, want 3 (2 good + 1 error proxy)", len(widgets))
	}

	// Verify we have an error widget for the failed one
	errorCount := 0
	for _, w := range widgets {
		if _, ok := w.(*ErrorWidget); ok {
			errorCount++
		}
	}
	if errorCount != 1 {
		t.Errorf("CreateWidgets() should have created 1 error widget, got %d", errorCount)
	}
}

// TestCreateWidgets_AllFailed tests that error proxies are created when ALL widgets fail
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

	// Should NOT error - error proxies are created instead
	if err != nil {
		t.Errorf("CreateWidgets() should not error when error proxies can be created, got: %v", err)
	}

	// Should return 2 error widgets (one for each failed widget)
	if len(widgets) != 2 {
		t.Errorf("CreateWidgets() should return 2 error widgets, got %d", len(widgets))
	}

	// All widgets should be error widgets
	for i, w := range widgets {
		if _, ok := w.(*ErrorWidget); !ok {
			t.Errorf("Widget %d should be ErrorWidget, got %T", i, w)
		}
	}
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
	case "clock":
		cfg.Text.Format = "%H:%M:%S"
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
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
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Format: "15:04",
			Size:   12,
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

// TestCreateWidgets_MixedEnabledDisabled tests mix of enabled and disabled widgets
func TestCreateWidgets_MixedEnabledDisabled(t *testing.T) {
	configs := []config.WidgetConfig{
		createDefaultConfig("clock"),
		{
			Type:    "battery",
			ID:      "disabled1",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("cpu"),
		{
			Type:    "network",
			ID:      "disabled2",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		createDefaultConfig("disk"),
	}

	widgets, err := CreateWidgets(configs)
	if err != nil {
		t.Fatalf("CreateWidgets() error = %v", err)
	}

	// Should only create 3 widgets (clock, cpu, disk)
	if len(widgets) != 3 {
		t.Errorf("CreateWidgets() returned %d widgets, want 3", len(widgets))
	}
}

// TestCreateWidgets_AllDisabled tests when all widgets are disabled
func TestCreateWidgets_AllDisabled(t *testing.T) {
	configs := []config.WidgetConfig{
		{
			Type:    "clock",
			ID:      "disabled1",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		{
			Type:    "battery",
			ID:      "disabled2",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
	}

	widgets, err := CreateWidgets(configs)

	// Should not error when all are disabled (no enabled widgets to create)
	if err != nil {
		t.Errorf("CreateWidgets() with all disabled should not error, got: %v", err)
	}

	// Should return empty list
	if len(widgets) != 0 {
		t.Errorf("CreateWidgets() returned %d widgets, want 0", len(widgets))
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

		_, err := CreateWidget(cfg)
		if err == nil {
			t.Fatal("CreateWidget() should return error for unknown type")
		}

		// Error should contain the type name and list of valid types
		errMsg := err.Error()
		if !strings.Contains(errMsg, "unknown widget type: nonexistent") {
			t.Errorf("Error message should contain 'unknown widget type: nonexistent', got %q", errMsg)
		}
		if !strings.Contains(errMsg, "valid:") {
			t.Errorf("Error message should contain list of valid types, got %q", errMsg)
		}
		if !strings.Contains(errMsg, "matrix") {
			t.Errorf("Error message should list 'matrix' as valid type, got %q", errMsg)
		}
	})

	t.Run("disabled widget error message", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Type:    "clock",
			ID:      "my_clock",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		}

		_, err := CreateWidget(cfg)
		if err == nil {
			t.Fatal("CreateWidget() should return error for disabled widget")
		}

		expectedMsg := "widget my_clock is disabled"
		if err.Error() != expectedMsg {
			t.Errorf("Error message = %q, want %q", err.Error(), expectedMsg)
		}
	})
}

// TestRegister_Reregister tests that re-registering a widget type logs a warning
func TestRegister_Reregister(t *testing.T) {
	// Create a test factory
	testFactory := func(cfg config.WidgetConfig) (Widget, error) {
		return nil, nil
	}

	// Register a unique test type
	uniqueType := "test_reregister_type_12345"
	Register(uniqueType, testFactory)

	// Re-register the same type - should log a warning (we just verify it doesn't panic)
	Register(uniqueType, testFactory)

	// Verify it's still in the registry
	types := RegisteredTypes()
	found := false
	for _, t := range types {
		if t == uniqueType {
			found = true
			break
		}
	}
	if !found {
		t.Error("Re-registered type should still be in registry")
	}
}

// TestAbbreviateError tests error message abbreviation for small screens
func TestAbbreviateError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "api_key pattern",
			errMsg:   "api_key is required for this widget",
			expected: "NO API KEY",
		},
		{
			name:     "lat/lon pattern",
			errMsg:   "lat/lon coordinates must be provided",
			expected: "NO COORDS",
		},
		{
			name:     "location pattern",
			errMsg:   "location is required",
			expected: "NO LOCATION",
		},
		{
			name:     "unknown widget type pattern",
			errMsg:   "unknown widget type: foobar",
			expected: "BAD TYPE",
		},
		{
			name:     "font error pattern",
			errMsg:   "failed to load font from path",
			expected: "FONT ERROR",
		},
		{
			name:     "parse error pattern",
			errMsg:   "failed to parse configuration",
			expected: "PARSE ERROR",
		},
		{
			name:     "timeout pattern",
			errMsg:   "timeout waiting for response",
			expected: "TIMEOUT",
		},
		{
			name:     "connection refused pattern",
			errMsg:   "connection refused by server",
			expected: "NO CONNECT",
		},
		{
			name:     "permission denied pattern",
			errMsg:   "permission denied accessing file",
			expected: "NO ACCESS",
		},
		{
			name:     "long unknown error - truncated to ERROR",
			errMsg:   "this is a very long error message that does not match any pattern",
			expected: "ERROR",
		},
		{
			name:     "short unknown error - uppercase",
			errMsg:   "oops",
			expected: "OOPS",
		},
		{
			name:     "exactly 12 chars - uppercase",
			errMsg:   "twelve chars",
			expected: "TWELVE CHARS",
		},
		{
			name:     "13 chars - truncated to ERROR",
			errMsg:   "thirteen char",
			expected: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abbreviateError(tt.errMsg)
			if result != tt.expected {
				t.Errorf("abbreviateError(%q) = %q, want %q", tt.errMsg, result, tt.expected)
			}
		})
	}
}

// TestRegisteredTypes tests the RegisteredTypes function
func TestRegisteredTypes(t *testing.T) {
	types := RegisteredTypes()

	// Should return a sorted list
	sorted := make([]string, len(types))
	copy(sorted, types)
	// Sort to verify
	for i := 0; i < len(sorted)-1; i++ {
		if sorted[i] > sorted[i+1] {
			t.Error("RegisteredTypes should return sorted list")
			break
		}
	}

	// Should contain known types (note: memory is in subpackage, not registered here)
	knownTypes := []string{"clock", "cpu", "matrix", "battery"}
	for _, known := range knownTypes {
		found := false
		for _, t := range types {
			if t == known {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RegisteredTypes should contain '%s'", known)
		}
	}
}

// TestRegisteredTypesList tests the RegisteredTypesList function
func TestRegisteredTypesList(t *testing.T) {
	list := RegisteredTypesList()

	// Should be comma-separated
	if !strings.Contains(list, ", ") {
		t.Error("RegisteredTypesList should be comma-separated")
	}

	// Should contain known types
	if !strings.Contains(list, "clock") {
		t.Error("RegisteredTypesList should contain 'clock'")
	}
}
