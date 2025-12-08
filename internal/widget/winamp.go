package widget

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"github.com/pozitronik/steelclock-go/internal/winamp"
	"golang.org/x/image/font"
)

func init() {
	Register("winamp", func(cfg config.WidgetConfig) (Widget, error) {
		return NewWinampWidget(cfg)
	})
}

// WinampWidget displays information from Winamp media player
type WinampWidget struct {
	*BaseWidget

	// Configuration
	format          string
	fontSize        int
	fontName        string
	horizAlign      string
	vertAlign       string
	padding         int
	placeholderMode string // "text" or "icon"
	placeholderText string

	// Scroll settings
	scrollEnabled bool
	scrollGap     int // gap between text repetitions (kept for rendering)

	// Auto-show event flags
	autoShowOnTrackChange bool
	autoShowOnPlay        bool
	autoShowOnPause       bool
	autoShowOnStop        bool
	autoShowOnSeek        bool

	// Runtime state
	client         winamp.Client
	fontFace       font.Face
	currentText    string
	previousTitle  string
	previousStatus winamp.PlaybackStatus
	previousPosMs  int // for seek detection
	mu             sync.RWMutex

	// Shared scroller
	scroller *shared.TextScroller
}

// NewWinampWidget creates a new Winamp widget
func NewWinampWidget(cfg config.WidgetConfig) (*WinampWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract text settings using helper
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Use larger default font for winamp
	fontSize := textSettings.FontSize
	if fontSize == 10 { // default value
		fontSize = 12
	}

	// Extract format from text.format (consistent with other widgets)
	format := "{title}"
	if cfg.Text != nil && cfg.Text.Format != "" {
		format = cfg.Text.Format
	}

	// Extract Winamp-specific settings (placeholder)
	placeholderMode := "icon"
	placeholderText := "No Winamp"

	if cfg.Winamp != nil {
		if cfg.Winamp.Placeholder != nil {
			if cfg.Winamp.Placeholder.Mode != "" {
				placeholderMode = cfg.Winamp.Placeholder.Mode
			}
			if cfg.Winamp.Placeholder.Text != "" {
				placeholderText = cfg.Winamp.Placeholder.Text
			}
		}
	}

	// Auto-show defaults: only track change is enabled by default
	autoShowOnTrackChange := true
	autoShowOnPlay := false
	autoShowOnPause := false
	autoShowOnStop := false
	autoShowOnSeek := false

	// Extract auto-show settings from root level
	if cfg.AutoShow != nil {
		if cfg.AutoShow.OnTrackChange != nil {
			autoShowOnTrackChange = *cfg.AutoShow.OnTrackChange
		}
		autoShowOnPlay = cfg.AutoShow.OnPlay
		autoShowOnPause = cfg.AutoShow.OnPause
		autoShowOnStop = cfg.AutoShow.OnStop
		autoShowOnSeek = cfg.AutoShow.OnSeek
	}

	// Extract scroll settings
	scrollEnabled := false
	scrollDirection := shared.ScrollLeft
	scrollSpeed := 30.0 // pixels per second
	scrollMode := shared.ScrollContinuous
	scrollPauseMs := 1000
	scrollGap := 20

	if cfg.Scroll != nil {
		scrollEnabled = cfg.Scroll.Enabled
		if cfg.Scroll.Direction != "" {
			scrollDirection = shared.ScrollDirection(cfg.Scroll.Direction)
		}
		if cfg.Scroll.Speed > 0 {
			scrollSpeed = cfg.Scroll.Speed
		}
		if cfg.Scroll.Mode != "" {
			scrollMode = shared.ScrollMode(cfg.Scroll.Mode)
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
	scroller := shared.NewTextScroller(shared.ScrollerConfig{
		Speed:     scrollSpeed,
		Mode:      scrollMode,
		Direction: scrollDirection,
		Gap:       scrollGap,
		PauseMs:   scrollPauseMs,
	})

	return &WinampWidget{
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
		autoShowOnSeek:        autoShowOnSeek,
		client:                winamp.NewClient(),
		fontFace:              fontFace,
		scroller:              scroller,
		previousStatus:        winamp.StatusStopped,
		previousPosMs:         -1,
	}, nil
}

// Update fetches current track information from Winamp
func (w *WinampWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update scroll position if scrolling is enabled and we have text
	if w.scrollEnabled && w.currentText != "" {
		pos := w.GetPosition()
		contentWidth := pos.W - w.padding*2
		textWidth, _ := bitmap.SmartMeasureText(w.currentText, w.fontFace, w.fontName)
		w.scroller.Update(textWidth, contentWidth)
	}

	// Get track info from Winamp
	info := w.client.GetTrackInfo()

	// Handle stopped/not running state
	if info == nil {
		// Check if we should show on stop (transition from playing/paused to stopped)
		if w.autoShowOnStop && w.previousStatus != winamp.StatusStopped {
			w.TriggerAutoHide()
		}
		w.currentText = ""
		w.previousStatus = winamp.StatusStopped
		w.previousPosMs = -1
		return nil
	}

	// Detect status changes
	statusChanged := info.Status != w.previousStatus

	// Check for play event (transition to playing state)
	if w.autoShowOnPlay && statusChanged && info.Status == winamp.StatusPlaying {
		w.TriggerAutoHide()
	}

	// Check for pause event (transition to paused state)
	if w.autoShowOnPause && statusChanged && info.Status == winamp.StatusPaused {
		w.TriggerAutoHide()
	}

	// Check for stop event (transition to stopped state)
	if w.autoShowOnStop && statusChanged && info.Status == winamp.StatusStopped {
		w.TriggerAutoHide()
	}

	// Check for seek event (position jumped significantly while playing)
	// Consider it a seek if position changed by more than 3 seconds (accounting for normal playback)
	if w.autoShowOnSeek && w.previousPosMs >= 0 && info.PositionMs >= 0 {
		posDiff := info.PositionMs - w.previousPosMs
		// If position jumped backwards, or jumped forward by more than 3 seconds
		if posDiff < -1000 || posDiff > 3000 {
			w.TriggerAutoHide()
		}
	}

	// Update previous status and position
	w.previousStatus = info.Status
	w.previousPosMs = info.PositionMs

	// Handle stopped state for display
	if info.Status == winamp.StatusStopped {
		w.currentText = ""
		return nil
	}

	// Format the output string
	w.currentText = w.formatOutput(info)

	// Check for track change and trigger auto-show
	// Only consider it a track change if:
	// 1. Title actually changed
	// 2. New title is not empty (empty title means stopping/transitioning, not a new track)
	// 3. Status didn't just change (avoids false triggers during stop/pause transitions
	//    when Winamp updates title before status)
	// 4. Current status is not Stopped (can't have a "new track" when stopped - handles
	//    timing where title updates before status during stop transition)
	if info.Title != w.previousTitle {
		if info.Title != "" && !statusChanged && info.Status != winamp.StatusStopped {
			// Reset scroll position on track change
			w.scroller.Reset()
			// Trigger auto-show if enabled
			if w.autoShowOnTrackChange {
				w.TriggerAutoHide()
			}
		}
		// Always update previousTitle to avoid re-triggering
		w.previousTitle = info.Title
	}

	return nil
}

// formatOutput replaces placeholders with actual values
func (w *WinampWidget) formatOutput(info *winamp.TrackInfo) string {
	if info == nil {
		return ""
	}

	result := w.format

	// Format shuffle/repeat as symbols or text
	shuffleStr := ""
	if info.Shuffle {
		shuffleStr = "S"
	}
	repeatStr := ""
	if info.Repeat {
		repeatStr = "R"
	}

	// Replace all placeholders
	replacements := map[string]string{
		"{title}":           info.Title,
		"{filename}":        info.FileName,
		"{filepath}":        info.FilePath,
		"{position}":        formatTime(info.PositionMs / 1000),
		"{duration}":        formatTime(info.DurationS),
		"{position_ms}":     fmt.Sprintf("%d", info.PositionMs),
		"{duration_s}":      fmt.Sprintf("%d", info.DurationS),
		"{bitrate}":         fmt.Sprintf("%d", info.Bitrate),
		"{samplerate}":      fmt.Sprintf("%d", info.SampleRate),
		"{channels}":        fmt.Sprintf("%d", info.Channels),
		"{status}":          info.Status.String(),
		"{track_num}":       fmt.Sprintf("%d", info.TrackNumber),
		"{playlist_length}": fmt.Sprintf("%d", info.PlaylistLength),
		"{shuffle}":         shuffleStr,
		"{repeat}":          repeatStr,
		"{version}":         info.Version,
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// formatTime converts seconds to MM:SS format
func formatTime(seconds int) string {
	if seconds < 0 {
		return "--:--"
	}
	mins := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// Render creates an image of the widget
func (w *WinampWidget) Render() (image.Image, error) {
	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw border if enabled
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	w.mu.RLock()
	currentText := w.currentText
	scrollOffset := w.scroller.GetOffset()
	w.mu.RUnlock()

	// Check if Winamp is running and playing
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

// renderPlaceholder renders the placeholder when Winamp is not playing
func (w *WinampWidget) renderPlaceholder(img *image.Gray) {
	pos := w.GetPosition()

	switch w.placeholderMode {
	case "icon":
		// Draw Winamp icon centered
		iconSet := glyphs.WinampIcons8x8
		icon := glyphs.GetIcon(iconSet, "winamp")
		if icon != nil {
			// Center the icon
			x := (pos.W - icon.Width) / 2
			y := (pos.H - icon.Height) / 2
			glyphs.DrawGlyph(img, icon, x, y, color.Gray{Y: 255})
		}
	case "text":
		bitmap.SmartDrawAlignedText(img, w.placeholderText, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}
}

// renderScrollingText renders text with scroll offset
func (w *WinampWidget) renderScrollingText(img *image.Gray, text string, offset float64) {
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
		if scrollCfg.Mode == shared.ScrollContinuous {
			bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, scrollX, textY, contentX, contentY, contentW, contentH)

			// Draw second instance for seamless loop
			if scrollCfg.Direction == shared.ScrollLeft {
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
