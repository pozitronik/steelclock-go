package cpu

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
)

// TestWidget_WithMockProvider demonstrates how to use mock providers for testing.
//
// ANALYSIS: The current architecture supports mock injection via direct field assignment
// after widget creation. Since tests are in the same package, they can access the
// unexported cpuProvider field.
//
// This pattern works but has drawbacks:
// 1. Requires creating widget first (calls real provider in New() for core count)
// 2. Provider swap happens after construction, not during
// 3. Tests must know about internal implementation details
//
// The alternative (constructor injection) would be cleaner but:
// 1. Adds complexity to New() signature
// 2. Requires nil-checking and fallback logic
// 3. Changes the public API
//
// DECISION: Current approach is acceptable given the trade-offs.
// Document the pattern for future test writers.
func TestWidget_WithMockProvider(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_mock",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Inject mock provider (possible because test is in same package)
	mockProvider := &metrics.MockCPU{
		PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
			return []float64{75.5}, nil // Controlled value
		},
	}
	widget.cpuProvider = mockProvider

	// Now Update() uses our mock
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify the exact value we set
	widget.mu.RLock()
	usage := widget.currentUsageSingle
	widget.mu.RUnlock()

	if usage != 75.5 {
		t.Errorf("currentUsageSingle = %f, want 75.5", usage)
	}
}

// TestWidget_MockProvider_PerCore tests per-core mode with mock provider
func TestWidget_MockProvider_PerCore(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_mock_percore",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:    "bar_vertical",
		PerCore: &config.PerCoreConfig{Enabled: true},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Mock with specific per-core values
	expectedValues := []float64{10.0, 30.0, 50.0, 70.0}
	mockProvider := &metrics.MockCPU{
		PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
			if perCore {
				return expectedValues, nil
			}
			return []float64{40.0}, nil
		},
	}
	widget.cpuProvider = mockProvider

	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	widget.mu.RLock()
	actualValues := widget.currentUsagePerCore
	widget.mu.RUnlock()

	if len(actualValues) != len(expectedValues) {
		t.Fatalf("per-core values length = %d, want %d", len(actualValues), len(expectedValues))
	}

	for i, expected := range expectedValues {
		if actualValues[i] != expected {
			t.Errorf("core %d usage = %f, want %f", i, actualValues[i], expected)
		}
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
				Type:    "cpu",
				ID:      "test_cpu_edge",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Mode: "text",
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			widget.cpuProvider = &metrics.MockCPU{
				PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
					return []float64{tt.mockValue}, nil
				},
			}

			err = widget.Update()
			if err != nil {
				t.Errorf("Update() error = %v", err)
			}

			widget.mu.RLock()
			actual := widget.currentUsageSingle
			widget.mu.RUnlock()

			if actual != tt.expectedValue {
				t.Errorf("value = %f, want %f", actual, tt.expectedValue)
			}
		})
	}
}

// TestWidget_MockProvider_ErrorHandling tests error handling with mock
func TestWidget_MockProvider_ErrorHandling(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "cpu",
		ID:      "test_cpu_error",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Mock that returns error
	expectedErr := metrics.MockCPU{
		PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
			return nil, &testError{"simulated CPU error"}
		},
	}
	widget.cpuProvider = &expectedErr

	err = widget.Update()
	if err == nil {
		t.Error("Update() should return error when provider fails")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
