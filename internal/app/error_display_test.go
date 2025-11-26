package app

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestErrorDisplayRefreshRateConstant(t *testing.T) {
	// Verify the constant has expected value
	if ErrorDisplayRefreshRateMs != 500 {
		t.Errorf("ErrorDisplayRefreshRateMs = %d, want 500", ErrorDisplayRefreshRateMs)
	}
}

func TestStartWithErrorDisplayNoClient(t *testing.T) {
	// Test error display when no client exists and no lastGoodConfig
	app := NewApp("nonexistent.json")

	// Should fail because we can't create a client
	err := app.startWithErrorDisplay("TEST ERROR", 128, 40)
	if err == nil {
		// If it succeeds, we need to clean up
		if app.comp != nil {
			app.comp.Stop()
		}
		t.Skip("Backend available, cannot test error path")
	}

	// Expected to fail with client creation error
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestStartWithErrorDisplayDimensions(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		height  int
		message string
	}{
		{"standard dimensions", 128, 40, "ERROR"},
		{"larger display", 256, 64, "CONFIG"},
		{"minimal display", 32, 8, "ERR"},
		{"zero width", 0, 40, "TEST"},
		{"zero height", 128, 0, "TEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp("config.json")

			// This will fail without a backend, but we're testing the code path
			err := app.startWithErrorDisplay(tt.message, tt.width, tt.height)
			if err == nil {
				if app.comp != nil {
					app.comp.Stop()
				}
				t.Skip("Backend available")
			}
			// Expected failure path - just verify no panic
		})
	}
}

func TestStartWithErrorDisplayMessages(t *testing.T) {
	messages := []string{
		"CONFIG",
		"NO WIDGETS",
		"ERROR",
		"",
		"A very long error message that exceeds normal display width",
	}

	for _, msg := range messages {
		t.Run("message_"+msg, func(t *testing.T) {
			app := NewApp("config.json")

			// This will fail without a backend, but we're testing the code path
			err := app.startWithErrorDisplay(msg, 128, 40)
			if err == nil {
				if app.comp != nil {
					app.comp.Stop()
				}
				t.Skip("Backend available")
			}
			// Just verify no panic with various messages
		})
	}
}

func TestStartWithErrorDisplayWithLastGoodConfig(t *testing.T) {
	app := NewApp("config.json")

	// Set a lastGoodConfig to test that code path
	app.lastGoodConfig = &config.Config{
		GameName:        "LAST_GOOD",
		GameDisplayName: "Last Good",
		Backend:         "gamesense",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}

	// This will fail without a backend, but we're testing the code path
	err := app.startWithErrorDisplay("TEST", 128, 40)
	if err == nil {
		if app.comp != nil {
			app.comp.Stop()
		}
		t.Skip("Backend available")
	}
	// Expected failure path - just verify no panic and correct error handling
}
