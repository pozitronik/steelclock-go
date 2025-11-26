//go:build windows

package widget

import (
	"fmt"
	"log"
	"sync"

	"github.com/moutend/go-wca/pkg/wca"
)

// Shared volume reader instance (singleton)
var (
	sharedVolumeReader     *VolumeReaderWCA
	sharedVolumeReaderOnce sync.Once
	sharedVolumeReaderErr  error
)

// GetSharedVolumeReader returns the shared VolumeReaderWCA instance
func GetSharedVolumeReader() (*VolumeReaderWCA, error) {
	sharedVolumeReaderOnce.Do(func() {
		vr := &VolumeReaderWCA{}
		if err := vr.initialize(); err != nil {
			sharedVolumeReaderErr = fmt.Errorf("failed to initialize: %w", err)
			return
		}
		sharedVolumeReader = vr
	})

	if sharedVolumeReaderErr != nil {
		return nil, sharedVolumeReaderErr
	}

	return sharedVolumeReader, nil
}

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

	log.Printf("[VOLUME-WCA] Ensuring COM is initialized")

	// Ensure COM is initialized on this thread
	err := EnsureCOMInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize COM: %w", err)
	}

	log.Printf("[VOLUME-WCA] COM is ready")

	// Note: We don't own COM cleanup - it's managed per-thread by EnsureCOMInitialized

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		return err
	}
	vr.mmde = mmde

	log.Printf("[VOLUME-WCA] Device enumerator created")

	// Get default audio endpoint
	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		vr.cleanup()
		return err
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
	SafeReleaseAudioEndpointVolume(&vr.aev)
	SafeReleaseMMDevice(&vr.mmd)
	SafeReleaseMMDeviceEnumerator(&vr.mmde)
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
	vr.initialized = false

	log.Printf("[VOLUME-WCA] Volume reader closed")
}

// newVolumeReader creates a platform-specific volume reader (Windows implementation using go-wca)
// Each widget gets its own reader instance to ensure proper lifecycle management
func newVolumeReader() (volumeReader, error) {
	return NewVolumeReaderWCA()
}
