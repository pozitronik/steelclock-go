//go:build windows

package widget

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

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
	mu             sync.Mutex
	initialized    bool
	comInitialized bool // Track if we initialized COM (vs already initialized)
	threadLocked   bool // Track if we locked the OS thread
	ami            *wca.IAudioMeterInformation
	mmd            *wca.IMMDevice
	mmde           *wca.IMMDeviceEnumerator
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
	mr.comInitialized = false // We don't own the COM initialization
	mr.threadLocked = false   // We don't own the thread lock

	// Create device enumerator
	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		if mr.comInitialized {
			ole.CoUninitialize()
		}
		if mr.threadLocked {
			runtime.UnlockOSThread()
			mr.threadLocked = false
		}
		return fmt.Errorf("CoCreateInstance failed: %w", err)
	}
	mr.mmde = mmde

	log.Printf("[METER-WCA] Device enumerator created")

	// Get default audio endpoint
	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		mr.cleanup()
		return fmt.Errorf("GetDefaultAudioEndpoint failed: %w", err)
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

// GetMeterData reads current audio meter values
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
		return nil, fmt.Errorf("GetPeakValue failed: %w", err)
	}
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
	if mr.ami != nil {
		mr.ami.Release()
		mr.ami = nil
		log.Printf("[METER-WCA] IAudioMeterInformation released")
	}

	if mr.mmd != nil {
		mr.mmd.Release()
		mr.mmd = nil
		log.Printf("[METER-WCA] IMMDevice released")
	}

	if mr.mmde != nil {
		mr.mmde.Release()
		mr.mmde = nil
		log.Printf("[METER-WCA] IMMDeviceEnumerator released")
	}
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

	// Only uninitialize COM if we initialized it (not if it was already initialized)
	if mr.comInitialized {
		ole.CoUninitialize()
		mr.comInitialized = false
		log.Printf("[METER-WCA] COM uninitialized")
	}

	// Only unlock the thread if we locked it
	if mr.threadLocked {
		runtime.UnlockOSThread()
		mr.threadLocked = false
		log.Printf("[METER-WCA] OS thread unlocked")
	}

	mr.initialized = false

	log.Printf("[METER-WCA] Meter reader closed")
}

// newMeterReader creates a platform-specific meter reader (Windows implementation using go-wca)
func newMeterReader() (meterReader, error) {
	return NewMeterReaderWCA()
}
