//go:build !windows
// +build !windows

package widget

import (
	"fmt"
)

// unixVolumeReader is a stub implementation for Unix systems
// Volume widget is currently only supported on Windows
type unixVolumeReader struct{}

// GetVolume returns an error indicating volume reading is not supported on Unix
func (r *unixVolumeReader) GetVolume() (vol float64, muted bool, err error) {
	return 0, false, fmt.Errorf("volume widget is not supported on this platform (Unix/Linux/macOS)")
}

// Close does nothing on Unix (no resources to clean up)
func (r *unixVolumeReader) Close() {
	// No cleanup needed
}

// newVolumeReader creates a platform-specific volume reader (Unix stub)
func newVolumeReader() (volumeReader, error) {
	// Return stub that will error on GetVolume calls
	// This allows the widget to initialize but not function
	return &unixVolumeReader{}, nil
}
