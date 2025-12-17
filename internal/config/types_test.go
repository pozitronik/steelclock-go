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
