//go:build windows

package volume

import (
	wcautil "github.com/pozitronik/steelclock-go/internal/wca"
)

// newVolumeReader creates a platform-specific volume reader (Windows implementation using go-wca)
// Each widget gets its own reader instance to ensure proper lifecycle management
func newVolumeReader() (Reader, error) {
	return wcautil.NewVolumeReaderWCA()
}
