//go:build !windows && !linux

package widget

import (
	"fmt"
)

// stubMeterReader is a stub implementation for unsupported platforms
type stubMeterReader struct{}

// GetMeterData returns an error indicating meter reading is not supported
func (r *stubMeterReader) GetMeterData(clippingThreshold, silenceThreshold float64) (*MeterData, error) {
	return nil, fmt.Errorf("volume meter widget is not supported on this platform")
}

// Close does nothing (no resources to clean up)
func (r *stubMeterReader) Close() {}

// newMeterReader creates a stub meter reader for unsupported platforms
func newMeterReader() (meterReader, error) {
	return &stubMeterReader{}, nil
}
