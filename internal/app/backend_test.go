package app

import (
	"errors"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestCreateBackendClientInvalidBackend(t *testing.T) {
	// Test with empty/default backend (should try backends by priority)
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	// This will fail because no backend is available, but we can verify
	// it returns BackendUnavailableError
	_, _, err := CreateBackendClient(cfg)
	if err == nil {
		t.Skip("A backend is available, cannot test error path")
	}

	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendClientGameSense(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "gamesense",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	_, backendName, err := CreateBackendClient(cfg)
	if err == nil {
		if backendName != "gamesense" {
			t.Errorf("expected backend name 'gamesense', got %q", backendName)
		}
		t.Skip("GameSense is running, cannot test error path")
	}

	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendClientDirect(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "direct",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	_, backendName, err := CreateBackendClient(cfg)
	if err == nil {
		if backendName != "direct" {
			t.Errorf("expected backend name 'direct', got %q", backendName)
		}
		t.Skip("Direct driver available, cannot test error path")
	}

	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendClientAny(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "any",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	_, _, err := CreateBackendClient(cfg)
	if err == nil {
		t.Skip("A backend is available, cannot test error path")
	}

	// With "any" mode, should try all backends and return error
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendByNameInvalidVID(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "direct",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
		DirectDriver: &config.DirectDriverConfig{
			VID: "invalid", // Invalid hex
			PID: "1234",
		},
	}

	_, err := CreateBackendByName("direct", cfg)
	if err == nil {
		t.Fatal("expected error for invalid VID")
	}

	// All errors from CreateBackendByName are wrapped in BackendUnavailableError
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}

	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestCreateBackendByNameInvalidPID(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "direct",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
		DirectDriver: &config.DirectDriverConfig{
			VID: "1038",
			PID: "ZZZZ", // Invalid hex
		},
	}

	_, err := CreateBackendByName("direct", cfg)
	if err == nil {
		t.Fatal("expected error for invalid PID")
	}

	// All errors from CreateBackendByName are wrapped in BackendUnavailableError
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendByNameValidHexParsing(t *testing.T) {
	tests := []struct {
		name    string
		vid     string
		pid     string
		wantErr bool
	}{
		{"lowercase hex", "1038", "12aa", false},
		{"uppercase hex", "1038", "12AA", false},
		{"mixed case", "1038", "12Aa", false},
		{"zero values", "0000", "0000", false},
		{"max values", "ffff", "FFFF", false},
		{"empty vid", "", "1234", false},
		{"empty pid", "1038", "", false},
		{"both empty", "", "", false},
		{"invalid vid char", "103g", "1234", true},
		{"invalid pid char", "1038", "123g", true},
		{"too long vid", "10380", "1234", true},
		{"negative vid", "-1038", "1234", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				GameName:        "TEST",
				GameDisplayName: "Test",
				Backend:         "direct",
				Display: config.DisplayConfig{
					Width:  128,
					Height: 40,
				},
				DirectDriver: &config.DirectDriverConfig{
					VID: tt.vid,
					PID: tt.pid,
				},
			}

			_, err := CreateBackendByName("direct", cfg)

			// We expect either a parse error or a device not found error
			if tt.wantErr && err == nil {
				t.Error("expected error for invalid hex")
			}
			// All errors are wrapped in BackendUnavailableError
		})
	}
}

func TestCreateBackendByNameDefaultInterface(t *testing.T) {
	// Test that default interface is used when not specified
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "direct",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
		// No DirectDriver config - should use defaults
	}

	// This will fail because device isn't found, but we test the code path
	_, err := CreateBackendByName("direct", cfg)
	if err == nil {
		t.Skip("Direct driver succeeded unexpectedly")
	}

	// Should be BackendUnavailableError for device not found
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendByNameCustomInterface(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "direct",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
		DirectDriver: &config.DirectDriverConfig{
			Interface: "mi_02",
		},
	}

	_, err := CreateBackendByName("direct", cfg)
	if err == nil {
		t.Skip("Direct driver succeeded unexpectedly")
	}

	// Should be BackendUnavailableError
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendByNameGameSenseError(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST_GAME",
		GameDisplayName: "Test Game",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	_, err := CreateBackendByName("gamesense", cfg)
	if err == nil {
		t.Skip("GameSense is running")
	}

	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateBackendByNameUnknownBackend(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST_GAME",
		GameDisplayName: "Test Game",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	_, err := CreateBackendByName("unknown", cfg)
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}

	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}
