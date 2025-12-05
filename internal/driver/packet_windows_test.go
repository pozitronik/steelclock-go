//go:build windows

package driver

import (
	"testing"
)

// Windows packet format: ReportID(1) + CMD(1) + Data(640) + Padding(1) = 643 bytes
// After Windows HID driver strips ReportID, device receives 642 bytes (matching HID descriptor)
const (
	testPacketSize = 643 // Total packet including Report ID
	testCmdOffset  = 1
	testDataOffset = 2
)

func TestBuildPacket_Size(t *testing.T) {
	pixelData := make([]byte, 640) // 128x40/8 = 640 bytes

	packet := buildPacket(pixelData, 128, 40)

	// Expected: ReportID(1) + CMD(1) + Data(640) + Padding(1) = 643 bytes
	if len(packet) != testPacketSize {
		t.Errorf("buildPacket() size = %d, want %d", len(packet), testPacketSize)
	}
}

func TestBuildPacket_ReportID(t *testing.T) {
	packet := buildPacket(make([]byte, 640), 128, 40)

	if packet[0] != 0x00 {
		t.Errorf("packet[0] (ReportID) = 0x%02X, want 0x00", packet[0])
	}
}

func TestBuildPacket_Command(t *testing.T) {
	packet := buildPacket(make([]byte, 640), 128, 40)

	if packet[testCmdOffset] != 0x61 {
		t.Errorf("packet[%d] (CMD) = 0x%02X, want 0x61", testCmdOffset, packet[testCmdOffset])
	}
}

func TestBuildPacket_DataCopy(t *testing.T) {
	pixelData := make([]byte, 640)
	for i := range pixelData {
		pixelData[i] = byte(i % 256)
	}

	packet := buildPacket(pixelData, 128, 40)

	// Check that pixel data is copied correctly starting at byte 2
	for i := 0; i < len(pixelData); i++ {
		if packet[testDataOffset+i] != pixelData[i] {
			t.Errorf("packet[%d] = 0x%02X, want 0x%02X", testDataOffset+i, packet[testDataOffset+i], pixelData[i])
			break
		}
	}
}

func TestBuildPacket_ShortData(t *testing.T) {
	// Less data than expected - should be padded with zeros
	pixelData := make([]byte, 100)
	for i := range pixelData {
		pixelData[i] = 0xFF
	}

	packet := buildPacket(pixelData, 128, 40)

	// First 100 bytes of data should be 0xFF
	for i := 0; i < 100; i++ {
		if packet[testDataOffset+i] != 0xFF {
			t.Errorf("packet[%d] = 0x%02X, want 0xFF", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}

	// Remaining bytes should be 0x00 (default)
	for i := 100; i < 640; i++ {
		if packet[testDataOffset+i] != 0x00 {
			t.Errorf("packet[%d] = 0x%02X, want 0x00 (padding)", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}
}

func TestBuildPacket_LongData(t *testing.T) {
	// More data than expected - should be truncated
	pixelData := make([]byte, 1000)
	for i := range pixelData {
		pixelData[i] = 0xAA
	}

	packet := buildPacket(pixelData, 128, 40)

	// Packet should still be correct size (643 for 128x40)
	if len(packet) != testPacketSize {
		t.Errorf("buildPacket() size with long data = %d, want %d", len(packet), testPacketSize)
	}

	// Data should be truncated to 640 bytes
	for i := 0; i < 640; i++ {
		if packet[testDataOffset+i] != 0xAA {
			t.Errorf("packet[%d] = 0x%02X, want 0xAA", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}

	// Trailing padding byte should be zero
	if packet[642] != 0x00 {
		t.Errorf("packet[642] (trailing padding) = 0x%02X, want 0x00", packet[642])
	}
}

func TestBuildPacket_DifferentResolution(t *testing.T) {
	// Test with a hypothetical 256x64 display
	width := 256
	height := 64
	dataSize := width * height / 8 // 2048 bytes

	pixelData := make([]byte, dataSize)
	packet := buildPacket(pixelData, width, height)

	// ReportID(1) + CMD(1) + Data + Padding(1) = 1 + 642 base, but data varies
	// For 256x64: 1 + 1 + 2048 + 1 = 2051 bytes
	expected := 1 + 1 + dataSize + 1
	if len(packet) != expected {
		t.Errorf("buildPacket() size for %dx%d = %d, want %d", width, height, len(packet), expected)
	}
}

func BenchmarkBuildPacket(b *testing.B) {
	pixelData := make([]byte, 640)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildPacket(pixelData, 128, 40)
	}
}

func BenchmarkBuildPacket_LargeData(b *testing.B) {
	pixelData := make([]byte, 2048) // 256x64 display

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildPacket(pixelData, 256, 64)
	}
}
