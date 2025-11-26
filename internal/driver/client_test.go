package driver

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/gamesense"
)

// Verify Client implements gamesense.API interface at compile time
var _ gamesense.API = (*Client)(nil)

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

	err := client.SendScreenData("event", make([]int, 640))
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
	resolutionData := map[string][]int{
		"image-data-256x64": make([]int, 2048),
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

	err := client.SendMultipleScreenData("event", [][]int{})
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
	_ = client.SendScreenData("event", make([]int, 640))
	if !client.disconnectLogged {
		t.Error("SendScreenData() should set disconnectLogged on disconnect")
	}

	// Second call should still work (not panic)
	_ = client.SendScreenData("event", make([]int, 640))
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
	resolutionData := map[string][]int{
		"image-data-128x40": make([]int, 640),
	}

	// Will fail because device not connected, but won't fail due to key not found
	err := client.SendScreenDataMultiRes("event", resolutionData)
	if err == nil {
		t.Error("SendScreenDataMultiRes() should fail due to disconnected device")
	}
	// Check it's a "not connected" error, not a "resolution not found" error
	expectedErr := "device not connected"
	if err != nil && err.Error() != expectedErr {
		// Could be "resolution not found" if key format is wrong
		if err.Error() == "resolution 128x40 not found in data" {
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

	frames := [][]int{
		make([]int, 640),
		make([]int, 640),
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
			data := make([]int, tc.size)
			// Should not panic even with wrong sizes
			err := client.SendScreenData("event", data)
			if err == nil {
				t.Error("SendScreenData() should fail due to disconnected device")
			}
		})
	}
}
