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
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

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
		&ClawdLargeIdle,
		&ClawdLargeWave,
		&ClawdMediumIdle,
		&ClawdMediumThinking,
		&ClawdMediumWorking,
		&ClawdMediumHappy,
		&ClawdMediumSad,
		&ClawdMediumSleeping,
		&ClawdSmallIdle,
		&ClawdSmallThinking,
		&ClawdSmallHappy,
		&ClawdSmallWorking,
		&ClawdSmallSad,
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
		tool     string
		wantNil  bool
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
		},
	}

	w, _ := New(cfg)

	// Test that all states produce valid sprites
	for _, state := range states {
		sprite := w.getClawdSprite(state, false)
		if sprite == nil {
			t.Errorf("getClawdSprite(%s) returned nil", state)
		}

		smallSprite := w.getSmallClawdSprite(state, false)
		if smallSprite == nil {
			t.Errorf("getSmallClawdSprite(%s) returned nil", state)
		}
	}

	// Test celebrating state
	sprite := w.getClawdSprite(StateIdle, true)
	if sprite != &ClawdMediumHappy {
		t.Error("Celebrating should use happy sprite")
	}
}
