//go:build linux

package driver

import (
	"testing"
)

// Linux packet format: CMD(1) + Data(640) + Padding = 642 fixed
const (
	testPacketSize   = 642
	testCmdOffset    = 0
	testDataOffset   = 1
	testHasReportID  = false
	testPaddingStart = 641 // After data
	testPaddingEnd   = 642
)

func TestBuildPacket_Size(t *testing.T) {
	pixelData := make([]byte, 640) // 128x40/8 = 640 bytes

	packet := buildPacket(pixelData, 128, 40)

	// Expected: 642 bytes fixed (matching HID descriptor)
	if len(packet) != testPacketSize {
		t.Errorf("buildPacket() size = %d, want %d", len(packet), testPacketSize)
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

	// On Linux: byte 641 should be zero padding (after data)
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

	// Check that pixel data is copied correctly
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

	// Packet should still be correct size (642 fixed)
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
}

func TestBuildPacket_DifferentResolution(t *testing.T) {
	// Test with a hypothetical 256x64 display
	// Note: On Linux, packet size is fixed at 642 bytes to match HID descriptor
	// This test verifies data is still copied correctly even with different resolution
	width := 256
	height := 64
	dataSize := width * height / 8 // 2048 bytes

	pixelData := make([]byte, dataSize)
	for i := range pixelData {
		pixelData[i] = 0xBB
	}
	packet := buildPacket(pixelData, width, height)

	// Packet size is fixed at 642 on Linux
	if len(packet) != testPacketSize {
		t.Errorf("buildPacket() size for %dx%d = %d, want %d", width, height, len(packet), testPacketSize)
	}

	// Data should be truncated to fit (only first 641 bytes after CMD)
	maxData := testPacketSize - 1 // 641 bytes max for data
	if dataSize > maxData {
		dataSize = maxData
	}
	for i := 0; i < dataSize; i++ {
		if packet[testDataOffset+i] != 0xBB {
			t.Errorf("packet[%d] = 0x%02X, want 0xBB", testDataOffset+i, packet[testDataOffset+i])
			break
		}
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
