//go:build !windows && !linux

package clipboard

import "fmt"

// stubClipboardReader is a no-op implementation for unsupported platforms.
type stubClipboardReader struct{}

// newClipboardReader returns an error on unsupported platforms.
func newClipboardReader() (ClipboardReader, error) {
	return nil, fmt.Errorf("clipboard widget is not supported on this platform")
}

// HasChanged always returns false for the stub.
func (r *stubClipboardReader) HasChanged() bool {
	return false
}

// Read returns an error for the stub.
func (r *stubClipboardReader) Read() (string, ContentType, error) {
	return "", TypeUnknown, fmt.Errorf("clipboard not supported on this platform")
}

// Close is a no-op for the stub.
func (r *stubClipboardReader) Close() error {
	return nil
}
