package app

import (
	"errors"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestCreateBackendClientInvalidBackend(t *testing.T) {
	// Test with empty/default backend (should try gamesense)
	cfg := &config.Config{
		GameName:        "TEST",
		GameDisplayName: "Test",
		Backend:         "",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	// This will fail because GameSense isn't running, but we can verify
	// it returns BackendUnavailableError
	_, err := CreateBackendClient(cfg)
	if err == nil {
		t.Skip("GameSense is running, cannot test error path")
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

	_, err := CreateBackendClient(cfg)
	if err == nil {
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

	_, err := CreateBackendClient(cfg)
	if err == nil {
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

	_, err := CreateBackendClient(cfg)
	if err == nil {
		t.Skip("A backend is available, cannot test error path")
	}

	// With "any" mode, should try both and return error
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateDirectClientInvalidVID(t *testing.T) {
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

	_, err := CreateDirectClient(cfg)
	if err == nil {
		t.Fatal("expected error for invalid VID")
	}

	// Should NOT be BackendUnavailableError - it's a config error
	var backendErr *BackendUnavailableError
	if errors.As(err, &backendErr) {
		t.Error("invalid VID should not be BackendUnavailableError")
	}

	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestCreateDirectClientInvalidPID(t *testing.T) {
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

	_, err := CreateDirectClient(cfg)
	if err == nil {
		t.Fatal("expected error for invalid PID")
	}

	// Should NOT be BackendUnavailableError - it's a config error
	var backendErr *BackendUnavailableError
	if errors.As(err, &backendErr) {
		t.Error("invalid PID should not be BackendUnavailableError")
	}
}

func TestCreateDirectClientValidHexParsing(t *testing.T) {
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

			_, err := CreateDirectClient(cfg)

			// We expect either a parse error or a device not found error
			if tt.wantErr {
				if err == nil {
					t.Error("expected error for invalid hex")
				}
				// Verify it's not a BackendUnavailableError (config error, not device error)
				var backendErr *BackendUnavailableError
				if errors.As(err, &backendErr) {
					t.Error("config parse error should not be BackendUnavailableError")
				}
			}
			// If no parse error expected, we might still get device not found
		})
	}
}

func TestCreateDirectClientDefaultInterface(t *testing.T) {
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
	_, err := CreateDirectClient(cfg)
	if err == nil {
		t.Skip("Direct driver succeeded unexpectedly")
	}

	// Should be BackendUnavailableError for device not found
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateDirectClientCustomInterface(t *testing.T) {
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

	_, err := CreateDirectClient(cfg)
	if err == nil {
		t.Skip("Direct driver succeeded unexpectedly")
	}

	// Should still be BackendUnavailableError
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}

func TestCreateGameSenseClientError(t *testing.T) {
	cfg := &config.Config{
		GameName:        "TEST_GAME",
		GameDisplayName: "Test Game",
	}

	_, err := CreateGameSenseClient(cfg)
	if err == nil {
		t.Skip("GameSense is running")
	}

	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T: %v", err, err)
	}
}
