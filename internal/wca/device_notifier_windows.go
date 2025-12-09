//go:build windows

package wca

import (
	"log"
	"sync"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// IIDIMMNotificationClient is the interface ID for IMMNotificationClient
var IIDIMMNotificationClient = ole.NewGUID("{7991EEC9-7E89-4D85-8390-6C703CEC60C0}")

// EDataFlow values for audio endpoint direction
//
//goland:noinspection ALL
const (
	ERender  = 0 // Audio rendering (playback)
	ECapture = 1 // Audio capture (recording)
	EAll     = 2 // Both render and capture
)

// ERole values for audio endpoint role
//
//goland:noinspection ALL
const (
	EConsole       = 0 // Games, system sounds, voice commands
	EMultimedia    = 1 // Music, movies, narration
	ECommunication = 2 // Voice communications
)

// DeviceNotifier manages audio device change notifications using IMMNotificationClient
type DeviceNotifier struct {
	mu          sync.RWMutex
	mmde        *wca.IMMDeviceEnumerator
	client      *notificationClient
	subscribers []chan struct{}
	started     bool
}

// Global singleton for device notifications
var (
	globalDeviceNotifier *DeviceNotifier
	deviceNotifierMu     sync.Mutex
)

// notificationClient implements IMMNotificationClient COM interface
type notificationClient struct {
	lpVtbl   *notificationClientVtbl
	refCount uint32
	notifier *DeviceNotifier
}

// notificationClientVtbl is the vtable for IMMNotificationClient
type notificationClientVtbl struct {
	QueryInterface         uintptr
	AddRef                 uintptr
	Release                uintptr
	OnDeviceStateChanged   uintptr
	OnDeviceAdded          uintptr
	OnDeviceRemoved        uintptr
	OnDefaultDeviceChanged uintptr
	OnPropertyValueChanged uintptr
}

// GetDeviceNotifier returns the global device notifier singleton
// Creates and starts it on first call
func GetDeviceNotifier() (*DeviceNotifier, error) {
	deviceNotifierMu.Lock()
	defer deviceNotifierMu.Unlock()

	if globalDeviceNotifier != nil {
		return globalDeviceNotifier, nil
	}

	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
	}

	if err := dn.start(); err != nil {
		return nil, err
	}

	globalDeviceNotifier = dn
	return globalDeviceNotifier, nil
}

// start initializes the device notifier and registers for notifications
func (dn *DeviceNotifier) start() error {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	if dn.started {
		return nil
	}

	// Initialize COM on this thread
	if err := EnsureCOMInitialized(); err != nil {
		return err
	}

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		return err
	}
	dn.mmde = mmde

	// Create notification client
	dn.client = newNotificationClient(dn)

	// Register for notifications
	// RegisterEndpointNotificationCallback is at vtable offset 6
	hr, _, _ := syscall.SyscallN(
		dn.mmde.VTable().RegisterEndpointNotificationCallback,
		uintptr(unsafe.Pointer(dn.mmde)),
		uintptr(unsafe.Pointer(dn.client)),
	)
	if hr != 0 {
		log.Printf("[DEVICE-NOTIFIER] Warning: RegisterEndpointNotificationCallback failed: 0x%08X", hr)
		// Continue anyway - we'll still work, just without proactive notifications
	} else {
		log.Printf("[DEVICE-NOTIFIER] Registered for audio device change notifications")
	}

	dn.started = true
	return nil
}

// Subscribe returns a channel that will receive a signal when the audio device changes
// The channel is buffered (capacity 1) to prevent blocking
func (dn *DeviceNotifier) Subscribe() <-chan struct{} {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	ch := make(chan struct{}, 1)
	dn.subscribers = append(dn.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber channel
func (dn *DeviceNotifier) Unsubscribe(ch <-chan struct{}) {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	for i, sub := range dn.subscribers {
		if sub == ch {
			dn.subscribers = append(dn.subscribers[:i], dn.subscribers[i+1:]...)
			close(sub)
			return
		}
	}
}

// notifySubscribers sends a signal to all subscribers (non-blocking)
func (dn *DeviceNotifier) notifySubscribers() {
	dn.mu.RLock()
	defer dn.mu.RUnlock()

	for _, ch := range dn.subscribers {
		select {
		case ch <- struct{}{}:
		default:
			// Channel full, subscriber already notified
		}
	}
}

// newNotificationClient creates a new IMMNotificationClient implementation
func newNotificationClient(notifier *DeviceNotifier) *notificationClient {
	client := &notificationClient{
		refCount: 1,
		notifier: notifier,
	}

	client.lpVtbl = &notificationClientVtbl{
		QueryInterface:         syscall.NewCallback(queryInterface),
		AddRef:                 syscall.NewCallback(addRef),
		Release:                syscall.NewCallback(release),
		OnDeviceStateChanged:   syscall.NewCallback(onDeviceStateChanged),
		OnDeviceAdded:          syscall.NewCallback(onDeviceAdded),
		OnDeviceRemoved:        syscall.NewCallback(onDeviceRemoved),
		OnDefaultDeviceChanged: syscall.NewCallback(onDefaultDeviceChanged),
		OnPropertyValueChanged: syscall.NewCallback(onPropertyValueChanged),
	}

	return client
}

// COM interface implementations

func queryInterface(this *notificationClient, riid *ole.GUID, ppvObject *unsafe.Pointer) uintptr {
	if ole.IsEqualGUID(riid, ole.IID_IUnknown) || ole.IsEqualGUID(riid, IIDIMMNotificationClient) {
		*ppvObject = unsafe.Pointer(this)
		this.refCount++
		return 0 // S_OK
	}
	*ppvObject = nil
	return 0x80004002 // E_NOINTERFACE
}

func addRef(this *notificationClient) uintptr {
	this.refCount++
	return uintptr(this.refCount)
}

func release(this *notificationClient) uintptr {
	this.refCount--
	return uintptr(this.refCount)
}

func onDeviceStateChanged(this *notificationClient, _ *uint16, dwNewState uint32) uintptr {
	// Device enabled/disabled - notify subscribers
	if this.notifier != nil {
		log.Printf("[DEVICE-NOTIFIER] Device state changed (state: %d)", dwNewState)
		this.notifier.notifySubscribers()
	}
	return 0 // S_OK
}

func onDeviceAdded(_ *notificationClient, _ *uint16) uintptr {
	// New device added - not critical for our use case
	return 0 // S_OK
}

func onDeviceRemoved(this *notificationClient, _ *uint16) uintptr {
	// Device removed - notify subscribers
	if this.notifier != nil {
		log.Printf("[DEVICE-NOTIFIER] Device removed")
		this.notifier.notifySubscribers()
	}
	return 0 // S_OK
}

func onDefaultDeviceChanged(this *notificationClient, flow uint32, role uint32, _ *uint16) uintptr {
	// Default device changed - this is the main event we care about
	if this.notifier != nil {
		// Only care about render devices (playback)
		if flow == ERender || flow == EAll {
			log.Printf("[DEVICE-NOTIFIER] Default audio device changed (flow: %d, role: %d)", flow, role)
			this.notifier.notifySubscribers()
		}
	}
	return 0 // S_OK
}

func onPropertyValueChanged(_ *notificationClient, _ *uint16, _ uintptr) uintptr {
	// Property changed - not critical for our use case
	return 0 // S_OK
}

// Stop unregisters the notification client and cleans up resources
func (dn *DeviceNotifier) Stop() {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	if !dn.started {
		return
	}

	// Unregister notification callback
	if dn.mmde != nil && dn.client != nil {
		hr, _, _ := syscall.SyscallN(
			dn.mmde.VTable().UnregisterEndpointNotificationCallback,
			uintptr(unsafe.Pointer(dn.mmde)),
			uintptr(unsafe.Pointer(dn.client)),
		)
		if hr != 0 {
			log.Printf("[DEVICE-NOTIFIER] Warning: UnregisterEndpointNotificationCallback failed: 0x%08X", hr)
		}
	}

	// Close all subscriber channels
	for _, ch := range dn.subscribers {
		close(ch)
	}
	dn.subscribers = nil

	// Release device enumerator
	if dn.mmde != nil {
		dn.mmde.Release()
		dn.mmde = nil
	}

	dn.started = false
	log.Printf("[DEVICE-NOTIFIER] Stopped")
}
