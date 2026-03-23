package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStringOrSlice_UnmarshalJSON_String(t *testing.T) {
	// Test unmarshaling a single string
	jsonData := `"single value"`
	var s StringOrSlice
	err := json.Unmarshal([]byte(jsonData), &s)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	expected := StringOrSlice{"single value"}
	if !reflect.DeepEqual(s, expected) {
		t.Errorf("UnmarshalJSON() = %v, want %v", s, expected)
	}
}

func TestStringOrSlice_UnmarshalJSON_Array(t *testing.T) {
	// Test unmarshaling an array of strings
	jsonData := `["value1", "value2", "value3"]`
	var s StringOrSlice
	err := json.Unmarshal([]byte(jsonData), &s)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	expected := StringOrSlice{"value1", "value2", "value3"}
	if !reflect.DeepEqual(s, expected) {
		t.Errorf("UnmarshalJSON() = %v, want %v", s, expected)
	}
}

func TestStringOrSlice_UnmarshalJSON_EmptyArray(t *testing.T) {
	jsonData := `[]`
	var s StringOrSlice
	err := json.Unmarshal([]byte(jsonData), &s)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if len(s) != 0 {
		t.Errorf("UnmarshalJSON() = %v, want empty slice", s)
	}
}

func TestStringOrSlice_UnmarshalJSON_Invalid(t *testing.T) {
	// Test with invalid JSON (number)
	jsonData := `123`
	var s StringOrSlice
	err := json.Unmarshal([]byte(jsonData), &s)
	if err == nil {
		t.Error("UnmarshalJSON() should return error for invalid input")
	}
}

func TestStringOrSlice_MarshalJSON_SingleValue(t *testing.T) {
	// Test marshaling a single-element slice (should produce a string)
	s := StringOrSlice{"single value"}
	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	expected := `"single value"`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}

func TestStringOrSlice_MarshalJSON_MultipleValues(t *testing.T) {
	// Test marshaling a multi-element slice (should produce an array)
	s := StringOrSlice{"value1", "value2"}
	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	expected := `["value1","value2"]`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}

func TestStringOrSlice_MarshalJSON_EmptySlice(t *testing.T) {
	// Test marshaling an empty slice
	s := StringOrSlice{}
	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	expected := `[]`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}

func TestStringOrSlice_RoundTrip_String(t *testing.T) {
	// Test round-trip: unmarshal string, marshal back
	original := `"test string"`
	var s StringOrSlice
	if err := json.Unmarshal([]byte(original), &s); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if string(data) != original {
		t.Errorf("Round-trip failed: got %s, want %s", string(data), original)
	}
}

func TestStringOrSlice_RoundTrip_Array(t *testing.T) {
	// Test round-trip: unmarshal array, marshal back
	original := `["a","b","c"]`
	var s StringOrSlice
	if err := json.Unmarshal([]byte(original), &s); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if string(data) != original {
		t.Errorf("Round-trip failed: got %s, want %s", string(data), original)
	}
}

func TestIntOrRange_UnmarshalJSON_Integer(t *testing.T) {
	jsonData := `50`
	var r IntOrRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if r.Min != 50 || r.Max != 50 {
		t.Errorf("UnmarshalJSON() = {Min: %d, Max: %d}, want {Min: 50, Max: 50}", r.Min, r.Max)
	}
}

func TestIntOrRange_UnmarshalJSON_Object(t *testing.T) {
	jsonData := `{"min": 10, "max": 100}`
	var r IntOrRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if r.Min != 10 || r.Max != 100 {
		t.Errorf("UnmarshalJSON() = {Min: %d, Max: %d}, want {Min: 10, Max: 100}", r.Min, r.Max)
	}
}

func TestIntOrRange_UnmarshalJSON_Invalid(t *testing.T) {
	jsonData := `"not a number"`
	var r IntOrRange
	err := json.Unmarshal([]byte(jsonData), &r)
	if err == nil {
		t.Error("UnmarshalJSON() should return error for invalid input")
	}
}

func TestIntOrRange_MarshalJSON_SingleValue(t *testing.T) {
	r := IntOrRange{Min: 42, Max: 42}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	expected := `42`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}

func TestIntOrRange_MarshalJSON_Range(t *testing.T) {
	r := IntOrRange{Min: 10, Max: 100}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Unmarshal to check values since map order might vary
	var result map[string]int
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["min"] != 10 || result["max"] != 100 {
		t.Errorf("MarshalJSON() result = %v, want {min: 10, max: 100}", result)
	}
}

func TestIntOrRange_IsRange(t *testing.T) {
	tests := []struct {
		name     string
		r        IntOrRange
		expected bool
	}{
		{"single value", IntOrRange{Min: 50, Max: 50}, false},
		{"range", IntOrRange{Min: 10, Max: 100}, true},
		{"zero range", IntOrRange{Min: 0, Max: 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.IsRange(); got != tt.expected {
				t.Errorf("IsRange() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIntOrRange_Value(t *testing.T) {
	tests := []struct {
		name     string
		r        IntOrRange
		expected int
	}{
		{"single value", IntOrRange{Min: 50, Max: 50}, 50},
		{"range returns zero", IntOrRange{Min: 10, Max: 100}, 0},
		{"zero value", IntOrRange{Min: 0, Max: 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Value(); got != tt.expected {
				t.Errorf("Value() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIntOrRange_RoundTrip_Integer(t *testing.T) {
	original := `42`
	var r IntOrRange
	if err := json.Unmarshal([]byte(original), &r); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if string(data) != original {
		t.Errorf("Round-trip failed: got %s, want %s", string(data), original)
	}
}

// --- GetDevices / ConfigForDevice / DeviceConfig tests ---

func TestGetDevices_SingleDeviceMode(t *testing.T) {
	cfg := &Config{
		Display:      DisplayConfig{Width: 128, Height: 40},
		Backend:      "direct",
		DirectDriver: &DirectDriverConfig{Interface: "mi_01"},
		Widgets:      []WidgetConfig{{Type: "clock"}},
	}

	devices := cfg.GetDevices()

	if len(devices) != 1 {
		t.Fatalf("GetDevices() returned %d devices, want 1", len(devices))
	}

	dev := devices[0]
	if dev.ID != "default" {
		t.Errorf("device ID = %q, want %q", dev.ID, "default")
	}
	if dev.Display.Width != 128 || dev.Display.Height != 40 {
		t.Errorf("device display = %dx%d, want 128x40", dev.Display.Width, dev.Display.Height)
	}
	if dev.Backend != "direct" {
		t.Errorf("device backend = %q, want %q", dev.Backend, "direct")
	}
	if dev.DirectDriver == nil || dev.DirectDriver.Interface != "mi_01" {
		t.Error("device DirectDriver not preserved")
	}
	if len(dev.Widgets) != 1 || dev.Widgets[0].Type != "clock" {
		t.Error("device Widgets not preserved")
	}
}

func TestGetDevices_MultiDeviceMode(t *testing.T) {
	cfg := &Config{
		Devices: []DeviceConfig{
			{ID: "keyboard", Display: DisplayConfig{Width: 128, Height: 40}},
			{ID: "gamedac", Display: DisplayConfig{Width: 128, Height: 64}},
		},
	}

	devices := cfg.GetDevices()

	if len(devices) != 2 {
		t.Fatalf("GetDevices() returned %d devices, want 2", len(devices))
	}
	if devices[0].ID != "keyboard" {
		t.Errorf("devices[0].ID = %q, want %q", devices[0].ID, "keyboard")
	}
	if devices[1].ID != "gamedac" {
		t.Errorf("devices[1].ID = %q, want %q", devices[1].ID, "gamedac")
	}
	if devices[1].Display.Height != 64 {
		t.Errorf("devices[1].Display.Height = %d, want 64", devices[1].Display.Height)
	}
}

func TestGetDevices_EmptyDevicesUsesTopLevel(t *testing.T) {
	cfg := &Config{
		Devices: []DeviceConfig{},
		Display: DisplayConfig{Width: 128, Height: 40},
		Widgets: []WidgetConfig{{Type: "clock"}},
	}

	devices := cfg.GetDevices()

	if len(devices) != 1 {
		t.Fatalf("GetDevices() returned %d devices, want 1", len(devices))
	}
	if devices[0].ID != "default" {
		t.Errorf("device ID = %q, want %q", devices[0].ID, "default")
	}
}

func TestConfigForDevice_MergesGlobalFields(t *testing.T) {
	global := &Config{
		GameName:        "STEELCLOCK",
		GameDisplayName: "SteelClock",
		RefreshRateMs:   100,
		Display:         DisplayConfig{Width: 128, Height: 40},
		Backend:         "gamesense",
		DirectDriver:    &DirectDriverConfig{Interface: "mi_01"},
		Widgets:         []WidgetConfig{{Type: "clock"}},
		Devices:         []DeviceConfig{{ID: "gamedac"}},
	}

	dev := DeviceConfig{
		ID:           "gamedac",
		Display:      DisplayConfig{Width: 128, Height: 64},
		Backend:      "direct",
		DirectDriver: &DirectDriverConfig{VID: "1038", PID: "12cb", Interface: "mi_04"},
		Widgets:      []WidgetConfig{{Type: "cpu"}, {Type: "memory"}},
	}

	merged := global.ConfigForDevice(dev)

	if merged.GameName != "STEELCLOCK" {
		t.Errorf("GameName = %q, want %q", merged.GameName, "STEELCLOCK")
	}
	if merged.RefreshRateMs != 100 {
		t.Errorf("RefreshRateMs = %d, want 100", merged.RefreshRateMs)
	}
	if merged.Display.Height != 64 {
		t.Errorf("Display.Height = %d, want 64", merged.Display.Height)
	}
	if merged.Backend != "direct" {
		t.Errorf("Backend = %q, want %q", merged.Backend, "direct")
	}
	if merged.DirectDriver.PID != "12cb" {
		t.Errorf("DirectDriver.PID = %q, want %q", merged.DirectDriver.PID, "12cb")
	}
	if len(merged.Widgets) != 2 {
		t.Errorf("Widgets count = %d, want 2", len(merged.Widgets))
	}
	if merged.Devices != nil {
		t.Error("Devices should be nil in merged config")
	}
}

func TestConfigForDevice_EmptyBackendInheritsGlobal(t *testing.T) {
	global := &Config{Backend: "gamesense"}

	dev := DeviceConfig{Backend: ""}

	merged := global.ConfigForDevice(dev)

	if merged.Backend != "gamesense" {
		t.Errorf("Backend = %q, want %q (inherited)", merged.Backend, "gamesense")
	}
}

func TestConfigForDevice_NilDriversInheritGlobal(t *testing.T) {
	global := &Config{
		DirectDriver: &DirectDriverConfig{Interface: "mi_01"},
		WebClient:    &WebClientConfig{TargetFPS: 30},
	}

	dev := DeviceConfig{DirectDriver: nil, WebClient: nil}

	merged := global.ConfigForDevice(dev)

	if merged.DirectDriver == nil || merged.DirectDriver.Interface != "mi_01" {
		t.Error("DirectDriver should be inherited from global")
	}
	if merged.WebClient == nil || merged.WebClient.TargetFPS != 30 {
		t.Error("WebClient should be inherited from global")
	}
}

func TestConfigForDevice_DoesNotMutateOriginal(t *testing.T) {
	global := &Config{
		Backend: "gamesense",
		Display: DisplayConfig{Width: 128, Height: 40},
		Widgets: []WidgetConfig{{Type: "clock"}},
	}

	dev := DeviceConfig{
		Backend: "direct",
		Display: DisplayConfig{Width: 128, Height: 64},
		Widgets: []WidgetConfig{{Type: "cpu"}},
	}

	_ = global.ConfigForDevice(dev)

	if global.Backend != "gamesense" {
		t.Errorf("original Backend mutated to %q", global.Backend)
	}
	if global.Display.Height != 40 {
		t.Errorf("original Display.Height mutated to %d", global.Display.Height)
	}
}

func TestDeviceConfig_JSONRoundTrip(t *testing.T) {
	jsonData := `{"id":"gamedac","display":{"width":128,"height":64},"backend":"direct","direct_driver":{"vid":"1038","pid":"12cb","interface":"mi_04","brightness":5},"widgets":[{"type":"clock","position":{"x":0,"y":0,"w":128,"h":64}}]}`

	var dev DeviceConfig
	if err := json.Unmarshal([]byte(jsonData), &dev); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if dev.ID != "gamedac" {
		t.Errorf("ID = %q, want %q", dev.ID, "gamedac")
	}
	if dev.Display.Width != 128 || dev.Display.Height != 64 {
		t.Errorf("Display = %dx%d, want 128x64", dev.Display.Width, dev.Display.Height)
	}
	if dev.DirectDriver == nil || dev.DirectDriver.Brightness == nil || *dev.DirectDriver.Brightness != 5 {
		t.Error("DirectDriver.Brightness not parsed correctly")
	}
}

func TestDirectDriverConfig_BrightnessField(t *testing.T) {
	jsonData := `{"vid":"1038","pid":"12cb","brightness":7}`

	var cfg DirectDriverConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if cfg.Brightness == nil {
		t.Fatal("Brightness should not be nil")
	}
	if *cfg.Brightness != 7 {
		t.Errorf("Brightness = %d, want 7", *cfg.Brightness)
	}
}

func TestDirectDriverConfig_BrightnessNil(t *testing.T) {
	jsonData := `{"vid":"1038"}`

	var cfg DirectDriverConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if cfg.Brightness != nil {
		t.Errorf("Brightness should be nil when omitted, got %d", *cfg.Brightness)
	}
}
