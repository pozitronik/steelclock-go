// Package claudecode provides a widget displaying Claude Code status with the Clawd mascot.
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
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/webeditor"
	"github.com/pozitronik/steelclock-go/internal/widget"
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
	State       State     `json:"state"`
	Tool        string    `json:"tool,omitempty"`
	ToolPreview string    `json:"preview,omitempty"`
	Message     string    `json:"message,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Session     struct {
		StartedAt   time.Time `json:"started_at,omitempty"`
		ToolCalls   int       `json:"tool_calls,omitempty"`
		TokensUsed  int       `json:"tokens_used,omitempty"`
		TokensLimit int       `json:"tokens_limit,omitempty"`
	} `json:"session,omitempty"`
}

// Config holds widget configuration
type Config struct {
	DisplayMode    string `json:"display_mode"`    // "intro", "full", "compact", "minimal"
	ShowStats      bool   `json:"show_stats"`      // Show token/tool counts
	ShowToolIcon   bool   `json:"show_tool_icon"`  // Show tool-specific icon
	IntroOnStart   bool   `json:"intro_on_start"`  // Play intro animation on widget start
	IntroDurationS int    `json:"intro_duration"`  // Intro duration in seconds
	IdleAnimations bool   `json:"idle_animations"` // Enable idle animations (blinking, etc.)
}

// Widget displays Claude Code status with the Clawd mascot
type Widget struct {
	*widget.BaseWidget
	cfg          Config
	textRenderer *render.HorizontalTextRenderer

	// Status
	status     StatusData
	lastStatus StatusData
	statusMu   sync.RWMutex

	// Animation state
	animFrame      int
	blinkCountdown int
	idleVariant    int
	lastFrameTime  time.Time
	showingIntro   bool
	introStartTime time.Time

	// Celebration state
	celebrateUntil time.Time
	sparklePhase   int

	// Random for idle animations
	rng *rand.Rand
}

// New creates a new Claude Code status widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Parse configuration
	widgetCfg := parseConfig(cfg)

	// Create text renderer
	textSettings := helper.GetTextSettings()
	fontFace, err := bitmap.LoadFont(textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	textRenderer := render.NewHorizontalTextRenderer(render.HorizontalTextRendererConfig{
		FontFace:   fontFace,
		FontName:   textSettings.FontName,
		HorizAlign: textSettings.HorizAlign,
		VertAlign:  textSettings.VertAlign,
	})

	w := &Widget{
		BaseWidget:     base,
		cfg:            widgetCfg,
		textRenderer:   textRenderer,
		status:         StatusData{State: StateNotRunning},
		blinkCountdown: 50 + rand.Intn(100), // Random blink timing
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
		lastFrameTime:  time.Now(),
	}

	// Start with intro if configured
	if widgetCfg.IntroOnStart {
		w.showingIntro = true
		w.introStartTime = time.Now()
	}

	return w, nil
}

func parseConfig(cfg config.WidgetConfig) Config {
	c := Config{
		DisplayMode:    "full",
		ShowStats:      true,
		ShowToolIcon:   true,
		IntroOnStart:   true,
		IntroDurationS: 3,
		IdleAnimations: true,
	}

	if cfg.ClaudeCode != nil {
		if cfg.ClaudeCode.DisplayMode != "" {
			c.DisplayMode = cfg.ClaudeCode.DisplayMode
		}
		if cfg.ClaudeCode.ShowStats != nil {
			c.ShowStats = *cfg.ClaudeCode.ShowStats
		}
		if cfg.ClaudeCode.ShowToolIcon != nil {
			c.ShowToolIcon = *cfg.ClaudeCode.ShowToolIcon
		}
		if cfg.ClaudeCode.IntroOnStart != nil {
			c.IntroOnStart = *cfg.ClaudeCode.IntroOnStart
		}
		if cfg.ClaudeCode.IntroDuration > 0 {
			c.IntroDurationS = cfg.ClaudeCode.IntroDuration
		}
		if cfg.ClaudeCode.IdleAnimations != nil {
			c.IdleAnimations = *cfg.ClaudeCode.IdleAnimations
		}
	}

	return c
}

// Update reads the current Claude Code status from the web editor's in-memory store
func (w *Widget) Update() error {
	w.statusMu.Lock()
	defer w.statusMu.Unlock()

	// Save previous status for change detection
	w.lastStatus = w.status

	// Get status from the web editor's in-memory store
	// This is populated by POST requests to /api/claude-status
	httpStatus := webeditor.GetClaudeStatus()
	if httpStatus == nil {
		// No status available - Claude Code not running or no hooks configured
		w.status = StatusData{State: StateNotRunning}
		return nil
	}

	// Convert webeditor status to widget status
	w.status = StatusData{
		State:       State(httpStatus.State),
		Tool:        httpStatus.Tool,
		ToolPreview: httpStatus.ToolPreview,
		Message:     httpStatus.Message,
		Timestamp:   httpStatus.Timestamp,
	}
	w.status.Session.StartedAt = httpStatus.Session.StartedAt
	w.status.Session.ToolCalls = httpStatus.Session.ToolCalls
	w.status.Session.TokensUsed = httpStatus.Session.TokensUsed
	w.status.Session.TokensLimit = httpStatus.Session.TokensLimit

	// Trigger celebration on transition from tool run to idle (task completed)
	if w.lastStatus.State == StateToolRun && w.status.State == State("idle") {
		w.celebrateUntil = time.Now().Add(2 * time.Second)
	}

	return nil
}

// Render draws Clawd and status information
func (w *Widget) Render() (image.Image, error) {
	if w.ShouldHide() {
		return nil, nil
	}

	img := w.CreateCanvas()
	w.ApplyBorder(img)

	now := time.Now()
	deltaTime := now.Sub(w.lastFrameTime)
	w.lastFrameTime = now

	// Update animation frame
	w.animFrame++
	w.updateIdleAnimation(deltaTime)

	// Check if we should show intro
	if w.showingIntro {
		introDuration := time.Duration(w.cfg.IntroDurationS) * time.Second
		if now.Sub(w.introStartTime) < introDuration {
			w.renderIntro(img)
			return img, nil
		}
		w.showingIntro = false
	}

	// Get current status
	w.statusMu.RLock()
	status := w.status
	celebrating := now.Before(w.celebrateUntil)
	w.statusMu.RUnlock()

	// Choose render mode
	switch w.cfg.DisplayMode {
	case "minimal":
		w.renderMinimal(img, status, celebrating)
	case "compact":
		w.renderCompact(img, status, celebrating)
	default: // "full"
		w.renderFull(img, status, celebrating)
	}

	return img, nil
}

func (w *Widget) updateIdleAnimation(dt time.Duration) {
	if !w.cfg.IdleAnimations {
		return
	}

	// Blink countdown
	w.blinkCountdown--
	if w.blinkCountdown <= 0 {
		// Trigger blink
		w.blinkCountdown = 50 + w.rng.Intn(150) // Next blink in 50-200 frames
		w.idleVariant = 1                        // Blink frame
	} else if w.blinkCountdown > 45 {
		w.idleVariant = 0 // Normal frame
	}
}

// renderIntro renders the intro animation with large Clawd
func (w *Widget) renderIntro(img *image.Gray) {
	pos := w.GetPosition()
	elapsed := time.Since(w.introStartTime)
	duration := time.Duration(w.cfg.IntroDurationS) * time.Second

	// Choose sprite based on animation phase
	var sprite *ClawdSprite
	phase := float64(elapsed) / float64(duration)

	if phase < 0.3 {
		// Fade in / appear
		sprite = &ClawdLargeIdle
	} else if phase < 0.7 {
		// Wave animation
		waveFrame := (w.animFrame / 8) % 2
		if waveFrame == 0 {
			sprite = &ClawdLargeIdle
		} else {
			sprite = &ClawdLargeWave
		}
	} else {
		// Return to idle before transition
		sprite = &ClawdLargeIdle
	}

	// Center the sprite
	x := (pos.W - sprite.Width) / 2
	y := (pos.H - sprite.Height) / 2

	drawSprite(img, sprite, x, y)
}

// renderFull renders the full display mode with Clawd, status, and stats
func (w *Widget) renderFull(img *image.Gray, status StatusData, celebrating bool) {
	pos := w.GetPosition()

	// Get appropriate Clawd sprite
	sprite := w.getClawdSprite(status.State, celebrating)

	// Draw Clawd on the left
	clawdX := 2
	clawdY := (pos.H - sprite.Height) / 2
	drawSprite(img, sprite, clawdX, clawdY)

	// Draw thinking dots if thinking
	if status.State == StateThinking {
		dotsFrame := (w.animFrame / 6) % len(ThinkingDots)
		dotsSprite := &ThinkingDots[dotsFrame]
		drawSprite(img, dotsSprite, clawdX+sprite.Width+1, clawdY-2)
	}

	// Draw sleeping Zs if not running
	if status.State == StateNotRunning {
		zsFrame := (w.animFrame / 12) % len(SleepingZs)
		zsSprite := &SleepingZs[zsFrame]
		drawSprite(img, zsSprite, clawdX+sprite.Width-2, clawdY-4)
	}

	// Draw celebration sparkles
	if celebrating {
		w.sparklePhase = (w.sparklePhase + 1) % (len(Sparkles) * 3)
		sparkleIdx := w.sparklePhase / 3
		sparkle := &Sparkles[sparkleIdx]
		// Draw sparkles around Clawd
		drawSprite(img, sparkle, clawdX-2, clawdY-2)
		drawSprite(img, sparkle, clawdX+sprite.Width, clawdY-2)
	}

	// Text area starts after Clawd
	textX := clawdX + sprite.Width + 4
	textWidth := pos.W - textX - 2

	// Draw status text
	statusText := w.getStatusText(status)
	bounds := image.Rect(textX, 2, textX+textWidth, 12)
	w.textRenderer.Render(img, statusText, 0, bounds)

	// Draw tool info or secondary text
	if status.State == StateToolRun && status.Tool != "" {
		// Draw tool icon if enabled
		if w.cfg.ShowToolIcon {
			if icon := GetToolIcon(status.Tool); icon != nil {
				iconY := 14
				drawSprite(img, icon, textX, iconY)
				textX += icon.Width + 2
			}
		}

		// Draw tool preview
		preview := status.Tool
		if status.ToolPreview != "" {
			preview = truncateString(status.ToolPreview, 20)
		}
		bounds = image.Rect(textX, 14, pos.W-2, 24)
		w.textRenderer.Render(img, preview, 0, bounds)
	}

	// Draw stats at bottom if enabled
	if w.cfg.ShowStats && status.Session.ToolCalls > 0 {
		stats := fmt.Sprintf("%d tools", status.Session.ToolCalls)
		if status.Session.TokensUsed > 0 {
			stats = fmt.Sprintf("%dK | %s", status.Session.TokensUsed/1000, stats)
		}
		bounds = image.Rect(textX, pos.H-10, pos.W-2, pos.H-2)
		w.textRenderer.Render(img, stats, 0, bounds)
	}
}

// renderCompact renders a compact display
func (w *Widget) renderCompact(img *image.Gray, status StatusData, celebrating bool) {
	pos := w.GetPosition()

	// Small Clawd on left
	sprite := w.getSmallClawdSprite(status.State, celebrating)
	clawdY := (pos.H - sprite.Height) / 2
	drawSprite(img, sprite, 2, clawdY)

	// Status text on right
	statusText := w.getShortStatusText(status)
	bounds := image.Rect(12, 2, pos.W-2, pos.H-2)
	w.textRenderer.Render(img, statusText, 0, bounds)
}

// renderMinimal renders just a tiny Clawd indicator
func (w *Widget) renderMinimal(img *image.Gray, status StatusData, celebrating bool) {
	pos := w.GetPosition()

	sprite := w.getSmallClawdSprite(status.State, celebrating)
	x := (pos.W - sprite.Width) / 2
	y := (pos.H - sprite.Height) / 2
	drawSprite(img, sprite, x, y)

	// Add thinking dots for thinking state
	if status.State == StateThinking {
		dotsFrame := (w.animFrame / 6) % len(ThinkingDots)
		dotsSprite := &ThinkingDots[dotsFrame]
		drawSprite(img, dotsSprite, x+sprite.Width+1, y-1)
	}
}

func (w *Widget) getClawdSprite(state State, celebrating bool) *ClawdSprite {
	if celebrating {
		return &ClawdMediumHappy
	}

	switch state {
	case StateNotRunning:
		return &ClawdMediumSleeping
	case StateThinking:
		return &ClawdMediumThinking
	case StateToolRun:
		return &ClawdMediumWorking
	case StateSuccess:
		return &ClawdMediumHappy
	case StateError:
		return &ClawdMediumSad
	default: // StateIdle
		return &ClawdMediumIdle
	}
}

func (w *Widget) getSmallClawdSprite(state State, celebrating bool) *ClawdSprite {
	if celebrating {
		return &ClawdSmallHappy
	}

	switch state {
	case StateThinking:
		return &ClawdSmallThinking
	case StateToolRun:
		return &ClawdSmallWorking
	case StateError:
		return &ClawdSmallSad
	case StateSuccess:
		return &ClawdSmallHappy
	default:
		return &ClawdSmallIdle
	}
}

func (w *Widget) getStatusText(status StatusData) string {
	switch status.State {
	case StateNotRunning:
		return "Zzz... (sleeping)"
	case StateIdle:
		return "Ready!"
	case StateThinking:
		return "Thinking..."
	case StateToolRun:
		return "Working..."
	case StateSuccess:
		return "Done!"
	case StateError:
		return "Oops!"
	default:
		return "..."
	}
}

func (w *Widget) getShortStatusText(status StatusData) string {
	switch status.State {
	case StateNotRunning:
		return "Zzz"
	case StateIdle:
		return "Ready"
	case StateThinking:
		return "Hmm..."
	case StateToolRun:
		if status.Tool != "" {
			return status.Tool
		}
		return "Working"
	case StateSuccess:
		return "Done!"
	case StateError:
		return "Error"
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
