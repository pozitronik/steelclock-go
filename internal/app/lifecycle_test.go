package app

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewLifecycleManager(t *testing.T) {
	lm := NewLifecycleManager()

	if lm == nil {
		t.Fatal("NewLifecycleManager returned nil")
	}

	if !lm.isFirstStart {
		t.Error("isFirstStart should be true initially")
	}

	if lm.retryCancel == nil {
		t.Error("retryCancel channel not initialized")
	}

	if lm.lastGoodConfig != nil {
		t.Error("lastGoodConfig should be nil initially")
	}

	if lm.currentBackend != "" {
		t.Errorf("currentBackend should be empty, got %q", lm.currentBackend)
	}
}

func TestLifecycleManagerGetDisplayDimensions(t *testing.T) {
	lm := NewLifecycleManager()

	// Without config, should return defaults
	w, h := lm.GetDisplayDimensions()
	if w != 128 || h != 40 {
		t.Errorf("default dimensions = %dx%d, want 128x40", w, h)
	}
}

func TestLifecycleManagerGetDisplayDimensionsWithConfig(t *testing.T) {
	lm := NewLifecycleManager()

	// Set a config with custom dimensions
	lm.mu.Lock()
	lm.lastGoodConfig = &config.Config{
		Display: config.DisplayConfig{
			Width:  256,
			Height: 64,
		},
	}
	lm.mu.Unlock()

	w, h := lm.GetDisplayDimensions()
	if w != 256 || h != 64 {
		t.Errorf("dimensions with config = %dx%d, want 256x64", w, h)
	}
}

func TestLifecycleManagerStopWithNilComponents(t *testing.T) {
	lm := NewLifecycleManager()

	// Should not panic
	lm.Stop()
	lm.Shutdown()
}

func TestLifecycleManagerGetLastGoodConfig(t *testing.T) {
	lm := NewLifecycleManager()

	// Initially nil
	if lm.GetLastGoodConfig() != nil {
		t.Error("lastGoodConfig should be nil initially")
	}

	// Set a config
	cfg := &config.Config{
		GameName: "TEST",
	}
	lm.mu.Lock()
	lm.lastGoodConfig = cfg
	lm.mu.Unlock()

	// Now should return it
	if lm.GetLastGoodConfig() != cfg {
		t.Error("GetLastGoodConfig should return the set config")
	}
}

//goland:noinspection GoBoolExpressions
func TestErrorDisplayRefreshRateConstant(t *testing.T) {
	// Verify the constant has expected value
	if ErrorDisplayRefreshRateMs != 500 {
		t.Errorf("ErrorDisplayRefreshRateMs = %d, want 500", ErrorDisplayRefreshRateMs)
	}
}

func TestStartErrorDisplayNoClient(t *testing.T) {
	// Test error display when no client exists and no lastGoodConfig
	lm := NewLifecycleManager()

	// Should fail because we can't create a client
	err := lm.StartErrorDisplay("TEST ERROR", 128, 40)
	if err == nil {
		// If it succeeds, we need to clean up
		lm.Stop()
		t.Skip("Backend available, cannot test error path")
	}

	// Expected to fail with client creation error
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestStartErrorDisplayDimensions(t *testing.T) {
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
			lm := NewLifecycleManager()

			// This will fail without a backend, but we're testing the code path
			err := lm.StartErrorDisplay(tt.message, tt.width, tt.height)
			if err == nil {
				lm.Stop()
				t.Skip("Backend available")
			}
			// Expected failure path - just verify no panic
		})
	}
}

func TestStartErrorDisplayMessages(t *testing.T) {
	messages := []string{
		"CONFIG",
		"NO WIDGETS",
		"ERROR",
		"",
		"A very long error message that exceeds normal display width",
	}

	for _, msg := range messages {
		t.Run("message_"+msg, func(t *testing.T) {
			lm := NewLifecycleManager()

			// This will fail without a backend, but we're testing the code path
			err := lm.StartErrorDisplay(msg, 128, 40)
			if err == nil {
				lm.Stop()
				t.Skip("Backend available")
			}
			// Just verify no panic with various messages
		})
	}
}

func TestStartErrorDisplayWithLastGoodConfig(t *testing.T) {
	lm := NewLifecycleManager()

	// Set a lastGoodConfig to test that code path
	lm.mu.Lock()
	lm.lastGoodConfig = &config.Config{
		GameName:        "LAST_GOOD",
		GameDisplayName: "Last Good",
		Backend:         "gamesense",
		Display: config.DisplayConfig{
			Width:  128,
			Height: 40,
		},
	}
	lm.mu.Unlock()

	// This will fail without a backend, but we're testing the code path
	err := lm.StartErrorDisplay("TEST", 128, 40)
	if err == nil {
		lm.Stop()
		t.Skip("Backend available")
	}
	// Expected failure path - just verify no panic and correct error handling
}
