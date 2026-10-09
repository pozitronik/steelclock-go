package hotplug

import (
	"sync/atomic"
	"testing"
	"time"
)

const (
	testSettle = 30 * time.Millisecond
	testLate   = 90 * time.Millisecond
)

func TestWatcher_DebouncesBurstIntoTwoNotifications(t *testing.T) {
	var calls atomic.Int32
	w := newWatcher(func() { calls.Add(1) }, testSettle, testLate)
	defer w.stopDebouncer()

	for i := 0; i < 10; i++ { // one device announcing several interfaces
		w.signal()
		time.Sleep(time.Millisecond)
	}

	time.Sleep(testSettle + 20*time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("after the burst settled: %d notification(s), want 1", got)
	}
	time.Sleep(testLate + 20*time.Millisecond)
	if got := calls.Load(); got != 2 {
		t.Fatalf("after the late recheck: %d notification(s), want 2", got)
	}
	time.Sleep(testLate + 20*time.Millisecond)
	if got := calls.Load(); got != 2 {
		t.Errorf("without new events: %d notification(s), want still 2", got)
	}
}

func TestWatcher_NewEventRestartsSettle(t *testing.T) {
	var calls atomic.Int32
	w := newWatcher(func() { calls.Add(1) }, testSettle, testLate)
	defer w.stopDebouncer()

	w.signal()
	time.Sleep(testSettle / 2)
	w.signal() // more interfaces arriving
	time.Sleep(testSettle / 2)
	if got := calls.Load(); got != 0 {
		t.Errorf("notified %d time(s) before events settled", got)
	}
}

func TestWatcher_StopPreventsNotifications(t *testing.T) {
	var calls atomic.Int32
	w := newWatcher(func() { calls.Add(1) }, testSettle, testLate)

	w.signal()
	w.Stop()
	time.Sleep(testSettle + testLate + 40*time.Millisecond)

	if got := calls.Load(); got != 0 {
		t.Errorf("notified %d time(s) after Stop", got)
	}
}

func TestWatcher_SignalNeverBlocks(t *testing.T) {
	w := newWatcher(func() {}, time.Hour, time.Hour)
	defer w.stopDebouncer()
	for i := 0; i < 1000; i++ {
		w.signal()
	}
}

func TestInterfaceLinkMatches(t *testing.T) {
	tests := []struct {
		name string
		link string
		want bool
	}{
		{"SteelSeries keyboard", `\\?\HID#VID_1038&PID_1612&MI_01#7&2a8e1b0&0&0000#{4d1e55b2-f16f-11cf-88cb-001111000030}`, true},
		{"lower case", `\\?\hid#vid_1038&pid_12cb&mi_04#8&1#{4d1e55b2-f16f-11cf-88cb-001111000030}`, true},
		{"other vendor", `\\?\HID#VID_046D&PID_C52B&MI_02#7&1#{4d1e55b2-f16f-11cf-88cb-001111000030}`, false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interfaceLinkMatches(tt.link, 0x1038); got != tt.want {
				t.Errorf("interfaceLinkMatches(%q) = %v, want %v", tt.link, got, tt.want)
			}
		})
	}
}

func uevent(fields ...string) []byte {
	msg := fields[0]
	for _, f := range fields[1:] {
		msg += "\x00" + f
	}
	return []byte(msg + "\x00")
}

func TestUeventMatches(t *testing.T) {
	const ssPath = "/devices/pci0000:00/0000:00:14.0/usb1/1-2/1-2:1.1/0003:1038:1612.0005/hidraw/hidraw3"
	tests := []struct {
		name string
		msg  []byte
		want bool
	}{
		{"SteelSeries hidraw added", uevent("add@"+ssPath, "ACTION=add", "DEVPATH="+ssPath, "SUBSYSTEM=hidraw", "DEVNAME=hidraw3"), true},
		{"SteelSeries hidraw removed", uevent("remove@"+ssPath, "ACTION=remove", "DEVPATH="+ssPath, "SUBSYSTEM=hidraw"), true},
		{"other vendor", uevent("add@/x", "ACTION=add", "DEVPATH=/devices/usb1/1-3/1-3:1.0/0003:046D:C52B.0001/hidraw/hidraw0", "SUBSYSTEM=hidraw"), false},
		{"not a hidraw node", uevent("add@/x", "ACTION=add", "DEVPATH=/devices/usb1/1-2/1-2:1.1/0003:1038:1612.0005", "SUBSYSTEM=hid", "HID_ID=0003:00001038:00001612"), false},
		{"other action", uevent("change@"+ssPath, "ACTION=change", "DEVPATH="+ssPath, "SUBSYSTEM=hidraw"), false},
		{"garbage", []byte("libudev\x00\xfe\xed"), false},
		{"empty", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ueventMatches(tt.msg, 0x1038); got != tt.want {
				t.Errorf("ueventMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
