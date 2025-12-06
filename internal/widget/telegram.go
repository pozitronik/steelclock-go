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
)

// TelegramWidget displays Telegram notifications
type TelegramWidget struct {
	*BaseWidget
	mu sync.RWMutex

	// Configuration
	displayMode          string
	showSender           bool
	showChat             bool
	showTime             bool
	timeFormat           string
	truncateLength       int
	notificationDuration float64
	scrollSpeed          float64
	showUnreadBadge      bool
	textColor            uint8

	// Telegram client
	client *tgclient.Client
	ctx    context.Context
	cancel context.CancelFunc

	// State
	messages          []tgclient.MessageInfo
	lastNotification  *tgclient.MessageInfo
	notificationStart time.Time
	scrollOffset      float64
	connectionError   error
	connecting        bool
	lastConnectionTry time.Time
	reconnectInterval time.Duration

	// Display dimensions
	width  int
	height int

	// Font
	glyphSet *glyphs.GlyphSet
}

// NewTelegramWidget creates a new Telegram notification widget
func NewTelegramWidget(cfg config.WidgetConfig) (*TelegramWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	if cfg.Telegram == nil {
		return nil, fmt.Errorf("telegram configuration is required")
	}

	// Default configuration
	displayMode := "last_message"
	showSender := true
	showChat := true
	showTime := false
	timeFormat := "15:04"
	truncateLength := 50
	notificationDuration := 3.0
	scrollSpeed := 1.0
	showUnreadBadge := true
	textColor := uint8(255)

	if cfg.Telegram.Display != nil {
		d := cfg.Telegram.Display
		if d.Mode != "" {
			displayMode = d.Mode
		}
		if d.ShowSender != nil {
			showSender = *d.ShowSender
		}
		if d.ShowChat != nil {
			showChat = *d.ShowChat
		}
		if d.ShowTime != nil {
			showTime = *d.ShowTime
		}
		if d.TimeFormat != "" {
			timeFormat = d.TimeFormat
		}
		if d.TruncateLength > 0 {
			truncateLength = d.TruncateLength
		}
		if d.NotificationDuration > 0 {
			notificationDuration = d.NotificationDuration
		}
		if d.ScrollSpeed > 0 {
			scrollSpeed = d.ScrollSpeed
		}
		if d.UnreadBadge != nil {
			showUnreadBadge = *d.UnreadBadge
		}
		if d.TextColor > 0 {
			textColor = uint8(d.TextColor)
		}
	}

	// Create Telegram client
	client, err := tgclient.NewClient(cfg.Telegram)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram client: %w", err)
	}

	// Get internal font
	glyphSet := bitmap.GetInternalFontByName("5x7")
	if glyphSet == nil {
		glyphSet = glyphs.Font5x7
	}

	w := &TelegramWidget{
		BaseWidget:           base,
		displayMode:          displayMode,
		showSender:           showSender,
		showChat:             showChat,
		showTime:             showTime,
		timeFormat:           timeFormat,
		truncateLength:       truncateLength,
		notificationDuration: notificationDuration,
		scrollSpeed:          scrollSpeed,
		showUnreadBadge:      showUnreadBadge,
		textColor:            textColor,
		client:               client,
		messages:             make([]tgclient.MessageInfo, 0),
		reconnectInterval:    30 * time.Second,
		width:                pos.W,
		height:               pos.H,
		glyphSet:             glyphSet,
	}

	// Set message callback
	client.SetMessageCallback(func(msg tgclient.MessageInfo) {
		w.mu.Lock()
		defer w.mu.Unlock()

		// Update messages list
		w.messages = w.client.GetMessages()

		// Trigger notification display
		if w.displayMode == "notification" {
			w.lastNotification = &msg
			w.notificationStart = time.Now()
		}
	})

	// Set error callback
	client.SetErrorCallback(func(err error) {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.connectionError = err
	})

	return w, nil
}

// Update handles widget state updates
func (w *TelegramWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Handle connection
	if !w.client.IsConnected() && !w.connecting {
		// Check if we should try reconnecting
		if time.Since(w.lastConnectionTry) > w.reconnectInterval {
			w.connecting = true
			w.lastConnectionTry = time.Now()
			w.connectionError = nil

			// Connect in background
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

	// Update scroll offset for ticker mode
	if w.displayMode == "ticker" {
		w.scrollOffset += w.scrollSpeed
	}

	// Check notification timeout
	if w.displayMode == "notification" && w.lastNotification != nil {
		if time.Since(w.notificationStart).Seconds() > w.notificationDuration {
			w.lastNotification = nil
		}
	}

	// Refresh messages from client
	if w.client.IsConnected() {
		w.messages = w.client.GetMessages()
	}

	return nil
}

// Render draws the widget
func (w *TelegramWidget) Render() (image.Image, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw based on state
	if w.connecting {
		w.drawText(img, "Connecting...", 2, w.height/2-3)
	} else if w.connectionError != nil {
		errMsg := w.connectionError.Error()
		if len(errMsg) > 20 {
			errMsg = errMsg[:20] + "..."
		}
		w.drawText(img, "Error: "+errMsg, 2, w.height/2-3)
	} else if !w.client.IsConnected() {
		w.drawText(img, "Disconnected", 2, w.height/2-3)
	} else {
		// Draw based on display mode
		switch w.displayMode {
		case "last_message":
			w.renderLastMessage(img)
		case "unread_count":
			w.renderUnreadCount(img)
		case "ticker":
			w.renderTicker(img)
		case "notification":
			w.renderNotification(img)
		default:
			w.renderLastMessage(img)
		}
	}

	// Draw border if enabled
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, nil
}

// renderLastMessage displays the most recent message
func (w *TelegramWidget) renderLastMessage(img *image.Gray) {
	if len(w.messages) == 0 {
		w.drawText(img, "No messages", 2, w.height/2-3)
		return
	}

	msg := w.messages[0]
	y := 2

	// Draw sender/chat info on first line
	header := w.formatHeader(msg)
	if header != "" {
		w.drawText(img, header, 2, y)
		y += 9
	}

	// Draw message text
	text := msg.Text
	if len(text) > w.truncateLength {
		text = text[:w.truncateLength-3] + "..."
	}
	w.drawText(img, text, 2, y)

	// Draw unread badge if enabled
	if w.showUnreadBadge {
		unread := w.client.GetUnreadCount()
		if unread > 0 {
			w.drawBadge(img, unread)
		}
	}
}

// renderUnreadCount displays unread message count
func (w *TelegramWidget) renderUnreadCount(img *image.Gray) {
	unread := w.client.GetUnreadCount()

	// Draw large centered count
	countStr := fmt.Sprintf("%d", unread)
	if unread == 0 {
		countStr = "0"
	}

	// Center the text
	charWidth := 6
	textWidth := len(countStr) * charWidth
	x := (w.width - textWidth) / 2
	y := w.height/2 - 3

	w.drawText(img, countStr, x, y)

	// Draw "unread" label below
	label := "unread"
	labelWidth := len(label) * charWidth
	labelX := (w.width - labelWidth) / 2
	w.drawTextSmall(img, label, labelX, y+10)
}

// renderTicker displays scrolling message ticker
func (w *TelegramWidget) renderTicker(img *image.Gray) {
	if len(w.messages) == 0 {
		w.drawText(img, "No messages", 2, w.height/2-3)
		return
	}

	// Build ticker text from all messages
	var ticker string
	for i, msg := range w.messages {
		if i > 0 {
			ticker += " | "
		}
		header := w.formatHeader(msg)
		if header != "" {
			ticker += header + ": "
		}
		ticker += msg.Text
	}

	// Calculate scroll position
	charWidth := 6
	tickerWidth := len(ticker) * charWidth
	offset := int(w.scrollOffset) % (tickerWidth + w.width)

	// Draw scrolling text
	x := w.width - offset
	y := w.height/2 - 3
	w.drawText(img, ticker, x, y)
}

// renderNotification displays flash notification for new messages
func (w *TelegramWidget) renderNotification(img *image.Gray) {
	if w.lastNotification == nil {
		// Show last message when no active notification
		w.renderLastMessage(img)
		return
	}

	msg := *w.lastNotification

	// Flash effect based on time
	elapsed := time.Since(w.notificationStart).Seconds()
	flash := (int(elapsed*4) % 2) == 0

	// Draw with flash effect
	brightness := w.textColor
	if !flash {
		brightness = brightness / 2
	}

	y := 2

	// Draw header
	header := w.formatHeader(msg)
	if header != "" {
		w.drawTextWithBrightness(img, header, 2, y, brightness)
		y += 9
	}

	// Draw message
	text := msg.Text
	if len(text) > w.truncateLength {
		text = text[:w.truncateLength-3] + "..."
	}
	w.drawTextWithBrightness(img, text, 2, y, brightness)
}

// formatHeader creates the header string for a message
func (w *TelegramWidget) formatHeader(msg tgclient.MessageInfo) string {
	var parts []string

	if w.showTime {
		parts = append(parts, msg.Time.Format(w.timeFormat))
	}

	if w.showChat && msg.ChatType != tgclient.ChatTypePrivate {
		parts = append(parts, msg.ChatTitle)
	}

	if w.showSender && msg.SenderName != "" {
		parts = append(parts, msg.SenderName)
	}

	if len(parts) == 0 {
		return ""
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// drawBadge draws unread count badge in top-right corner
func (w *TelegramWidget) drawBadge(img *image.Gray, count int) {
	countStr := fmt.Sprintf("%d", count)
	if count > 99 {
		countStr = "99+"
	}

	charWidth := 4
	padding := 2
	badgeWidth := len(countStr)*charWidth + padding*2
	badgeHeight := 7

	x := w.width - badgeWidth - 1
	y := 1

	// Draw badge background
	for dy := 0; dy < badgeHeight; dy++ {
		for dx := 0; dx < badgeWidth; dx++ {
			img.SetGray(x+dx, y+dy, color.Gray{Y: 255})
		}
	}

	// Draw count in black
	w.drawTextWithColor(img, countStr, x+padding, y, 0)
}

// drawText draws text at position
func (w *TelegramWidget) drawText(img *image.Gray, text string, x, y int) {
	w.drawTextWithBrightness(img, text, x, y, w.textColor)
}

// drawTextSmall draws smaller text (using 3x5 font if available)
func (w *TelegramWidget) drawTextSmall(img *image.Gray, text string, x, y int) {
	smallFont := bitmap.GetInternalFontByName("3x5")
	if smallFont == nil {
		smallFont = w.glyphSet
	}
	c := color.Gray{Y: w.textColor}

	for i, ch := range text {
		glyph := glyphs.GetGlyph(smallFont, ch)
		if glyph == nil {
			continue
		}

		charX := x + i*4
		for row := 0; row < glyph.Height && row < len(glyph.Data); row++ {
			for col := 0; col < glyph.Width && col < len(glyph.Data[row]); col++ {
				if glyph.Data[row][col] {
					px, py := charX+col, y+row
					if px >= 0 && px < w.width && py >= 0 && py < w.height {
						img.Set(px, py, c)
					}
				}
			}
		}
	}
}

// drawTextWithBrightness draws text with specific brightness
func (w *TelegramWidget) drawTextWithBrightness(img *image.Gray, text string, x, y int, brightness uint8) {
	w.drawTextWithColor(img, text, x, y, brightness)
}

// drawTextWithColor draws text with specific color value
func (w *TelegramWidget) drawTextWithColor(img *image.Gray, text string, x, y int, brightness uint8) {
	c := color.Gray{Y: brightness}

	for i, ch := range text {
		glyph := glyphs.GetGlyph(w.glyphSet, ch)
		if glyph == nil {
			continue
		}

		charX := x + i*6
		for row := 0; row < glyph.Height && row < len(glyph.Data); row++ {
			for col := 0; col < glyph.Width && col < len(glyph.Data[row]); col++ {
				if glyph.Data[row][col] {
					px, py := charX+col, y+row
					if px >= 0 && px < w.width && py >= 0 && py < w.height {
						img.Set(px, py, c)
					}
				}
			}
		}
	}
}

// Stop cleans up resources
func (w *TelegramWidget) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancel != nil {
		w.cancel()
	}
	if w.client != nil {
		w.client.Disconnect()
	}
}
