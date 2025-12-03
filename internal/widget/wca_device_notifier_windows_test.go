//go:build windows

package widget

import (
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
)

// TestDeviceNotifierSubscribeUnsubscribe tests the subscription mechanism
func TestDeviceNotifierSubscribeUnsubscribe(t *testing.T) {
	// Create a DeviceNotifier directly (bypassing COM initialization)
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true, // Pretend it's started
	}

	// Subscribe
	ch1 := dn.Subscribe()
	ch2 := dn.Subscribe()

	if len(dn.subscribers) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", len(dn.subscribers))
	}

	// Unsubscribe first channel
	dn.Unsubscribe(ch1)

	if len(dn.subscribers) != 1 {
		t.Errorf("Expected 1 subscriber after unsubscribe, got %d", len(dn.subscribers))
	}

	// Verify ch1 is closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("Expected ch1 to be closed")
		}
	default:
		t.Error("Expected ch1 to be closed and readable")
	}

	// Unsubscribe second channel
	dn.Unsubscribe(ch2)

	if len(dn.subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", len(dn.subscribers))
	}
}

// TestDeviceNotifierUnsubscribeNonExistent tests unsubscribing a channel that was never subscribed
func TestDeviceNotifierUnsubscribeNonExistent(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	// Subscribe one channel
	ch1 := dn.Subscribe()

	// Create a channel that was never subscribed
	ch2 := make(chan struct{}, 1)

	// Try to unsubscribe the non-existent channel - should not panic
	dn.Unsubscribe(ch2)

	// Original subscriber should still be there
	if len(dn.subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(dn.subscribers))
	}

	// Cleanup
	dn.Unsubscribe(ch1)
}

// TestNotifySubscribers tests the notification mechanism
func TestNotifySubscribers(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	ch1 := dn.Subscribe()
	ch2 := dn.Subscribe()

	// Notify subscribers
	dn.notifySubscribers()

	// Both channels should receive notification
	select {
	case <-ch1:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("ch1 did not receive notification")
	}

	select {
	case <-ch2:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("ch2 did not receive notification")
	}
}

// TestNotifySubscribersNonBlocking tests that notification doesn't block when channel is full
func TestNotifySubscribersNonBlocking(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	ch := dn.Subscribe()

	// Fill the channel (capacity is 1)
	dn.notifySubscribers()

	// Second notification should not block
	done := make(chan bool)
	go func() {
		dn.notifySubscribers()
		done <- true
	}()

	select {
	case <-done:
		// OK - notification completed without blocking
	case <-time.After(100 * time.Millisecond):
		t.Error("notifySubscribers blocked when channel was full")
	}

	// Drain the channel
	<-ch
}

// TestNotifySubscribersConcurrent tests concurrent subscribe/unsubscribe/notify operations
func TestNotifySubscribersConcurrent(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	var wg sync.WaitGroup
	const numGoroutines = 10
	const numOperations = 100

	// Start goroutines that subscribe, receive, and unsubscribe
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				ch := dn.Subscribe()
				// Drain if there's a notification
				select {
				case <-ch:
				default:
				}
				dn.Unsubscribe(ch)
			}
		}()
	}

	// Start goroutines that send notifications
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				dn.notifySubscribers()
			}
		}()
	}

	wg.Wait()

	// All subscribers should be cleaned up
	dn.mu.RLock()
	remaining := len(dn.subscribers)
	dn.mu.RUnlock()

	if remaining != 0 {
		t.Errorf("Expected 0 subscribers after concurrent operations, got %d", remaining)
	}
}

// TestNewNotificationClient tests creation of notification client
func TestNewNotificationClient(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)

	if client == nil {
		t.Fatal("newNotificationClient returned nil")
	}

	if client.refCount != 1 {
		t.Errorf("Expected initial refCount of 1, got %d", client.refCount)
	}

	if client.notifier != dn {
		t.Error("Client notifier does not match")
	}

	if client.lpVtbl == nil {
		t.Error("Client vtable is nil")
	}

	// Verify vtable has all callbacks set
	if client.lpVtbl.QueryInterface == 0 {
		t.Error("QueryInterface callback not set")
	}
	if client.lpVtbl.AddRef == 0 {
		t.Error("AddRef callback not set")
	}
	if client.lpVtbl.Release == 0 {
		t.Error("Release callback not set")
	}
	if client.lpVtbl.OnDeviceStateChanged == 0 {
		t.Error("OnDeviceStateChanged callback not set")
	}
	if client.lpVtbl.OnDeviceAdded == 0 {
		t.Error("OnDeviceAdded callback not set")
	}
	if client.lpVtbl.OnDeviceRemoved == 0 {
		t.Error("OnDeviceRemoved callback not set")
	}
	if client.lpVtbl.OnDefaultDeviceChanged == 0 {
		t.Error("OnDefaultDeviceChanged callback not set")
	}
	if client.lpVtbl.OnPropertyValueChanged == 0 {
		t.Error("OnPropertyValueChanged callback not set")
	}
}

// TestAddRefRelease tests COM reference counting
func TestAddRefRelease(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)

	// Initial refCount should be 1
	if client.refCount != 1 {
		t.Errorf("Expected initial refCount of 1, got %d", client.refCount)
	}

	// AddRef should increment
	newCount := addRef(client)
	if newCount != 2 {
		t.Errorf("Expected refCount 2 after AddRef, got %d", newCount)
	}
	if client.refCount != 2 {
		t.Errorf("Expected client.refCount 2 after AddRef, got %d", client.refCount)
	}

	// Release should decrement
	newCount = release(client)
	if newCount != 1 {
		t.Errorf("Expected refCount 1 after Release, got %d", newCount)
	}
	if client.refCount != 1 {
		t.Errorf("Expected client.refCount 1 after Release, got %d", client.refCount)
	}
}

// TestQueryInterface tests COM QueryInterface implementation
func TestQueryInterface(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)
	var ppvObject unsafe.Pointer

	// Query for IUnknown - should succeed
	hr := queryInterface(client, ole.IID_IUnknown, &ppvObject)
	if hr != 0 {
		t.Errorf("QueryInterface for IUnknown failed with hr=0x%08X", hr)
	}
	if ppvObject != unsafe.Pointer(client) {
		t.Error("QueryInterface for IUnknown returned wrong pointer")
	}

	// Query for IMMNotificationClient - should succeed
	ppvObject = nil
	hr = queryInterface(client, IidImmnotificationclient, &ppvObject)
	if hr != 0 {
		t.Errorf("QueryInterface for IMMNotificationClient failed with hr=0x%08X", hr)
	}
	if ppvObject != unsafe.Pointer(client) {
		t.Error("QueryInterface for IMMNotificationClient returned wrong pointer")
	}

	// Query for unknown GUID - should fail with E_NOINTERFACE
	ppvObject = nil
	unknownGUID := ole.NewGUID("{00000000-0000-0000-0000-000000000000}")
	hr = queryInterface(client, unknownGUID, &ppvObject)
	if hr != 0x80004002 { // E_NOINTERFACE
		t.Errorf("QueryInterface for unknown GUID should return E_NOINTERFACE (0x80004002), got 0x%08X", hr)
	}
	if ppvObject != nil {
		t.Error("QueryInterface for unknown GUID should set ppvObject to nil")
	}
}

// TestOnDeviceStateChanged tests the device state changed callback
func TestOnDeviceStateChanged(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)
	ch := dn.Subscribe()

	// Call the callback
	hr := onDeviceStateChanged(client, nil, 1)
	if hr != 0 {
		t.Errorf("onDeviceStateChanged returned non-zero hr: 0x%08X", hr)
	}

	// Should have notified subscriber
	select {
	case <-ch:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("onDeviceStateChanged did not notify subscriber")
	}
}

// TestOnDeviceRemoved tests the device removed callback
func TestOnDeviceRemoved(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)
	ch := dn.Subscribe()

	// Call the callback
	hr := onDeviceRemoved(client, nil)
	if hr != 0 {
		t.Errorf("onDeviceRemoved returned non-zero hr: 0x%08X", hr)
	}

	// Should have notified subscriber
	select {
	case <-ch:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("onDeviceRemoved did not notify subscriber")
	}
}

// TestOnDefaultDeviceChanged tests the default device changed callback
func TestOnDefaultDeviceChanged(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)

	tests := []struct {
		name         string
		flow         uint32
		role         uint32
		shouldNotify bool
	}{
		{"eRender eConsole", eRender, eConsole, true},
		{"eRender eMultimedia", eRender, eMultimedia, true},
		{"eRender eCommunication", eRender, eCommunication, true},
		{"eAll eConsole", eAll, eConsole, true},
		{"eCapture eConsole", eCapture, eConsole, false}, // Capture devices are ignored
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := dn.Subscribe()
			defer dn.Unsubscribe(ch)

			hr := onDefaultDeviceChanged(client, tt.flow, tt.role, nil)
			if hr != 0 {
				t.Errorf("onDefaultDeviceChanged returned non-zero hr: 0x%08X", hr)
			}

			select {
			case <-ch:
				if !tt.shouldNotify {
					t.Error("Should not have notified for this flow type")
				}
			case <-time.After(50 * time.Millisecond):
				if tt.shouldNotify {
					t.Error("Should have notified for this flow type")
				}
			}
		})
	}
}

// TestOnDeviceAdded tests that device added callback returns S_OK but doesn't notify
func TestOnDeviceAdded(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)
	ch := dn.Subscribe()

	// Call the callback
	hr := onDeviceAdded(client, nil)
	if hr != 0 {
		t.Errorf("onDeviceAdded returned non-zero hr: 0x%08X", hr)
	}

	// Should NOT notify subscriber (device added is not critical)
	select {
	case <-ch:
		t.Error("onDeviceAdded should not notify subscribers")
	case <-time.After(50 * time.Millisecond):
		// OK - no notification expected
	}
}

// TestOnPropertyValueChanged tests that property changed callback returns S_OK but doesn't notify
func TestOnPropertyValueChanged(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     true,
	}

	client := newNotificationClient(dn)
	ch := dn.Subscribe()

	// Call the callback
	hr := onPropertyValueChanged(client, nil, 0)
	if hr != 0 {
		t.Errorf("onPropertyValueChanged returned non-zero hr: 0x%08X", hr)
	}

	// Should NOT notify subscriber (property changes are not critical)
	select {
	case <-ch:
		t.Error("onPropertyValueChanged should not notify subscribers")
	case <-time.After(50 * time.Millisecond):
		// OK - no notification expected
	}
}

// TestCallbacksWithNilNotifier tests callbacks don't panic when notifier is nil
func TestCallbacksWithNilNotifier(t *testing.T) {
	client := &notificationClient{
		refCount: 1,
		notifier: nil, // Intentionally nil
	}

	// None of these should panic
	hr := onDeviceStateChanged(client, nil, 0)
	if hr != 0 {
		t.Errorf("onDeviceStateChanged with nil notifier returned non-zero hr: 0x%08X", hr)
	}

	hr = onDeviceRemoved(client, nil)
	if hr != 0 {
		t.Errorf("onDeviceRemoved with nil notifier returned non-zero hr: 0x%08X", hr)
	}

	hr = onDefaultDeviceChanged(client, eRender, eConsole, nil)
	if hr != 0 {
		t.Errorf("onDefaultDeviceChanged with nil notifier returned non-zero hr: 0x%08X", hr)
	}
}

// TestDeviceNotifierStopWithoutStart tests Stop on unstarted notifier
func TestDeviceNotifierStopWithoutStart(t *testing.T) {
	dn := &DeviceNotifier{
		subscribers: make([]chan struct{}, 0),
		started:     false,
	}

	// Should not panic
	dn.Stop()

	if dn.started {
		t.Error("Stop should not set started to true")
	}
}

// TestEDataFlowConstants verifies the EDataFlow constants match Windows API
func TestEDataFlowConstants(t *testing.T) {
	// These values are defined by Windows and must not change
	if eRender != 0 {
		t.Errorf("eRender should be 0, got %d", eRender)
	}
	if eCapture != 1 {
		t.Errorf("eCapture should be 1, got %d", eCapture)
	}
	if eAll != 2 {
		t.Errorf("eAll should be 2, got %d", eAll)
	}
}

// TestERoleConstants verifies the ERole constants match Windows API
func TestERoleConstants(t *testing.T) {
	// These values are defined by Windows and must not change
	if eConsole != 0 {
		t.Errorf("eConsole should be 0, got %d", eConsole)
	}
	if eMultimedia != 1 {
		t.Errorf("eMultimedia should be 1, got %d", eMultimedia)
	}
	if eCommunication != 2 {
		t.Errorf("eCommunication should be 2, got %d", eCommunication)
	}
}
