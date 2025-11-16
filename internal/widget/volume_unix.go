//go:build !windows
// +build !windows

package widget

import (
	"github.com/itchyny/volume-go"
)

// getSystemVolumeImpl gets the system volume on Unix-like systems
func getSystemVolumeImpl() (vol float64, muted bool, err error) {
	// Get current volume level (0-100)
	volumeLevel, err := volume.GetVolume()
	if err != nil {
		return 0, false, err
	}

	// Get mute status
	isMuted, err := volume.GetMuted()
	if err != nil {
		return float64(volumeLevel), false, err
	}

	return float64(volumeLevel), isMuted, nil
}
