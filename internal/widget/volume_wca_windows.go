//go:build windows
// +build windows

package widget

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// VolumeReaderWCA manages Windows Core Audio using go-wca library
// with proper COM lifecycle (initialize once, not per call)
type VolumeReaderWCA struct {
	mu          sync.Mutex
	initialized bool
	aev         *wca.IAudioEndpointVolume
	mmd         *wca.IMMDevice
	mmde        *wca.IMMDeviceEnumerator
}

// NewVolumeReaderWCA creates a volume reader using go-wca with proper lifecycle
func NewVolumeReaderWCA() (*VolumeReaderWCA, error) {
	vr := &VolumeReaderWCA{}

	if err := vr.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	return vr, nil
}

// initialize sets up COM once for the lifetime of this reader
func (vr *VolumeReaderWCA) initialize() error {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	log.Printf("[VOLUME-WCA] Initializing COM (ONCE per goroutine)")

	// Initialize COM - ONCE, not per call
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return fmt.Errorf("CoInitializeEx failed: %w", err)
	}

	log.Printf("[VOLUME-WCA] COM initialized")

	// Create device enumerator
	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		ole.CoUninitialize()
		return fmt.Errorf("CoCreateInstance failed: %w", err)
	}
	vr.mmde = mmde

	log.Printf("[VOLUME-WCA] Device enumerator created")

	// Get default audio endpoint
	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		vr.cleanup()
		return fmt.Errorf("GetDefaultAudioEndpoint failed: %w", err)
	}
	vr.mmd = mmd

	log.Printf("[VOLUME-WCA] Default audio device obtained")

	// Activate IAudioEndpointVolume
	var aev *wca.IAudioEndpointVolume
	if err := mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		vr.cleanup()
		return fmt.Errorf("Activate failed: %w", err)
	}
	vr.aev = aev

	vr.initialized = true

	log.Printf("[VOLUME-WCA] IAudioEndpointVolume activated successfully")

	return nil
}

// GetVolume reads the current master volume level (0-100) and mute status
func (vr *VolumeReaderWCA) GetVolume() (volume float64, muted bool, err error) {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	if !vr.initialized {
		return 0, false, fmt.Errorf("not initialized")
	}

	// Get master volume level (0.0 - 1.0 scalar)
	var level float32
	if err := vr.aev.GetMasterVolumeLevelScalar(&level); err != nil {
		return 0, false, fmt.Errorf("GetMasterVolumeLevelScalar failed: %w", err)
	}

	// Get mute status
	var isMuted bool
	if err := vr.aev.GetMute(&isMuted); err != nil {
		return 0, false, fmt.Errorf("GetMute failed: %w", err)
	}

	// Convert to 0-100 scale
	volume = float64(level) * 100.0
	muted = isMuted

	return volume, muted, nil
}

// cleanup releases all COM objects
func (vr *VolumeReaderWCA) cleanup() {
	if vr.aev != nil {
		vr.aev.Release()
		vr.aev = nil
		log.Printf("[VOLUME-WCA] IAudioEndpointVolume released")
	}

	if vr.mmd != nil {
		vr.mmd.Release()
		vr.mmd = nil
		log.Printf("[VOLUME-WCA] IMMDevice released")
	}

	if vr.mmde != nil {
		vr.mmde.Release()
		vr.mmde = nil
		log.Printf("[VOLUME-WCA] IMMDeviceEnumerator released")
	}
}

// Close releases all COM resources and uninitializes COM
func (vr *VolumeReaderWCA) Close() {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	if !vr.initialized {
		return
	}

	log.Printf("[VOLUME-WCA] Closing volume reader")

	vr.cleanup()
	ole.CoUninitialize()
	vr.initialized = false

	log.Printf("[VOLUME-WCA] COM uninitialized, volume reader closed")
}

// newVolumeReader creates a platform-specific volume reader (Windows implementation using go-wca)
func newVolumeReader() (volumeReader, error) {
	return NewVolumeReaderWCA()
}
