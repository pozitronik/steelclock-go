package widget

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	tgclient "github.com/pozitronik/steelclock-go/internal/telegram"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"golang.org/x/image/font"
)

func init() {
	Register("telegram_counter", func(cfg config.WidgetConfig) (Widget, error) {
		return NewTelegramCounterWidget(cfg)
	})
}

// Display mode constants
const (
	telegramCounterModeBadge = "badge"
	telegramCounterModeText  = "text"
)

// TelegramCounterWidget displays unread message count
type TelegramCounterWidget struct {
	*BaseWidget
	mu sync.RWMutex

	// Telegram client (shared via registry)
	client  *tgclient.Client
	authCfg *config.TelegramAuthConfig

	// Display settings
	fontFace   font.Face
	fontName   string
	fontSize   int
	mode       string // "badge" or "text"
	textFormat string // format string with tokens like {unread}, {mentions}, etc.

	// Badge colors (for badge mode)
	badgeForeground int // 0-255 grayscale, -1 for transparent
	badgeBackground int // 0-255 grayscale, -1 for transparent

	// State
	unreadCount   int
	unreadStats   tgclient.UnreadStats
	lastFetch     time.Time
	fetchInterval time.Duration

	// Shared modules
	connection     *shared.ConnectionManager
	blink          *shared.BlinkAnimator
	statusRenderer *shared.StatusRenderer

	// Display dimensions
	width  int
	height int
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
		mode = telegramCounterModeBadge // default mode
	}

	blinkMode := shared.BlinkNever
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

	// Create connection manager
	connManager := shared.NewConnectionManager(client, 30*time.Second, 60*time.Second)

	w := &TelegramCounterWidget{
		BaseWidget:      base,
		client:          client,
		authCfg:         cfg.Auth,
		mode:            mode,
		textFormat:      textFormat,
		badgeForeground: badgeForeground,
		badgeBackground: badgeBackground,
		fontFace:        fontFace,
		fontName:        fontName,
		fontSize:        fontSize,
		fetchInterval:   5 * time.Second, // Fetch unread count every 5 seconds
		width:           pos.W,
		height:          pos.H,
		connection:      connManager,
		blink:           shared.NewBlinkAnimator(blinkMode, 500*time.Millisecond),
		statusRenderer:  shared.NewStatusRenderer("5x7"),
	}

	// Add error callback (using Add instead of Set for proper multi-widget support)
	client.AddErrorCallback(func(err error) {
		// Connection manager handles errors internally
	})

	return w, nil
}

// Update handles widget state updates
func (w *TelegramCounterWidget) Update() error {
	now := time.Now()

	w.mu.Lock()
	defer w.mu.Unlock()

	// Update blink state (pass unread count for progressive mode)
	w.blink.UpdateWithTime(now, w.unreadCount)

	// Handle connection via shared manager
	w.connection.Update()

	// Fetch unread count periodically
	if w.connection.IsConnected() {
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

	// Create canvas with background
	img := w.CreateCanvas()

	// Draw based on state
	if w.connection.IsConnecting() {
		w.statusRenderer.DrawCentered(img, "...", 0, 0, w.width, w.height)
	} else if w.connection.GetError() != nil {
		w.statusRenderer.DrawCentered(img, "ERR", 0, 0, w.width, w.height)
	} else if !w.connection.IsConnected() {
		// Show "..." on initial state (before first connection attempt)
		// to avoid brief "---" flash
		if w.connection.IsInitialState() {
			w.statusRenderer.DrawCentered(img, "...", 0, 0, w.width, w.height)
		} else {
			w.statusRenderer.DrawCentered(img, "---", 0, 0, w.width, w.height)
		}
	} else if w.unreadCount > 0 {
		// Only display when there are unread messages
		// Check if we should skip rendering due to blink
		if !w.blink.ShouldRender() {
			// Skip rendering when blinking off
		} else {
			w.renderUnreadCount(img)
		}
	}

	// Draw border if enabled
	w.ApplyBorder(img)

	return img, nil
}

// renderUnreadCount renders the unread count in the configured format
func (w *TelegramCounterWidget) renderUnreadCount(img *image.Gray) {
	switch w.mode {
	case telegramCounterModeBadge:
		// Draw Telegram icon (paper airplane) centered
		w.drawTelegramIcon(img, (w.width-w.getIconSize())/2, (w.height-w.getIconSize())/2)
	case telegramCounterModeText:
		// Formatted text with tokens (may include {icon})
		w.renderFormattedText(img)
	default: // telegramCounterModeBadge by default
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
	if w.mode == telegramCounterModeText {
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

	// Draw the icon: true pixels are foreground, false pixels are background
	// -1 means transparent (don't draw)
	bitmap.DrawGlyphWithBackground(img, icon, x, y, w.badgeForeground, w.badgeBackground)
}

// Stop cleans up resources
func (w *TelegramCounterWidget) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Release client via registry (will disconnect when ref count reaches 0)
	if w.authCfg != nil {
		tgclient.ReleaseClient(w.authCfg)
		w.authCfg = nil
	}
}
