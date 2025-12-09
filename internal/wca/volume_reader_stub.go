//go:build !windows

package wca

import (
	"fmt"
)

// VolumeReaderWCA stub for non-Windows platforms
type VolumeReaderWCA struct{}

// GetSharedVolumeReader returns an error on non-Windows platforms
func GetSharedVolumeReader() (*VolumeReaderWCA, error) {
	return nil, fmt.Errorf("volume reader is not supported on this platform")
}

// NewVolumeReaderWCA returns an error on non-Windows platforms
func NewVolumeReaderWCA() (*VolumeReaderWCA, error) {
	return nil, fmt.Errorf("volume reader is not supported on this platform")
}

// GetVolume returns an error indicating volume reading is not supported
func (vr *VolumeReaderWCA) GetVolume() (volume float64, muted bool, err error) {
	return 0, false, fmt.Errorf("volume reader is not supported on this platform")
}

// Close does nothing on non-Windows platforms
func (vr *VolumeReaderWCA) Close() {}

// Reinitialize returns an error on non-Windows platforms
func (vr *VolumeReaderWCA) Reinitialize() error {
	return fmt.Errorf("volume reader is not supported on this platform")
}

// NeedsReinitialize returns false on non-Windows platforms
func (vr *VolumeReaderWCA) NeedsReinitialize() bool {
	return false
}

// IsInitialized returns false on non-Windows platforms
func (vr *VolumeReaderWCA) IsInitialized() bool {
	return false
}
