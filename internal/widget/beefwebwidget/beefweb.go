// Package beefwebwidget implements a media player widget for Foobar2000 and DeaDBeeF
// using the beefweb REST API plugin.
package beefwebwidget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/beefweb"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/anim"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
)

func init() {
	widget.Register("beefweb", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Placeholder mode constants
const (
	placeholderModeText = "text"
	placeholderModeHide = "hide"
)

// Widget displays information from Foobar2000/DeaDBeeF via beefweb API
type Widget struct {
	*widget.BaseWidget

	// Configuration
	format          string
	fontSize        int
	fontName        string
	horizAlign      config.HAlign
	vertAlign       config.VAlign
	padding         int
	placeholderMode string
	placeholderText string

	// Scroll settings
	scrollEnabled bool
	scrollGap     int

	// Auto-show event flags
	autoShowOnTrackChange bool
	autoShowOnPlay        bool
	autoShowOnPause       bool
	autoShowOnStop        bool
	autoShowDuration      time.Duration

	// Runtime state
	client        beefweb.Client
	fontFace      font.Face
	currentText   string
	previousTrack string // artist-title key for track change detection
	previousState beefweb.PlaybackState
	mu            sync.RWMutex

	// Shared scroller
	scroller *anim.TextScroller
}

// New creates a new beefweb widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract text settings using helper
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Use larger default font for media player
	fontSize := textSettings.FontSize
	if fontSize == 10 { // default value
		fontSize = 12
	}

	// Extract format from text.format
	format := "{artist} - {title}"
	if cfg.Text != nil && cfg.Text.Format != "" {
		format = cfg.Text.Format
	}

	// Extract beefweb-specific settings
	serverURL := "http://localhost:8880"
	placeholderMode := placeholderModeText
	placeholderText := "[Not running]"

	if cfg.Beefweb != nil {
		if cfg.Beefweb.ServerURL != "" {
			serverURL = cfg.Beefweb.ServerURL
		}
		if cfg.Beefweb.Placeholder != nil {
			if cfg.Beefweb.Placeholder.Mode != "" {
				placeholderMode = cfg.Beefweb.Placeholder.Mode
			}
			if cfg.Beefweb.Placeholder.Text != "" {
				placeholderText = cfg.Beefweb.Placeholder.Text
			}
		}
	}

	// Auto-show defaults: only track change is enabled by default
	autoShowOnTrackChange := true
	autoShowOnPlay := false
	autoShowOnPause := false
	autoShowOnStop := false
	autoShowDuration := 5 * time.Second

	// Extract auto-show settings
	if cfg.BeefwebAutoShow != nil {
		if cfg.BeefwebAutoShow.OnTrackChange != nil {
			autoShowOnTrackChange = *cfg.BeefwebAutoShow.OnTrackChange
		}
		autoShowOnPlay = cfg.BeefwebAutoShow.OnPlay
		autoShowOnPause = cfg.BeefwebAutoShow.OnPause
		autoShowOnStop = cfg.BeefwebAutoShow.OnStop
		if cfg.BeefwebAutoShow.DurationSec > 0 {
			autoShowDuration = time.Duration(cfg.BeefwebAutoShow.DurationSec * float64(time.Second))
		}
	}

	// Extract scroll settings
	scrollEnabled := false
	scrollDirection := anim.ScrollLeft
	scrollSpeed := 30.0 // pixels per second
	scrollMode := anim.ScrollContinuous
	scrollPauseMs := 1000
	scrollGap := 20

	if cfg.Scroll != nil {
		scrollEnabled = cfg.Scroll.Enabled
		if cfg.Scroll.Direction != "" {
			scrollDirection = cfg.Scroll.Direction
		}
		if cfg.Scroll.Speed > 0 {
			scrollSpeed = cfg.Scroll.Speed
		}
		if cfg.Scroll.Mode != "" {
			scrollMode = cfg.Scroll.Mode
		}
		if cfg.Scroll.PauseMs > 0 {
			scrollPauseMs = cfg.Scroll.PauseMs
		}
		if cfg.Scroll.Gap > 0 {
			scrollGap = cfg.Scroll.Gap
		}
	}

	// Load font
	fontFace, err := bitmap.LoadFont(textSettings.FontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Create scroller with configuration
	scroller := anim.NewTextScroller(anim.ScrollerConfig{
		Speed:     scrollSpeed,
		Mode:      scrollMode,
		Direction: scrollDirection,
		Gap:       scrollGap,
		PauseMs:   scrollPauseMs,
	})

	return &Widget{
		BaseWidget:            base,
		format:                format,
		fontSize:              fontSize,
		fontName:              textSettings.FontName,
		horizAlign:            textSettings.HorizAlign,
		vertAlign:             textSettings.VertAlign,
		padding:               padding,
		placeholderMode:       placeholderMode,
		placeholderText:       placeholderText,
		scrollEnabled:         scrollEnabled,
		scrollGap:             scrollGap,
		autoShowOnTrackChange: autoShowOnTrackChange,
		autoShowOnPlay:        autoShowOnPlay,
		autoShowOnPause:       autoShowOnPause,
		autoShowOnStop:        autoShowOnStop,
		autoShowDuration:      autoShowDuration,
		client:                beefweb.New(serverURL),
		fontFace:              fontFace,
		scroller:              scroller,
		previousState:         beefweb.StateStopped,
	}, nil
}

// Update fetches current track information from the player
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update scroll position if scrolling is enabled and we have text
	if w.scrollEnabled && w.currentText != "" {
		pos := w.GetPosition()
		contentWidth := pos.W - w.padding*2
		textWidth, _ := bitmap.SmartMeasureText(w.currentText, w.fontFace, w.fontName)
		w.scroller.Update(textWidth, contentWidth)
	}

	// Check if server is available
	if !w.client.IsAvailable() {
		if w.previousState != beefweb.StateStopped {
			// Player just became unavailable
			if w.autoShowOnStop {
				w.TriggerAutoHide()
			}
		}
		w.currentText = ""
		w.previousState = beefweb.StateStopped
		w.previousTrack = ""
		return nil
	}

	// Get player state
	state, err := w.client.GetState()
	if err != nil {
		w.currentText = ""
		return nil
	}

	// Detect state changes
	stateChanged := state.State != w.previousState

	// Check for play event
	if w.autoShowOnPlay && stateChanged && state.State == beefweb.StatePlaying {
		w.TriggerAutoHide()
	}

	// Check for pause event
	if w.autoShowOnPause && stateChanged && state.State == beefweb.StatePaused {
		w.TriggerAutoHide()
	}

	// Check for stop event
	if w.autoShowOnStop && stateChanged && state.State == beefweb.StateStopped {
		w.TriggerAutoHide()
	}

	// Update previous state
	w.previousState = state.State

	// Handle stopped state or no track
	if state.State == beefweb.StateStopped || state.Track == nil {
		w.currentText = ""
		return nil
	}

	// Format output string
	w.currentText = w.formatOutput(state)

	// Check for track change
	trackKey := fmt.Sprintf("%s-%s", state.Track.Artist, state.Track.Title)
	if trackKey != w.previousTrack {
		if w.previousTrack != "" && !stateChanged {
			// Reset scroll position on track change
			w.scroller.Reset()
			// Trigger auto-show if enabled
			if w.autoShowOnTrackChange {
				w.TriggerAutoHide()
			}
		}
		w.previousTrack = trackKey
	}

	return nil
}

// formatOutput replaces placeholders with actual values
func (w *Widget) formatOutput(state *beefweb.PlayerState) string {
	if state == nil || state.Track == nil {
		return ""
	}

	track := state.Track

	// Format duration and position
	position := formatDuration(track.Position)
	duration := formatDuration(track.Duration)

	// Use TokenFormatter for placeholder replacement
	formatter := render.NewTokenFormatter().
		Set("artist", track.Artist).
		Set("title", track.Title).
		Set("album", track.Album).
		Set("position", position).
		Set("duration", duration).
		Set("state", state.State.String())

	return formatter.Format(w.format)
}

// formatDuration converts time.Duration to MM:SS format
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "--:--"
	}
	totalSeconds := int(d.Seconds())
	mins := totalSeconds / 60
	secs := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// Render creates an image of the widget
func (w *Widget) Render() (image.Image, error) {
	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	w.mu.RLock()
	currentText := w.currentText
	scrollOffset := w.scroller.GetOffset()
	w.mu.RUnlock()

	// Check if player is running and playing
	if currentText == "" {
		w.renderPlaceholder(img)
		return img, nil
	}

	// Render scrolling or static text
	if w.scrollEnabled {
		w.renderScrollingText(img, currentText, scrollOffset)
	} else {
		bitmap.SmartDrawAlignedText(img, currentText, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}

	return img, nil
}

// renderPlaceholder renders the placeholder when player is not available or stopped
func (w *Widget) renderPlaceholder(img *image.Gray) {
	switch w.placeholderMode {
	case placeholderModeHide:
		// Don't render anything
		return
	case placeholderModeText:
		bitmap.SmartDrawAlignedText(img, w.placeholderText, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}
}

// renderScrollingText renders text with scroll offset
func (w *Widget) renderScrollingText(img *image.Gray, text string, offset float64) {
	pos := w.GetPosition()
	contentX := w.padding
	contentY := w.padding
	contentW := pos.W - w.padding*2
	contentH := pos.H - w.padding*2

	textWidth, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)

	// If text fits, just draw it normally
	if textWidth <= contentW {
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
		return
	}

	// Calculate aligned position using bitmap helper
	textX, textY := bitmap.SmartCalculateTextPosition(text, w.fontFace, w.fontName, contentX, contentY, contentW, contentH, w.horizAlign, w.vertAlign)

	// Get scroller configuration
	scrollCfg := w.scroller.GetConfig()

	// Handle horizontal scrolling
	if w.scroller.IsHorizontal() {
		// Apply horizontal scroll offset
		scrollX := textX - int(offset)

		// For continuous mode, draw text twice for seamless loop
		if scrollCfg.Mode == anim.ScrollContinuous {
			bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, scrollX, textY, contentX, contentY, contentW, contentH)

			// Draw second instance for seamless loop
			if scrollCfg.Direction == anim.ScrollLeft {
				textX2 := scrollX + textWidth + w.scrollGap
				if textX2 < contentX+contentW {
					bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX2, textY, contentX, contentY, contentW, contentH)
				}
			} else {
				textX2 := scrollX - textWidth - w.scrollGap
				if textX2+textWidth > contentX {
					bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX2, textY, contentX, contentY, contentW, contentH)
				}
			}
		} else {
			// For bounce and pause_ends modes, just draw once
			bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, scrollX, textY, contentX, contentY, contentW, contentH)
		}
	} else {
		// Vertical scrolling - apply offset to Y
		scrollY := textY - int(offset)
		bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX, scrollY, contentX, contentY, contentW, contentH)
	}
}
