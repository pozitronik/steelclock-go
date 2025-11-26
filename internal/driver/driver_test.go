package driver

import (
	"testing"
)

func TestNewDriver_DefaultInterface(t *testing.T) {
	driver := NewDriver(Config{})

	if driver.config.Interface != "mi_01" {
		t.Errorf("default interface = %q, want %q", driver.config.Interface, "mi_01")
	}
}

func TestNewDriver_DefaultWidth(t *testing.T) {
	driver := NewDriver(Config{})

	if driver.config.Width != 128 {
		t.Errorf("default width = %d, want 128", driver.config.Width)
	}
}

func TestNewDriver_DefaultHeight(t *testing.T) {
	driver := NewDriver(Config{})

	if driver.config.Height != 40 {
		t.Errorf("default height = %d, want 40", driver.config.Height)
	}
}

func TestNewDriver_PreservesCustomInterface(t *testing.T) {
	driver := NewDriver(Config{Interface: "mi_02"})

	if driver.config.Interface != "mi_02" {
		t.Errorf("interface = %q, want %q", driver.config.Interface, "mi_02")
	}
}

func TestNewDriver_PreservesCustomWidth(t *testing.T) {
	driver := NewDriver(Config{Width: 256})

	if driver.config.Width != 256 {
		t.Errorf("width = %d, want 256", driver.config.Width)
	}
}

func TestNewDriver_PreservesCustomHeight(t *testing.T) {
	driver := NewDriver(Config{Height: 64})

	if driver.config.Height != 64 {
		t.Errorf("height = %d, want 64", driver.config.Height)
	}
}

func TestNewDriver_PreservesVIDPID(t *testing.T) {
	driver := NewDriver(Config{
		VID: 0x1038,
		PID: 0x1612,
	})

	if driver.config.VID != 0x1038 {
		t.Errorf("VID = 0x%04X, want 0x1038", driver.config.VID)
	}
	if driver.config.PID != 0x1612 {
		t.Errorf("PID = 0x%04X, want 0x1612", driver.config.PID)
	}
}

func TestNewDriver_InitialState(t *testing.T) {
	driver := NewDriver(Config{})

	if driver.connected {
		t.Error("new driver should not be connected")
	}
	if driver.IsConnected() {
		t.Error("IsConnected() should return false for new driver")
	}
}

func TestHIDDriver_DeviceInfo_Empty(t *testing.T) {
	driver := NewDriver(Config{})

	info := driver.DeviceInfo()
	if info.VID != 0 {
		t.Errorf("DeviceInfo().VID = %d, want 0 for unconnected driver", info.VID)
	}
	if info.PID != 0 {
		t.Errorf("DeviceInfo().PID = %d, want 0 for unconnected driver", info.PID)
	}
	if info.Path != "" {
		t.Errorf("DeviceInfo().Path = %q, want empty for unconnected driver", info.Path)
	}
}

func TestHIDDriver_Close_NotConnected(t *testing.T) {
	driver := NewDriver(Config{})

	// Should not error when closing an unconnected driver
	err := driver.Close()
	if err != nil {
		t.Errorf("Close() on unconnected driver returned error: %v", err)
	}
}

func TestHIDDriver_SendFrame_NotConnected(t *testing.T) {
	driver := NewDriver(Config{})

	err := driver.SendFrame(make([]byte, 640))
	if err == nil {
		t.Error("SendFrame() on unconnected driver should return error")
	}
}

func TestHIDDriver_Reconnect_NoDevice(t *testing.T) {
	// Use a non-existent PID to ensure no device is found
	driver := NewDriver(Config{
		VID: 0x1038,
		PID: 0xFFFF, // Non-existent PID
	})

	// On Unix, this will fail with ErrNotSupported
	// On Windows without matching device, it will fail to find device
	err := driver.Reconnect()
	if err == nil {
		// Very unlikely with fake PID
		t.Log("Reconnect() succeeded unexpectedly with non-existent PID")
	} else {
		t.Logf("Reconnect() failed as expected: %v", err)
	}
}

func TestHIDDriver_Reconnect_AutoDetect(t *testing.T) {
	// Test auto-detect reconnection behavior
	driver := NewDriver(Config{})

	err := driver.Reconnect()
	if err != nil {
		// No device available - expected on Unix or Windows without keyboard
		t.Logf("Reconnect() with auto-detect failed (expected if no device): %v", err)
		return
	}

	// Device found and reconnected
	defer func() { _ = driver.Close() }()

	if !driver.IsConnected() {
		t.Error("Reconnect() succeeded but IsConnected() returns false")
	}

	info := driver.DeviceInfo()
	t.Logf("Reconnected to device: path=%s", info.Path)
}

func TestBuildPacket_Size(t *testing.T) {
	pixelData := make([]byte, 640) // 128x40/8 = 640 bytes

	packet := buildPacket(pixelData, 128, 40)

	// Expected: ReportID(1) + CMD(1) + Padding(16) + Data(640) = 658 bytes
	expected := 18 + 640
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

	if packet[1] != 0x61 {
		t.Errorf("packet[1] (CMD) = 0x%02X, want 0x61", packet[1])
	}
}

func TestBuildPacket_Padding(t *testing.T) {
	packet := buildPacket(make([]byte, 640), 128, 40)

	// Bytes 2-17 should be zero padding
	for i := 2; i < 18; i++ {
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
		if packet[18+i] != pixelData[i] {
			t.Errorf("packet[%d] = 0x%02X, want 0x%02X", 18+i, packet[18+i], pixelData[i])
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
		if packet[18+i] != 0xFF {
			t.Errorf("packet[%d] = 0x%02X, want 0xFF", 18+i, packet[18+i])
			break
		}
	}

	// Remaining bytes should be 0x00 (default)
	for i := 100; i < 640; i++ {
		if packet[18+i] != 0x00 {
			t.Errorf("packet[%d] = 0x%02X, want 0x00 (padding)", 18+i, packet[18+i])
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
	expected := 18 + 640
	if len(packet) != expected {
		t.Errorf("buildPacket() size with long data = %d, want %d", len(packet), expected)
	}

	// Data should be truncated to 640 bytes
	for i := 0; i < 640; i++ {
		if packet[18+i] != 0xAA {
			t.Errorf("packet[%d] = 0x%02X, want 0xAA", 18+i, packet[18+i])
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

	expected := 18 + dataSize
	if len(packet) != expected {
		t.Errorf("buildPacket() size for %dx%d = %d, want %d", width, height, len(packet), expected)
	}
}

func TestConfig_ZeroValues(t *testing.T) {
	cfg := Config{}

	if cfg.VID != 0 {
		t.Errorf("zero Config.VID = %d, want 0", cfg.VID)
	}
	if cfg.PID != 0 {
		t.Errorf("zero Config.PID = %d, want 0", cfg.PID)
	}
	if cfg.Interface != "" {
		t.Errorf("zero Config.Interface = %q, want empty", cfg.Interface)
	}
	if cfg.Width != 0 {
		t.Errorf("zero Config.Width = %d, want 0", cfg.Width)
	}
	if cfg.Height != 0 {
		t.Errorf("zero Config.Height = %d, want 0", cfg.Height)
	}
}

func TestDeviceInfo_Fields(t *testing.T) {
	info := DeviceInfo{
		VID:          0x1038,
		PID:          0x1612,
		Path:         "/dev/hidraw0",
		ProductName:  "Apex 7",
		Manufacturer: "SteelSeries",
		Interface:    "mi_01",
	}

	if info.VID != 0x1038 {
		t.Errorf("DeviceInfo.VID = 0x%04X, want 0x1038", info.VID)
	}
	if info.PID != 0x1612 {
		t.Errorf("DeviceInfo.PID = 0x%04X, want 0x1612", info.PID)
	}
	if info.Path != "/dev/hidraw0" {
		t.Errorf("DeviceInfo.Path = %q, want /dev/hidraw0", info.Path)
	}
	if info.ProductName != "Apex 7" {
		t.Errorf("DeviceInfo.ProductName = %q, want Apex 7", info.ProductName)
	}
	if info.Manufacturer != "SteelSeries" {
		t.Errorf("DeviceInfo.Manufacturer = %q, want SteelSeries", info.Manufacturer)
	}
	if info.Interface != "mi_01" {
		t.Errorf("DeviceInfo.Interface = %q, want mi_01", info.Interface)
	}
}

func TestInvalidHandle(t *testing.T) {
	// InvalidHandle should be a meaningful "invalid" value
	if InvalidHandle == 0 {
		// On Unix, InvalidHandle is -1
		t.Logf("InvalidHandle = %v (note: 0 is also valid as invalid marker on some platforms)", InvalidHandle)
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
