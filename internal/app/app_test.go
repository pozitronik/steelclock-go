package app

import (
	"errors"
	"testing"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
	}{
		{
			name:       "creates app with config path",
			configPath: "config.json",
		},
		{
			name:       "creates app with absolute path",
			configPath: "/path/to/config.json",
		},
		{
			name:       "creates app with empty path",
			configPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(tt.configPath)

			if app == nil {
				t.Fatal("NewApp returned nil")
			}

			if app.configPath != tt.configPath {
				t.Errorf("configPath = %q, want %q", app.configPath, tt.configPath)
			}

			if app.retryCancel == nil {
				t.Error("retryCancel channel not initialized")
			}

			// Verify other fields are nil/zero initially
			if app.comp != nil {
				t.Error("comp should be nil initially")
			}
			if app.client != nil {
				t.Error("client should be nil initially")
			}
			if app.trayMgr != nil {
				t.Error("trayMgr should be nil initially")
			}
			if app.lastGoodConfig != nil {
				t.Error("lastGoodConfig should be nil initially")
			}
			if app.currentBackend != "" {
				t.Errorf("currentBackend should be empty, got %q", app.currentBackend)
			}
		})
	}
}

func TestBackendUnavailableError(t *testing.T) {
	originalErr := errors.New("connection refused")
	err := &BackendUnavailableError{Err: originalErr}

	// Test Error() method
	expectedMsg := "backend unavailable: connection refused"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}

	// Test Unwrap() method
	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// Test errors.Is/As compatibility
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Error("errors.As should match BackendUnavailableError")
	}

	if !errors.Is(err, originalErr) {
		t.Error("errors.Is should find wrapped error")
	}
}

func TestNoWidgetsError(t *testing.T) {
	err := &NoWidgetsError{}

	expectedMsg := "no widgets enabled in configuration"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}

	// Test errors.As compatibility
	var noWidgetsErr *NoWidgetsError
	wrappedErr := error(err)
	if !errors.As(wrappedErr, &noWidgetsErr) {
		t.Error("errors.As should match NoWidgetsError")
	}
}

func TestBackendUnavailableErrorWrapping(t *testing.T) {
	// Test nested error wrapping
	innerErr := errors.New("tcp connect failed")
	backendErr := &BackendUnavailableError{Err: innerErr}

	// Verify the error chain works correctly
	if !errors.Is(backendErr, innerErr) {
		t.Error("BackendUnavailableError should wrap inner error")
	}

	// Test with nil inner error
	nilErr := &BackendUnavailableError{Err: nil}
	if nilErr.Unwrap() != nil {
		t.Error("Unwrap() with nil Err should return nil")
	}
	if nilErr.Error() != "backend unavailable: <nil>" {
		t.Errorf("Error() with nil = %q", nilErr.Error())
	}
}

func TestAppConstants(t *testing.T) {
	// Verify constants have expected values
	if EventName != "STEELCLOCK_DISPLAY" {
		t.Errorf("EventName = %q, want %q", EventName, "STEELCLOCK_DISPLAY")
	}

	if DeviceType != "screened-128x40" {
		t.Errorf("DeviceType = %q, want %q", DeviceType, "screened-128x40")
	}

	if DeveloperName != "Pozitronik" {
		t.Errorf("DeveloperName = %q, want %q", DeveloperName, "Pozitronik")
	}
}

func TestAppStopWithNilComponents(t *testing.T) {
	// Test that Stop() doesn't panic with nil components
	app := NewApp("config.json")

	// Should not panic
	app.Stop()
	app.stopAndWait()
}

func TestAppMutexProtection(t *testing.T) {
	// Test that concurrent Stop calls don't cause race conditions
	app := NewApp("config.json")

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			app.Stop()
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
