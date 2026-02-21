package autostart

import (
	"path/filepath"
	"testing"
)

func TestGetAppPaths(t *testing.T) {
	exePath, exeDir, err := getAppPaths()
	if err != nil {
		t.Fatalf("getAppPaths() error = %v", err)
	}

	if exePath == "" {
		t.Error("getAppPaths() exePath is empty")
	}

	if exeDir == "" {
		t.Error("getAppPaths() exeDir is empty")
	}

	// exeDir must be the parent of exePath
	if filepath.Dir(exePath) != exeDir {
		t.Errorf("getAppPaths() exeDir = %q, want parent of %q (%q)", exeDir, exePath, filepath.Dir(exePath))
	}
}

func TestToggleConsistency(t *testing.T) {
	// Toggle should return the opposite of the current state.
	// We cannot assert the exact outcome because it depends on platform
	// permissions, but we can verify it doesn't panic and returns coherent results.
	initial, err := IsEnabled()
	if err != nil {
		t.Skipf("IsEnabled() not supported: %v", err)
	}

	newState, err := Toggle()
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}

	if newState == initial {
		t.Errorf("Toggle() returned %v, same as initial state %v", newState, initial)
	}

	// Restore original state
	restored, err := Toggle()
	if err != nil {
		t.Fatalf("Toggle() restore error = %v", err)
	}

	if restored != initial {
		t.Errorf("Toggle() restore returned %v, want %v", restored, initial)
	}
}
