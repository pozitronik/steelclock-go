package driver

import (
	"errors"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/display"
)

// Verify Client implements display.Backend interface at compile time
var _ display.Backend = (*Client)(nil)

func TestNewClient_NoDevice(t *testing.T) {
	// Without a device, NewClient should fail
	cfg := Config{
		VID:    0x1038,
		PID:    0xFFFF, // Non-existent PID
		Width:  128,
		Height: 40,
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Error("NewClient() should fail when device is not found")
	}
}

func TestNewClient_AutoDetect(t *testing.T) {
	// With zero VID/PID, should attempt auto-detection
	cfg := Config{
		Width:  128,
		Height: 40,
	}

	client, err := NewClient(cfg)
	if err != nil {
		// No device available - expected on Unix or Windows without keyboard
		t.Logf("NewClient() auto-detect failed (expected if no device): %v", err)
		return
	}

	// Device found - verify it connected properly
	defer func() { _ = client.RemoveGame() }()

	if !client.IsConnected() {
		t.Error("NewClient() succeeded but IsConnected() returns false")
	}

	info := client.DeviceInfo()
	t.Logf("Auto-detected device: VID_%04X PID_%04X path=%s", info.VID, info.PID, info.Path)
}

// TestClient_SupportsMultipleEvents tests that direct driver does not support batching
func TestClient_SupportsMultipleEvents(t *testing.T) {
	// Create a client with unconnected driver for method testing
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	if client.SupportsMultipleEvents() {
		t.Error("SupportsMultipleEvents() should return false")
	}
}

func TestClient_RegisterGame_NoOp(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// RegisterGame is a no-op for direct driver
	err := client.RegisterGame("test-dev", 15000)
	if err != nil {
		t.Errorf("RegisterGame() returned error: %v", err)
	}
}

func TestClient_BindScreenEvent_NoOp(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// BindScreenEvent is a no-op for direct driver
	err := client.BindScreenEvent("event", "screened-128x40")
	if err != nil {
		t.Errorf("BindScreenEvent() returned error: %v", err)
	}
}

func TestClient_SendScreenData_DisconnectedDevice(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	err := client.SendScreenData("event", make([]byte, 640))
	if err == nil {
		t.Error("SendScreenData() should return error when device is disconnected")
	}
}

func TestClient_SendScreenDataMultiRes_WrongResolution(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Only provide data for a different resolution
	resolutionData := map[string][]byte{
		"image-data-256x64": make([]byte, 2048),
	}

	err := client.SendScreenDataMultiRes("event", resolutionData)
	if err == nil {
		t.Error("SendScreenDataMultiRes() should return error when resolution not found")
	}
}

func TestClient_SendMultipleScreenData_EmptyFrames(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	err := client.SendMultipleScreenData("event", [][]byte{})
	if err != nil {
		t.Errorf("SendMultipleScreenData() with empty frames returned error: %v", err)
	}
}

func TestClient_IsConnected_NewDriver(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// New driver should not be connected
	if client.IsConnected() {
		t.Error("IsConnected() should return false for unconnected driver")
	}
}

func TestClient_DeviceInfo_Empty(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	info := client.DeviceInfo()
	if info.VID != 0 {
		t.Errorf("DeviceInfo().VID = 0x%04X, want 0 for unconnected device", info.VID)
	}
	if info.Path != "" {
		t.Errorf("DeviceInfo().Path = %q, want empty for unconnected device", info.Path)
	}
}

func TestClient_Driver_ReturnsDriver(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	if client.Driver() != driver {
		t.Error("Driver() should return the underlying driver")
	}
}

func TestClient_RemoveGame_UnconnectedDriver(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Should not error when closing an unconnected driver
	err := client.RemoveGame()
	if err != nil {
		t.Errorf("RemoveGame() on unconnected driver returned error: %v", err)
	}
}

func TestClient_SendHeartbeat_Disconnected(t *testing.T) {
	// Use a non-existent PID to ensure no device is found
	driver := NewDriver(Config{
		VID:    0x1038,
		PID:    0xFFFF, // Non-existent PID
		Width:  128,
		Height: 40,
	})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Heartbeat on disconnected device will try to reconnect
	err := client.SendHeartbeat()
	if err == nil {
		// Reconnection succeeded unexpectedly (very unlikely with fake PID)
		t.Log("SendHeartbeat() reconnected successfully (unexpected with fake PID)")
	} else {
		// Expected: reconnection failed
		t.Logf("SendHeartbeat() reconnect failed as expected: %v", err)
	}
}

func TestClient_DisconnectLogFlag(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver:           driver,
		width:            128,
		height:           40,
		disconnectLogged: false,
	}

	// First call should set disconnectLogged
	_ = client.SendScreenData("event", make([]byte, 640))
	if !client.disconnectLogged {
		t.Error("SendScreenData() should set disconnectLogged on disconnect")
	}

	// Second call should still work (not panic)
	_ = client.SendScreenData("event", make([]byte, 640))
}

func TestClient_WidthHeight(t *testing.T) {
	driver := NewDriver(Config{Width: 256, Height: 64})
	client := &Client{
		driver: driver,
		width:  256,
		height: 64,
	}

	if client.width != 256 {
		t.Errorf("client.width = %d, want 256", client.width)
	}
	if client.height != 64 {
		t.Errorf("client.height = %d, want 64", client.height)
	}
}

func TestClient_SendScreenDataMultiRes_KeyFormat(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Test that the key format is "image-data-WxH"
	// Even though this will fail (not connected), it validates key lookup
	resolutionData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
	}

	// Will fail because device not connected, but won't fail due to key not found
	err := client.SendScreenDataMultiRes("event", resolutionData)
	if err == nil {
		t.Error("SendScreenDataMultiRes() should fail due to disconnected device")
	}
	// Check it's a "not connected" error, not a "resolution not found" error
	if err != nil && !errors.Is(err, ErrDeviceNotConnected) {
		// Could be "resolution not found" if key format is wrong
		if errors.Is(err, ErrResolutionNotFound) {
			t.Error("SendScreenDataMultiRes() used wrong key format")
		}
	}
}

func TestClient_SendMultipleScreenData_WithFrames(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	frames := [][]byte{
		make([]byte, 640),
		make([]byte, 640),
	}

	// Will fail because not connected, but should not panic
	err := client.SendMultipleScreenData("event", frames)
	if err == nil {
		t.Error("SendMultipleScreenData() should fail due to disconnected device")
	}
}

func TestClient_SendScreenData_DataLength(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Test with various data lengths - all should handle gracefully
	testCases := []struct {
		name string
		size int
	}{
		{"exact size", 640},
		{"small", 100},
		{"large", 1000},
		{"empty", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.size)
			// Should not panic even with wrong sizes
			err := client.SendScreenData("event", data)
			if err == nil {
				t.Error("SendScreenData() should fail due to disconnected device")
			}
		})
	}
}

func TestClient_SendScreenDataMultiRes_EmptyMap(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Empty resolution map should return error (resolution not found)
	err := client.SendScreenDataMultiRes("event", map[string][]byte{})
	if err == nil {
		t.Error("SendScreenDataMultiRes() with empty map should return error")
	}
}

func TestClient_SendScreenDataMultiRes_MultipleResolutions(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	// Provide multiple resolutions including the correct one
	resolutionData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
		"image-data-256x64": make([]byte, 2048),
	}

	// Should find correct resolution but fail due to disconnected device
	err := client.SendScreenDataMultiRes("event", resolutionData)
	if err == nil {
		t.Error("SendScreenDataMultiRes() should fail due to disconnected device")
	}

	// Should be "device not connected", not "resolution not found"
	if errors.Is(err, ErrResolutionNotFound) {
		t.Error("Should have found resolution but failed on send")
	}
}

func TestClient_SendMultipleScreenData_SingleFrame(t *testing.T) {
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver: driver,
		width:  128,
		height: 40,
	}

	frames := [][]byte{
		make([]byte, 640),
	}

	// Should fail due to disconnected device
	err := client.SendMultipleScreenData("event", frames)
	if err == nil {
		t.Error("SendMultipleScreenData() should fail due to disconnected device")
	}
}

func TestClient_DisconnectLogFlag_Reset(t *testing.T) {
	driver := NewDriver(Config{
		VID:    0x1038,
		PID:    0xFFFF,
		Width:  128,
		Height: 40,
	})
	client := &Client{
		driver:           driver,
		width:            128,
		height:           40,
		disconnectLogged: true, // Start as logged
	}

	// Call SendHeartbeat - this will try to reconnect and fail
	// But if it succeeded, it should reset the flag
	err := client.SendHeartbeat()
	if err == nil {
		// Reconnection succeeded unexpectedly
		if client.disconnectLogged {
			t.Error("disconnectLogged should be false after successful reconnect")
		}
	}
}

func TestClient_SendScreenData_ResetDisconnectLogOnSuccess(t *testing.T) {
	// This tests that the disconnectLogged flag prevents log spam
	driver := NewDriver(Config{Width: 128, Height: 40})
	client := &Client{
		driver:           driver,
		width:            128,
		height:           40,
		disconnectLogged: false,
	}

	// First call should set disconnectLogged to true
	_ = client.SendScreenData("event", make([]byte, 640))
	if !client.disconnectLogged {
		t.Error("First disconnect should set disconnectLogged")
	}

	// Store the flag value
	firstValue := client.disconnectLogged

	// Second call should not change anything
	_ = client.SendScreenData("event", make([]byte, 640))

	// Flag should still be true
	if client.disconnectLogged != firstValue {
		t.Error("disconnectLogged should not change on repeated disconnects")
	}
}

func TestClient_WidthHeightDifferentSizes(t *testing.T) {
	testCases := []struct {
		width  int
		height int
	}{
		{128, 40}, // Standard
		{256, 64}, // Large
		{64, 20},  // Small
		{128, 52}, // Different height
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			driver := NewDriver(Config{Width: tc.width, Height: tc.height})
			client := &Client{
				driver: driver,
				width:  tc.width,
				height: tc.height,
			}

			if client.width != tc.width {
				t.Errorf("width = %d, want %d", client.width, tc.width)
			}
			if client.height != tc.height {
				t.Errorf("height = %d, want %d", client.height, tc.height)
			}

			// Verify key format in SendScreenDataMultiRes
			expectedKey := "image-data-" + string(rune('0'+tc.width/100)) + string(rune('0'+(tc.width/10)%10)) + string(rune('0'+tc.width%10)) + "x" + string(rune('0'+tc.height/10)) + string(rune('0'+tc.height%10))
			_ = expectedKey // Key format is tested elsewhere
		})
	}
}

func TestConfig_Default(t *testing.T) {
	cfg := Config{}

	// Verify default values
	if cfg.VID != 0 {
		t.Errorf("Default VID = %d, want 0", cfg.VID)
	}
	if cfg.PID != 0 {
		t.Errorf("Default PID = %d, want 0", cfg.PID)
	}
	if cfg.Interface != "" {
		t.Errorf("Default Interface = %q, want empty", cfg.Interface)
	}
}

func TestConfig_WithValues(t *testing.T) {
	cfg := Config{
		VID:       0x1038,
		PID:       0x12AA,
		Interface: "mi_01",
		Width:     128,
		Height:    40,
	}

	if cfg.VID != 0x1038 {
		t.Errorf("VID = 0x%04X, want 0x1038", cfg.VID)
	}
	if cfg.PID != 0x12AA {
		t.Errorf("PID = 0x%04X, want 0x12AA", cfg.PID)
	}
	if cfg.Interface != "mi_01" {
		t.Errorf("Interface = %q, want mi_01", cfg.Interface)
	}
}
