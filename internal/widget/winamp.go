package widget

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/winamp"
	"golang.org/x/image/font"
)

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
	scrollEnabled   bool
	scrollDirection string  // "left", "right", "up", "down"
	scrollSpeed     float64 // pixels per second
	scrollMode      string  // "continuous", "bounce", "pause_ends"
	scrollPauseMs   int     // pause duration at ends
	scrollGap       int     // gap between text repetitions

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
	lastUpdateTime time.Time

	// Scroll state
	scrollOffset     float64
	scrollDirection2 int // 1 or -1 for bounce mode
	scrollPauseUntil time.Time
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
	scrollDirection := "left"
	scrollSpeed := 30.0 // pixels per second
	scrollMode := "continuous"
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
		scrollDirection:       scrollDirection,
		scrollSpeed:           scrollSpeed,
		scrollMode:            scrollMode,
		scrollPauseMs:         scrollPauseMs,
		scrollGap:             scrollGap,
		autoShowOnTrackChange: autoShowOnTrackChange,
		autoShowOnPlay:        autoShowOnPlay,
		autoShowOnPause:       autoShowOnPause,
		autoShowOnStop:        autoShowOnStop,
		autoShowOnSeek:        autoShowOnSeek,
		client:                winamp.NewClient(),
		fontFace:              fontFace,
		lastUpdateTime:        time.Now(),
		scrollDirection2:      1,
		previousStatus:        winamp.StatusStopped,
		previousPosMs:         -1,
	}, nil
}

// Update fetches current track information from Winamp
func (w *WinampWidget) Update() error {
	now := time.Now()

	w.mu.Lock()
	defer w.mu.Unlock()

	// Update scroll position based on elapsed time
	if w.scrollEnabled && w.currentText != "" {
		w.updateScrollPosition(now)
	}
	w.lastUpdateTime = now

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
	if info.Title != w.previousTitle {
		w.previousTitle = info.Title
		// Reset scroll position on track change
		w.scrollOffset = 0
		w.scrollDirection2 = 1
		w.scrollPauseUntil = time.Time{}
		// Trigger auto-show if enabled
		if w.autoShowOnTrackChange {
			w.TriggerAutoHide()
		}
	}

	return nil
}

// updateScrollPosition calculates the new scroll offset based on elapsed time
func (w *WinampWidget) updateScrollPosition(now time.Time) {
	elapsed := now.Sub(w.lastUpdateTime).Seconds()

	// Check if we're in a pause
	if !w.scrollPauseUntil.IsZero() && now.Before(w.scrollPauseUntil) {
		return
	}
	w.scrollPauseUntil = time.Time{}

	// Calculate text width
	textWidth, _ := bitmap.MeasureText(w.currentText, w.fontFace)
	pos := w.GetPosition()
	contentWidth := pos.W - w.padding*2

	// Only scroll if text is wider than content area
	if textWidth <= contentWidth {
		w.scrollOffset = 0
		return
	}

	// Calculate movement
	movement := w.scrollSpeed * elapsed

	switch w.scrollMode {
	case "continuous":
		w.updateContinuousScroll(movement, textWidth)
	case "bounce":
		w.updateBounceScroll(movement, textWidth, contentWidth, now)
	case "pause_ends":
		w.updatePauseEndsScroll(movement, textWidth, contentWidth, now)
	}
}

// updateContinuousScroll handles continuous/marquee scrolling
func (w *WinampWidget) updateContinuousScroll(movement float64, textWidth int) {
	totalWidth := float64(textWidth + w.scrollGap)

	switch w.scrollDirection {
	case "left":
		w.scrollOffset += movement
		if w.scrollOffset >= totalWidth {
			w.scrollOffset -= totalWidth
		}
	case "right":
		w.scrollOffset -= movement
		if w.scrollOffset <= -totalWidth {
			w.scrollOffset += totalWidth
		}
	case "up", "down":
		// For vertical scrolling, use the same logic, but it applies to Y
		if w.scrollDirection == "up" {
			w.scrollOffset += movement
		} else {
			w.scrollOffset -= movement
		}
		// Reset when text has scrolled completely
		textHeight := w.fontSize + w.scrollGap
		if w.scrollOffset >= float64(textHeight) || w.scrollOffset <= float64(-textHeight) {
			w.scrollOffset = 0
		}
	}
}

// updateBounceScroll handles bounce scrolling (reverse at edges)
func (w *WinampWidget) updateBounceScroll(movement float64, textWidth, contentWidth int, now time.Time) {
	maxOffset := float64(textWidth - contentWidth)

	w.scrollOffset += movement * float64(w.scrollDirection2)

	if w.scrollOffset >= maxOffset {
		w.scrollOffset = maxOffset
		w.scrollDirection2 = -1
		w.scrollPauseUntil = now.Add(time.Duration(w.scrollPauseMs) * time.Millisecond)
	} else if w.scrollOffset <= 0 {
		w.scrollOffset = 0
		w.scrollDirection2 = 1
		w.scrollPauseUntil = now.Add(time.Duration(w.scrollPauseMs) * time.Millisecond)
	}
}

// updatePauseEndsScroll handles scroll with pause at ends
func (w *WinampWidget) updatePauseEndsScroll(movement float64, textWidth, contentWidth int, now time.Time) {
	maxOffset := float64(textWidth - contentWidth)

	switch w.scrollDirection {
	case "left":
		w.scrollOffset += movement
		if w.scrollOffset >= maxOffset {
			w.scrollOffset = 0
			w.scrollPauseUntil = now.Add(time.Duration(w.scrollPauseMs) * time.Millisecond)
		}
	case "right":
		w.scrollOffset -= movement
		if w.scrollOffset <= -maxOffset {
			w.scrollOffset = 0
			w.scrollPauseUntil = now.Add(time.Duration(w.scrollPauseMs) * time.Millisecond)
		}
	}
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
	scrollOffset := w.scrollOffset
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
		bitmap.DrawAlignedText(img, currentText, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
	}

	return img, nil
}

// renderPlaceholder renders the placeholder when Winamp is not playing
func (w *WinampWidget) renderPlaceholder(img *image.Gray) {
	pos := w.GetPosition()

	switch w.placeholderMode {
	case "icon":
		// Draw Winamp icon centered
		iconSet := glyphs.GetWinampIconSet(pos.H - w.padding*2)
		icon := glyphs.GetIcon(iconSet, "winamp")
		if icon != nil {
			// Center the icon
			x := (pos.W - icon.Width) / 2
			y := (pos.H - icon.Height) / 2
			glyphs.DrawGlyph(img, icon, x, y, color.Gray{Y: 255})
		}
	case "text":
		bitmap.DrawAlignedText(img, w.placeholderText, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
	}
}

// renderScrollingText renders text with scroll offset
func (w *WinampWidget) renderScrollingText(img *image.Gray, text string, offset float64) {
	pos := w.GetPosition()
	contentX := w.padding
	contentY := w.padding
	contentW := pos.W - w.padding*2
	contentH := pos.H - w.padding*2

	textWidth, _ := bitmap.MeasureText(text, w.fontFace)

	// If text fits, just draw it normally
	if textWidth <= contentW {
		bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
		return
	}

	// Calculate Y position for text baseline
	metrics := w.fontFace.Metrics()
	ascent := metrics.Ascent.Ceil()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()

	var textY int
	switch w.vertAlign {
	case "top":
		textY = contentY + ascent
	case "bottom":
		textY = contentY + contentH - textHeight + ascent
	default: // center
		textY = contentY + (contentH-textHeight)/2 + ascent
	}

	// Handle horizontal scrolling
	if w.scrollDirection == "left" || w.scrollDirection == "right" {
		// Draw text at offset position
		textX := contentX - int(offset)

		// For continuous mode, we may need to draw text twice for seamless loop
		if w.scrollMode == "continuous" {
			// Draw first instance
			bitmap.DrawTextAtPosition(img, text, w.fontFace, textX, textY, contentX, contentY, contentW, contentH)

			// Draw second instance for seamless loop
			if w.scrollDirection == "left" {
				textX2 := textX + textWidth + w.scrollGap
				if textX2 < contentX+contentW {
					bitmap.DrawTextAtPosition(img, text, w.fontFace, textX2, textY, contentX, contentY, contentW, contentH)
				}
			} else {
				textX2 := textX - textWidth - w.scrollGap
				if textX2+textWidth > contentX {
					bitmap.DrawTextAtPosition(img, text, w.fontFace, textX2, textY, contentX, contentY, contentW, contentH)
				}
			}
		} else {
			// For bounce and pause_ends modes, just draw once
			bitmap.DrawTextAtPosition(img, text, w.fontFace, textX, textY, contentX, contentY, contentW, contentH)
		}
	} else {
		// Vertical scrolling (up/down)
		var textX int
		switch w.horizAlign {
		case "right":
			textX = contentX + contentW - textWidth
		case "center":
			textX = contentX + (contentW-textWidth)/2
		default: // left
			textX = contentX
		}

		scrollY := textY - int(offset)
		bitmap.DrawTextAtPosition(img, text, w.fontFace, textX, scrollY, contentX, contentY, contentW, contentH)
	}
}
