//go:build !windows && !linux
// +build !windows,!linux

package widget

import (
	"fmt"
)

// VolumeReaderWCA stub for non-Windows platforms
type VolumeReaderWCA struct{}

// GetSharedVolumeReader returns an error on Unix platforms
func GetSharedVolumeReader() (*VolumeReaderWCA, error) {
	return nil, fmt.Errorf("volume reader is not supported on this platform (Unix/Linux/macOS)")
}

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
