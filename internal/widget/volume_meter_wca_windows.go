//go:build windows
// +build windows

package widget

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

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

	// Lock this goroutine to the current OS thread for COM apartment threading
	// COM requires thread affinity - once initialized on a thread, all COM calls
	// must happen on that same thread
	runtime.LockOSThread()
	mr.threadLocked = true

	log.Printf("[METER-WCA] Initializing COM (ONCE per goroutine)")

	// Initialize COM - ONCE, not per call
	// Note: CoInitializeEx may return an error if COM is already initialized on this thread
	// In test environments, this is expected and we handle it gracefully
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		// Check if the error is because COM is already initialized (S_FALSE or RPC_E_CHANGED_MODE)
		// In that case, we can continue - COM is ready to use
		errMsg := err.Error()
		if errMsg == "Incorrect function." || errMsg == "Cannot change thread mode after it is set." {
			log.Printf("[METER-WCA] COM already initialized on this thread (expected in test environment)")
			// Don't set comInitialized since we didn't initialize it
		} else {
			runtime.UnlockOSThread()
			mr.threadLocked = false
			return fmt.Errorf("CoInitializeEx failed: %w", err)
		}
	} else {
		log.Printf("[METER-WCA] COM initialized successfully")
		mr.comInitialized = true
	}

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

	// NOTE: GetMeteringChannelCount and GetChannelsPeakValues are not implemented in go-wca
	// (they return E_NOTIMPL). For now, we only support mono metering using overall peak.
	// TODO: Implement proper COM calls for per-channel metering when needed.
	data.ChannelCount = 1 // Assume mono for now
	data.ChannelPeaks = []float64{data.Peak}

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
