//go:build !windows && !linux
// +build !windows,!linux

package widget

import (
	"fmt"
)

// unixMeterReader is a stub implementation for Unix systems
// Volume meter widget is currently only supported on Windows
type unixMeterReader struct{}

// GetMeterData returns an error indicating meter reading is not supported on Unix
func (r *unixMeterReader) GetMeterData(clippingThreshold, silenceThreshold float64) (*MeterData, error) {
	return nil, fmt.Errorf("volume meter widget is not supported on this platform (Unix/Linux/macOS)")
}

// Close does nothing on Unix (no resources to clean up)
func (r *unixMeterReader) Close() {
	// No cleanup needed
}

// newMeterReader creates a platform-specific meter reader (Unix stub)
func newMeterReader() (meterReader, error) {
	// Return stub that will error on GetMeterData calls
	// This allows the widget to initialize but not function
	return &unixMeterReader{}, nil
}
