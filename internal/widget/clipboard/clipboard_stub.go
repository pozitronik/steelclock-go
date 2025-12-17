//go:build !windows && !linux

package clipboard

import "fmt"

// stubReader is a no-op implementation for unsupported platforms.
type stubReader struct{}

// newReader returns an error on unsupported platforms.
func newReader() (Reader, error) {
	return nil, fmt.Errorf("clipboard widget is not supported on this platform")
}

// HasChanged always returns false for the stub.
func (r *stubReader) HasChanged() bool {
	return false
}

// Read returns an error for the stub.
func (r *stubReader) Read() (string, ContentType, error) {
	return "", TypeUnknown, fmt.Errorf("clipboard not supported on this platform")
}

// Close is a no-op for the stub.
func (r *stubReader) Close() error {
	return nil
}
