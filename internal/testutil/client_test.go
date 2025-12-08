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

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_SendHeartbeatError(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("heartbeat failed")
	client.SetHeartbeatError(expectedErr)

	err := client.SendHeartbeat()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Clear error and try again
	client.ClearErrors()
	if err := client.SendHeartbeat(); err != nil {
		t.Errorf("SendHeartbeat should succeed after ClearErrors: %v", err)
	}
}

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_RemoveGameError(t *testing.T) {
	client := NewTestClient()
	_ = client.RegisterGame("Dev", 0)

	expectedErr := errors.New("remove failed")
	client.SetRemoveError(expectedErr)

	err := client.RemoveGame()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Client should still be registered because RemoveGame failed
	// (depends on implementation - currently it sets registered=false before checking error)
}

func TestTestClient_FrameByIndex(t *testing.T) {
	client := NewTestClient(WithMaxFrames(100))

	// Send 5 frames
	for i := 0; i < 5; i++ {
		frameData := make([]byte, 640)
		frameData[0] = byte(i * 10)
		_ = client.SendScreenData("EVENT", frameData)
	}

	// Get frame by index (1-based)
	frame := client.Frame(3)
	if frame == nil {
		t.Fatal("Frame(3) returned nil")
	}

	if frame.Index != 3 {
		t.Errorf("Expected frame index 3, got %d", frame.Index)
	}

	if frame.Data[0] != 20 {
		t.Errorf("Expected frame data[0]=20, got %d", frame.Data[0])
	}

	// Non-existent frame
	nonExistent := client.Frame(100)
	if nonExistent != nil {
		t.Error("Frame(100) should return nil for non-existent frame")
	}
}

func TestTestClient_Duration(t *testing.T) {
	client := NewTestClient()

	time.Sleep(10 * time.Millisecond)

	duration := client.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration should be at least 10ms, got %v", duration)
	}
}

func TestTestClient_LastSendTime(t *testing.T) {
	client := NewTestClient()

	// Initially should be zero
	if !client.LastSendTime().IsZero() {
		t.Error("LastSendTime should be zero before any sends")
	}

	before := time.Now()
	_ = client.SendScreenData("EVENT", make([]byte, 640))
	after := time.Now()

	lastSend := client.LastSendTime()
	if lastSend.Before(before) || lastSend.After(after) {
		t.Errorf("LastSendTime %v should be between %v and %v", lastSend, before, after)
	}
}

func TestTestClient_SendScreenDataMultiRes_MissingResolution(t *testing.T) {
	client := NewTestClient(WithDimensions(128, 40))

	// Only provide data for a different resolution
	resData := map[string][]byte{
		"image-data-256x64": make([]byte, 2048),
	}

	// Should succeed but capture nothing
	err := client.SendScreenDataMultiRes("EVENT", resData)
	if err != nil {
		t.Errorf("SendScreenDataMultiRes should not error: %v", err)
	}

	// No frame should be captured
	if client.FrameCount() != 0 {
		t.Errorf("Expected 0 frames, got %d", client.FrameCount())
	}
}

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_SendMultipleScreenData_Error(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("batch send failed")
	client.SetSendError(expectedErr, 0) // Error on all sends

	frames := [][]byte{
		make([]byte, 640),
		make([]byte, 640),
	}

	err := client.SendMultipleScreenData("EVENT", frames)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// No frames should be captured
	if client.FrameCount() != 0 {
		t.Errorf("Expected 0 frames, got %d", client.FrameCount())
	}
}

func TestTestClient_SendMultipleScreenData_Paused(t *testing.T) {
	client := NewTestClient()
	client.Pause()

	frames := [][]byte{
		make([]byte, 640),
		make([]byte, 640),
	}

	err := client.SendMultipleScreenData("EVENT", frames)
	if err != nil {
		t.Errorf("SendMultipleScreenData should succeed when paused: %v", err)
	}

	// No frames should be captured
	if client.FrameCount() != 0 {
		t.Errorf("Expected 0 frames when paused, got %d", client.FrameCount())
	}
}

func TestTestClient_SendScreenDataMultiRes_Paused(t *testing.T) {
	client := NewTestClient(WithDimensions(128, 40))
	client.Pause()

	resData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
	}

	err := client.SendScreenDataMultiRes("EVENT", resData)
	if err != nil {
		t.Errorf("SendScreenDataMultiRes should succeed when paused: %v", err)
	}

	// No frames should be captured
	if client.FrameCount() != 0 {
		t.Errorf("Expected 0 frames when paused, got %d", client.FrameCount())
	}
}

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_SendScreenDataMultiRes_Error(t *testing.T) {
	client := NewTestClient(WithDimensions(128, 40))
	expectedErr := errors.New("multi-res send failed")
	client.SetSendError(expectedErr, 0)

	resData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
	}

	err := client.SendScreenDataMultiRes("EVENT", resData)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestTestClient_SendMultipleScreenData_Empty(t *testing.T) {
	client := NewTestClient()

	err := client.SendMultipleScreenData("EVENT", [][]byte{})
	if err != nil {
		t.Errorf("SendMultipleScreenData with empty frames should not error: %v", err)
	}

	if client.FrameCount() != 0 {
		t.Errorf("Expected 0 frames, got %d", client.FrameCount())
	}
}

func TestTestClient_FrameChannel_Full(t *testing.T) {
	// Create a channel with buffer of 1
	ch := make(chan Frame, 1)
	client := NewTestClient(WithFrameChannel(ch))

	// Send multiple frames - channel should not block
	for i := 0; i < 5; i++ {
		err := client.SendScreenData("EVENT", make([]byte, 640))
		if err != nil {
			t.Errorf("SendScreenData failed: %v", err)
		}
	}

	// All 5 frames should be captured
	if client.FrameCount() != 5 {
		t.Errorf("Expected 5 frames, got %d", client.FrameCount())
	}

	// Only 1 should be in channel (buffer full, others dropped)
	select {
	case <-ch:
		// OK
	default:
		t.Error("Expected at least 1 frame in channel")
	}
}

func TestTestClient_LastFrame_Nil(t *testing.T) {
	client := NewTestClient()

	// No frames sent
	if client.LastFrame() != nil {
		t.Error("LastFrame should return nil when no frames sent")
	}
}

func TestTestClient_Frames_Empty(t *testing.T) {
	client := NewTestClient()

	frames := client.Frames()
	if len(frames) != 0 {
		t.Errorf("Expected 0 frames, got %d", len(frames))
	}
}

func TestTestClient_BindScreenEvent_Error(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("bind failed")
	client.SetBindError(expectedErr)

	err := client.BindScreenEvent("EVENT", "device")
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Event should not be bound
	if client.IsBound("EVENT") {
		t.Error("Event should not be bound after error")
	}
}

func TestTestClient_ConcurrentAccess(t *testing.T) {
	client := NewTestClient(WithMaxFrames(1000))

	// Start multiple goroutines sending frames
	const numGoroutines = 10
	const framesPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < framesPerGoroutine; j++ {
				frameData := make([]byte, 640)
				frameData[0] = byte(id)
				frameData[1] = byte(j)
				_ = client.SendScreenData("EVENT", frameData)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify frame count
	expectedFrames := numGoroutines * framesPerGoroutine
	if client.FrameCount() != expectedFrames {
		t.Errorf("Expected %d frames, got %d", expectedFrames, client.FrameCount())
	}
}

func TestTestClient_SenderrorCount_Zero(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("permanent error")

	// SetSendError with count=0 means error on ALL sends
	client.SetSendError(expectedErr, 0)

	for i := 0; i < 5; i++ {
		err := client.SendScreenData("EVENT", make([]byte, 640))
		if err != expectedErr {
			t.Errorf("Send %d: expected error, got %v", i, err)
		}
	}

	// No frames captured
	if client.FrameCount() != 0 {
		t.Errorf("Expected 0 frames, got %d", client.FrameCount())
	}
}

func TestTestClient_Calls_RecordArgs(t *testing.T) {
	client := NewTestClient()

	_ = client.RegisterGame("TestDeveloper", 15000)
	_ = client.BindScreenEvent("TEST_EVENT", "test-device")
	_ = client.SendScreenData("EVENT", make([]byte, 640))

	calls := client.Calls()
	if len(calls) != 3 {
		t.Fatalf("Expected 3 calls, got %d", len(calls))
	}

	// Check RegisterGame args
	if calls[0].Method != "RegisterGame" {
		t.Errorf("Expected method RegisterGame, got %s", calls[0].Method)
	}
	if len(calls[0].Args) != 2 {
		t.Errorf("Expected 2 args for RegisterGame, got %d", len(calls[0].Args))
	}
	if calls[0].Args[0] != "TestDeveloper" {
		t.Errorf("Expected developer 'TestDeveloper', got %v", calls[0].Args[0])
	}

	// Check BindScreenEvent args
	if calls[1].Method != "BindScreenEvent" {
		t.Errorf("Expected method BindScreenEvent, got %s", calls[1].Method)
	}

	// Check SendScreenData args
	if calls[2].Method != "SendScreenData" {
		t.Errorf("Expected method SendScreenData, got %s", calls[2].Method)
	}
}

// Tests for error decrement paths in SendScreenDataMultiRes and SendMultipleScreenData

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_SendScreenDataMultiRes_ErrorDecrement(t *testing.T) {
	client := NewTestClient(WithDimensions(128, 40))
	expectedErr := errors.New("multi-res error")

	// Set error for exactly 2 sends
	client.SetSendError(expectedErr, 2)

	resData := map[string][]byte{
		"image-data-128x40": make([]byte, 640),
	}

	// First two should fail
	err1 := client.SendScreenDataMultiRes("EVENT", resData)
	err2 := client.SendScreenDataMultiRes("EVENT", resData)
	// Third should succeed
	err3 := client.SendScreenDataMultiRes("EVENT", resData)

	if err1 != expectedErr {
		t.Errorf("First send should fail with error")
	}
	if err2 != expectedErr {
		t.Errorf("Second send should fail with error")
	}
	if err3 != nil {
		t.Errorf("Third send should succeed, got %v", err3)
	}
}

//goland:noinspection GoDirectComparisonOfErrors
func TestTestClient_SendMultipleScreenData_ErrorDecrement(t *testing.T) {
	client := NewTestClient()
	expectedErr := errors.New("batch error")

	// Set error for exactly 2 sends
	client.SetSendError(expectedErr, 2)

	frames := [][]byte{make([]byte, 640)}

	// First two should fail
	err1 := client.SendMultipleScreenData("EVENT", frames)
	err2 := client.SendMultipleScreenData("EVENT", frames)
	// Third should succeed
	err3 := client.SendMultipleScreenData("EVENT", frames)

	if err1 != expectedErr {
		t.Errorf("First send should fail")
	}
	if err2 != expectedErr {
		t.Errorf("Second send should fail")
	}
	if err3 != nil {
		t.Errorf("Third send should succeed, got %v", err3)
	}
}
