//go:build !windows

package fps

import "fmt"

// newReader returns an error on non-Windows platforms (RTSS is Windows-only).
func newReader() (Reader, error) {
	return nil, fmt.Errorf("FPS monitoring is not supported on this platform (RTSS is Windows-only)")
}
