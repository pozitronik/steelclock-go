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

func TestKnownDevices_AllWidth128(t *testing.T) {
	// All known devices have 128px wide displays
	for _, device := range KnownDevices {
		if device.DisplaySize.Width != 128 {
			t.Errorf("device %s has display width %d, expected 128",
				device.Name, device.DisplaySize.Width)
		}
	}
}

func TestKnownDevices_ApexDisplaySize(t *testing.T) {
	// Apex keyboards (nil NewProtocol) have 128x40 displays
	for _, device := range KnownDevices {
		if device.NewProtocol != nil {
			continue // Skip non-Apex devices
		}
		if device.DisplaySize.Height != 40 {
			t.Errorf("Apex device %s has display height %d, expected 40",
				device.Name, device.DisplaySize.Height)
		}
	}
}

func TestKnownDevices_NovaProDisplaySize(t *testing.T) {
	// Nova Pro devices have 128x64 displays
	for _, device := range KnownDevices {
		if device.NewProtocol == nil {
			continue // Skip Apex devices
		}
		if device.DisplaySize.Height != 64 {
			t.Errorf("Nova Pro device %s has display height %d, expected 64",
				device.Name, device.DisplaySize.Height)
		}
	}
}

func TestDeviceInterface(t *testing.T) {
	tests := []struct {
		name string
		dev  KnownDevice
		want string
	}{
		{
			name: "Apex device (nil protocol) defaults to mi_01",
			dev:  KnownDevice{Name: "Apex Pro", NewProtocol: nil},
			want: "mi_01",
		},
		{
			name: "Nova Pro device uses mi_04",
			dev:  KnownDevice{Name: "Nova Pro", NewProtocol: func() Protocol { return &NovaProProtocol{} }},
			want: "mi_04",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := tt.dev
			if got := deviceInterface(&dev); got != tt.want {
				t.Errorf("deviceInterface(%s) = %q, want %q", tt.dev.Name, got, tt.want)
			}
		})
	}
}

func TestKnownDevices_NovaProOmni(t *testing.T) {
	// The Omni must resolve to the Nova Pro protocol on mi_04 at 128x64,
	// otherwise the direct driver would talk the wrong (Apex) protocol to it.
	var omni *KnownDevice
	for i := range KnownDevices {
		if KnownDevices[i].PID == 0x2290 {
			omni = &KnownDevices[i]
			break
		}
	}
	if omni == nil {
		t.Fatal("Arctis Nova Pro Omni (PID 0x2290) not found in KnownDevices")
	}
	if got := deviceInterface(omni); got != "mi_04" {
		t.Errorf("Omni interface = %q, want mi_04", got)
	}
	if _, ok := resolveProtocol(omni.VID, omni.PID).(*NovaProProtocol); !ok {
		t.Errorf("Omni protocol = %T, want *NovaProProtocol", resolveProtocol(omni.VID, omni.PID))
	}
	if omni.DisplaySize.Width != 128 || omni.DisplaySize.Height != 64 {
		t.Errorf("Omni display = %dx%d, want 128x64", omni.DisplaySize.Width, omni.DisplaySize.Height)
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
		// Apex keyboards
		0x1612: "Apex 7",
		0x1618: "Apex 7 TKL",
		0x1610: "Apex Pro",
		0x1614: "Apex Pro TKL",
		0x161C: "Apex 5",
		0x1630: "Apex Pro (2023)",
		0x1632: "Apex Pro TKL (2023)",
		// Nova Pro / GameDAC Gen 2
		0x12cb: "Arctis Nova Pro (Wired)",
		0x12cd: "Arctis Nova Pro Wireless (Base Station)",
		0x12e0: "Arctis Nova Pro Wireless (USB-C Dongle)",
		0x12e5: "Arctis Nova Pro Wireless (Xbox)",
		0x225d: "Arctis Nova 5P (USB-C Dongle)",
		0x2290: "Arctis Nova Pro Omni",
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
