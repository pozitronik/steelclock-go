package webclient

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("with target FPS", func(t *testing.T) {
		c := NewClient(Config{TargetFPS: 30, Width: 128, Height: 40})
		if c == nil {
			t.Fatal("NewClient returned nil")
		}
		if c.config.Width != 128 || c.config.Height != 40 {
			t.Errorf("dimensions = %dx%d, want 128x40", c.config.Width, c.config.Height)
		}
		if c.minFrameInterval == 0 {
			t.Error("minFrameInterval should be non-zero for FPS > 0")
		}
		// 30 FPS = ~33ms per frame
		expected := time.Second / 30
		if c.minFrameInterval != expected {
			t.Errorf("minFrameInterval = %v, want %v", c.minFrameInterval, expected)
		}
	})

	t.Run("unlimited FPS", func(t *testing.T) {
		c := NewClient(Config{TargetFPS: 0, Width: 64, Height: 20})
		if c.minFrameInterval != 0 {
			t.Errorf("minFrameInterval = %v, want 0 for unlimited FPS", c.minFrameInterval)
		}
	})

	t.Run("subscribers initialized", func(t *testing.T) {
		c := NewClient(Config{TargetFPS: 30, Width: 128, Height: 40})
		if c.subscribers == nil {
			t.Error("subscribers map should be initialized")
		}
		if c.SubscriberCount() != 0 {
			t.Errorf("initial subscriber count = %d, want 0", c.SubscriberCount())
		}
	})
}

func TestGetConfig(t *testing.T) {
	cfg := Config{TargetFPS: 60, Width: 128, Height: 64}
	c := NewClient(cfg)
	got := c.GetConfig()
	if got != cfg {
		t.Errorf("GetConfig() = %+v, want %+v", got, cfg)
	}
}

func TestSendScreenData_StoresFrame(t *testing.T) {
	c := NewClient(Config{TargetFPS: 0, Width: 128, Height: 40})

	data := []byte{0xFF, 0x00, 0xAA, 0x55}
	err := c.SendScreenData("SCREEN", data)
	if err != nil {
		t.Fatalf("SendScreenData() error = %v", err)
	}

	frame, frameNum, ts := c.GetCurrentFrame()
	if frame == nil {
		t.Fatal("GetCurrentFrame() returned nil after SendScreenData")
	}
	if len(frame) != len(data) {
		t.Errorf("frame length = %d, want %d", len(frame), len(data))
	}
	for i := range data {
		if frame[i] != data[i] {
			t.Errorf("frame[%d] = %d, want %d", i, frame[i], data[i])
		}
	}
	if frameNum != 1 {
		t.Errorf("frameNumber = %d, want 1", frameNum)
	}
	if ts.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestGetCurrentFrame_NoData(t *testing.T) {
	c := NewClient(Config{TargetFPS: 30, Width: 128, Height: 40})

	frame, frameNum, ts := c.GetCurrentFrame()
	if frame != nil {
		t.Errorf("expected nil frame, got %v", frame)
	}
	if frameNum != 0 {
		t.Errorf("frameNumber = %d, want 0", frameNum)
	}
	if !ts.IsZero() {
		t.Error("timestamp should be zero")
	}
}

func TestGetCurrentFrame_ReturnsCopy(t *testing.T) {
	c := NewClient(Config{TargetFPS: 0, Width: 128, Height: 40})
	_ = c.SendScreenData("SCREEN", []byte{1, 2, 3})

	frame1, _, _ := c.GetCurrentFrame()
	frame2, _, _ := c.GetCurrentFrame()

	// Modifying one should not affect the other
	frame1[0] = 99
	if frame2[0] == 99 {
		t.Error("GetCurrentFrame should return a copy, not a reference")
	}
}

func TestSendScreenData_IncrementsFrameNumber(t *testing.T) {
	c := NewClient(Config{TargetFPS: 0, Width: 128, Height: 40})

	_ = c.SendScreenData("SCREEN", []byte{1})
	_, n1, _ := c.GetCurrentFrame()

	_ = c.SendScreenData("SCREEN", []byte{2})
	_, n2, _ := c.GetCurrentFrame()

	_ = c.SendScreenData("SCREEN", []byte{3})
	_, n3, _ := c.GetCurrentFrame()

	if n1 != 1 || n2 != 2 || n3 != 3 {
		t.Errorf("frame numbers = %d, %d, %d, want 1, 2, 3", n1, n2, n3)
	}
}

func TestSendMultipleScreenData_LastFrame(t *testing.T) {
	c := NewClient(Config{TargetFPS: 0, Width: 128, Height: 40})

	frames := [][]byte{{1, 1, 1}, {2, 2, 2}, {3, 3, 3}}
	err := c.SendMultipleScreenData("SCREEN", frames)
	if err != nil {
		t.Fatalf("SendMultipleScreenData() error = %v", err)
	}

	frame, _, _ := c.GetCurrentFrame()
	if frame == nil {
		t.Fatal("frame is nil")
	}
	// Should store the last frame
	if frame[0] != 3 {
		t.Errorf("stored frame[0] = %d, want 3 (last frame)", frame[0])
	}
}

func TestSendMultipleScreenData_Empty(t *testing.T) {
	c := NewClient(Config{TargetFPS: 0, Width: 128, Height: 40})
	err := c.SendMultipleScreenData("SCREEN", [][]byte{})
	if err != nil {
		t.Fatalf("expected nil error for empty frames, got %v", err)
	}
}

func TestSendScreenDataMultiRes(t *testing.T) {
	c := NewClient(Config{TargetFPS: 0, Width: 128, Height: 40})

	data := map[string][]byte{
		"image-data-128x40": {0xAA, 0xBB},
	}
	err := c.SendScreenDataMultiRes("SCREEN", data)
	if err != nil {
		t.Fatalf("SendScreenDataMultiRes() error = %v", err)
	}

	frame, _, _ := c.GetCurrentFrame()
	if frame == nil {
		t.Fatal("frame is nil after SendScreenDataMultiRes")
	}
}

func TestNoOpMethods(t *testing.T) {
	c := NewClient(Config{TargetFPS: 30, Width: 128, Height: 40})

	if err := c.SendHeartbeat(); err != nil {
		t.Errorf("SendHeartbeat() error = %v", err)
	}
	if err := c.RegisterGame("dev", 5000); err != nil {
		t.Errorf("RegisterGame() error = %v", err)
	}
	if err := c.BindScreenEvent("SCREEN", "type"); err != nil {
		t.Errorf("BindScreenEvent() error = %v", err)
	}
	if err := c.RemoveGame(); err != nil {
		t.Errorf("RemoveGame() error = %v", err)
	}
	if c.SupportsMultipleEvents() {
		t.Error("SupportsMultipleEvents() should return false")
	}
}

func TestRateLimiting(t *testing.T) {
	// 10 FPS = 100ms between frames
	c := NewClient(Config{TargetFPS: 10, Width: 128, Height: 40})

	// First frame: always stored and broadcast
	_ = c.SendScreenData("SCREEN", []byte{1})
	_, n1, _ := c.GetCurrentFrame()

	// Immediate second frame: stored but rate-limited (no broadcast, still increments frame counter)
	_ = c.SendScreenData("SCREEN", []byte{2})
	frame, n2, _ := c.GetCurrentFrame()

	if n1 != 1 {
		t.Errorf("first frame number = %d, want 1", n1)
	}
	if n2 != 2 {
		t.Errorf("second frame number = %d, want 2 (still stored even if rate-limited)", n2)
	}
	// The second frame's data should be stored even though broadcast was skipped
	if frame[0] != 2 {
		t.Errorf("stored frame = %d, want 2 (latest data)", frame[0])
	}

	// Wait for rate limit window to pass, then send third frame
	time.Sleep(110 * time.Millisecond)
	_ = c.SendScreenData("SCREEN", []byte{3})
	frame, n3, _ := c.GetCurrentFrame()
	if n3 != 3 {
		t.Errorf("third frame number = %d, want 3", n3)
	}
	if frame[0] != 3 {
		t.Errorf("stored frame = %d, want 3", frame[0])
	}
}

func TestFrameMessage_Structure(t *testing.T) {
	msg := FrameMessage{
		Type:        "frame",
		Width:       128,
		Height:      40,
		Frame:       []byte{0xFF},
		FrameNumber: 42,
		Timestamp:   1234567890,
	}

	if msg.Type != "frame" {
		t.Errorf("Type = %q", msg.Type)
	}
	if msg.Width != 128 || msg.Height != 40 {
		t.Errorf("dimensions = %dx%d", msg.Width, msg.Height)
	}
	if msg.FrameNumber != 42 {
		t.Errorf("FrameNumber = %d", msg.FrameNumber)
	}
}
