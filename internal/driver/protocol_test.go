package driver

import (
	"testing"
)

func TestApexProtocol_Interface(t *testing.T) {
	p := &ApexProtocol{}
	if p.Interface() != "mi_01" {
		t.Errorf("Interface() = %q, want %q", p.Interface(), "mi_01")
	}
}

func TestApexProtocol_DeviceFamily(t *testing.T) {
	p := &ApexProtocol{}
	if p.DeviceFamily() != "Apex Keyboard" {
		t.Errorf("DeviceFamily() = %q, want %q", p.DeviceFamily(), "Apex Keyboard")
	}
}

func TestApexProtocol_ImplementsProtocol(t *testing.T) {
	var _ Protocol = (*ApexProtocol)(nil)
}

func TestResolveProtocol_UnknownDevice(t *testing.T) {
	p := resolveProtocol(0xFFFF, 0xFFFF)
	if _, ok := p.(*ApexProtocol); !ok {
		t.Errorf("resolveProtocol for unknown device should return *ApexProtocol, got %T", p)
	}
}

func TestResolveProtocol_ZeroVIDPID(t *testing.T) {
	p := resolveProtocol(0, 0)
	if _, ok := p.(*ApexProtocol); !ok {
		t.Errorf("resolveProtocol with zero VID/PID should return *ApexProtocol, got %T", p)
	}
}

func TestResolveProtocol_KnownApexDevice(t *testing.T) {
	// Apex 7 (PID 0x1612) has no custom protocol, should get ApexProtocol
	p := resolveProtocol(SteelSeriesVID, 0x1612)
	if _, ok := p.(*ApexProtocol); !ok {
		t.Errorf("resolveProtocol for Apex 7 should return *ApexProtocol, got %T", p)
	}
}

func TestResolveProtocol_AllKnownDevices(t *testing.T) {
	for _, dev := range KnownDevices {
		p := resolveProtocol(dev.VID, dev.PID)
		if p == nil {
			t.Errorf("resolveProtocol(%04X, %04X) returned nil for %s", dev.VID, dev.PID, dev.Name)
		}
	}
}

func TestNewDriver_ProtocolInitialized(t *testing.T) {
	d := NewDriver(Config{})
	if d.protocol == nil {
		t.Error("protocol should be initialized")
	}
	if _, ok := d.protocol.(*ApexProtocol); !ok {
		t.Errorf("default protocol should be *ApexProtocol, got %T", d.protocol)
	}
}

func TestNewDriver_ProtocolFromVIDPID(t *testing.T) {
	// Known Apex device should get ApexProtocol
	d := NewDriver(Config{VID: SteelSeriesVID, PID: 0x1612})
	if _, ok := d.protocol.(*ApexProtocol); !ok {
		t.Errorf("protocol for Apex 7 should be *ApexProtocol, got %T", d.protocol)
	}
}

func TestNewDriver_InterfaceFromProtocol(t *testing.T) {
	// When no interface specified, protocol's default should be used
	d := NewDriver(Config{})
	if d.config.Interface != "mi_01" {
		t.Errorf("default interface from protocol = %q, want %q", d.config.Interface, "mi_01")
	}
}

func TestNewDriver_InterfaceOverridesProtocol(t *testing.T) {
	// Custom interface should override protocol's default
	d := NewDriver(Config{Interface: "mi_04"})
	if d.config.Interface != "mi_04" {
		t.Errorf("custom interface = %q, want %q", d.config.Interface, "mi_04")
	}
}
