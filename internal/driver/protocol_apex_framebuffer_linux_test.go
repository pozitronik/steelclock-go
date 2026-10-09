//go:build linux

package driver

import (
	"testing"
)

// Linux framebuffer packet format (mirrors the community reference):
// ReportID(1) + CMD(2: 1F 81) + Data(640) + Padding(1) = 644 bytes fixed.
const (
	fbTestPacketSize = 644
	fbTestDataOffset = 3
)

func TestBuildApexPacket_Framebuffer_Size(t *testing.T) {
	packet := buildApexPacket(NewApexFramebufferProtocol(), make([]byte, 640), 128, 40)

	if len(packet) != fbTestPacketSize {
		t.Errorf("buildApexPacket() size = %d, want %d", len(packet), fbTestPacketSize)
	}
}

func TestBuildApexPacket_Framebuffer_Header(t *testing.T) {
	packet := buildApexPacket(NewApexFramebufferProtocol(), make([]byte, 640), 128, 40)

	if packet[0] != 0x00 {
		t.Errorf("packet[0] (ReportID) = 0x%02X, want 0x00", packet[0])
	}
	if packet[1] != 0x1F || packet[2] != 0x81 {
		t.Errorf("packet[1:3] (CMD) = 0x%02X 0x%02X, want 0x1F 0x81", packet[1], packet[2])
	}
}

func TestBuildApexPacket_Framebuffer_Data(t *testing.T) {
	tests := []struct {
		name     string
		dataLen  int
		fill     byte
		wantData int // bytes expected to carry fill; the rest must be zero
	}{
		{"exact", 640, 0x5A, 640},
		{"short data is zero-padded", 100, 0xFF, 100},
		{"long data is truncated", 1000, 0xAA, 640},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixelData := make([]byte, tt.dataLen)
			for i := range pixelData {
				pixelData[i] = tt.fill
			}

			packet := buildApexPacket(NewApexFramebufferProtocol(), pixelData, 128, 40)

			if len(packet) != fbTestPacketSize {
				t.Fatalf("size = %d, want %d", len(packet), fbTestPacketSize)
			}
			for i := 0; i < 640; i++ {
				want := byte(0)
				if i < tt.wantData {
					want = tt.fill
				}
				if packet[fbTestDataOffset+i] != want {
					t.Fatalf("packet[%d] = 0x%02X, want 0x%02X", fbTestDataOffset+i, packet[fbTestDataOffset+i], want)
				}
			}
			if packet[fbTestPacketSize-1] != 0x00 {
				t.Errorf("trailing byte = 0x%02X, want 0x00", packet[fbTestPacketSize-1])
			}
		})
	}
}
