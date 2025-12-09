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
