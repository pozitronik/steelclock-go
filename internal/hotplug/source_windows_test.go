//go:build windows

package hotplug

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestCMNotifyFilterSize(t *testing.T) {
	// sizeof(CM_NOTIFY_FILTER) on both 32- and 64-bit Windows.
	if got := unsafe.Sizeof(cmNotifyFilter{}); got != 416 {
		t.Errorf("sizeof(cmNotifyFilter) = %d, want 416", got)
	}
}

func TestSymbolicLink(t *testing.T) {
	link := `\\?\HID#VID_1038&PID_1612&MI_01#7&2a8e1b0&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`
	chars, err := windows.UTF16FromString(link)
	if err != nil {
		t.Fatal(err)
	}
	// CM_NOTIFY_EVENT_DATA: FilterType, Reserved, ClassGuid, then the link.
	buf := make([]byte, symbolicLinkOffset+2*len(chars))
	for i, c := range chars {
		buf[symbolicLinkOffset+2*i] = byte(c)
		buf[symbolicLinkOffset+2*i+1] = byte(c >> 8)
	}

	if got := symbolicLink(unsafe.Pointer(&buf[0]), uintptr(len(buf))); got != link {
		t.Errorf("symbolicLink() = %q, want %q", got, link)
	}
	if got := symbolicLink(nil, 100); got != "" {
		t.Errorf("symbolicLink(nil) = %q, want empty", got)
	}
	if got := symbolicLink(unsafe.Pointer(&buf[0]), symbolicLinkOffset); got != "" {
		t.Errorf("symbolicLink() without a link = %q, want empty", got)
	}
}

func TestNotificationCallback_SignalsOnlyForVendor(t *testing.T) {
	signalled := 0
	sourcesMu.Lock()
	sources[999] = &source{vendorID: 0x1038, signal: func() { signalled++ }}
	sourcesMu.Unlock()
	t.Cleanup(func() { removeSource(999) })

	event := func(link string) ([]byte, uintptr) {
		chars, _ := windows.UTF16FromString(link)
		buf := make([]byte, symbolicLinkOffset+2*len(chars))
		for i, c := range chars {
			buf[symbolicLinkOffset+2*i] = byte(c)
			buf[symbolicLinkOffset+2*i+1] = byte(c >> 8)
		}
		return buf, uintptr(len(buf))
	}

	ss, ssLen := event(`\\?\HID#VID_1038&PID_1612&MI_01#x`)
	other, otherLen := event(`\\?\HID#VID_046D&PID_C52B&MI_02#x`)

	notificationCallback(0, 999, cmNotifyActionDeviceInterfaceArrival, unsafe.Pointer(&ss[0]), ssLen)
	notificationCallback(0, 999, cmNotifyActionDeviceInterfaceRemoval, unsafe.Pointer(&ss[0]), ssLen)
	notificationCallback(0, 999, cmNotifyActionDeviceInterfaceArrival, unsafe.Pointer(&other[0]), otherLen)
	notificationCallback(0, 999, 2 /* query remove */, unsafe.Pointer(&ss[0]), ssLen)
	notificationCallback(0, 12345 /* unknown source */, cmNotifyActionDeviceInterfaceArrival, unsafe.Pointer(&ss[0]), ssLen)

	if signalled != 2 {
		t.Errorf("signalled %d time(s), want 2 (SteelSeries arrival and removal)", signalled)
	}
}

func TestStart_RegistersAndUnregisters(t *testing.T) {
	w, err := Start(0x1038, func() {})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	w.Stop()

	sourcesMu.Lock()
	n := len(sources)
	sourcesMu.Unlock()
	if n != 0 {
		t.Errorf("%d source(s) left registered after Stop", n)
	}
}
