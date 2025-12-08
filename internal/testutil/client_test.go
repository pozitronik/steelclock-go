package testutil

import (
	"errors"
	"testing"
	"time"
)

func TestNewTestClient(t *testing.T) {
	client := NewTestClient()

	if client == nil {
		t.Fatal("NewTestClient returned nil")
	}

	if client.width != 128 || client.height != 40 {
		t.Errorf("Expected default dimensions 128x40, got %dx%d", client.width, client.height)
	}

	if client.maxFrames != 100 {
		t.Errorf("Expected default maxFrames 100, got %d", client.maxFrames)
	}
}

func TestNewTestClientWithOptions(t *testing.T) {
	client := NewTestClient(
		WithDimensions(64, 20),
		WithMaxFrames(50),
		WithMultipleEventsSupport(true),
	)

	if client.width != 64 || client.height != 20 {
		t.Errorf("Expected dimensions 64x20, got %dx%d", client.width, client.height)
	}

	if client.maxFrames != 50 {
		t.Errorf("Expected maxFrames 50, got %d", client.maxFrames)
	}

	if !client.supportsMultipleEvents {
		t.Error("Expected supportsMultipleEvents to be true")
	}
}

func TestTestClient_RegisterGame(t *testing.T) {
	client := NewTestClient()

	err := client.RegisterGame("Test Developer", 15000)
	if err != nil {
		t.Errorf("RegisterGame failed: %v", err)
	}

	if !client.IsRegistered() {
		t.Error("Client should be registered")
	}

	if client.CallCount("RegisterGame") != 1 {
		t.Errorf("Expected 1 RegisterGame call, got %d", client.CallCount("RegisterGame"))
	}
}

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_RegisterGameError(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("registration failed")
	client.SetRegisterError(expectedErr)

	err := client.RegisterGame("Test", 0)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if client.IsRegistered() {
		t.Error("Client should not be registered after error")
	}
}

func TestTestClient_BindScreenEvent(t *testing.T) {
	client := NewTestClient()

	err := client.BindScreenEvent("SCREEN_UPDATE", "screened-128x40")
	if err != nil {
		t.Errorf("BindScreenEvent failed: %v", err)
	}

	if !client.IsBound("SCREEN_UPDATE") {
		t.Error("Event should be bound")
	}

	if client.IsBound("OTHER_EVENT") {
		t.Error("Other event should not be bound")
	}
}

func TestTestClient_SendScreenData(t *testing.T) {
	client := NewTestClient()

	// Create test frame data
	frameData := make([]byte, 640)
	frameData[0] = 0xFF // Set first byte

	err := client.SendScreenData("EVENT", frameData)
	if err != nil {
		t.Errorf("SendScreenData failed: %v", err)
	}

	if client.FrameCount() != 1 {
		t.Errorf("Expected 1 frame, got %d", client.FrameCount())
	}

	lastFrame := client.LastFrame()
	if lastFrame == nil {
		t.Fatal("LastFrame returned nil")
	}

	if lastFrame.Data[0] != 0xFF {
		t.Errorf("Frame data mismatch: expected 0xFF, got %d", lastFrame.Data[0])
	}

	if lastFrame.EventName != "EVENT" {
		t.Errorf("Expected event name 'EVENT', got '%s'", lastFrame.EventName)
	}
}

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_SendErrorInjection(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("send failed")

	// Set error for 2 sends
	client.SetSendError(expectedErr, 2)

	// First two should fail
	err1 := client.SendScreenData("EVENT", make([]byte, 640))
	err2 := client.SendScreenData("EVENT", make([]byte, 640))
	// Third should succeed
	err3 := client.SendScreenData("EVENT", make([]byte, 640))

	if err1 != expectedErr {
		t.Errorf("First send should fail with error")
	}
	if err2 != expectedErr {
		t.Errorf("Second send should fail with error")
	}
	if err3 != nil {
		t.Errorf("Third send should succeed, got %v", err3)
	}

	// Only one frame should be captured (the successful one)
	if client.FrameCount() != 1 {
		t.Errorf("Expected 1 frame, got %d", client.FrameCount())
	}
}

func TestTestClient_FrameHistory(t *testing.T) {
	client := NewTestClient(WithMaxFrames(3))

	// Send 5 frames
	for i := 0; i < 5; i++ {
		frameData := make([]byte, 640)
		frameData[0] = byte(i)
		_ = client.SendScreenData("EVENT", frameData)
	}

	// Should have 5 total, but only 3 in history
	if client.FrameCount() != 5 {
		t.Errorf("Expected total frameCount 5, got %d", client.FrameCount())
	}

	frames := client.Frames()
	if len(frames) != 3 {
		t.Errorf("Expected 3 frames in history, got %d", len(frames))
	}

	// Should have frames 3, 4, 5 (0-indexed: values 2, 3, 4)
	if frames[0].Data[0] != 2 {
		t.Errorf("Expected first frame data[0]=2, got %d", frames[0].Data[0])
	}
}

func TestTestClient_Pause(t *testing.T) {
	client := NewTestClient()

	_ = client.SendScreenData("EVENT", make([]byte, 640))
	client.Pause()
	_ = client.SendScreenData("EVENT", make([]byte, 640))
	_ = client.SendScreenData("EVENT", make([]byte, 640))
	client.Resume()
	_ = client.SendScreenData("EVENT", make([]byte, 640))

	// Only 2 frames should be captured (before pause and after resume)
	if client.FrameCount() != 2 {
		t.Errorf("Expected 2 frames, got %d", client.FrameCount())
	}
}

func TestTestClient_Reset(t *testing.T) {
	client := NewTestClient()

	_ = client.RegisterGame("Dev", 0)
	_ = client.BindScreenEvent("EVENT", "device")
	_ = client.SendScreenData("EVENT", make([]byte, 640))

	client.Reset()

	if client.IsRegistered() {
		t.Error("Should not be registered after reset")
	}
	if client.IsBound("EVENT") {
		t.Error("Event should not be bound after reset")
	}
	if client.FrameCount() != 0 {
		t.Error("Frame count should be 0 after reset")
	}
	if len(client.Calls()) != 0 {
		t.Error("Calls should be empty after reset")
	}
}

func TestTestClient_FrameChannel(t *testing.T) {
	ch := make(chan Frame, 10)
	client := NewTestClient(WithFrameChannel(ch))

	frameData := make([]byte, 640)
	frameData[0] = 42
	_ = client.SendScreenData("EVENT", frameData)

	select {
	case frame := <-ch:
		if frame.Data[0] != 42 {
			t.Errorf("Expected frame data[0]=42, got %d", frame.Data[0])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for frame on channel")
	}
}

func TestTestClient_SendMultipleScreenData(t *testing.T) {
	client := NewTestClient()

	frames := [][]byte{
		make([]byte, 640),
		make([]byte, 640),
		make([]byte, 640),
	}
	frames[0][0] = 1
	frames[1][0] = 2
	frames[2][0] = 3

	err := client.SendMultipleScreenData("EVENT", frames)
	if err != nil {
		t.Errorf("SendMultipleScreenData failed: %v", err)
	}

	if client.FrameCount() != 3 {
		t.Errorf("Expected 3 frames, got %d", client.FrameCount())
	}

	capturedFrames := client.Frames()
	for i, f := range capturedFrames {
		if f.Data[0] != byte(i+1) {
			t.Errorf("Frame %d: expected data[0]=%d, got %d", i, i+1, f.Data[0])
		}
	}
}

func TestTestClient_SendScreenDataMultiRes(t *testing.T) {
	client := NewTestClient(WithDimensions(128, 40))

	resData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
		"image-data-128x52": make([]byte, 832),
	}
	resData["image-data-128x40"][0] = 99

	err := client.SendScreenDataMultiRes("EVENT", resData)
	if err != nil {
		t.Errorf("SendScreenDataMultiRes failed: %v", err)
	}

	lastFrame := client.LastFrame()
	if lastFrame == nil {
		t.Fatal("No frame captured")
	}

	if lastFrame.Data[0] != 99 {
		t.Errorf("Expected data[0]=99, got %d", lastFrame.Data[0])
	}
}

func TestTestClient_SupportsMultipleEvents(t *testing.T) {
	clientNoSupport := NewTestClient()
	clientWithSupport := NewTestClient(WithMultipleEventsSupport(true))

	if clientNoSupport.SupportsMultipleEvents() {
		t.Error("Expected SupportsMultipleEvents to be false by default")
	}

	if !clientWithSupport.SupportsMultipleEvents() {
		t.Error("Expected SupportsMultipleEvents to be true when configured")
	}
}

func TestTestClient_RemoveGame(t *testing.T) {
	client := NewTestClient()
	_ = client.RegisterGame("Dev", 0)

	if !client.IsRegistered() {
		t.Error("Should be registered")
	}

	err := client.RemoveGame()
	if err != nil {
		t.Errorf("RemoveGame failed: %v", err)
	}

	if client.IsRegistered() {
		t.Error("Should not be registered after RemoveGame")
	}
}

func TestTestClient_ClearErrors(t *testing.T) {
	client := NewTestClient()
	client.SetRegisterError(errors.New("error"))
	client.SetBindError(errors.New("error"))
	client.SetSendError(errors.New("error"), 5)

	client.ClearErrors()

	if err := client.RegisterGame("Dev", 0); err != nil {
		t.Error("RegisterGame should succeed after ClearErrors")
	}
	if err := client.BindScreenEvent("EVENT", "device"); err != nil {
		t.Error("BindScreenEvent should succeed after ClearErrors")
	}
	if err := client.SendScreenData("EVENT", make([]byte, 640)); err != nil {
		t.Error("SendScreenData should succeed after ClearErrors")
	}
}
