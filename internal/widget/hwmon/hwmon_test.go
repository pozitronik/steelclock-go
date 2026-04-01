package hwmon

import (
	"errors"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
)

func baseCfg() config.WidgetConfig {
	return config.WidgetConfig{
		Type:    "hwmon",
		ID:      "test_hwmon",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}
}

func mockSensors() []metrics.HWMonStat {
	return []metrics.HWMonStat{
		{SensorID: "/amdcpu/0/temperature/2", Name: "Core (Tctl/Tdie)", Type: "Temperature", Value: 65.0, Unit: "°C"},
		{SensorID: "/amdcpu/0/load/0", Name: "CPU Total", Type: "Load", Value: 15.0, Unit: "%"},
		{SensorID: "/gpu-nvidia/0/temperature/0", Name: "GPU Core", Type: "Temperature", Value: 48.0, Unit: "°C"},
		{SensorID: "/gpu-nvidia/0/load/0", Name: "GPU Core", Type: "Load", Value: 26.0, Unit: "%"},
		{SensorID: "/nvme/0/temperature/0", Name: "Composite Temperature", Type: "Temperature", Value: 42.0, Unit: "°C"},
	}
}

func TestWidget_Update_SingleValue(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{SensorType: "Temperature"}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		t.Fatal("hasData should be true after Update")
	}

	// Average of 3 temperature sensors: (65 + 48 + 42) / 3 = 51.67
	expectedRaw := (65.0 + 48.0 + 42.0) / 3.0
	if diff := w.rawValue - expectedRaw; diff > 0.1 || diff < -0.1 {
		t.Errorf("rawValue = %.2f, want ~%.2f", w.rawValue, expectedRaw)
	}

	if w.rawUnit != "°C" {
		t.Errorf("rawUnit = %q, want %q", w.rawUnit, "°C")
	}
}

func TestWidget_Update_BySensorID(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{SensorID: "/gpu-nvidia/0/temperature/0"}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.rawValue != 48.0 {
		t.Errorf("rawValue = %.2f, want 48.0 (GPU Core)", w.rawValue)
	}
}

func TestWidget_Update_PerCore(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{SensorType: "Temperature"}
	cfg.PerCore = &config.PerCoreConfig{Enabled: true}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.sensorCount != 3 {
		t.Errorf("sensorCount = %d, want 3 (temperature sensors)", w.sensorCount)
	}

	if len(w.rawValues) != 3 {
		t.Fatalf("rawValues has %d entries, want 3", len(w.rawValues))
	}

	expected := []float64{65.0, 48.0, 42.0}
	for i, e := range expected {
		if w.rawValues[i] != e {
			t.Errorf("rawValues[%d] = %.1f, want %.1f", i, w.rawValues[i], e)
		}
	}
}

func TestWidget_Normalization(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		min    float64
		max    float64
		expect float64
	}{
		{"default scale at 75", 75.0, 0, 100, 75.0},
		{"zero", 0, 0, 100, 0},
		{"at max", 100, 0, 100, 100},
		{"above max (clamped)", 120, 0, 100, 100},
		{"below min (clamped)", -10, 0, 100, 0},
		{"custom range midpoint", 50, 0, 200, 25.0},
		{"non-zero min", 900, 800, 1000, 50.0},
		{"non-zero min below", 700, 800, 1000, 0},
		{"non-zero min above", 1200, 800, 1000, 100},
		{"equal min max", 50, 50, 50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Widget{minVal: tt.min, maxVal: tt.max}
			got := w.normalize(tt.value)
			if diff := got - tt.expect; diff > 0.01 || diff < -0.01 {
				t.Errorf("normalize(%.1f) = %.2f, want %.2f", tt.value, got, tt.expect)
			}
		})
	}
}

func TestWidget_FilterSensors(t *testing.T) {
	sensors := mockSensors()

	tests := []struct {
		name         string
		sensorID     string
		sensorType   string
		sensorFilter string
		wantCount    int
	}{
		{"no filters", "", "", "", 5},
		{"by sensor_id", "/amdcpu/0/load/0", "", "", 1},
		{"by sensor_id not found", "/nonexistent", "", "", 0},
		{"by type Temperature", "", "Temperature", "", 3},
		{"by type Load", "", "Load", "", 2},
		{"by type case insensitive", "", "temperature", "", 3},
		{"by filter amdcpu", "", "", "amdcpu", 2},
		{"by filter GPU", "", "", "gpu", 2},
		{"by filter name", "", "", "Composite", 1},
		{"type + filter combined", "", "Temperature", "gpu", 1},
		{"type + filter no match", "", "Load", "nvme", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Widget{
				sensorID:     tt.sensorID,
				sensorType:   tt.sensorType,
				sensorFilter: tt.sensorFilter,
			}
			got := w.filterSensors(sensors)
			if len(got) != tt.wantCount {
				t.Errorf("filterSensors() returned %d sensors, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestWidget_ProviderError_MarksUnavailable(t *testing.T) {
	cfg := baseCfg()
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	callCount := 0
	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			callCount++
			return nil, errors.New("connection refused")
		},
	}

	err = w.Update()
	if err != nil {
		t.Errorf("Update() should not propagate error, got: %v", err)
	}

	w.mu.RLock()
	unavailable := w.unavailable
	msg := w.unavailableMsg
	w.mu.RUnlock()

	if !unavailable {
		t.Error("widget should be marked unavailable after provider error")
	}
	if msg != "No sensors" {
		t.Errorf("unavailableMsg = %q, want %q", msg, "No sensors")
	}

	// Second call should skip polling entirely
	err = w.Update()
	if err != nil {
		t.Errorf("second Update() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("provider called %d times, want 1 (should skip after unavailable)", callCount)
	}
}

func TestWidget_Render_Unavailable(t *testing.T) {
	cfg := baseCfg()
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return nil, errors.New("not reachable")
		},
	}

	_ = w.Update()

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestWidget_Render_AllModes(t *testing.T) {
	modes := []string{"text", "bar", "graph", "gauge"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := baseCfg()
			cfg.Mode = mode
			cfg.HWMon = &config.HWMonConfig{SensorID: "/amdcpu/0/temperature/2"}
			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			w.hwmonProvider = &metrics.MockHWMon{
				SensorsFunc: func() ([]metrics.HWMonStat, error) {
					return mockSensors(), nil
				},
			}

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
		})
	}
}

func TestWidget_Render_NoData(t *testing.T) {
	cfg := baseCfg()
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestWidget_Render_PerCore(t *testing.T) {
	cfg := baseCfg()
	cfg.Mode = "bar"
	cfg.HWMon = &config.HWMonConfig{SensorType: "Temperature"}
	cfg.PerCore = &config.PerCoreConfig{Enabled: true}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

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
}

func TestWidget_CustomMinMax(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{
		SensorID: "/amdcpu/0/temperature/2",
		Min:      20,
		Max:      120,
	}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	// 65°C with range [20, 120]: (65-20)/(120-20)*100 = 45%
	expected := 45.0
	if diff := w.currentNorm - expected; diff > 0.1 || diff < -0.1 {
		t.Errorf("currentNorm = %.2f, want %.2f", w.currentNorm, expected)
	}
}

func TestWidget_FormatValue(t *testing.T) {
	w := &Widget{}

	tests := []struct {
		value float64
		unit  string
		want  string
	}{
		{74.0, "°C", "74°C"},
		{6.8, "%", "7%"},
		{13.9, "W", "13.9W"},
		{3924.0, "MHz", "3924MHz"},
		{1.05, "V", "1.05V"},
		{0.5, "A", "0.50A"},
		{45.0, "", "45.0"},
		{42.5, "GB", "42.5GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := w.formatValue(tt.value, tt.unit)
			if got != tt.want {
				t.Errorf("formatValue(%.1f, %q) = %q, want %q", tt.value, tt.unit, got, tt.want)
			}
		})
	}
}

func TestWidget_TextFormatString(t *testing.T) {
	tests := []struct {
		unit   string
		expect string
	}{
		{"°C", "%.0f°C"},
		{"%", "%.0f%%"},
		{"W", "%.1fW"},
		{"MHz", "%.0fMHz"},
		{"V", "%.2fV"},
		{"", "%.1f"},
		{"GB", "%.1fGB"},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			w := &Widget{rawUnit: tt.unit}
			got := w.textFormatString()
			if got != tt.expect {
				t.Errorf("textFormatString() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestWidget_UserTextFormat(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{SensorID: "/amdcpu/0/temperature/2"}
	cfg.Text = &config.TextConfig{Format: "CPU: %.0f°C"}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.userTextFormat != "CPU: %.0f°C" {
		t.Errorf("userTextFormat = %q, want %q", w.userTextFormat, "CPU: %.0f°C")
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

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
}

func TestWidget_UserTextFormat_Empty(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{SensorID: "/amdcpu/0/temperature/2"}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.userTextFormat != "" {
		t.Errorf("userTextFormat should be empty by default, got %q", w.userTextFormat)
	}
}

func TestWidget_DefaultConfig(t *testing.T) {
	cfg := baseCfg()
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.minVal != 0 {
		t.Errorf("default minVal = %f, want 0", w.minVal)
	}
	if w.maxVal != 100 {
		t.Errorf("default maxVal = %f, want 100", w.maxVal)
	}
}

func TestWidget_NoMatchingSensors(t *testing.T) {
	cfg := baseCfg()
	cfg.HWMon = &config.HWMonConfig{SensorID: "/nonexistent/sensor"}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.hwmonProvider = &metrics.MockHWMon{
		SensorsFunc: func() ([]metrics.HWMonStat, error) {
			return mockSensors(), nil
		},
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.hasData {
		t.Error("hasData should be false when no sensors match")
	}
}
