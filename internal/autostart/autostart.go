package autostart

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotSupported is returned on platforms where autostart is not implemented.
var ErrNotSupported = errors.New("autostart is not supported on this platform")

// IsEnabled checks whether the application is registered for autostart.
func IsEnabled() (bool, error) {
	return isEnabled()
}

// Enable registers the application for autostart.
func Enable() error {
	return enable()
}

// Disable removes the application from autostart.
func Disable() error {
	return disable()
}

// Toggle switches the autostart state and returns the new state.
func Toggle() (enabled bool, err error) {
	current, err := IsEnabled()
	if err != nil {
		return false, err
	}

	if current {
		return false, Disable()
	}
	return true, Enable()
}

// getAppPaths returns the resolved executable path and its parent directory.
func getAppPaths() (exePath, exeDir string, err error) {
	raw, err := os.Executable()
	if err != nil {
		return "", "", err
	}

	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return "", "", err
	}

	return resolved, filepath.Dir(resolved), nil
}
