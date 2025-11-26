package driver

import (
	"testing"
)

//goland:noinspection GoBoolExpressions
func TestSteelSeriesVID(t *testing.T) {
	// SteelSeries VID should be 0x1038
	if SteelSeriesVID != 0x1038 {
		t.Errorf("SteelSeriesVID = 0x%04X, want 0x1038", SteelSeriesVID)
	}
}

func TestKnownDevices_NotEmpty(t *testing.T) {
	if len(KnownDevices) == 0 {
		t.Error("KnownDevices should not be empty")
	}
}

func TestKnownDevices_AllHaveSteelSeriesVID(t *testing.T) {
	for _, device := range KnownDevices {
		if device.VID != SteelSeriesVID {
			t.Errorf("device %s has VID 0x%04X, want 0x%04X (SteelSeriesVID)",
				device.Name, device.VID, SteelSeriesVID)
		}
	}
}

func TestKnownDevices_AllHaveValidPID(t *testing.T) {
	for _, device := range KnownDevices {
		if device.PID == 0 {
			t.Errorf("device %s has invalid PID 0x0000", device.Name)
		}
	}
}

func TestKnownDevices_AllHaveNames(t *testing.T) {
	for i, device := range KnownDevices {
		if device.Name == "" {
			t.Errorf("KnownDevices[%d] has empty name (PID 0x%04X)", i, device.PID)
		}
	}
}

func TestKnownDevices_AllHaveValidDisplaySize(t *testing.T) {
	for _, device := range KnownDevices {
		if device.DisplaySize.Width <= 0 {
			t.Errorf("device %s has invalid display width %d", device.Name, device.DisplaySize.Width)
		}
		if device.DisplaySize.Height <= 0 {
			t.Errorf("device %s has invalid display height %d", device.Name, device.DisplaySize.Height)
		}
	}
}

func TestKnownDevices_DisplaySize128x40(t *testing.T) {
	// All known Apex keyboards have 128x40 displays
	for _, device := range KnownDevices {
		if device.DisplaySize.Width != 128 {
			t.Errorf("device %s has display width %d, expected 128",
				device.Name, device.DisplaySize.Width)
		}
		if device.DisplaySize.Height != 40 {
			t.Errorf("device %s has display height %d, expected 40",
				device.Name, device.DisplaySize.Height)
		}
	}
}

func TestKnownDevices_UniquePIDs(t *testing.T) {
	seen := make(map[uint16]string)
	for _, device := range KnownDevices {
		if existing, ok := seen[device.PID]; ok {
			t.Errorf("duplicate PID 0x%04X: %s and %s", device.PID, existing, device.Name)
		}
		seen[device.PID] = device.Name
	}
}

func TestKnownDevices_ContainsExpectedDevices(t *testing.T) {
	// Verify some expected devices are present
	expectedDevices := map[uint16]string{
		0x1612: "Apex 7",
		0x1618: "Apex 7 TKL",
		0x1610: "Apex Pro",
		0x1614: "Apex Pro TKL",
		0x161C: "Apex 5",
		0x1630: "Apex Pro (2023)",
		0x1632: "Apex Pro TKL (2023)",
	}

	deviceByPID := make(map[uint16]KnownDevice)
	for _, device := range KnownDevices {
		deviceByPID[device.PID] = device
	}

	for pid, expectedName := range expectedDevices {
		device, ok := deviceByPID[pid]
		if !ok {
			t.Errorf("expected device PID 0x%04X (%s) not found in KnownDevices", pid, expectedName)
			continue
		}
		if device.Name != expectedName {
			t.Errorf("device PID 0x%04X has name %q, expected %q", pid, device.Name, expectedName)
		}
	}
}
