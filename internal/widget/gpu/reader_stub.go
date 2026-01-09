//go:build !windows

package gpu

import "fmt"

// newReader returns an error on non-Windows platforms
func newReader() (Reader, error) {
	return nil, fmt.Errorf("GPU monitoring is not supported on this platform (Windows PDH required)")
}
