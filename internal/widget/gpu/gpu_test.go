package gpu

import (
	"fmt"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
)

// mockReader is a test implementation of Reader
type mockReader struct {
	metricValue float64
	adapters    []AdapterInfo
	returnErr   error
}

func (m *mockReader) GetMetric(_ int, _ string) (float64, error) {
	if m.returnErr != nil {
		return 0, m.returnErr
	}
	return m.metricValue, nil
}

func (m *mockReader) ListAdapters() ([]AdapterInfo, error) {
	return m.adapters, nil
}

func (m *mockReader) Close() {}

// newTestWidget creates a GPU widget with the given mock reader pre-injected.
// This avoids the two-step pattern of creating a widget then replacing internal fields.
func newTestWidget(t *testing.T, mode string, reader Reader) *Widget {
	t.Helper()
	cfg := config.WidgetConfig{
		Type:    "gpu",
		ID:      "test_gpu",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:  mode,
		Graph: &config.GraphConfig{History: 30},
	}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	w.reader = reader
	w.readerFailed = false
	return w
}

// TestNew tests successful GPU widget creation
func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "gpu",
		ID:      "test_gpu",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		GPU: &config.GPUConfig{
			Adapter: 0,
			Metric:  "utilization",
		},
		Text: &config.TextConfig{
			Size:  10,
			Align: &config.AlignConfig{H: "center", V: "center"},
		},
		Colors: &config.ColorsConfig{
			Fill: config.IntPtr(255),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget == nil {
		t.Fatal("New() returned nil")
	}

	if widget.Name() != "test_gpu" {
		t.Errorf("Name() = %s, want test_gpu", widget.Name())
	}
}

// TestNew_Defaults tests that defaults are properly applied
func TestNew_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "gpu",
		ID:      "test_gpu_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		// Intentionally leave fields empty to test defaults
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify defaults
	if widget.displayMode != render.DisplayModeText {
		t.Errorf("default displayMode = %s, want text", widget.displayMode)
	}

	if widget.metric != MetricUtilization {
		t.Errorf("default metric = %s, want %s", widget.metric, MetricUtilization)
	}

	if widget.adapter != 0 {
		t.Errorf("default adapter = %d, want 0", widget.adapter)
	}

	if widget.historyLen != 30 {
		t.Errorf("default historyLen = %d, want 30", widget.historyLen)
	}
}

// TestNew_WithGPUConfig tests GPU widget with explicit config
func TestNew_WithGPUConfig(t *testing.T) {
	tests := []struct {
		name    string
		adapter int
		metric  string
	}{
		{"adapter 0 utilization", 0, MetricUtilization},
		{"adapter 1 utilization", 1, MetricUtilization},
		{"adapter 0 3d", 0, MetricUtilization3D},
		{"adapter 0 copy", 0, MetricUtilizationCopy},
		{"adapter 0 encode", 0, MetricUtilizationEncode},
		{"adapter 0 decode", 0, MetricUtilizationDecode},
		{"adapter 0 memory dedicated", 0, MetricMemoryDedicated},
		{"adapter 0 memory shared", 0, MetricMemoryShared},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "gpu",
				ID:      "test_gpu_config",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Mode: "gauge",
				GPU: &config.GPUConfig{
					Adapter: tt.adapter,
					Metric:  tt.metric,
				},
			}

			widget, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if widget.adapter != tt.adapter {
				t.Errorf("adapter = %d, want %d", widget.adapter, tt.adapter)
			}

			if widget.metric != tt.metric {
				t.Errorf("metric = %s, want %s", widget.metric, tt.metric)
			}
		})
	}
}

// TestNew_InvalidMetric tests that invalid metrics are rejected at construction time
func TestNew_InvalidMetric(t *testing.T) {
	tests := []struct {
		name   string
		metric string
	}{
		{"unknown metric", "nonexistent"},
		{"empty with explicit config", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip empty string test - empty metric falls back to default "utilization"
			if tt.metric == "" {
				return
			}
			cfg := config.WidgetConfig{
				Type:    "gpu",
				ID:      "test_gpu_invalid",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				GPU: &config.GPUConfig{Metric: tt.metric},
			}
			_, err := New(cfg)
			if err == nil {
				t.Errorf("New() should return error for metric %q", tt.metric)
			}
		})
	}
}

// TestNew_SupportedMetricsCompleteness verifies that all metric constants
// are present in the supportedMetrics map
func TestNew_SupportedMetricsCompleteness(t *testing.T) {
	expectedSupported := []string{
		MetricUtilization,
		MetricUtilization3D,
		MetricUtilizationCopy,
		MetricUtilizationEncode,
		MetricUtilizationDecode,
		MetricMemoryDedicated,
		MetricMemoryShared,
	}

	for _, metric := range expectedSupported {
		if !supportedMetrics[metric] {
			t.Errorf("metric %q should be in supportedMetrics", metric)
		}
	}
}

// TestWidget_Render_BeforeUpdate tests rendering before first update
func TestWidget_Render_BeforeUpdate(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "gpu",
		ID:      "test_gpu_render",
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

	// Render without prior update should return empty canvas (not panic)
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestWidget_Render_AllModes tests rendering in all display modes
func TestWidget_Render_AllModes(t *testing.T) {
	modes := []struct {
		mode        string
		displayMode render.DisplayMode
	}{
		{"text", render.DisplayModeText},
		{"bar", render.DisplayModeBar},
		{"graph", render.DisplayModeGraph},
		{"gauge", render.DisplayModeGauge},
	}

	mock := &mockReader{metricValue: 50.0}

	for _, tt := range modes {
		t.Run(tt.mode, func(t *testing.T) {
			w := newTestWidget(t, tt.mode, mock)

			if w.displayMode != tt.displayMode {
				t.Errorf("displayMode = %s, want %s", w.displayMode, tt.displayMode)
			}

			// Update with mock data, then render
			if err := w.Update(); err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}

			if img == nil {
				t.Error("Render() returned nil image")
			}

			bounds := img.Bounds()
			if bounds.Dx() != 128 || bounds.Dy() != 40 {
				t.Errorf("Image bounds = %v, want 128x40", bounds)
			}
		})
	}
}

// TestWidget_WithMockReader tests the widget with a mock reader
func TestWidget_WithMockReader(t *testing.T) {
	mock := &mockReader{
		metricValue: 75.5,
		adapters: []AdapterInfo{
			{Index: 0, Name: "Test GPU 0"},
			{Index: 1, Name: "Test GPU 1"},
		},
	}
	w := newTestWidget(t, "text", mock)

	// Update should use mock reader
	err := w.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify value was set
	w.mu.RLock()
	value := w.currentValue
	hasData := w.hasData
	w.mu.RUnlock()

	if !hasData {
		t.Error("Update() did not set hasData")
	}

	if value != 75.5 {
		t.Errorf("currentValue = %f, want 75.5", value)
	}

	// Test render with data
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestWidget_Update_ReaderError tests Update when reader returns an error
func TestWidget_Update_ReaderError(t *testing.T) {
	mock := &mockReader{returnErr: fmt.Errorf("PDH error")}
	w := newTestWidget(t, "text", mock)

	err := w.Update()
	if err == nil {
		t.Error("Update() should return error when reader fails")
	}
}

// TestWidget_Stop tests cleanup
func TestWidget_Stop(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "gpu",
		ID:      "test_gpu_stop",
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

	// Stop should not panic
	widget.Stop()
}

// TestWidget_ValueClamping tests that values are clamped to 0-100
func TestWidget_ValueClamping(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"normal value", 50.0, 50.0},
		{"zero", 0.0, 0.0},
		{"max", 100.0, 100.0},
		{"over max", 150.0, 100.0},
		{"negative", -10.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockReader{metricValue: tt.input}
			w := newTestWidget(t, "text", mock)

			err := w.Update()
			if err != nil {
				t.Errorf("Update() error = %v", err)
			}

			w.mu.RLock()
			value := w.currentValue
			w.mu.RUnlock()

			if value != tt.expected {
				t.Errorf("currentValue = %f, want %f", value, tt.expected)
			}
		})
	}
}

// TestNew_TextFormat tests that text format is correctly extracted from config
func TestNew_TextFormat(t *testing.T) {
	tests := []struct {
		name       string
		textConfig *config.TextConfig
		wantFormat string
	}{
		{"no text config", nil, ""},
		{"empty format", &config.TextConfig{Size: 10}, ""},
		{"custom format", &config.TextConfig{Format: "GPU %.0f%%", Size: 10}, "GPU %.0f%%"},
		{"vram format", &config.TextConfig{Format: "VRAM %.0f%%", Size: 10}, "VRAM %.0f%%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "gpu",
				ID:      "test_gpu_format",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				Mode: "bar",
				Text: tt.textConfig,
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if w.textFormat != tt.wantFormat {
				t.Errorf("textFormat = %q, want %q", w.textFormat, tt.wantFormat)
			}
		})
	}
}
