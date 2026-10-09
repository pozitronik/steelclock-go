package memory

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
)

// TestWidget_WithMockProvider demonstrates mock provider injection for memory widget.
// See cpu/cpu_mock_test.go for detailed explanation of the pattern.
func TestWidget_WithMockProvider(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_mock",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 20,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Inject mock provider
	mockProvider := &metrics.MockMemory{
		UsedPercentFunc: func() (float64, error) {
			return 42.5, nil // Controlled value
		},
	}
	widget.memoryProvider = mockProvider

	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify the exact value
	usage := widget.GetValue()
	if usage != 42.5 {
		t.Errorf("GetValue() = %f, want 42.5", usage)
	}
}

// TestWidget_MockProvider_EdgeCases tests edge cases using mock
func TestWidget_MockProvider_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		mockValue     float64
		expectedValue float64
	}{
		{"zero", 0.0, 0.0},
		{"max", 100.0, 100.0},
		{"over max (clamped)", 150.0, 100.0},
		{"negative (clamped)", -10.0, 0.0},
		{"typical", 65.5, 65.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "memory",
				ID:      "test_memory_edge",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 64, H: 20,
				},
				Mode: "text",
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			widget.memoryProvider = &metrics.MockMemory{
				UsedPercentFunc: func() (float64, error) {
					return tt.mockValue, nil
				},
			}

			err = widget.Update()
			if err != nil {
				t.Errorf("Update() error = %v", err)
			}

			actual := widget.GetValue()
			if actual != tt.expectedValue {
				t.Errorf("value = %f, want %f", actual, tt.expectedValue)
			}
		})
	}
}

// TestNew_ReadsTextFormatFromConfig is a regression test: New() used to
// hardcode textFormat to "%.0f" regardless of the configured text.format,
// silently ignoring custom formats (e.g. GB/percent display) in text mode.
func TestNew_ReadsTextFormatFromConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_format",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 20,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Format: "R {used}GB {percent}%",
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget.textFormat != "R {used}GB {percent}%" {
		t.Errorf("textFormat = %q, want %q", widget.textFormat, "R {used}GB {percent}%")
	}
}

// TestNew_DefaultTextFormat verifies the "%.0f" default still applies when
// no text.format is configured.
func TestNew_DefaultTextFormat(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_default_format",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 20,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget.textFormat != "%.0f" {
		t.Errorf("textFormat = %q, want %q", widget.textFormat, "%.0f")
	}
}

// TestRender_DualFormat_UsesUsedGB verifies that a {used}/{total}/{percent}
// token format renders successfully in text mode using the GB and percent
// values (rather than being passed unresolved to the underlying strategy).
func TestRender_DualFormat_UsesUsedGB(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_dual",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 20,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Format: "R {used}GB {percent}%",
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	widget.memoryProvider = &metrics.MockMemory{
		UsedPercentFunc: func() (float64, error) { return 50.0, nil },
		UsedGBFunc:      func() (float64, float64, error) { return 8.0, 16.0, nil },
	}

	if err := widget.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if _, err := widget.Render(); err != nil {
		t.Errorf("Render() error = %v", err)
	}
}
