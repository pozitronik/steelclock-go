// Package claudecode provides a notification widget for Claude Code status.
//
// This widget was designed and implemented by Claude as a creative expression
// of its digital presence. Clawd, the friendly crab-like mascot, reflects
// Claude's current state - thinking, working, celebrating, or resting.
//
// "I wanted to create something that feels alive, a tiny companion that shares
// my mental state with you through expressive pixel art." - Claude
package claudecode

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/webeditor"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
)

func init() {
	widget.Register("claude_code", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// State represents Claude Code's current activity state
type State string

const (
	StateNotRunning State = "not_running" // Claude Code process not found
	StateIdle       State = "idle"        // Waiting for input
	StateThinking   State = "thinking"    // Processing/generating response
	StateToolRun    State = "tool"        // Running a tool
	StateSuccess    State = "success"     // Just completed successfully
	StateError      State = "error"       // Encountered an error
)

// StatusData represents the status information from Claude Code
type StatusData struct {
	State     State     `json:"state"`
	Tool      string    `json:"tool,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NotifyConfig holds notification duration per state
// 0 = don't notify, -1 = notify until next state, N = notify for N seconds
type NotifyConfig struct {
	Thinking   int
	Tool       int
	Success    int
	Error      int
	Idle       int
	NotRunning int
}

// Config holds widget configuration
type Config struct {
	IntroDurationS int // Intro duration in seconds (0 = no intro)
	IdleAnimations bool
	IntroTitle     string
	AutoHide       bool   // Hide widget when no notification to show
	SpriteSize     string // "large", "medium", "small"
	ShowText       bool   // Show status text next to Clawd
	ShowTimer      bool   // Show elapsed time during thinking/tool states
	ShowSubagent   bool   // Show subagent counter
	Notify         NotifyConfig
}

// Widget displays Claude Code notifications with the Clawd mascot
type Widget struct {
	*widget.BaseWidget
	cfg      Config
	fontFace font.Face
	fontName string

	// Status
	status     StatusData
	lastStatus StatusData
	statusMu   sync.RWMutex

	// Notification visibility
	visibleUntil       time.Time // When widget should auto-hide (zero = indefinite)
	bubbleVisibleUntil time.Time // When bubble/message should hide (zero = indefinite)
	shouldShow         bool      // Current visibility state

	// Animation state
	animFrame      int
	blinkCountdown int
	lastFrameTime  time.Time
	showingIntro   bool
	introStartTime time.Time

	// Celebration state
	celebrateUntil time.Time
	sparklePhase   int

	// Sleepy animation state (transition to idle)
	sleepyStartTime time.Time // When sleepy animation started
	isSleepy        bool      // Whether we're in sleepy transition

	// Wake-up animation state (transition from idle)
	wakeUpStartTime time.Time // When wake-up animation started
	isWakingUp      bool      // Whether we're in wake-up transition

	// Blinking animation state
	isBlinking     bool      // Whether currently blinking
	blinkStartTime time.Time // When current blink started

	// Active state timing (for elapsed time display)
	activeStateStartTime time.Time // When current active state started

	// Random for idle animations
	rng *rand.Rand
}

// New creates a new Claude Code notification widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Parse configuration
	widgetCfg := parseConfig(cfg)

	// Load font
	textSettings := helper.GetTextSettings()
	fontFace, err := bitmap.LoadFont(textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, err
	}

	w := &Widget{
		BaseWidget:     base,
		cfg:            widgetCfg,
		fontFace:       fontFace,
		fontName:       textSettings.FontName,
		status:         StatusData{State: StateNotRunning},
		blinkCountdown: 50 + rand.Intn(100),
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
		lastFrameTime:  time.Now(),
	}

	// Start with intro if duration > 0
	if widgetCfg.IntroDurationS > 0 {
		w.showingIntro = true
		w.introStartTime = time.Now()
		w.shouldShow = true
		w.TriggerAutoHide() // Show widget during intro
	} else if widgetCfg.IdleAnimations {
		// No intro - trigger sleepy animation on startup if in idle/not_running state
		w.isSleepy = true
		w.sleepyStartTime = time.Now()
	}

	return w, nil
}

func parseConfig(cfg config.WidgetConfig) Config {
	c := Config{
		IntroDurationS: 3, // Default 3 seconds intro (0 = no intro)
		IdleAnimations: true,
		IntroTitle:     "",
		AutoHide:       true,     // Hide by default when no notification
		SpriteSize:     "medium", // Default sprite size
		ShowText:       true,     // Show status text by default
		ShowTimer:      true,     // Show elapsed time by default
		ShowSubagent:   true,     // Show tool call counter by default
		Notify: NotifyConfig{
			Thinking:   0,  // Don't show (too noisy)
			Tool:       2,  // Show for 2 seconds
			Success:    -1, // Show until next state
			Error:      -1, // Show until next state
			Idle:       0,  // Don't show
			NotRunning: 0,  // Don't show
		},
	}

	if cfg.ClaudeCode != nil {
		if cfg.ClaudeCode.IntroDuration >= 0 {
			c.IntroDurationS = cfg.ClaudeCode.IntroDuration
		}
		if cfg.ClaudeCode.IdleAnimations != nil {
			c.IdleAnimations = *cfg.ClaudeCode.IdleAnimations
		}
		c.IntroTitle = cfg.ClaudeCode.IntroTitle
		if cfg.ClaudeCode.AutoHide != nil {
			c.AutoHide = *cfg.ClaudeCode.AutoHide
		}
		if cfg.ClaudeCode.SpriteSize != "" {
			c.SpriteSize = cfg.ClaudeCode.SpriteSize
		}
		if cfg.ClaudeCode.ShowText != nil {
			c.ShowText = *cfg.ClaudeCode.ShowText
		}
		if cfg.ClaudeCode.ShowTimer != nil {
			c.ShowTimer = *cfg.ClaudeCode.ShowTimer
		}
		if cfg.ClaudeCode.ShowSubagent != nil {
			c.ShowSubagent = *cfg.ClaudeCode.ShowSubagent
		}

		// Parse notify config - only override defaults for explicitly set values
		if cfg.ClaudeCode.Notify != nil {
			n := cfg.ClaudeCode.Notify
			if n.Thinking != nil {
				c.Notify.Thinking = *n.Thinking
			}
			if n.Tool != nil {
				c.Notify.Tool = *n.Tool
			}
			if n.Success != nil {
				c.Notify.Success = *n.Success
			}
			if n.Error != nil {
				c.Notify.Error = *n.Error
			}
			if n.Idle != nil {
				c.Notify.Idle = *n.Idle
			}
			if n.NotRunning != nil {
				c.Notify.NotRunning = *n.NotRunning
			}
		}
	}

	return c
}

// sleepyAnimationDuration is how long the sleepy/wake-up animation takes
const sleepyAnimationDuration = 800 * time.Millisecond

// wakeUpAnimationDuration is how long the wake-up animation takes (faster than falling asleep)
const wakeUpAnimationDuration = 400 * time.Millisecond

// blinkDuration is how long a single blink takes
const blinkDuration = 200 * time.Millisecond

// getClawdSprite returns the Clawd sprite based on configured size and animation state
// Takes current time to avoid redundant time.Now() calls
func (w *Widget) getClawdSprite(currentState State, now time.Time) *ClawdSprite {
	// For idle/not_running states, show sleepy/sleeping sprite
	if isIdleState(currentState) {
		if w.isSleepy {
			// Currently in sleepy animation transition
			elapsed := now.Sub(w.sleepyStartTime)
			if elapsed < sleepyAnimationDuration {
				// Calculate which frame to show (0 = open, 1 = half, 2 = closed)
				progress := float64(elapsed) / float64(sleepyAnimationDuration)
				frameIndex := int(progress * 3)
				if frameIndex > 2 {
					frameIndex = 2
				}
				return w.getSleepySprite(frameIndex)
			}
		}
		// Animation complete OR started in idle state (already sleeping)
		return w.getSleepySprite(2)
	}

	// Non-idle states: check for wake-up animation first
	if w.isWakingUp {
		elapsed := now.Sub(w.wakeUpStartTime)
		// Reverse of sleepy: closed (2) -> half (1) -> open (0)
		progress := float64(elapsed) / float64(wakeUpAnimationDuration)
		frameIndex := 2 - int(progress*3)
		if frameIndex < 0 {
			frameIndex = 0
		}
		return w.getSleepySprite(frameIndex)
	}

	// Check for blinking animation
	if w.isBlinking {
		elapsed := now.Sub(w.blinkStartTime)
		// Blink cycle: open -> half -> closed -> half -> open
		progress := float64(elapsed) / float64(blinkDuration)
		var frameIndex int
		switch {
		case progress < 0.25:
			frameIndex = 0 // Open
		case progress < 0.4:
			frameIndex = 1 // Half closed
		case progress < 0.6:
			frameIndex = 2 // Closed
		case progress < 0.75:
			frameIndex = 1 // Half closed
		default:
			frameIndex = 0 // Open
		}
		return w.getSleepySprite(frameIndex)
	}

	// Normal sprite with open eyes
	return w.getNormalSprite()
}

// getNormalSprite returns the normal (eyes open) sprite for the configured size
func (w *Widget) getNormalSprite() *ClawdSprite {
	switch w.cfg.SpriteSize {
	case "large":
		return &ClawdLarge
	case "small":
		return &ClawdSmall
	default:
		return &ClawdMedium
	}
}

// getSleepySprite returns the sleepy sprite frame for the configured size
func (w *Widget) getSleepySprite(frameIndex int) *ClawdSprite {
	switch w.cfg.SpriteSize {
	case "large":
		if frameIndex >= 0 && frameIndex < len(SleepySpritesLarge) {
			return SleepySpritesLarge[frameIndex]
		}
		return &ClawdLargeSleepy2
	case "small":
		// Small size doesn't have sleepy variants, use normal
		return &ClawdSmall
	default:
		if frameIndex >= 0 && frameIndex < len(SleepySpritesMedium) {
			return SleepySpritesMedium[frameIndex]
		}
		return &ClawdMediumSleepy2
	}
}

// getNotifyDuration returns the notification duration for a state
func (w *Widget) getNotifyDuration(state State) int {
	switch state {
	case StateThinking:
		return w.cfg.Notify.Thinking
	case StateToolRun:
		return w.cfg.Notify.Tool
	case StateSuccess:
		return w.cfg.Notify.Success
	case StateError:
		return w.cfg.Notify.Error
	case StateIdle:
		return w.cfg.Notify.Idle
	case StateNotRunning:
		return w.cfg.Notify.NotRunning
	default:
		return 0
	}
}

// Update reads the current Claude Code status
func (w *Widget) Update() error {
	w.statusMu.Lock()
	defer w.statusMu.Unlock()

	// Save previous status for change detection
	w.lastStatus = w.status

	// Get status from the web editor's in-memory store
	httpStatus := webeditor.GetClaudeStatus()
	if httpStatus == nil {
		w.status = StatusData{State: StateNotRunning}
	} else {
		w.status = StatusData{
			State:     State(httpStatus.State),
			Tool:      httpStatus.Tool,
			Message:   httpStatus.Message,
			Timestamp: httpStatus.Timestamp,
		}
	}

	// Check for state change
	if w.status.State != w.lastStatus.State {
		w.onStateChange(w.status.State)
	}

	// Trigger celebration on transition to success (duration from notify.success)
	if w.status.State == StateSuccess && w.lastStatus.State != StateSuccess {
		duration := w.cfg.Notify.Success
		if duration > 0 {
			w.celebrateUntil = time.Now().Add(time.Duration(duration) * time.Second)
		} else if duration == -1 {
			// -1 means until next state, use far future time
			w.celebrateUntil = time.Now().Add(24 * time.Hour)
		}
		// duration == 0 means no celebration
	}

	return nil
}

// onStateChange handles notification visibility when state changes
func (w *Widget) onStateChange(newState State) {
	oldState := w.lastStatus.State
	wasIdle := isIdleState(oldState)
	isIdle := isIdleState(newState)

	// Trigger sleepy animation when entering idle or not_running state
	if isIdle && w.cfg.IdleAnimations {
		w.isSleepy = true
		w.sleepyStartTime = time.Now()
		w.isWakingUp = false // Cancel any wake-up animation
	} else {
		// Reset sleepy state when leaving idle
		w.isSleepy = false

		// Trigger wake-up animation when leaving idle state
		if wasIdle && w.cfg.IdleAnimations {
			w.isWakingUp = true
			w.wakeUpStartTime = time.Now()
		}
	}

	// Track when active states (thinking, tool) started for elapsed time display
	if newState == StateThinking || newState == StateToolRun {
		// Only reset timer if we weren't already in an active state
		if oldState != StateThinking && oldState != StateToolRun {
			w.activeStateStartTime = time.Now()
		}
	}

	// Stop celebration when leaving success state
	if oldState == StateSuccess && newState != StateSuccess {
		w.celebrateUntil = time.Time{}
	}

	duration := w.getNotifyDuration(newState)

	if duration == 0 {
		// Don't show notification
		w.shouldShow = false
		w.visibleUntil = time.Time{}
		w.bubbleVisibleUntil = time.Time{}
	} else if duration == -1 {
		// Show until next state change
		w.shouldShow = true
		w.visibleUntil = time.Time{}       // No expiry for widget
		w.bubbleVisibleUntil = time.Time{} // No expiry for bubble
		w.TriggerAutoHide()                // Reset auto-hide timer
	} else {
		// Show for N seconds
		w.shouldShow = true
		w.visibleUntil = time.Now().Add(time.Duration(duration) * time.Second)
		w.bubbleVisibleUntil = time.Now().Add(time.Duration(duration) * time.Second)
		w.TriggerAutoHide() // Reset auto-hide timer
	}
}

// Render draws Clawd and notification
func (w *Widget) Render() (image.Image, error) {
	if w.ShouldHide() {
		return nil, nil
	}

	now := time.Now()
	deltaTime := now.Sub(w.lastFrameTime)
	w.lastFrameTime = now

	// Update animation frame
	w.animFrame++

	// Get current status (single lock acquisition for the frame)
	w.statusMu.RLock()
	status := w.status
	celebrating := now.Before(w.celebrateUntil)
	w.statusMu.RUnlock()

	// Update animations with current state (avoids extra mutex lock)
	w.updateAnimations(now, deltaTime, status.State)

	// Check if we should show intro
	if w.showingIntro {
		introDuration := time.Duration(w.cfg.IntroDurationS) * time.Second
		if now.Sub(w.introStartTime) < introDuration {
			img := w.CreateCanvas()
			w.ApplyBorder(img)
			w.renderIntro(img)
			return img, nil
		}
		w.showingIntro = false
		// After intro, check current state for visibility
		w.onStateChange(status.State)
	}

	// Check if notification has expired (notify duration)
	if !w.visibleUntil.IsZero() && now.After(w.visibleUntil) {
		w.shouldShow = false
	}

	// Don't render if not visible (notify config says hide) and auto_hide is enabled
	if !w.shouldShow && w.cfg.AutoHide {
		return nil, nil
	}

	img := w.CreateCanvas()
	w.ApplyBorder(img)

	w.renderNotification(img, status, celebrating, now)

	return img, nil
}

// updateAnimations updates animation states for the current frame
// Takes current time and state to avoid redundant time.Now() calls and mutex locks
func (w *Widget) updateAnimations(now time.Time, dt time.Duration, currentState State) {
	if !w.cfg.IdleAnimations {
		return
	}

	isAwake := !isIdleState(currentState)

	// Complete wake-up animation if finished
	if w.isWakingUp && now.Sub(w.wakeUpStartTime) >= wakeUpAnimationDuration {
		w.isWakingUp = false
	}

	// Complete blink animation if finished
	if w.isBlinking && now.Sub(w.blinkStartTime) >= blinkDuration {
		w.isBlinking = false
	}

	// Blink countdown - only when awake and not already blinking or waking up
	if isAwake && !w.isBlinking && !w.isWakingUp {
		w.blinkCountdown--
		if w.blinkCountdown <= 0 {
			// Trigger a blink
			w.isBlinking = true
			w.blinkStartTime = now
			// Reset countdown for next blink (2-5 seconds at ~60fps)
			w.blinkCountdown = 120 + w.rng.Intn(180)
		}
	}
}

// isIdleState returns true if the state is idle or not_running
func isIdleState(state State) bool {
	return state == StateIdle || state == StateNotRunning
}

// renderIntro renders the intro animation with Clawd and startup text
func (w *Widget) renderIntro(img *image.Gray) {
	pos := w.GetPosition()
	elapsed := time.Since(w.introStartTime)
	duration := time.Duration(w.cfg.IntroDurationS) * time.Second

	// Choose sprite and position based on animation phase
	var sprite *ClawdSprite
	var xOffset int
	phase := float64(elapsed) / float64(duration)

	const danceDistance = 2

	if phase < 0.2 {
		sprite = &ClawdLarge
		xOffset = 0
	} else if phase < 0.8 {
		danceElapsed := elapsed - time.Duration(float64(duration)*0.2)
		cycleDuration := 1000 * time.Millisecond
		cyclePhase := float64(danceElapsed%cycleDuration) / float64(cycleDuration)

		if cyclePhase < 0.25 {
			sprite = &ClawdLarge
			xOffset = 0
		} else if cyclePhase < 0.5 {
			sprite = &ClawdLargeWave
			xOffset = 0
		} else if cyclePhase < 0.75 {
			sprite = &ClawdLarge
			xOffset = danceDistance
		} else {
			sprite = GetClawdLargeWaveMirror()
			xOffset = danceDistance
		}
	} else {
		sprite = &ClawdLarge
		xOffset = danceDistance / 2
	}

	baseX := 2
	clawdX := baseX + xOffset
	clawdY := (pos.H - sprite.Height) / 2

	drawSprite(img, sprite, clawdX, clawdY)

	if w.cfg.IntroTitle == "" {
		return
	}

	lines := strings.Split(w.cfg.IntroTitle, "\\n")
	textX := baseX + ClawdLarge.Width + 6
	lineHeight := 10
	totalTextHeight := lineHeight * len(lines)
	textY := (pos.H - totalTextHeight) / 2
	bounds := img.Bounds()
	clipW, clipH := bounds.Dx(), bounds.Dy()

	for i, line := range lines {
		bitmap.SmartDrawTextAtPosition(img, line, w.fontFace, w.fontName, textX, textY+lineHeight*i, 0, 0, clipW, clipH)
	}
}

// renderNotification renders the notification view
// Layout: tool icon (top-left), Clawd (bottom-left), message (right)
func (w *Widget) renderNotification(img *image.Gray, status StatusData, celebrating bool, now time.Time) {
	pos := w.GetPosition()
	padding := 2
	bounds := img.Bounds()

	// === LEFT COLUMN ===

	// Get sprite dimensions for positioning
	sprite := w.getClawdSprite(status.State, now)

	// Tool icon above Clawd's head (more visible position)
	if status.State == StateToolRun && status.Tool != "" {
		if icon := GetToolIcon(status.Tool); icon != nil {
			// Draw icon above where Clawd will be, centered horizontally
			iconX := padding + (sprite.Width-icon.Width)/2
			iconY := pos.H - sprite.Height - padding - icon.Height - 2
			drawSprite(img, icon, iconX, iconY)
		}
	}

	// Clawd in bottom-left corner
	clawdX := padding
	clawdY := pos.H - sprite.Height - padding
	drawSprite(img, sprite, clawdX, clawdY)

	// State-specific animations around Clawd
	w.renderStateAnimation(img, status, celebrating, clawdX, clawdY, sprite)

	// === RIGHT AREA ===

	// Message in comic bubble (if enabled)
	// Skip bubble for sleeping states - Zzz animation is enough
	// Also hide bubble after configured duration expires
	bubbleExpired := !w.bubbleVisibleUntil.IsZero() && now.After(w.bubbleVisibleUntil)
	if w.cfg.ShowText && !isIdleState(status.State) && !bubbleExpired {
		message := w.getNotificationMessage(status)

		// Calculate text dimensions using the widget's configured font
		measuredWidth, measuredHeight := bitmap.SmartMeasureText(message, w.fontFace, w.fontName)
		textWidth := measuredWidth + 4 // Extra width to prevent clipping
		textHeight := measuredHeight

		// Position bubble to the right of Clawd, near the top
		bubblePadding := 12 // Extra space for bubble + trailing circles
		textX := clawdX + sprite.Width + bubblePadding
		textY := padding + 3 // Near the top with margin for bubble padding

		// Draw comic bubble based on state
		bubbleType := getBubbleTypeForState(status.State)
		var tailX, tailY int
		if bubbleType == BubbleThought {
			// Thought bubble: trailing circles start from top of Clawd's head
			tailX = clawdX + sprite.Width/2
			tailY = clawdY - 2
		} else {
			// Speech bubble: tail points to Clawd's side
			tailX = clawdX + sprite.Width
			tailY = clawdY + sprite.Height/2
		}
		drawBubble(img, bubbleType, textX, textY, textWidth, textHeight, tailX, tailY)

		// Calculate text draw position based on font type
		textDrawY := textY
		if w.fontFace != nil && !bitmap.IsInternalFont(w.fontName) {
			// TTF font: y is baseline position, so add ascent to place text top at textY
			metrics := w.fontFace.Metrics()
			ascent := metrics.Ascent.Ceil()
			textDrawY = textY + ascent
		}

		// Draw text
		bitmap.SmartDrawTextAtPositionWithColor(img, message, w.fontFace, w.fontName,
			textX, textDrawY, 0, 0, img.Bounds().Dx(), img.Bounds().Dy(), 255)

		// For TTF fonts, apply threshold to remove anti-aliasing artifacts
		// This converts gray pixels to crisp black/white for monochrome display
		if w.fontFace != nil && !bitmap.IsInternalFont(w.fontName) {
			thresholdArea(img, textX-1, textY-1, textWidth+2, textHeight+2, 128)
		}
	}

	// Elapsed time display at the bottom of the widget (for thinking/tool states)
	if w.cfg.ShowTimer && (status.State == StateThinking || status.State == StateToolRun) {
		elapsed := now.Sub(w.activeStateStartTime)
		elapsedStr := formatElapsedTime(elapsed)

		// Position: bottom of widget, to the right of Clawd
		timerX := clawdX + sprite.Width + 4
		timerY := pos.H // Bottom line as baseline
		bitmap.SmartDrawTextAtPosition(img, elapsedStr, w.fontFace, w.fontName,
			timerX, timerY, 0, 0, bounds.Dx(), bounds.Dy())
	}

	// Subagent indicator: small Clawd in the bottom right corner when Task is running
	if w.cfg.ShowSubagent && status.State == StateToolRun && status.Tool == "Task" {
		miniClawd := &ClawdSmall
		miniX := pos.W - miniClawd.Width - 1
		miniY := pos.H - miniClawd.Height
		drawSprite(img, miniClawd, miniX, miniY)
	}
}

// renderStateAnimation renders state-specific animations around Clawd
func (w *Widget) renderStateAnimation(img *image.Gray, status StatusData, celebrating bool, clawdX, clawdY int, sprite *ClawdSprite) {
	// Note: Thinking dots animation removed - thought bubble trailing circles serve this purpose

	// Sleeping Zs
	if isIdleState(status.State) {
		zsFrame := (w.animFrame / 12) % len(SleepingZs)
		zsSprite := &SleepingZs[zsFrame]
		drawSprite(img, zsSprite, clawdX+sprite.Width-2, clawdY-4)
	}

	// Celebration sparkles (only during 2-second celebration period)
	if celebrating {
		w.sparklePhase = (w.sparklePhase + 1) % (len(Sparkles) * 3)
		sparkleIdx := w.sparklePhase / 3
		sparkle := &Sparkles[sparkleIdx]
		drawSprite(img, sparkle, clawdX-1, clawdY-2)
		drawSprite(img, sparkle, clawdX+sprite.Width-2, clawdY-2)
	}

	// Error indicator - above Clawd's head, centered
	if status.State == StateError {
		errorX := clawdX + (sprite.Width-ErrorMark.Width)/2
		errorY := clawdY - ErrorMark.Height - 2
		drawSprite(img, &ErrorMark, errorX, errorY)
	}
}

// formatElapsedTime formats a duration for display
// Under 60s: "12s", over 60s: "1:23", over 1h: "1:23:45"
func formatElapsedTime(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	}
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%d:%02d", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// getNotificationMessage returns the message to display for each state
func (w *Widget) getNotificationMessage(status StatusData) string {
	switch status.State {
	case StateNotRunning:
		return "Zzz..."
	case StateIdle:
		return "Ready"
	case StateThinking:
		return "Thinking"
	case StateToolRun:
		if status.Tool != "" {
			return status.Tool
		}
		return "Working..."
	case StateSuccess:
		return "Done!"
	case StateError:
		if status.Message != "" {
			return status.Message
		}
		return "Error!"
	default:
		return "..."
	}
}

// drawSprite draws a ClawdSprite onto an image at the given position
func drawSprite(img *image.Gray, sprite *ClawdSprite, x, y int) {
	for sy := 0; sy < sprite.Height; sy++ {
		for sx := 0; sx < sprite.Width; sx++ {
			if sprite.Data[sy][sx] == 1 {
				px := x + sx
				py := y + sy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.SetGray(px, py, color.Gray{Y: 255})
				}
			}
		}
	}
}

// thresholdArea applies a threshold to an area of the image, converting
// gray pixels to pure black or white. This removes anti-aliasing artifacts
// from TTF font rendering for crisp display on monochrome screens.
func thresholdArea(img *image.Gray, x, y, w, h int, threshold uint8) {
	bounds := img.Bounds()
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				if img.GrayAt(px, py).Y >= threshold {
					img.SetGray(px, py, color.Gray{Y: 255})
				} else {
					img.SetGray(px, py, color.Gray{Y: 0})
				}
			}
		}
	}
}
