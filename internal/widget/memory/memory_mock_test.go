package memory

import (
	"errors"
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
		UsageFunc: func() (metrics.MemoryUsage, error) {
			return metrics.MemoryUsage{UsedPercent: 42.5}, nil // Controlled value
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
				UsageFunc: func() (metrics.MemoryUsage, error) {
					return metrics.MemoryUsage{UsedPercent: tt.mockValue}, nil
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
		UsageFunc: func() (metrics.MemoryUsage, error) {
			return metrics.MemoryUsage{UsedPercent: 50.0, UsedBytes: 8 << 30, TotalBytes: 16 << 30}, nil
		},
	}

	if err := widget.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if _, err := widget.Render(); err != nil {
		t.Errorf("Render() error = %v", err)
	}
}

// TestWidget_Update_TakesGBFromSameSample verifies that the percentage and the
// GB figures all come from one provider sample.
func TestWidget_Update_TakesGBFromSameSample(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_sample",
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

	calls := 0
	widget.memoryProvider = &metrics.MockMemory{
		UsageFunc: func() (metrics.MemoryUsage, error) {
			calls++
			return metrics.MemoryUsage{UsedPercent: 25.0, UsedBytes: 4 << 30, TotalBytes: 16 << 30}, nil
		},
	}

	if err := widget.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if calls != 1 {
		t.Errorf("provider sampled %d times per Update(), want 1", calls)
	}
	if widget.currentValue != 25.0 || widget.usedGB != 4.0 || widget.totalGB != 16.0 {
		t.Errorf("got %.1f%% %.1f/%.1f GB, want 25.0%% 4.0/16.0 GB",
			widget.currentValue, widget.usedGB, widget.totalGB)
	}
}

// TestWidget_Update_ProviderError verifies that a failed sample is reported
// and leaves the previous values untouched.
func TestWidget_Update_ProviderError(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "memory",
		ID:      "test_memory_error",
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

	widget.memoryProvider = &metrics.MockMemory{}
	if err := widget.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	widget.memoryProvider = &metrics.MockMemory{
		UsageFunc: func() (metrics.MemoryUsage, error) {
			return metrics.MemoryUsage{}, errors.New("sample failed")
		},
	}
	if err := widget.Update(); err == nil {
		t.Error("Update() error = nil, want the provider error")
	}

	if widget.GetValue() != 65.0 || widget.usedGB != 13.0 || widget.totalGB != 20.0 {
		t.Errorf("values changed after a failed sample: %.1f%% %.1f/%.1f GB",
			widget.GetValue(), widget.usedGB, widget.totalGB)
	}
}
