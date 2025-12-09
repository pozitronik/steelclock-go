package widget

import (
	"image"
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

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

// TestCreateWidgets_AllDisabled tests when all widgets are disabled
func TestCreateWidgets_AllDisabled(t *testing.T) {
	configs := []config.WidgetConfig{
		{
			Type:    "test_widget_disabled_1",
			ID:      "disabled1",
			Enabled: config.BoolPtr(false),
			Position: config.PositionConfig{
				X: 0, Y: 0, W: 128, H: 40,
			},
		},
		{
			Type:    "test_widget_disabled_2",
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

// TestRegisteredTypes_Mechanism tests the RegisteredTypes function mechanism
func TestRegisteredTypes_Mechanism(t *testing.T) {
	// Register a test type
	testType := "test_mechanism_type"
	Register(testType, func(cfg config.WidgetConfig) (Widget, error) {
		return nil, nil
	})

	types := RegisteredTypes()

	// Should be sorted
	sorted := make([]string, len(types))
	copy(sorted, types)
	for i := 0; i < len(sorted)-1; i++ {
		if sorted[i] > sorted[i+1] {
			t.Error("RegisteredTypes should return sorted list")
			break
		}
	}

	// Our test type should be in the list
	found := false
	for _, t := range types {
		if t == testType {
			found = true
			break
		}
	}
	if !found {
		t.Error("Test type should be in registered types")
	}

	t.Logf("Found %d registered types", len(types))
}

// TestRegisteredTypesList_Mechanism tests the RegisteredTypesList function mechanism
func TestRegisteredTypesList_Mechanism(t *testing.T) {
	// Register two test types to ensure comma separation
	Register("test_list_type_a", func(cfg config.WidgetConfig) (Widget, error) {
		return nil, nil
	})
	Register("test_list_type_b", func(cfg config.WidgetConfig) (Widget, error) {
		return nil, nil
	})

	list := RegisteredTypesList()

	// Should be comma-separated when multiple types exist
	if !strings.Contains(list, ", ") {
		t.Error("RegisteredTypesList should be comma-separated when multiple types exist")
	}

	t.Logf("Registered types list: %s", list)
}

// TestCreateWidget_WithTestFactory tests widget creation with a test-registered factory
func TestCreateWidget_WithTestFactory(t *testing.T) {
	testType := "test_factory_widget"

	// Register a test factory that creates a simple widget
	Register(testType, func(cfg config.WidgetConfig) (Widget, error) {
		base := NewBaseWidget(cfg)
		return &testWidget{BaseWidget: base}, nil
	})

	cfg := config.WidgetConfig{
		Type:    testType,
		ID:      "test_id",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	widget, err := CreateWidget(cfg)
	if err != nil {
		t.Errorf("CreateWidget() error = %v", err)
	}

	if widget == nil {
		t.Error("CreateWidget() returned nil widget")
	}

	if widget.Name() != "test_id" {
		t.Errorf("Widget name = %q, want %q", widget.Name(), "test_id")
	}
}

// testWidget is a minimal widget implementation for testing
type testWidget struct {
	*BaseWidget
}

func (w *testWidget) Render() (image.Image, error) {
	return w.CreateCanvas(), nil
}

func (w *testWidget) Update() error {
	return nil
}
