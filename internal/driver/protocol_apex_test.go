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
	}{
		{"legacy (zero value)", &ApexProtocol{}, []byte{0x61}, "Apex Keyboard", false},
		{"framebuffer", NewApexFramebufferProtocol(), []byte{0x1F, 0x81}, "Apex Keyboard (framebuffer command)", true},
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
