package driver

import (
	"bytes"
	"testing"
)

func TestApexProtocol_Variants(t *testing.T) {
	tests := []struct {
		name        string
		protocol    *ApexProtocol
		wantCommand []byte
		wantFamily  string
		wantPadding bool
		wantFeature bool // strict feature-report transport
	}{
		{"legacy (zero value)", &ApexProtocol{}, []byte{0x61}, "Apex Keyboard", false, false},
		{"framebuffer", NewApexFramebufferProtocol(), []byte{0x1F, 0x81}, "Apex Keyboard (framebuffer command)", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.protocol.command(); !bytes.Equal(got, tt.wantCommand) {
				t.Errorf("command() = % X, want % X", got, tt.wantCommand)
			}
			if got := tt.protocol.DeviceFamily(); got != tt.wantFamily {
				t.Errorf("DeviceFamily() = %q, want %q", got, tt.wantFamily)
			}
			if got := tt.protocol.Interface(); got != "mi_01" {
				t.Errorf("Interface() = %q, want %q", got, "mi_01")
			}
			if got := protocolWantsReportLengthPadding(tt.protocol); got != tt.wantPadding {
				t.Errorf("protocolWantsReportLengthPadding() = %v, want %v", got, tt.wantPadding)
			}
			if got := protocolRequiresFeatureReport(tt.protocol); got != tt.wantFeature {
				t.Errorf("protocolRequiresFeatureReport() = %v, want %v", got, tt.wantFeature)
			}
			if protocolDetectsScreenInterface(tt.protocol) {
				t.Error("Apex protocols must not detect the screen interface by capability")
			}
		})
	}
}

func TestResolveProtocol_ApexVariants(t *testing.T) {
	tests := []struct {
		name            string
		pid             uint16
		wantFramebuffer bool
	}{
		{"Apex 7 uses the legacy command", 0x1612, false},
		{"Apex Pro TKL (2023) PID 0x1632 uses the legacy command", 0x1632, false},
		{"Apex Pro TKL 2023 PID 0x1628 uses the framebuffer command", 0x1628, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := resolveProtocol(SteelSeriesVID, tt.pid).(*ApexProtocol)
			if !ok {
				t.Fatalf("resolveProtocol(0x%04X) = %T, want *ApexProtocol", tt.pid, p)
			}
			if p.framebuffer != tt.wantFramebuffer {
				t.Errorf("resolveProtocol(0x%04X).framebuffer = %v, want %v", tt.pid, p.framebuffer, tt.wantFramebuffer)
			}
		})
	}
}

func TestProtocolRequiresFeatureReport_NovaProDefaultsFalse(t *testing.T) {
	if protocolRequiresFeatureReport(&NovaProProtocol{}) {
		t.Error("NovaProProtocol should keep the default transport")
	}
}

func TestApexProtocol_BuildReturnToUIPacket(t *testing.T) {
	t.Run("legacy has no return-to-UI command", func(t *testing.T) {
		p := &ApexProtocol{}
		if packet := p.BuildReturnToUIPacket(); packet != nil {
			t.Errorf("BuildReturnToUIPacket() = % X, want nil", packet)
		}
		if protocolReturnsToUIWithOutputReport(p) {
			t.Error("legacy variant should not use an output report")
		}
	})

	t.Run("framebuffer sends 1F 82 as a 65-byte output report", func(t *testing.T) {
		p := NewApexFramebufferProtocol()
		packet := p.BuildReturnToUIPacket()
		if len(packet) != 65 {
			t.Fatalf("len = %d, want 65", len(packet))
		}
		want := make([]byte, 65)
		want[1], want[2] = 0x1F, 0x82
		if !bytes.Equal(packet, want) {
			t.Errorf("packet = % X, want % X", packet, want)
		}
		if !protocolReturnsToUIWithOutputReport(p) {
			t.Error("framebuffer variant should return to UI with an output report")
		}
	})
}
