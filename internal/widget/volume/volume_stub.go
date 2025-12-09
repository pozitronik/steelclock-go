//go:build !windows && !linux

package volume

import (
	"fmt"
)

// ReaderWCA stub for unsupported platforms
type ReaderWCA struct{}

// GetSharedVolumeReader returns an error on unsupported platforms
func GetSharedVolumeReader() (*ReaderWCA, error) {
	return nil, fmt.Errorf("volume reader is not supported on this platform")
}

// stubReader is a stub implementation for unsupported platforms
type stubReader struct{}

// GetVolume returns an error indicating volume reading is not supported
func (r *stubReader) GetVolume() (vol float64, muted bool, err error) {
	return 0, false, fmt.Errorf("volume widget is not supported on this platform")
}

// Close does nothing (no resources to clean up)
func (r *stubReader) Close() {}

// newVolumeReader creates a stub volume reader for unsupported platforms
func newVolumeReader() (Reader, error) {
	return &stubReader{}, nil
}
