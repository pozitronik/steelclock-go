//go:build !windows && !linux

package screenmirror

import (
	"fmt"
	"image"
)

// stubCapture is a no-op implementation for unsupported platforms.
type stubCapture struct{}

// newScreenCapture returns an error on unsupported platforms.
func newScreenCapture(_ CaptureConfig) (ScreenCapture, error) {
	return nil, fmt.Errorf("screen_mirror widget is not supported on this platform")
}

// Capture returns an error for the stub.
func (s *stubCapture) Capture() (*image.RGBA, error) {
	return nil, fmt.Errorf("screen capture not supported on this platform")
}

// Close is a no-op for the stub.
func (s *stubCapture) Close() {}

// IsAvailable always returns false for the stub.
func (s *stubCapture) IsAvailable() bool {
	return false
}

// GetDisplayInfo returns empty info for the stub.
func (s *stubCapture) GetDisplayInfo() DisplayInfo {
	return DisplayInfo{}
}
