//go:build darwin

package volume

/*
#cgo LDFLAGS: -framework CoreAudio

#include <CoreAudio/CoreAudio.h>

// getDefaultOutputDeviceID returns the default output audio device ID
static AudioDeviceID getDefaultOutputDeviceID() {
	AudioDeviceID deviceID = kAudioObjectUnknown;
	UInt32 size = sizeof(deviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	OSStatus err = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);
	if (err != noErr) return kAudioObjectUnknown;
	return deviceID;
}

// getVolumeScalar returns the volume scalar (0.0-1.0) for the given device
// Tries master channel first, then averages L/R
static float getVolumeScalar(AudioDeviceID deviceID) {
	Float32 volume = 0.0;
	UInt32 size = sizeof(volume);

	// Try master channel (element 0)
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMain
	};

	OSStatus err = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &volume);
	if (err == noErr) return volume;

	// Fallback: average L (element 1) and R (element 2)
	Float32 left = 0.0, right = 0.0;
	addr.mElement = 1;
	err = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &left);
	if (err != noErr) return -1.0;

	addr.mElement = 2;
	err = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &right);
	if (err != noErr) return left; // mono device

	return (left + right) / 2.0;
}

// getMuteState returns 1 if muted, 0 if not, -1 on error
static int getMuteState(AudioDeviceID deviceID) {
	UInt32 muted = 0;
	UInt32 size = sizeof(muted);

	// Try master channel
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyMute,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMain
	};

	OSStatus err = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &muted);
	if (err == noErr) return (int)muted;

	// Fallback: check channel 1
	addr.mElement = 1;
	err = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &muted);
	if (err == noErr) return (int)muted;

	return 0; // Assume not muted if we can't read it
}
*/
import "C"

import (
	"fmt"
	"log"
	"sync"
)

// DarwinReader reads system volume using CoreAudio
type DarwinReader struct {
	mu sync.Mutex
}

// NewDarwinReader creates a new macOS volume reader
func NewDarwinReader() (*DarwinReader, error) {
	// Verify we can get the default output device
	deviceID := C.getDefaultOutputDeviceID()
	if deviceID == C.kAudioObjectUnknown {
		return nil, fmt.Errorf("no default audio output device found")
	}

	log.Printf("[VOLUME-DARWIN] Using CoreAudio")
	return &DarwinReader{}, nil
}

// GetVolume reads the current master volume level (0-100) and mute status
func (r *DarwinReader) GetVolume() (volume float64, muted bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	deviceID := C.getDefaultOutputDeviceID()
	if deviceID == C.kAudioObjectUnknown {
		return 0, false, fmt.Errorf("no default audio output device")
	}

	vol := C.getVolumeScalar(deviceID)
	if vol < 0 {
		return 0, false, fmt.Errorf("failed to read volume scalar")
	}

	muteState := C.getMuteState(deviceID)

	return float64(vol) * 100.0, muteState == 1, nil
}

// Close releases resources (no-op for CoreAudio property queries)
func (r *DarwinReader) Close() {}

// Reinitialize re-checks the default output device
func (r *DarwinReader) Reinitialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	deviceID := C.getDefaultOutputDeviceID()
	if deviceID == C.kAudioObjectUnknown {
		return fmt.Errorf("no default audio output device after reinitialization")
	}

	log.Printf("[VOLUME-DARWIN] Reinitialized with CoreAudio")
	return nil
}

// NeedsReinitialize returns false - CoreAudio handles device switching transparently
func (r *DarwinReader) NeedsReinitialize() bool {
	return false
}

// newVolumeReader creates a platform-specific volume reader (macOS implementation)
func newVolumeReader() (Reader, error) {
	return NewDarwinReader()
}
