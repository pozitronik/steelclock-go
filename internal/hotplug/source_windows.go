//go:build windows

package hotplug

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modCfgMgr32                  = syscall.NewLazyDLL("cfgmgr32.dll")
	procCMRegisterNotification   = modCfgMgr32.NewProc("CM_Register_Notification")
	procCMUnregisterNotification = modCfgMgr32.NewProc("CM_Unregister_Notification")
)

// hidInterfaceGUID is GUID_DEVINTERFACE_HID.
var hidInterfaceGUID = windows.GUID{
	Data1: 0x4D1E55B2,
	Data2: 0xF16F,
	Data3: 0x11CF,
	Data4: [8]byte{0x88, 0xCB, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
}

const (
	crSuccess                            = 0
	cmNotifyFilterTypeDeviceInterface    = 0
	cmNotifyActionDeviceInterfaceArrival = 0
	cmNotifyActionDeviceInterfaceRemoval = 1

	// symbolicLinkOffset is where CM_NOTIFY_EVENT_DATA.u.DeviceInterface.SymbolicLink
	// starts: FilterType (4) + Reserved (4) + ClassGuid (16).
	symbolicLinkOffset = 24
)

// cmNotifyFilter mirrors CM_NOTIFY_FILTER for a device-interface filter. The
// union's largest member is WCHAR InstanceId[200], so it spans 400 bytes.
type cmNotifyFilter struct {
	cbSize     uint32
	flags      uint32
	filterType uint32
	reserved   uint32
	classGUID  windows.GUID
	_          [400 - 16]byte
}

// source routes callbacks for one registration to its watcher.
type source struct {
	vendorID uint16
	signal   func()
}

var (
	// syscall.NewCallback slots are limited and never freed, so a single
	// callback serves every registration, keyed by the context value.
	callbackOnce sync.Once
	callbackPtr  uintptr

	sourcesMu sync.Mutex
	sources   = make(map[uintptr]*source)
	nextID    uintptr
)

// notificationCallback is the CM_NOTIFY_CALLBACK; it runs on a system thread
// pool thread.
func notificationCallback(_ uintptr, context uintptr, action uintptr, eventData unsafe.Pointer, eventDataSize uintptr) uintptr {
	if action != cmNotifyActionDeviceInterfaceArrival && action != cmNotifyActionDeviceInterfaceRemoval {
		return 0
	}
	sourcesMu.Lock()
	s := sources[context]
	sourcesMu.Unlock()
	if s != nil && interfaceLinkMatches(symbolicLink(eventData, eventDataSize), s.vendorID) {
		s.signal()
	}
	return 0 // ERROR_SUCCESS
}

// symbolicLink extracts the device interface path from CM_NOTIFY_EVENT_DATA.
func symbolicLink(eventData unsafe.Pointer, size uintptr) string {
	if eventData == nil || size <= symbolicLinkOffset {
		return ""
	}
	chars := unsafe.Slice((*uint16)(unsafe.Add(eventData, symbolicLinkOffset)), (size-symbolicLinkOffset)/2)
	return windows.UTF16ToString(chars)
}

// startSource registers for HID device interface arrival and removal
// notifications (Windows 8+). They need no window or message loop.
func startSource(vendorID uint16, signal func()) (func(), error) {
	if err := procCMRegisterNotification.Find(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	callbackOnce.Do(func() {
		callbackPtr = syscall.NewCallback(notificationCallback)
	})

	sourcesMu.Lock()
	nextID++
	id := nextID
	sources[id] = &source{vendorID: vendorID, signal: signal}
	sourcesMu.Unlock()

	filter := cmNotifyFilter{
		filterType: cmNotifyFilterTypeDeviceInterface,
		classGUID:  hidInterfaceGUID,
	}
	filter.cbSize = uint32(unsafe.Sizeof(filter))

	var handle uintptr
	r, _, _ := procCMRegisterNotification.Call(
		uintptr(unsafe.Pointer(&filter)),
		id,
		callbackPtr,
		uintptr(unsafe.Pointer(&handle)),
	)
	if r != crSuccess {
		removeSource(id)
		return nil, fmt.Errorf("CM_Register_Notification failed: CONFIGRET 0x%X", r)
	}

	return func() {
		// No callbacks run after CM_Unregister_Notification returns.
		_, _, _ = procCMUnregisterNotification.Call(handle)
		removeSource(id)
	}, nil
}

func removeSource(id uintptr) {
	sourcesMu.Lock()
	delete(sources, id)
	sourcesMu.Unlock()
}
