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

			if app.configMgr.GetConfigPath() != tt.configPath {
				t.Errorf("configMgr.GetConfigPath() = %q, want %q", app.configMgr.GetConfigPath(), tt.configPath)
			}

			// Verify lifecycle manager is initialized
			if app.lifecycle == nil {
				t.Error("lifecycle manager not initialized")
			}

			// Verify trayMgr is nil initially (created in Run())
			if app.trayMgr != nil {
				t.Error("trayMgr should be nil initially")
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
	if //goland:noinspection GoDirectComparisonOfErrors // Testing Unwrap() requires exact pointer equality
	unwrapped != originalErr {
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

//goland:noinspection GoBoolExpressions,GoBoolExpressions,GoBoolExpressions
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

func TestNewAppWithProfiles(t *testing.T) {
	// NewAppWithProfiles requires a ProfileManager, but we can test nil handling
	app := NewAppWithProfiles(nil)

	if app == nil {
		t.Fatal("NewAppWithProfiles returned nil")
	}

	if app.lifecycle == nil {
		t.Error("lifecycle manager not initialized")
	}

	if app.configMgr == nil {
		t.Error("configMgr not initialized")
	}
}

func TestBackendUnavailableErrorChain(t *testing.T) {
	// Test deep error chaining
	inner1 := errors.New("network error")
	inner2 := &BackendUnavailableError{Err: inner1}
	outer := &BackendUnavailableError{Err: inner2}

	// Should be able to unwrap through the chain
	if !errors.Is(outer, inner1) {
		t.Error("errors.Is should find inner1 through chain")
	}

	// errors.As should match at any level
	var backendErr *BackendUnavailableError
	if !errors.As(outer, &backendErr) {
		t.Error("errors.As should match BackendUnavailableError")
	}
}

func TestNoWidgetsErrorAsInterface(t *testing.T) {
	// Test that NoWidgetsError implements the error interface
	var err error = &NoWidgetsError{}

	if err == nil {
		t.Error("error should not be nil")
	}

	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestAppDoubleStop(t *testing.T) {
	app := NewApp("config.json")

	// Double stop should not panic
	app.Stop()
	app.Stop()
}

func TestAppStopBeforeStart(t *testing.T) {
	app := NewApp("config.json")

	// Stop before Start should work
	app.Stop()
}

func TestBackendUnavailableErrorWithNilChain(t *testing.T) {
	// Test behavior when wrapping nil
	err := &BackendUnavailableError{Err: nil}

	if err.Unwrap() != nil {
		t.Error("Unwrap of nil should return nil")
	}

	// Error message should still be valid
	msg := err.Error()
	if msg == "" {
		t.Error("Error message should not be empty even with nil inner error")
	}
}

func TestMultipleNoWidgetsErrors(t *testing.T) {
	// Multiple instances should have same message
	err1 := &NoWidgetsError{}
	err2 := &NoWidgetsError{}

	if err1.Error() != err2.Error() {
		t.Error("NoWidgetsError instances should have same message")
	}
}

func TestAppConfigPathPreserved(t *testing.T) {
	paths := []string{
		"config.json",
		"./config.json",
		"/absolute/path/config.json",
		"",
		"../relative/config.json",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			app := NewApp(path)
			if app.configMgr.GetConfigPath() != path {
				t.Errorf("config path = %q, want %q", app.configMgr.GetConfigPath(), path)
			}
		})
	}
}

func TestAppLifecycleManagerAccess(t *testing.T) {
	app := NewApp("config.json")

	// Verify internal lifecycle manager works
	w, h := app.lifecycle.GetDisplayDimensions()
	if w != 128 || h != 40 {
		t.Errorf("default dimensions = %dx%d, want 128x40", w, h)
	}
}
