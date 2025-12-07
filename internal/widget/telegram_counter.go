package widget

import (
	"context"
	"fmt"
	"image"
	"image/color"
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
	client    *tgclient.Client
	clientCfg *config.TelegramConfig
	ctx       context.Context
	cancel    context.CancelFunc

	// Display settings
	fontFace  font.Face
	fontName  string
	fontSize  int
	format    string // "count", "badge", "text"
	showZero  bool   // whether to show when count is 0
	blinkWhen string // "never", "always", "nonzero"

	// State
	unreadCount       int
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

	if cfg.Telegram == nil {
		return nil, fmt.Errorf("telegram configuration is required")
	}

	// Get or create shared Telegram client via registry
	client, err := tgclient.GetOrCreateClient(cfg.Telegram)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram client: %w", err)
	}

	// Parse counter-specific settings from telegram config
	format := "count"
	showZero := false
	blinkWhen := "never"
	fontName := ""
	fontSize := 16

	if cfg.Telegram.Counter != nil {
		if cfg.Telegram.Counter.Format != "" {
			format = cfg.Telegram.Counter.Format
		}
		if cfg.Telegram.Counter.ShowZero != nil {
			showZero = *cfg.Telegram.Counter.ShowZero
		}
		if cfg.Telegram.Counter.Blink != "" {
			blinkWhen = cfg.Telegram.Counter.Blink
		}
		if cfg.Telegram.Counter.Text != nil {
			if cfg.Telegram.Counter.Text.Font != "" {
				fontName = cfg.Telegram.Counter.Text.Font
			}
			if cfg.Telegram.Counter.Text.Size > 0 {
				fontSize = cfg.Telegram.Counter.Text.Size
			}
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
		clientCfg:         cfg.Telegram,
		format:            format,
		showZero:          showZero,
		blinkWhen:         blinkWhen,
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

	// Update blink state (toggle every 500ms)
	if now.Sub(w.lastBlink) >= 500*time.Millisecond {
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
	} else {
		// Check if we should display anything
		if w.unreadCount == 0 && !w.showZero {
			// Return empty/transparent widget
		} else {
			// Check blink
			shouldBlink := false
			switch w.blinkWhen {
			case "always":
				shouldBlink = true
			case "nonzero":
				shouldBlink = w.unreadCount > 0
			}

			if shouldBlink && !w.blinkState {
				// Skip rendering when blinking off
			} else {
				w.renderUnreadCount(img)
			}
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
	switch w.format {
	case "badge":
		// Draw Telegram icon (paper airplane) centered
		w.drawTelegramIcon(img)
	case "text":
		// Formatted text with count
		var text string
		if w.unreadCount == 1 {
			text = "1 unread"
		} else {
			text = fmt.Sprintf("%d unread", w.unreadCount)
		}
		textX, textY := bitmap.SmartCalculateTextPosition(text, w.fontFace, w.fontName, 0, 0, w.width, w.height, "center", "center")
		bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX, textY, 0, 0, w.width, w.height)
	default: // "count" - just the number
		text := fmt.Sprintf("%d", w.unreadCount)
		textX, textY := bitmap.SmartCalculateTextPosition(text, w.fontFace, w.fontName, 0, 0, w.width, w.height, "center", "center")
		bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX, textY, 0, 0, w.width, w.height)
	}
}

// drawTelegramIcon draws the Telegram paper airplane icon centered in the widget
func (w *TelegramCounterWidget) drawTelegramIcon(img *image.Gray) {
	// Select appropriate icon size based on widget dimensions
	// Use the smaller dimension to ensure icon fits
	iconSize := w.width
	if w.height < iconSize {
		iconSize = w.height
	}
	// Leave some margin (use 80% of available space)
	iconSet := glyphs.GetTelegramIcon(iconSize)
	icon := iconSet.Icons["telegram"]
	if icon == nil {
		return
	}

	// Center the icon
	x := (w.width - icon.Width) / 2
	y := (w.height - icon.Height) / 2

	c := color.Gray{Y: 255}

	for row := 0; row < icon.Height && row < len(icon.Data); row++ {
		for col := 0; col < icon.Width && col < len(icon.Data[row]); col++ {
			if icon.Data[row][col] {
				px, py := x+col, y+row
				if px >= 0 && px < w.width && py >= 0 && py < w.height {
					img.Set(px, py, c)
				}
			}
		}
	}
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
	if w.clientCfg != nil {
		tgclient.ReleaseClient(w.clientCfg)
		w.clientCfg = nil
	}
}
