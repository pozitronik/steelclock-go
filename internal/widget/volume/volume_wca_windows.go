//go:build windows

package volume

import (
	wcautil "github.com/pozitronik/steelclock-go/internal/wca"
)

// ReaderWCA is an alias for the shared WCA volume reader
type ReaderWCA = wcautil.VolumeReaderWCA

// GetSharedVolumeReader returns the shared volume reader instance
func GetSharedVolumeReader() (*ReaderWCA, error) {
	return wcautil.GetSharedVolumeReader()
}

// NewReaderWCA creates a new volume reader using WCA
func NewReaderWCA() (*ReaderWCA, error) {
	return wcautil.NewVolumeReaderWCA()
}

// newVolumeReader creates a platform-specific volume reader (Windows implementation using go-wca)
// Each widget gets its own reader instance to ensure proper lifecycle management
func newVolumeReader() (Reader, error) {
	return wcautil.NewVolumeReaderWCA()
}
