//go:build windows

package widget

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// Cooldown period between reinitializations to prevent rapid loops
const meterReinitCooldown = 10 * time.Second

// Direct COM calls for IAudioMeterInformation methods not implemented in go-wca
// These use syscall to directly call the vtable methods

// getMeteringChannelCount calls IAudioMeterInformation::GetMeteringChannelCount directly via vtable
func getMeteringChannelCount(ami *wca.IAudioMeterInformation) (uint32, error) {
	// Get the vtable pointer
	vtbl := *(**uintptr)(unsafe.Pointer(ami))

	// GetMeteringChannelCount is at offset 4 in the vtable
	// (0=QueryInterface, 1=AddRef, 2=Release, 3=GetPeakValue, 4=GetMeteringChannelCount)
	getMeteringChannelCountPtr := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(vtbl)) + 4*unsafe.Sizeof(uintptr(0))))

	var channelCount uint32
	ret, _, _ := syscall.SyscallN(
		getMeteringChannelCountPtr,
		uintptr(unsafe.Pointer(ami)),
		uintptr(unsafe.Pointer(&channelCount)),
	)

	if ret != 0 {
		return 0, ole.NewError(ret)
	}

	return channelCount, nil
}

// getChannelsPeakValues calls IAudioMeterInformation::GetChannelsPeakValues directly via vtable
func getChannelsPeakValues(ami *wca.IAudioMeterInformation, channelCount uint32, peaks []float32) error {
	if channelCount == 0 || len(peaks) == 0 {
		return nil
	}

	// Get the vtable pointer
	vtbl := *(**uintptr)(unsafe.Pointer(ami))

	// GetChannelsPeakValues is at offset 5 in the vtable
	getChannelsPeakValuesPtr := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(vtbl)) + 5*unsafe.Sizeof(uintptr(0))))

	ret, _, _ := syscall.SyscallN(
		getChannelsPeakValuesPtr,
		uintptr(unsafe.Pointer(ami)),
		uintptr(channelCount),
		uintptr(unsafe.Pointer(&peaks[0])),
	)

	if ret != 0 {
		return ole.NewError(ret)
	}

	return nil
}

// MeterReaderWCA manages Windows Core Audio meter using go-wca library
// with proper COM lifecycle (initialize once, not per call)
type MeterReaderWCA struct {
	mu              sync.Mutex
	initialized     bool
	ami             *wca.IAudioMeterInformation
	mmd             *wca.IMMDevice
	mmde            *wca.IMMDeviceEnumerator
	consecutiveErrs int       // Track consecutive errors for device change detection
	lastReinitTime  time.Time // Track last reinit to enforce cooldown
}

// NewMeterReaderWCA creates a meter reader using go-wca with proper lifecycle
func NewMeterReaderWCA() (*MeterReaderWCA, error) {
	mr := &MeterReaderWCA{}

	if err := mr.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	return mr, nil
}

// initialize sets up COM once for the lifetime of this reader
func (mr *MeterReaderWCA) initialize() error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	log.Printf("[METER-WCA] Ensuring COM is initialized")

	// Ensure COM is initialized on this thread
	err := EnsureCOMInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize COM: %w", err)
	}

	log.Printf("[METER-WCA] COM is ready")

	// Note: We don't own COM cleanup - it's managed per-thread by EnsureCOMInitialized

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		return err
	}
	mr.mmde = mmde

	log.Printf("[METER-WCA] Device enumerator created")

	// Get default audio endpoint
	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		mr.cleanup()
		return err
	}
	mr.mmd = mmd

	log.Printf("[METER-WCA] Default audio device obtained")

	// Activate IAudioMeterInformation
	var ami *wca.IAudioMeterInformation
	if err := mmd.Activate(wca.IID_IAudioMeterInformation, wca.CLSCTX_ALL, nil, &ami); err != nil {
		mr.cleanup()
		return fmt.Errorf("Activate IAudioMeterInformation failed: %w", err)
	}
	mr.ami = ami

	mr.initialized = true

	log.Printf("[METER-WCA] IAudioMeterInformation activated successfully")

	return nil
}

// Reinitialize recreates the meter reader with a new device
// Call this when device change is detected
func (mr *MeterReaderWCA) Reinitialize() error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Enforce cooldown to prevent rapid reinitialization loops
	timeSinceLastReinit := time.Since(mr.lastReinitTime)
	if timeSinceLastReinit < meterReinitCooldown {
		remaining := meterReinitCooldown - timeSinceLastReinit
		log.Printf("[METER-WCA] Reinit skipped - cooldown active (%.1fs remaining)", remaining.Seconds())
		return nil // Not an error, just skipped
	}

	// Cleanup old resources
	mr.cleanup()
	mr.initialized = false
	mr.consecutiveErrs = 0

	log.Printf("[METER-WCA] Reinitializing after device change...")

	// Reinitialize COM and get new device
	err := EnsureCOMInitialized()
	if err != nil {
		mr.lastReinitTime = time.Now() // Update time even on failure
		return fmt.Errorf("failed to reinitialize COM: %w", err)
	}

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		mr.lastReinitTime = time.Now()
		return err
	}
	mr.mmde = mmde

	// Get default audio endpoint
	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		mr.cleanup()
		mr.lastReinitTime = time.Now()
		return err
	}
	mr.mmd = mmd

	// Activate IAudioMeterInformation
	var ami *wca.IAudioMeterInformation
	if err := mmd.Activate(wca.IID_IAudioMeterInformation, wca.CLSCTX_ALL, nil, &ami); err != nil {
		mr.cleanup()
		mr.lastReinitTime = time.Now()
		return fmt.Errorf("Activate IAudioMeterInformation failed: %w", err)
	}
	mr.ami = ami

	mr.initialized = true
	mr.lastReinitTime = time.Now()
	log.Printf("[METER-WCA] Reinitialized successfully")
	return nil
}

// NeedsReinitialize returns true if the reader needs to be reinitialized
func (mr *MeterReaderWCA) NeedsReinitialize() bool {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	return !mr.initialized || mr.consecutiveErrs >= 3
}

// GetMeterData reads current audio meter values
// Automatically detects device changes and marks for reinitialization
func (mr *MeterReaderWCA) GetMeterData(clippingThreshold, silenceThreshold float64) (*MeterData, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if !mr.initialized {
		return nil, fmt.Errorf("not initialized")
	}

	data := &MeterData{}

	// Get overall peak value (0.0 - 1.0)
	var peak float32
	if err := mr.ami.GetPeakValue(&peak); err != nil {
		mr.consecutiveErrs++
		if mr.consecutiveErrs >= 3 {
			log.Printf("[METER-WCA] Multiple consecutive errors detected, device may have changed")
			mr.initialized = false // Mark for reinitialization
		}
		return nil, fmt.Errorf("GetPeakValue failed: %w", err)
	}

	// Reset error counter on successful read
	mr.consecutiveErrs = 0
	data.Peak = float64(peak)

	// Check for clipping and audio presence
	data.IsClipping = data.Peak >= clippingThreshold
	data.HasAudio = data.Peak > silenceThreshold

	// Get channel count using direct COM call
	channelCount, err := getMeteringChannelCount(mr.ami)
	if err != nil {
		// If channel count fails, fall back to mono
		log.Printf("[METER-WCA] GetMeteringChannelCount failed (falling back to mono): %v", err)
		data.ChannelCount = 1
		data.ChannelPeaks = []float64{data.Peak}
		return data, nil
	}
	data.ChannelCount = int(channelCount)

	// Get per-channel peak values using direct COM call
	if channelCount > 0 {
		channelPeaks := make([]float32, channelCount)
		if err := getChannelsPeakValues(mr.ami, channelCount, channelPeaks); err != nil {
			// If per-channel fails, fall back to using overall peak for all channels
			log.Printf("[METER-WCA] GetChannelsPeakValues failed (using overall peak): %v", err)
			data.ChannelPeaks = make([]float64, channelCount)
			for i := range data.ChannelPeaks {
				data.ChannelPeaks[i] = data.Peak
			}
		} else {
			// Convert to float64
			data.ChannelPeaks = make([]float64, channelCount)
			for i, p := range channelPeaks {
				data.ChannelPeaks[i] = float64(p)
			}
		}
	}

	return data, nil
}

// cleanup releases all COM objects
func (mr *MeterReaderWCA) cleanup() {
	SafeReleaseAudioMeterInformation(&mr.ami)
	SafeReleaseMMDevice(&mr.mmd)
	SafeReleaseMMDeviceEnumerator(&mr.mmde)
}

// Close releases all COM resources and uninitializes COM
func (mr *MeterReaderWCA) Close() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if !mr.initialized {
		return
	}

	log.Printf("[METER-WCA] Closing meter reader")

	mr.cleanup()
	mr.initialized = false

	log.Printf("[METER-WCA] Meter reader closed")
}

// newMeterReader creates a platform-specific meter reader (Windows implementation using go-wca)
func newMeterReader() (meterReader, error) {
	return NewMeterReaderWCA()
}
