package widget

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	tgclient "github.com/pozitronik/steelclock-go/internal/telegram"
	"golang.org/x/image/font"
)

// TelegramCounterWidget displays unread message count
type TelegramCounterWidget struct {
	*BaseWidget
	mu sync.RWMutex

	// Telegram client (shared via registry)
	client  *tgclient.Client
	authCfg *config.TelegramAuthConfig
	ctx     context.Context
	cancel  context.CancelFunc

	// Display settings
	fontFace   font.Face
	fontName   string
	fontSize   int
	mode       string // "badge" or "text"
	textFormat string // format string with tokens like {unread}, {mentions}, etc.
	blinkMode  string // "never", "always", "progressive"

	// Badge colors (for badge mode)
	badgeForeground int // 0-255 grayscale, -1 for transparent
	badgeBackground int // 0-255 grayscale, -1 for transparent

	// State
	unreadCount       int
	unreadStats       tgclient.UnreadStats
	connectionError   error
	connecting        bool
	lastConnectionTry time.Time
	reconnectInterval time.Duration
	lastFetch         time.Time
	fetchInterval     time.Duration

	// Blink state
	blinkState bool
	lastBlink  time.Time

	// Display dimensions
	width  int
	height int

	// Fallback internal font
	glyphSet *glyphs.GlyphSet
}

// NewTelegramCounterWidget creates a new Telegram unread count widget
func NewTelegramCounterWidget(cfg config.WidgetConfig) (*TelegramCounterWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	if cfg.Auth == nil {
		return nil, fmt.Errorf("telegram auth configuration is required")
	}

	// Create client config for registry
	clientCfg := &tgclient.ClientConfig{
		Auth:    cfg.Auth,
		Filters: cfg.Filters,
	}

	// Get or create shared Telegram client via registry
	client, err := tgclient.GetOrCreateClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram client: %w", err)
	}

	// Parse counter-specific settings from flat config
	mode := cfg.Mode
	if mode == "" {
		mode = "badge" // default mode
	}

	blinkMode := "never"
	badgeForeground := 255 // default: white
	badgeBackground := 0   // default: black

	if cfg.Badge != nil {
		if cfg.Badge.Blink != "" {
			blinkMode = cfg.Badge.Blink
		}
		if cfg.Badge.Colors != nil {
			badgeForeground = cfg.Badge.Colors.Foreground
			badgeBackground = cfg.Badge.Colors.Background
		}
	}

	// Parse text settings
	textFormat := "{unread} unread" // default format
	fontName := ""
	fontSize := 16
	if cfg.Text != nil {
		if cfg.Text.Format != "" {
			textFormat = cfg.Text.Format
		}
		if cfg.Text.Font != "" {
			fontName = cfg.Text.Font
		}
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
	}

	// Load font
	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Get internal font for fallback
	glyphSet := bitmap.GetInternalFontByName("5x7")
	if glyphSet == nil {
		glyphSet = glyphs.Font5x7
	}

	w := &TelegramCounterWidget{
		BaseWidget:        base,
		client:            client,
		authCfg:           cfg.Auth,
		mode:              mode,
		textFormat:        textFormat,
		blinkMode:         blinkMode,
		badgeForeground:   badgeForeground,
		badgeBackground:   badgeBackground,
		fontFace:          fontFace,
		fontName:          fontName,
		fontSize:          fontSize,
		reconnectInterval: 30 * time.Second,
		fetchInterval:     5 * time.Second, // Fetch unread count every 5 seconds
		width:             pos.W,
		height:            pos.H,
		glyphSet:          glyphSet,
		lastBlink:         time.Now(),
	}

	// Set error callback
	client.SetErrorCallback(func(err error) {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.connectionError = err
	})

	return w, nil
}

// Update handles widget state updates
func (w *TelegramCounterWidget) Update() error {
	now := time.Now()

	w.mu.Lock()
	defer w.mu.Unlock()

	// Update blink state based on mode
	blinkInterval := w.getBlinkInterval()
	if blinkInterval > 0 && now.Sub(w.lastBlink) >= blinkInterval {
		w.blinkState = !w.blinkState
		w.lastBlink = now
	}

	// Handle connection
	if !w.client.IsConnected() && !w.connecting {
		if time.Since(w.lastConnectionTry) > w.reconnectInterval {
			w.connecting = true
			w.lastConnectionTry = time.Now()
			w.connectionError = nil

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				err := w.client.Connect(ctx)

				w.mu.Lock()
				w.connecting = false
				if err != nil {
					w.connectionError = err
				}
				w.mu.Unlock()
			}()
		}
	}

	// Fetch unread count periodically
	if w.client.IsConnected() {
		if time.Since(w.lastFetch) >= w.fetchInterval {
			w.lastFetch = now
			// Fetch in background to avoid blocking
			go func() {
				if err := w.client.FetchUnreadCount(); err != nil {
					// Log error but don't fail - will retry next interval
				}
			}()
		}
		w.unreadCount = w.client.GetUnreadCount()
		w.unreadStats = w.client.GetUnreadStats()
	}

	return nil
}

// Render draws the widget
func (w *TelegramCounterWidget) Render() (image.Image, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw based on state
	if w.connecting {
		w.drawStatusText(img, "...")
	} else if w.connectionError != nil {
		w.drawStatusText(img, "ERR")
	} else if !w.client.IsConnected() {
		w.drawStatusText(img, "---")
	} else if w.unreadCount > 0 {
		// Only display when there are unread messages
		// Check if we should skip rendering due to blink
		if w.shouldSkipForBlink() {
			// Skip rendering when blinking off
		} else {
			w.renderUnreadCount(img)
		}
	}

	// Draw border if enabled
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, nil
}

// renderUnreadCount renders the unread count in the configured format
func (w *TelegramCounterWidget) renderUnreadCount(img *image.Gray) {
	switch w.mode {
	case "badge":
		// Draw Telegram icon (paper airplane) centered
		w.drawTelegramIcon(img, (w.width-w.getIconSize())/2, (w.height-w.getIconSize())/2)
	case "text":
		// Formatted text with tokens (may include {icon})
		w.renderFormattedText(img)
	default: // badge by default
		w.drawTelegramIcon(img, (w.width-w.getIconSize())/2, (w.height-w.getIconSize())/2)
	}
}

// formatText replaces tokens in the text format with actual values (except {icon})
func (w *TelegramCounterWidget) formatText() string {
	text := w.textFormat

	// Replace all supported tokens (except {icon} which is handled separately)
	replacements := map[string]int{
		"{unread}":         w.unreadStats.Total,
		"{total}":          w.unreadStats.Total,
		"{mentions}":       w.unreadStats.Mentions,
		"{reactions}":      w.unreadStats.Reactions,
		"{private}":        w.unreadStats.Private,
		"{groups}":         w.unreadStats.Groups,
		"{channels}":       w.unreadStats.Channels,
		"{muted}":          w.unreadStats.Muted,
		"{private_muted}":  w.unreadStats.PrivateMuted,
		"{groups_muted}":   w.unreadStats.GroupsMuted,
		"{channels_muted}": w.unreadStats.ChannelsMuted,
	}

	for token, value := range replacements {
		text = strings.ReplaceAll(text, token, fmt.Sprintf("%d", value))
	}

	return text
}

// getIconSize returns the icon size based on actual text height
func (w *TelegramCounterWidget) getIconSize() int {
	// For text mode, measure actual text height
	if w.mode == "text" {
		_, textHeight := bitmap.SmartMeasureText("X", w.fontFace, w.fontName)
		if textHeight > 0 {
			return textHeight
		}
		// Fallback to fontSize if measurement fails
		return w.fontSize
	}
	// For badge mode, use the smaller widget dimension
	iconSize := w.width
	if w.height < iconSize {
		iconSize = w.height
	}
	return iconSize
}

// renderFormattedText renders text with {icon} token support
func (w *TelegramCounterWidget) renderFormattedText(img *image.Gray) {
	text := w.formatText()

	// Check if text contains {icon} token
	if !strings.Contains(text, "{icon}") {
		// No icon - simple text rendering
		textX, textY := bitmap.SmartCalculateTextPosition(text, w.fontFace, w.fontName, 0, 0, w.width, w.height, "center", "center")
		bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX, textY, 0, 0, w.width, w.height)
		return
	}

	// Split text by {icon} and render parts with icons between them
	parts := strings.Split(text, "{icon}")
	iconSize := w.getIconSize()

	// Calculate total width
	totalWidth := 0
	for i, part := range parts {
		if part != "" {
			partWidth, _ := bitmap.SmartMeasureText(part, w.fontFace, w.fontName)
			totalWidth += partWidth
		}
		// Add icon width between parts (not after last part)
		if i < len(parts)-1 {
			totalWidth += iconSize
		}
	}

	// Calculate starting X position (centered)
	x := (w.width - totalWidth) / 2
	// Calculate Y position for text (centered)
	_, textY := bitmap.SmartCalculateTextPosition("X", w.fontFace, w.fontName, 0, 0, w.width, w.height, "left", "center")

	// Render each part with icons between
	for i, part := range parts {
		if part != "" {
			bitmap.SmartDrawTextAtPosition(img, part, w.fontFace, w.fontName, x, textY, 0, 0, w.width, w.height)
			partWidth, _ := bitmap.SmartMeasureText(part, w.fontFace, w.fontName)
			x += partWidth
		}
		// Draw icon between parts (not after last part)
		if i < len(parts)-1 {
			// Center icon vertically
			iconY := (w.height - iconSize) / 2
			w.drawTelegramIcon(img, x, iconY)
			x += iconSize
		}
	}
}

// drawTelegramIcon draws the Telegram paper airplane icon at specified position
func (w *TelegramCounterWidget) drawTelegramIcon(img *image.Gray, x, y int) {
	iconSize := w.getIconSize()

	iconSet := glyphs.GetTelegramIcon(iconSize)
	icon := iconSet.Icons["telegram"]
	if icon == nil {
		return
	}

	// Adjust position if icon size differs from requested size
	// (icon might be smaller than iconSize)
	x += (iconSize - icon.Width) / 2
	y += (iconSize - icon.Height) / 2

	// Determine colors based on config
	fgColor := color.Gray{Y: uint8(w.badgeForeground)}
	bgColor := color.Gray{Y: uint8(w.badgeBackground)}

	// Draw the icon: true pixels are foreground, false pixels are background
	// -1 means transparent (don't draw)
	for row := 0; row < icon.Height && row < len(icon.Data); row++ {
		for col := 0; col < icon.Width && col < len(icon.Data[row]); col++ {
			px, py := x+col, y+row
			if px >= 0 && px < w.width && py >= 0 && py < w.height {
				if icon.Data[row][col] {
					if w.badgeForeground >= 0 {
						img.Set(px, py, fgColor)
					}
				} else {
					if w.badgeBackground >= 0 {
						img.Set(px, py, bgColor)
					}
				}
			}
		}
	}
}

// getBlinkInterval returns the blink interval based on mode and unread count
// Returns 0 if blinking is disabled
func (w *TelegramCounterWidget) getBlinkInterval() time.Duration {
	switch w.blinkMode {
	case "always":
		return 500 * time.Millisecond
	case "progressive":
		if w.unreadCount <= 0 {
			return 0
		}
		// Scale from 1000ms (1 msg) to 100ms (10+ msgs)
		// Formula: interval = 1000 - (count-1) * 100, clamped to [100, 1000]
		intervalMs := 1000 - (w.unreadCount-1)*100
		if intervalMs < 100 {
			intervalMs = 100
		}
		if intervalMs > 1000 {
			intervalMs = 1000
		}
		return time.Duration(intervalMs) * time.Millisecond
	default: // "never"
		return 0
	}
}

// shouldSkipForBlink returns true if rendering should be skipped due to blink state
func (w *TelegramCounterWidget) shouldSkipForBlink() bool {
	if w.blinkMode == "never" {
		return false
	}
	return !w.blinkState
}

// drawStatusText draws status text centered using internal font
func (w *TelegramCounterWidget) drawStatusText(img *image.Gray, text string) {
	c := color.Gray{Y: 255}

	// Calculate total text width
	totalWidth := 0
	for _, ch := range text {
		glyph := glyphs.GetGlyph(w.glyphSet, ch)
		if glyph != nil {
			totalWidth += glyph.Width + 1
		}
	}
	if totalWidth > 0 {
		totalWidth-- // Remove trailing space
	}

	// Center the text
	x := (w.width - totalWidth) / 2
	y := (w.height - 7) / 2 // 7 is height of 5x7 font

	for _, ch := range text {
		glyph := glyphs.GetGlyph(w.glyphSet, ch)
		if glyph == nil {
			continue
		}
		for row := 0; row < glyph.Height && row < len(glyph.Data); row++ {
			for col := 0; col < glyph.Width && col < len(glyph.Data[row]); col++ {
				if glyph.Data[row][col] {
					px, py := x+col, y+row
					if px >= 0 && px < w.width && py >= 0 && py < w.height {
						img.Set(px, py, c)
					}
				}
			}
		}
		x += glyph.Width + 1
	}
}

// Stop cleans up resources
func (w *TelegramCounterWidget) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancel != nil {
		w.cancel()
	}
	// Release client via registry (will disconnect when ref count reaches 0)
	if w.authCfg != nil {
		tgclient.ReleaseClient(w.authCfg)
		w.authCfg = nil
	}
}
