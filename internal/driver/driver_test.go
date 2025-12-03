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

// Packet building tests are in platform-specific files:
// - packet_test_linux.go
// - packet_test_windows.go

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

// Benchmark tests for buildPacket are in platform-specific files:
// - packet_test_linux.go
// - packet_test_windows.go
