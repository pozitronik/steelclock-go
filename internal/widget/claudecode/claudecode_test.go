package claudecode

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test_claude",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w == nil {
		t.Fatal("New() returned nil widget")
	}

	if w.Name() != "test_claude" {
		t.Errorf("Name() = %s, want test_claude", w.Name())
	}
}

func TestRender_NotRunning(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test_render",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroOnStart: config.BoolPtr(false), // Skip intro for test
			Notify: &config.ClaudeCodeNotifyConfig{
				NotRunning: config.IntPtr(-1), // Show not_running state for test
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Manually set shouldShow since no intro and state change triggers it
	w.shouldShow = true

	// Update to get initial state
	_ = w.Update()

	// Render
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestRender_Intro(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test_intro",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroOnStart:  config.BoolPtr(true),
			IntroDuration: 1, // Short intro for test
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render during intro
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image during intro")
	}
}

func TestDrawSprite(t *testing.T) {
	// Test sprite drawing doesn't panic
	sprites := []*ClawdSprite{
		&ClawdLarge,
		&ClawdLargeWave,
		&ClawdLargeBounce,
		&ClawdMedium,
		&ClawdSmall,
	}

	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test_sprites",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroOnStart: config.BoolPtr(false),
		},
	}

	w, _ := New(cfg)
	img := w.CreateCanvas()

	for _, sprite := range sprites {
		drawSprite(img, sprite, 0, 0)
	}
}

func TestGetToolIcon(t *testing.T) {
	tests := []struct {
		tool    string
		wantNil bool
	}{
		{"Bash", false},
		{"Read", false},
		{"Edit", false},
		{"Write", false},
		{"Glob", false},
		{"Grep", false},
		{"WebFetch", false},
		{"WebSearch", false},
		{"Task", false},
		{"Unknown", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			icon := GetToolIcon(tt.tool)
			if (icon == nil) != tt.wantNil {
				t.Errorf("GetToolIcon(%q) nil = %v, want nil = %v", tt.tool, icon == nil, tt.wantNil)
			}
		})
	}
}

func TestStatusStates(t *testing.T) {
	states := []State{
		StateNotRunning,
		StateIdle,
		StateThinking,
		StateToolRun,
		StateSuccess,
		StateError,
	}

	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test_states",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroOnStart: config.BoolPtr(false),
			Notify: &config.ClaudeCodeNotifyConfig{
				Thinking:   config.IntPtr(-1), // Show all states for testing
				Tool:       config.IntPtr(-1),
				Success:    config.IntPtr(-1),
				Error:      config.IntPtr(-1),
				Idle:       config.IntPtr(-1),
				NotRunning: config.IntPtr(-1),
			},
		},
	}

	w, _ := New(cfg)

	// Test that all states produce valid messages
	for _, state := range states {
		status := StatusData{State: state}
		msg := w.getNotificationMessage(status)
		if msg == "" {
			t.Errorf("getNotificationMessage(%s) returned empty string", state)
		}
	}

	// Test notification duration lookup
	for _, state := range states {
		duration := w.getNotifyDuration(state)
		if duration != -1 {
			t.Errorf("getNotifyDuration(%s) = %d, want -1", state, duration)
		}
	}
}
