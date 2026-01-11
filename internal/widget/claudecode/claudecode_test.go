package claudecode

import (
	"testing"
	"time"

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
			IntroDuration: 0, // Skip intro for test
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
			IntroDuration: 0, // Skip intro
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
			IntroDuration: 0, // Skip intro
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

func TestFormatElapsedTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "0s"},
		{"seconds", 12 * time.Second, "12s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"one minute", 60 * time.Second, "1:00"},
		{"one minute 23 seconds", 83 * time.Second, "1:23"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59:59"},
		{"one hour", 60 * time.Minute, "1:00:00"},
		{"one hour 23 minutes 45 seconds", time.Hour + 23*time.Minute + 45*time.Second, "1:23:45"},
		{"multi hour", 2*time.Hour + 5*time.Minute + 30*time.Second, "2:05:30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsedTime(tt.duration)
			if got != tt.want {
				t.Errorf("formatElapsedTime(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestIsIdleState(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateNotRunning, true},
		{StateIdle, true},
		{StateThinking, false},
		{StateToolRun, false},
		{StateSuccess, false},
		{StateError, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := isIdleState(tt.state)
			if got != tt.want {
				t.Errorf("isIdleState(%s) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestGetSleepySprite(t *testing.T) {
	sizes := []string{"large", "medium", "small"}

	for _, size := range sizes {
		t.Run(size, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type: "claude_code",
				ID:   "test",
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				ClaudeCode: &config.ClaudeCodeConfig{
					IntroDuration: 0,
					SpriteSize:    size,
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			// Test all frame indices
			for i := 0; i < 3; i++ {
				sprite := w.getSleepySprite(i)
				if sprite == nil {
					t.Errorf("getSleepySprite(%d) returned nil for size %s", i, size)
				}
			}

			// Test out of bounds indices (should return last frame)
			sprite := w.getSleepySprite(10)
			if sprite == nil {
				t.Errorf("getSleepySprite(10) returned nil for size %s", size)
			}
		})
	}
}

func TestGetLookSprite(t *testing.T) {
	sizes := []string{"large", "medium", "small"}

	for _, size := range sizes {
		t.Run(size, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type: "claude_code",
				ID:   "test",
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				ClaudeCode: &config.ClaudeCodeConfig{
					IntroDuration: 0,
					SpriteSize:    size,
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			// Test look left
			spriteLeft := w.getLookSprite(true)
			if spriteLeft == nil {
				t.Errorf("getLookSprite(true) returned nil for size %s", size)
			}

			// Test look right
			spriteRight := w.getLookSprite(false)
			if spriteRight == nil {
				t.Errorf("getLookSprite(false) returned nil for size %s", size)
			}
		})
	}
}

func TestGetNormalSprite(t *testing.T) {
	tests := []struct {
		size       string
		wantWidth  int
		wantHeight int
	}{
		{"large", 34, 20},
		{"medium", 17, 10},
		{"small", 9, 6},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type: "claude_code",
				ID:   "test",
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				ClaudeCode: &config.ClaudeCodeConfig{
					IntroDuration: 0,
					SpriteSize:    tt.size,
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			sprite := w.getNormalSprite()
			if sprite == nil {
				t.Fatal("getNormalSprite() returned nil")
			}

			if sprite.Width != tt.wantWidth || sprite.Height != tt.wantHeight {
				t.Errorf("getNormalSprite() size = %dx%d, want %dx%d",
					sprite.Width, sprite.Height, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

func TestGetClawdSprite_IdleAnimations(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			SpriteSize:    "medium",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	now := time.Now()

	// Test sleepy transition (returns sleepy sprite during animation)
	w.isSleepy = true
	w.sleepyStartTime = now
	sprite := w.getClawdSprite(StateIdle, now)
	if sprite == nil {
		t.Error("getClawdSprite() returned nil during sleepy transition")
	}

	// Test looking left
	w.isSleepy = false
	w.isLookingLeft = true
	sprite = w.getClawdSprite(StateIdle, now)
	if sprite != &ClawdMediumLookLeft {
		t.Error("getClawdSprite() should return look left sprite")
	}

	// Test looking right
	w.isLookingLeft = false
	w.isLookingRight = true
	sprite = w.getClawdSprite(StateIdle, now)
	if sprite != &ClawdMediumLookRight {
		t.Error("getClawdSprite() should return look right sprite")
	}

	// Test wake-up animation
	w.isLookingRight = false
	w.isWakingUp = true
	w.wakeUpStartTime = now
	sprite = w.getClawdSprite(StateThinking, now)
	if sprite == nil {
		t.Error("getClawdSprite() returned nil during wake-up")
	}

	// Test blinking
	w.isWakingUp = false
	w.isBlinking = true
	w.blinkStartTime = now
	sprite = w.getClawdSprite(StateThinking, now)
	if sprite == nil {
		t.Error("getClawdSprite() returned nil during blink")
	}
}

func TestUpdateAnimations_SleepyCompletion(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start sleepy animation
	w.isSleepy = true
	w.sleepyStartTime = time.Now().Add(-sleepyAnimationDuration - time.Millisecond)

	// Update should complete the sleepy animation
	w.updateAnimations(time.Now(), StateIdle)

	if w.isSleepy {
		t.Error("updateAnimations() should set isSleepy to false after animation completes")
	}
}

func TestUpdateAnimations_WakeUpCompletion(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start wake-up animation
	w.isWakingUp = true
	w.wakeUpStartTime = time.Now().Add(-wakeUpAnimationDuration - time.Millisecond)

	// Update should complete the wake-up animation
	w.updateAnimations(time.Now(), StateThinking)

	if w.isWakingUp {
		t.Error("updateAnimations() should set isWakingUp to false after animation completes")
	}
}

func TestUpdateAnimations_BlinkCompletion(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start blink animation
	w.isBlinking = true
	w.blinkStartTime = time.Now().Add(-blinkDuration - time.Millisecond)

	// Update should complete the blink animation
	w.updateAnimations(time.Now(), StateThinking)

	if w.isBlinking {
		t.Error("updateAnimations() should set isBlinking to false after animation completes")
	}
}

func TestUpdateAnimations_LookCompletion(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set up completed look animation
	w.isSleepy = false
	w.isLookingLeft = true
	w.lookStartTime = time.Now().Add(-lookDuration - time.Millisecond)

	// Update should complete the look animation
	w.updateAnimations(time.Now(), StateIdle)

	if w.isLookingLeft {
		t.Error("updateAnimations() should set isLookingLeft to false after animation completes")
	}
}

func TestUpdateAnimations_BreathingPhase(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set up idle state without sleepy transition
	w.isSleepy = false
	w.breathingPhase = 0
	w.lastBreathTime = time.Now().Add(-100 * time.Millisecond)

	initialPhase := w.breathingPhase

	// Update should advance breathing phase
	w.updateAnimations(time.Now(), StateIdle)

	if w.breathingPhase == initialPhase {
		t.Error("updateAnimations() should advance breathingPhase during idle")
	}
}

func TestUpdateAnimations_IdleAnimationsDisabled(t *testing.T) {
	idleAnimations := false
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration:  0,
			IdleAnimations: &idleAnimations,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set some animation states
	w.isBlinking = true
	w.blinkStartTime = time.Now().Add(-blinkDuration - time.Millisecond)

	// Update should not change anything when idle animations disabled
	w.updateAnimations(time.Now(), StateThinking)

	// With idle animations disabled, isBlinking should remain unchanged
	// (the function returns early)
	if !w.isBlinking {
		t.Error("updateAnimations() should not modify state when idle animations disabled")
	}
}

func TestParseConfig_ShowTimer(t *testing.T) {
	showTimer := false
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			ShowTimer:     &showTimer,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.cfg.ShowTimer != false {
		t.Error("ShowTimer should be false when configured as false")
	}
}

func TestParseConfig_ShowSubagent(t *testing.T) {
	showSubagent := false
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			ShowSubagent:  &showSubagent,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.cfg.ShowSubagent != false {
		t.Error("ShowSubagent should be false when configured as false")
	}
}

func TestParseConfig_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Check defaults
	if w.cfg.ShowTimer != true {
		t.Error("ShowTimer should default to true")
	}
	if w.cfg.ShowSubagent != true {
		t.Error("ShowSubagent should default to true")
	}
	if w.cfg.SpriteSize != "medium" {
		t.Errorf("SpriteSize should default to 'medium', got %s", w.cfg.SpriteSize)
	}
	if w.cfg.IdleAnimations != true {
		t.Error("IdleAnimations should default to true")
	}
}

func TestOnStateChange_CelebrationTiming(t *testing.T) {
	successDuration := 5
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			Notify: &config.ClaudeCodeNotifyConfig{
				Success: &successDuration,
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Simulate transition to success
	w.lastStatus = StatusData{State: StateThinking}
	w.status = StatusData{State: StateSuccess}

	// Trigger celebration via Update (which calls onStateChange on state change)
	// We need to manually trigger the celebration logic
	now := time.Now()
	w.celebrateUntil = now.Add(time.Duration(successDuration) * time.Second)

	// Celebration should last for configured duration
	expectedEnd := now.Add(5 * time.Second)
	if w.celebrateUntil.Before(expectedEnd.Add(-100*time.Millisecond)) ||
		w.celebrateUntil.After(expectedEnd.Add(100*time.Millisecond)) {
		t.Errorf("celebrateUntil should be ~5 seconds from now, got %v", w.celebrateUntil.Sub(now))
	}
}

func TestOnStateChange_ActiveStateTimer(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Initial state
	w.lastStatus = StatusData{State: StateIdle}

	before := time.Now()
	w.onStateChange(StateThinking)
	after := time.Now()

	// activeStateStartTime should be set
	if w.activeStateStartTime.Before(before) || w.activeStateStartTime.After(after) {
		t.Error("activeStateStartTime should be set to current time on transition to active state")
	}
}

func TestOnStateChange_SleepyTransition(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Transition from active to idle
	w.lastStatus = StatusData{State: StateThinking}
	w.isSleepy = false

	w.onStateChange(StateIdle)

	if !w.isSleepy {
		t.Error("onStateChange() should set isSleepy to true when entering idle state")
	}
}

func TestOnStateChange_WakeUpTransition(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Transition from idle to active
	w.lastStatus = StatusData{State: StateIdle}
	w.isWakingUp = false

	w.onStateChange(StateThinking)

	if !w.isWakingUp {
		t.Error("onStateChange() should set isWakingUp to true when leaving idle state")
	}
}

func TestRender_WithTimer(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			Notify: &config.ClaudeCodeNotifyConfig{
				Thinking: config.IntPtr(-1),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set up thinking state with timer
	w.status = StatusData{State: StateThinking}
	w.shouldShow = true
	w.activeStateStartTime = time.Now().Add(-5 * time.Second)

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestRender_WithSubagent(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			Notify: &config.ClaudeCodeNotifyConfig{
				Tool: config.IntPtr(-1),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set up Task tool state (subagent)
	w.status = StatusData{State: StateToolRun, Tool: "Task"}
	w.shouldShow = true

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestRender_Celebration(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
			Notify: &config.ClaudeCodeNotifyConfig{
				Success: config.IntPtr(-1),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set up success state with celebration
	w.status = StatusData{State: StateSuccess}
	w.shouldShow = true
	w.celebrateUntil = time.Now().Add(5 * time.Second)

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestBubbleTypes(t *testing.T) {
	states := []struct {
		state    State
		wantType BubbleType
	}{
		{StateThinking, BubbleThought},
		{StateToolRun, BubbleSpeech},
		{StateSuccess, BubbleSpeech},
		{StateError, BubbleSpeech},
	}

	for _, tt := range states {
		t.Run(string(tt.state), func(t *testing.T) {
			got := getBubbleTypeForState(tt.state)
			if got != tt.wantType {
				t.Errorf("getBubbleTypeForState(%s) = %v, want %v", tt.state, got, tt.wantType)
			}
		})
	}
}

func TestDrawSprite_AllIdleSprites(t *testing.T) {
	sprites := []*ClawdSprite{
		&ClawdMediumLookLeft,
		&ClawdMediumLookRight,
		&ClawdLargeLookLeft,
		&ClawdLargeLookRight,
		&ClawdMediumSleepy1,
		&ClawdMediumSleepy2,
		&ClawdLargeSleepy1,
		&ClawdLargeSleepy2,
	}

	cfg := config.WidgetConfig{
		Type: "claude_code",
		ID:   "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		ClaudeCode: &config.ClaudeCodeConfig{
			IntroDuration: 0,
		},
	}

	w, _ := New(cfg)
	img := w.CreateCanvas()

	for _, sprite := range sprites {
		drawSprite(img, sprite, 0, 0)
	}
}
