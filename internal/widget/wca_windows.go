//go:build windows

package widget

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// comInitMutex protects COM initialization to prevent race conditions
var comInitMutex sync.Mutex

// EnsureCOMInitialized ensures COM is initialized on the calling thread.
// This function is safe to call multiple times from the same thread.
//
// IMPORTANT: COM is thread-specific. This function must be called on each thread
// that needs to use COM. The function handles both cases:
// - First call on a thread: initializes COM and locks the thread
// - Subsequent calls on same thread: no-op (COM already initialized)
// - Calls on different threads: each thread gets its own COM initialization
//
// Used by all Windows Core Audio (WCA) components:
// - AudioCaptureWCA (audio visualizer)
// - VolumeReaderWCA (volume widget)
// - MeterReaderWCA (volume meter widget)
func EnsureCOMInitialized() error {
	comInitMutex.Lock()
	defer comInitMutex.Unlock()

	// Lock this goroutine to the current OS thread for COM apartment threading
	// COM requires thread affinity - once initialized on a thread, all COM calls
	// must happen on that same thread
	runtime.LockOSThread()

	// Initialize COM with COINIT_APARTMENTTHREADED (STA - Single-Threaded Apartment)
	// CoInitializeEx returns:
	// - S_OK (0x00000000): Successfully initialized COM on this thread
	// - S_FALSE (0x00000001): COM already initialized on this thread (success)
	// - Error: Actual initialization failure
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		// Check if this is S_FALSE (0x00000001) - already initialized on this thread
		// Use errors.As to properly unwrap and check for ole.OleError
		var oleErr *ole.OleError
		if errors.As(err, &oleErr) {
			// S_FALSE = 0x00000001 (in COM, S_FALSE is a success code with value 1)
			// Some implementations might represent it as 0x80000001 (signed interpretation)
			if oleErr.Code() == 0x00000001 || oleErr.Code() == 0x80000001 {
				// S_FALSE: COM already initialized on this thread, this is OK
				log.Printf("[WCA] COM already initialized on this thread (S_FALSE)")
				return nil
			}
		}

		// Real initialization error - unlock thread and return
		runtime.UnlockOSThread()
		return fmt.Errorf("CoInitializeEx failed: %w", err)
	}

	// Successfully initialized COM (S_OK)
	log.Printf("[WCA] COM initialized successfully on this thread")
	return nil
}

// CreateDeviceEnumerator creates an IMMDeviceEnumerator instance.
// This is a common operation used by all WCA components to enumerate audio devices.
// Returns the enumerator or an error if creation fails.
func CreateDeviceEnumerator() (*wca.IMMDeviceEnumerator, error) {
	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return nil, fmt.Errorf("failed to create device enumerator: %w", err)
	}
	return mmde, nil
}

// GetDefaultRenderDevice retrieves the default audio render (output) endpoint.
// This is a common operation used by all WCA components to access the default speakers/headphones.
// Returns the device or an error if retrieval fails.
func GetDefaultRenderDevice(mmde *wca.IMMDeviceEnumerator) (*wca.IMMDevice, error) {
	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		return nil, fmt.Errorf("failed to get default audio device: %w", err)
	}
	return mmd, nil
}

// SafeReleaseAudioEndpointVolume safely releases an IAudioEndpointVolume interface.
func SafeReleaseAudioEndpointVolume(ptr **wca.IAudioEndpointVolume) {
	if ptr != nil && *ptr != nil {
		(*ptr).Release()
		*ptr = nil
		log.Printf("[WCA] IAudioEndpointVolume released")
	}
}

// SafeReleaseAudioMeterInformation safely releases an IAudioMeterInformation interface.
func SafeReleaseAudioMeterInformation(ptr **wca.IAudioMeterInformation) {
	if ptr != nil && *ptr != nil {
		(*ptr).Release()
		*ptr = nil
		log.Printf("[WCA] IAudioMeterInformation released")
	}
}

// SafeReleaseMMDevice safely releases an IMMDevice interface.
func SafeReleaseMMDevice(ptr **wca.IMMDevice) {
	if ptr != nil && *ptr != nil {
		(*ptr).Release()
		*ptr = nil
		log.Printf("[WCA] IMMDevice released")
	}
}

// SafeReleaseMMDeviceEnumerator safely releases an IMMDeviceEnumerator interface.
func SafeReleaseMMDeviceEnumerator(ptr **wca.IMMDeviceEnumerator) {
	if ptr != nil && *ptr != nil {
		(*ptr).Release()
		*ptr = nil
		log.Printf("[WCA] IMMDeviceEnumerator released")
	}
}

// SafeReleaseAudioClient safely releases an IAudioClient interface.
func SafeReleaseAudioClient(ptr **wca.IAudioClient) {
	if ptr != nil && *ptr != nil {
		(*ptr).Release()
		*ptr = nil
		log.Printf("[WCA] IAudioClient released")
	}
}

// SafeReleaseAudioCaptureClient safely releases an IAudioCaptureClient interface.
func SafeReleaseAudioCaptureClient(ptr **wca.IAudioCaptureClient) {
	if ptr != nil && *ptr != nil {
		(*ptr).Release()
		*ptr = nil
		log.Printf("[WCA] IAudioCaptureClient released")
	}
}
