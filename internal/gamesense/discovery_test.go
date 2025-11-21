package gamesense

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverServer_ValidCoreProps(t *testing.T) {
	// Create temp directory and coreProps.json
	tmpDir := t.TempDir()
	corePropsPath := filepath.Join(tmpDir, "coreProps.json")

	validJSON := `{"address": "127.0.0.1:12345"}`
	if err := os.WriteFile(corePropsPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to write temp coreProps.json: %v", err)
	}

	// Mock findCorePropsPathFunc to return our temp file
	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return corePropsPath, nil
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	address, err := DiscoverServer()
	if err != nil {
		t.Errorf("DiscoverServer() error = %v, want nil", err)
	}

	if address != "127.0.0.1:12345" {
		t.Errorf("DiscoverServer() = %s, want 127.0.0.1:12345", address)
	}
}

func TestDiscoverServer_MissingFile(t *testing.T) {
	// Mock findCorePropsPathFunc to return non-existent path
	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return "/nonexistent/coreProps.json", nil
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	_, err := DiscoverServer()
	if err == nil {
		t.Error("DiscoverServer() with missing file should return error")
	}
}

func TestDiscoverServer_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	corePropsPath := filepath.Join(tmpDir, "coreProps.json")

	invalidJSON := `{invalid json`
	if err := os.WriteFile(corePropsPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write temp coreProps.json: %v", err)
	}

	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return corePropsPath, nil
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	_, err := DiscoverServer()
	if err == nil {
		t.Error("DiscoverServer() with invalid JSON should return error")
	}
}

func TestDiscoverServer_MissingAddress(t *testing.T) {
	tmpDir := t.TempDir()
	corePropsPath := filepath.Join(tmpDir, "coreProps.json")

	noAddressJSON := `{"other_field": "value"}`
	if err := os.WriteFile(corePropsPath, []byte(noAddressJSON), 0644); err != nil {
		t.Fatalf("Failed to write temp coreProps.json: %v", err)
	}

	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return corePropsPath, nil
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	_, err := DiscoverServer()
	if err == nil {
		t.Error("DiscoverServer() with missing address field should return error")
	}
}

func TestDiscoverServer_EmptyAddress(t *testing.T) {
	tmpDir := t.TempDir()
	corePropsPath := filepath.Join(tmpDir, "coreProps.json")

	emptyAddressJSON := `{"address": ""}`
	if err := os.WriteFile(corePropsPath, []byte(emptyAddressJSON), 0644); err != nil {
		t.Fatalf("Failed to write temp coreProps.json: %v", err)
	}

	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return corePropsPath, nil
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	_, err := DiscoverServer()
	if err == nil {
		t.Error("DiscoverServer() with empty address should return error")
	}
}

func TestDiscoverServer_InvalidAddressFormat(t *testing.T) {
	testCases := []struct {
		name    string
		address string
	}{
		{"no port", "127.0.0.1"},
		{"too many colons", "127.0.0.1:12345:extra"},
		{"invalid port", "127.0.0.1:abc"},
		{"empty port", "127.0.0.1:"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			corePropsPath := filepath.Join(tmpDir, "coreProps.json")

			jsonContent := `{"address": "` + tc.address + `"}`
			if err := os.WriteFile(corePropsPath, []byte(jsonContent), 0644); err != nil {
				t.Fatalf("Failed to write temp coreProps.json: %v", err)
			}

			originalFunc := findCorePropsPathFunc
			findCorePropsPathFunc = func() (string, error) {
				return corePropsPath, nil
			}
			defer func() { findCorePropsPathFunc = originalFunc }()

			_, err := DiscoverServer()
			if err == nil {
				t.Errorf("DiscoverServer() with invalid address format %q should return error", tc.address)
			}
		})
	}
}

func TestDiscoverServer_ValidAddressFormats(t *testing.T) {
	testCases := []struct {
		name    string
		address string
	}{
		{"localhost", "localhost:12345"},
		{"IPv4", "127.0.0.1:12345"},
		{"hostname", "myhost:54321"},
		{"high port", "127.0.0.1:65535"},
		{"low port", "127.0.0.1:1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			corePropsPath := filepath.Join(tmpDir, "coreProps.json")

			jsonContent := `{"address": "` + tc.address + `"}`
			if err := os.WriteFile(corePropsPath, []byte(jsonContent), 0644); err != nil {
				t.Fatalf("Failed to write temp coreProps.json: %v", err)
			}

			originalFunc := findCorePropsPathFunc
			findCorePropsPathFunc = func() (string, error) {
				return corePropsPath, nil
			}
			defer func() { findCorePropsPathFunc = originalFunc }()

			address, err := DiscoverServer()
			if err != nil {
				t.Errorf("DiscoverServer() error = %v, want nil", err)
			}
			if address != tc.address {
				t.Errorf("DiscoverServer() = %s, want %s", address, tc.address)
			}
		})
	}
}

func TestDiscoverServer_FindCorePropsPathError(t *testing.T) {
	originalFunc := findCorePropsPathFunc
	findCorePropsPathFunc = func() (string, error) {
		return "", os.ErrNotExist
	}
	defer func() { findCorePropsPathFunc = originalFunc }()

	_, err := DiscoverServer()
	if err == nil {
		t.Error("DiscoverServer() should return error when findCorePropsPathFunc fails")
	}
}
