//go:build darwin

package driver

import (
	"testing"
)

// Darwin packet format: ReportID(1) + CMD(1) + Data(640) + Padding(1) = 643 bytes
// sendHIDFeatureReport strips ReportID; device receives CMD(1) + Data + Padding = 642 bytes
const (
	testPacketSize = 643 // Total packet including Report ID
	testCmdOffset  = 1
	testDataOffset = 2
)

func TestBuildApexPacket_Size(t *testing.T) {
	pixelData := make([]byte, 640) // 128x40/8 = 640 bytes

	packet := buildApexPacket(pixelData, 128, 40)

	if len(packet) != testPacketSize {
		t.Errorf("buildApexPacket() size = %d, want %d", len(packet), testPacketSize)
	}
}

func TestBuildApexPacket_ReportID(t *testing.T) {
	packet := buildApexPacket(make([]byte, 640), 128, 40)

	if packet[0] != 0x00 {
		t.Errorf("packet[0] (ReportID) = 0x%02X, want 0x00", packet[0])
	}
}

func TestBuildApexPacket_Command(t *testing.T) {
	packet := buildApexPacket(make([]byte, 640), 128, 40)

	if packet[testCmdOffset] != 0x61 {
		t.Errorf("packet[%d] (CMD) = 0x%02X, want 0x61", testCmdOffset, packet[testCmdOffset])
	}
}

func TestBuildApexPacket_DataCopy(t *testing.T) {
	pixelData := make([]byte, 640)
	for i := range pixelData {
		pixelData[i] = byte(i % 256)
	}

	packet := buildApexPacket(pixelData, 128, 40)

	for i := 0; i < len(pixelData); i++ {
		if packet[testDataOffset+i] != pixelData[i] {
			t.Errorf("packet[%d] = 0x%02X, want 0x%02X", testDataOffset+i, packet[testDataOffset+i], pixelData[i])
			break
		}
	}
}

func TestBuildApexPacket_ShortData(t *testing.T) {
	pixelData := make([]byte, 100)
	for i := range pixelData {
		pixelData[i] = 0xFF
	}

	packet := buildApexPacket(pixelData, 128, 40)

	for i := 0; i < 100; i++ {
		if packet[testDataOffset+i] != 0xFF {
			t.Errorf("packet[%d] = 0x%02X, want 0xFF", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}

	for i := 100; i < 640; i++ {
		if packet[testDataOffset+i] != 0x00 {
			t.Errorf("packet[%d] = 0x%02X, want 0x00 (padding)", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}
}

func TestBuildApexPacket_LongData(t *testing.T) {
	pixelData := make([]byte, 1000)
	for i := range pixelData {
		pixelData[i] = 0xAA
	}

	packet := buildApexPacket(pixelData, 128, 40)

	if len(packet) != testPacketSize {
		t.Errorf("buildApexPacket() size with long data = %d, want %d", len(packet), testPacketSize)
	}

	for i := 0; i < 640; i++ {
		if packet[testDataOffset+i] != 0xAA {
			t.Errorf("packet[%d] = 0x%02X, want 0xAA", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}

	if packet[642] != 0x00 {
		t.Errorf("packet[642] (trailing padding) = 0x%02X, want 0x00", packet[642])
	}
}

func TestBuildApexPacket_DifferentResolution(t *testing.T) {
	width := 256
	height := 64
	dataSize := width * height / 8 // 2048 bytes

	pixelData := make([]byte, dataSize)
	packet := buildApexPacket(pixelData, width, height)

	expected := 1 + 1 + dataSize + 1 // ReportID + CMD + Data + Padding
	if len(packet) != expected {
		t.Errorf("buildApexPacket() size for %dx%d = %d, want %d", width, height, len(packet), expected)
	}
}

func BenchmarkBuildApexPacket(b *testing.B) {
	pixelData := make([]byte, 640)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildApexPacket(pixelData, 128, 40)
	}
}

func BenchmarkBuildApexPacket_LargeData(b *testing.B) {
	pixelData := make([]byte, 2048)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildApexPacket(pixelData, 256, 64)
	}
}
