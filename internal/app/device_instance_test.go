package app

import (
	"testing"
)

func TestNewDeviceInstance(t *testing.T) {
	cancel := make(chan struct{})
	d := NewDeviceInstance("test-device", cancel)

	if d == nil {
		t.Fatal("NewDeviceInstance returned nil")
	}
	if d.id != "test-device" {
		t.Errorf("id = %q, want %q", d.id, "test-device")
	}
	if d.widgetMgr == nil {
		t.Error("widgetMgr should be initialized")
	}
	if d.retryCancel == nil {
		t.Error("retryCancel should be set")
	}
	if d.client != nil {
		t.Error("client should be nil initially")
	}
	if d.comp != nil {
		t.Error("comp should be nil initially")
	}
	if d.currentBackend != "" {
		t.Errorf("currentBackend should be empty, got %q", d.currentBackend)
	}
	if d.displayWidth != 0 || d.displayHeight != 0 {
		t.Errorf("display dimensions should be 0x0 initially, got %dx%d", d.displayWidth, d.displayHeight)
	}
}

func TestDeviceInstance_GetCurrentBackend_Empty(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	if d.GetCurrentBackend() != "" {
		t.Errorf("GetCurrentBackend() = %q, want empty", d.GetCurrentBackend())
	}
}

func TestDeviceInstance_GetWebClient_Nil(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	if d.GetWebClient() != nil {
		t.Error("GetWebClient() should be nil without webclient backend")
	}
}

func TestDeviceInstance_StopWithNilComponents(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	// Should not panic
	d.Stop()
}

func TestDeviceInstance_ShutdownWithNilComponents(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	// Should not panic
	d.Shutdown(false)
	d.Shutdown(true)
}

func TestDeviceInstance_DoubleStop(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	d.Stop()
	d.Stop()
}

func TestDeviceInstance_StopThenShutdown(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	d.Stop()
	d.Shutdown(false)
}

func TestDeviceInstance_ShowTransitionBanner_NilClient(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	// Should not panic with nil client
	d.ShowTransitionBanner("TestProfile")
}

func TestDeviceInstance_ShowWebClientModeMessage_NilClient(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	// Should not panic with nil client
	d.ShowWebClientModeMessage()
}

func TestDeviceInstance_ConcurrentStop(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			d.Stop()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDeviceInstance_ConcurrentAccessors(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			_ = d.GetCurrentBackend()
			_ = d.GetWebClient()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// uiReturnRecordingClient records how many frames had been sent when
// ReturnToUI was called.
type uiReturnRecordingClient struct {
	mockSplashClient
	uiReturnCalls      int
	framesAtUIReturn   int
	framesAfterReturns int
}

func (c *uiReturnRecordingClient) SendScreenData(eventName string, bitmapData []byte) error {
	if c.uiReturnCalls > 0 {
		c.framesAfterReturns++
	}
	return c.mockSplashClient.SendScreenData(eventName, bitmapData)
}

func (c *uiReturnRecordingClient) ReturnToUI() error {
	c.uiReturnCalls++
	c.framesAtUIReturn = c.framesSent
	return nil
}

func TestDeviceInstance_Shutdown_ReturnsToUIAfterExitMessage(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))
	client := &uiReturnRecordingClient{}
	d.client = client
	d.displayWidth, d.displayHeight = 128, 40

	d.Shutdown(false)

	if client.uiReturnCalls != 1 {
		t.Fatalf("ReturnToUI called %d times, want 1", client.uiReturnCalls)
	}
	if client.framesSent == 0 {
		t.Fatal("exit message sent no frames")
	}
	if client.framesAfterReturns != 0 || client.framesAtUIReturn != client.framesSent {
		t.Errorf("%d frame(s) sent after ReturnToUI; the exit message must finish first",
			client.framesAfterReturns)
	}
}
