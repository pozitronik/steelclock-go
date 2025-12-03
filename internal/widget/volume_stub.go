//go:build !windows && !linux

package widget

import (
	"fmt"
)

// VolumeReaderWCA stub for unsupported platforms
type VolumeReaderWCA struct{}

// GetSharedVolumeReader returns an error on unsupported platforms
func GetSharedVolumeReader() (*VolumeReaderWCA, error) {
	return nil, fmt.Errorf("volume reader is not supported on this platform")
}

// stubVolumeReader is a stub implementation for unsupported platforms
type stubVolumeReader struct{}

// GetVolume returns an error indicating volume reading is not supported
func (r *stubVolumeReader) GetVolume() (vol float64, muted bool, err error) {
	return 0, false, fmt.Errorf("volume widget is not supported on this platform")
}

// Close does nothing (no resources to clean up)
func (r *stubVolumeReader) Close() {}

// newVolumeReader creates a stub volume reader for unsupported platforms
func newVolumeReader() (volumeReader, error) {
	return &stubVolumeReader{}, nil
}
