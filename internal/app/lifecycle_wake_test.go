package app

import "testing"

func TestLifecycleManager_WakeDevices(t *testing.T) {
	m := NewLifecycleManager()
	a := NewDeviceInstance("a", m.retryCancel)
	b := NewDeviceInstance("b", m.retryCancel)
	m.devices = []*DeviceInstance{a, b}

	m.WakeDevices()

	for _, d := range []*DeviceInstance{a, b} {
		if len(d.wake) != 1 {
			t.Errorf("device %s: %d pending wake(s), want 1", d.id, len(d.wake))
		}
	}

	m.WakeDevices() // already pending: must not block
}

func TestLifecycleManager_WakeDevices_NoDevices(t *testing.T) {
	NewLifecycleManager().WakeDevices() // must not panic or block
}
