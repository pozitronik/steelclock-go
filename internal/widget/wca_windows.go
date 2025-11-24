//go:build windows

package widget

import (
	"fmt"
	"log"
	"runtime"
	"sync"

	ole "github.com/go-ole/go-ole"
)

// comInitMutex protects COM initialization to prevent race conditions
var (
	comInitMutex   sync.Mutex
	comInitialized bool
)

// EnsureCOMInitialized ensures COM is initialized on the calling thread.
// This function is safe to call multiple times from the same goroutine.
//
// IMPORTANT ASSUMPTION: All widgets that use COM are created in the same goroutine.
// This is guaranteed by the singleton pattern (GetSharedAudioCapture, GetSharedVolumeReader, etc.)
// which ensures widgets are created once in the first calling goroutine.
//
// COM is thread-specific, so this implementation relies on runtime.LockOSThread()
// to ensure the goroutine stays on the same OS thread after first initialization.
//
// Used by all Windows Core Audio (WCA) components:
// - AudioCaptureWCA (audio visualizer)
// - VolumeReaderWCA (volume widget)
// - MeterReaderWCA (volume meter widget)
func EnsureCOMInitialized() error {
	comInitMutex.Lock()
	defer comInitMutex.Unlock()

	// If already initialized, nothing to do
	if comInitialized {
		return nil
	}

	// Lock this goroutine to the current OS thread for COM apartment threading
	// COM requires thread affinity - once initialized on a thread, all COM calls
	// must happen on that same thread
	runtime.LockOSThread()

	// Initialize COM with COINIT_APARTMENTTHREADED (STA - Single-Threaded Apartment)
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		// Real initialization error - unlock thread and return
		runtime.UnlockOSThread()
		return fmt.Errorf("CoInitializeEx failed: %w", err)
	}

	// Successfully initialized COM
	comInitialized = true
	log.Printf("[WCA] COM initialized successfully on this thread")
	return nil
}
