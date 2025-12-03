//go:build windows

package driver

import (
	"testing"
)

// Windows packet format: ReportID(1) + CMD(1) + Padding(16) + Data = 18 + dataSize
const (
	testHeaderSize   = 18
	testCmdOffset    = 1
	testDataOffset   = 18
	testPaddingStart = 2
	testPaddingEnd   = 18
)

func TestBuildPacket_Size(t *testing.T) {
	pixelData := make([]byte, 640) // 128x40/8 = 640 bytes

	packet := buildPacket(pixelData, 128, 40)

	// Expected: ReportID(1) + CMD(1) + Padding(16) + Data(640) = 658 bytes on Windows
	expected := testHeaderSize + 640
	if len(packet) != expected {
		t.Errorf("buildPacket() size = %d, want %d", len(packet), expected)
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

func TestBuildPacket_Padding(t *testing.T) {
	packet := buildPacket(make([]byte, 640), 128, 40)

	// Bytes 2-17 should be zero padding
	for i := testPaddingStart; i < testPaddingEnd; i++ {
		if packet[i] != 0x00 {
			t.Errorf("packet[%d] (padding) = 0x%02X, want 0x00", i, packet[i])
		}
	}
}

func TestBuildPacket_DataCopy(t *testing.T) {
	pixelData := make([]byte, 640)
	for i := range pixelData {
		pixelData[i] = byte(i % 256)
	}

	packet := buildPacket(pixelData, 128, 40)

	// Check that pixel data is copied correctly starting at byte 18
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

	// Packet should still be correct size
	expected := testHeaderSize + 640
	if len(packet) != expected {
		t.Errorf("buildPacket() size with long data = %d, want %d", len(packet), expected)
	}

	// Data should be truncated to 640 bytes
	for i := 0; i < 640; i++ {
		if packet[testDataOffset+i] != 0xAA {
			t.Errorf("packet[%d] = 0x%02X, want 0xAA", testDataOffset+i, packet[testDataOffset+i])
			break
		}
	}
}

func TestBuildPacket_DifferentResolution(t *testing.T) {
	// Test with a hypothetical 256x64 display
	width := 256
	height := 64
	dataSize := width * height / 8 // 2048 bytes

	pixelData := make([]byte, dataSize)
	packet := buildPacket(pixelData, width, height)

	expected := testHeaderSize + dataSize
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
