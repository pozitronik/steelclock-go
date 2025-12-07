package widget

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	tgclient "github.com/pozitronik/steelclock-go/internal/telegram"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"golang.org/x/image/font"
)

// ElementAppearance holds processed appearance settings for header or message
type ElementAppearance struct {
	Enabled    bool
	Blink      bool
	FontFace   font.Face
	FontName   string
	FontSize   int
	HorizAlign string
	VertAlign  string
	// Scroll settings
	ScrollEnabled   bool
	ScrollDirection string
	ScrollSpeed     float64
	ScrollMode      string
	ScrollGap       int
	// Word break mode: "normal" or "break-all"
	WordBreak string
}

// ChatAppearance holds full appearance settings for a chat type
type ChatAppearance struct {
	Header      ElementAppearance
	Message     ElementAppearance
	Separator   config.SeparatorConfig
	Timeout     int
	Transitions config.TransitionConfig
}

// TelegramWidget displays Telegram notifications
type TelegramWidget struct {
	*BaseWidget
	mu sync.RWMutex

	// Telegram client (shared via registry)
	client  *tgclient.Client
	authCfg *config.TelegramAuthConfig // stored for releasing client
	ctx     context.Context
	cancel  context.CancelFunc

	// Appearance (single appearance for all chat types now)
	appearance ChatAppearance

	// State
	messages           []tgclient.MessageInfo
	currentMessage     *tgclient.MessageInfo
	messageStartTime   time.Time
	dismissedMessageID int // Track dismissed message to prevent re-showing after timeout
	connectionError    error
	connecting         bool
	lastConnectionTry  time.Time
	reconnectInterval  time.Duration

	// Scroll state (per element)
	headerScrollOffset  float64
	messageScrollOffset float64
	lastUpdateTime      time.Time

	// Blink state
	blinkState bool
	lastBlink  time.Time

	// Transition state
	transitionActive    bool
	transitionProgress  float64
	transitionStartTime time.Time
	oldFrame            *image.Gray
	activeTransition    string
	pixelOrder          []int

	// Display dimensions
	width  int
	height int

	// Status text renderer for connection/error messages
	statusRenderer *shared.StatusRenderer
}

// NewTelegramWidget creates a new Telegram notification widget
func NewTelegramWidget(cfg config.WidgetConfig) (*TelegramWidget, error) {
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

	// Parse appearance from config
	appearance, err := parseAppearance(cfg.Appearance)
	if err != nil {
		return nil, fmt.Errorf("failed to parse appearance: %w", err)
	}

	// Create status renderer for connection/error messages
	statusRenderer := shared.NewStatusRenderer("5x7")

	w := &TelegramWidget{
		BaseWidget:        base,
		client:            client,
		authCfg:           cfg.Auth,
		appearance:        appearance,
		messages:          make([]tgclient.MessageInfo, 0),
		reconnectInterval: 30 * time.Second,
		width:             pos.W,
		height:            pos.H,
		statusRenderer:    statusRenderer,
		lastUpdateTime:    time.Now(),
		lastBlink:         time.Now(),
	}

	// Set message callback
	client.SetMessageCallback(func(msg tgclient.MessageInfo) {
		w.mu.Lock()
		defer w.mu.Unlock()

		// Update messages list
		w.messages = w.client.GetMessages()

		// Start transition if we have a current message
		if w.currentMessage != nil {
			w.startTransition(msg.ChatType)
		}

		// Set new current message
		msgCopy := msg
		w.currentMessage = &msgCopy
		w.messageStartTime = time.Now()
		w.dismissedMessageID = 0 // Reset dismissed ID when new message arrives

		// Reset scroll offsets for new message
		w.headerScrollOffset = 0
		w.messageScrollOffset = 0
	})

	// Set error callback
	client.SetErrorCallback(func(err error) {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.connectionError = err
	})

	return w, nil
}

// parseAppearance converts config to processed appearance settings
func parseAppearance(appCfg *config.TelegramAppearanceConfig) (ChatAppearance, error) {
	appearance := ChatAppearance{
		Header: ElementAppearance{
			Enabled:         true,
			Blink:           false,
			FontName:        "",
			FontSize:        16,
			HorizAlign:      "left",
			VertAlign:       "top",
			ScrollEnabled:   true,
			ScrollDirection: "left",
			ScrollSpeed:     30,
			ScrollMode:      "continuous",
			ScrollGap:       20,
			WordBreak:       "normal",
		},
		Message: ElementAppearance{
			Enabled:         true,
			Blink:           false,
			FontName:        "",
			FontSize:        16,
			HorizAlign:      "left",
			VertAlign:       "top",
			ScrollEnabled:   true,
			ScrollDirection: "left",
			ScrollSpeed:     30,
			ScrollMode:      "continuous",
			ScrollGap:       20,
			WordBreak:       "normal",
		},
		Separator: config.SeparatorConfig{
			Color:     128,
			Thickness: 1,
		},
		Timeout: 0,
		Transitions: config.TransitionConfig{
			In:       "none",
			InSpeed:  0.5,
			Out:      "none",
			OutSpeed: 0.5,
		},
	}

	if appCfg == nil {
		// Load default fonts
		var err error
		appearance.Header.FontFace, err = bitmap.LoadFont("", appearance.Header.FontSize)
		if err != nil {
			return appearance, err
		}
		appearance.Message.FontFace, err = bitmap.LoadFont("", appearance.Message.FontSize)
		if err != nil {
			return appearance, err
		}
		return appearance, nil
	}

	app := appCfg

	// Parse header appearance
	if app.Header != nil {
		if app.Header.Enabled != nil {
			appearance.Header.Enabled = *app.Header.Enabled
		}
		appearance.Header.Blink = app.Header.Blink

		if app.Header.Text != nil {
			if app.Header.Text.Font != "" {
				appearance.Header.FontName = app.Header.Text.Font
			}
			if app.Header.Text.Size > 0 {
				appearance.Header.FontSize = app.Header.Text.Size
			}
			if app.Header.Text.Align != nil {
				if app.Header.Text.Align.H != "" {
					appearance.Header.HorizAlign = app.Header.Text.Align.H
				}
				if app.Header.Text.Align.V != "" {
					appearance.Header.VertAlign = app.Header.Text.Align.V
				}
			}
		}

		if app.Header.Scroll != nil {
			appearance.Header.ScrollEnabled = app.Header.Scroll.Enabled
			if app.Header.Scroll.Direction != "" {
				appearance.Header.ScrollDirection = app.Header.Scroll.Direction
			}
			if app.Header.Scroll.Speed > 0 {
				appearance.Header.ScrollSpeed = app.Header.Scroll.Speed
			}
			if app.Header.Scroll.Mode != "" {
				appearance.Header.ScrollMode = app.Header.Scroll.Mode
			}
			if app.Header.Scroll.Gap > 0 {
				appearance.Header.ScrollGap = app.Header.Scroll.Gap
			}
		}

		if app.Header.WordBreak != "" {
			appearance.Header.WordBreak = app.Header.WordBreak
		}
	}

	// Parse message appearance
	if app.Message != nil {
		if app.Message.Enabled != nil {
			appearance.Message.Enabled = *app.Message.Enabled
		}
		appearance.Message.Blink = app.Message.Blink

		if app.Message.Text != nil {
			if app.Message.Text.Font != "" {
				appearance.Message.FontName = app.Message.Text.Font
			}
			if app.Message.Text.Size > 0 {
				appearance.Message.FontSize = app.Message.Text.Size
			}
			if app.Message.Text.Align != nil {
				if app.Message.Text.Align.H != "" {
					appearance.Message.HorizAlign = app.Message.Text.Align.H
				}
				if app.Message.Text.Align.V != "" {
					appearance.Message.VertAlign = app.Message.Text.Align.V
				}
			}
		}

		if app.Message.Scroll != nil {
			appearance.Message.ScrollEnabled = app.Message.Scroll.Enabled
			if app.Message.Scroll.Direction != "" {
				appearance.Message.ScrollDirection = app.Message.Scroll.Direction
			}
			if app.Message.Scroll.Speed > 0 {
				appearance.Message.ScrollSpeed = app.Message.Scroll.Speed
			}
			if app.Message.Scroll.Mode != "" {
				appearance.Message.ScrollMode = app.Message.Scroll.Mode
			}
			if app.Message.Scroll.Gap > 0 {
				appearance.Message.ScrollGap = app.Message.Scroll.Gap
			}
		}

		if app.Message.WordBreak != "" {
			appearance.Message.WordBreak = app.Message.WordBreak
		}
	}

	// Parse separator
	if app.Separator != nil {
		appearance.Separator.Color = app.Separator.Color
		appearance.Separator.Thickness = app.Separator.Thickness
	}

	// Parse timeout
	appearance.Timeout = app.Timeout

	// Parse transitions
	if app.Transitions != nil {
		if app.Transitions.In != "" {
			appearance.Transitions.In = app.Transitions.In
		}
		if app.Transitions.InSpeed > 0 {
			appearance.Transitions.InSpeed = app.Transitions.InSpeed
		}
		if app.Transitions.Out != "" {
			appearance.Transitions.Out = app.Transitions.Out
		}
		if app.Transitions.OutSpeed > 0 {
			appearance.Transitions.OutSpeed = app.Transitions.OutSpeed
		}
	}

	// Load fonts
	var err error
	appearance.Header.FontFace, err = bitmap.LoadFont(appearance.Header.FontName, appearance.Header.FontSize)
	if err != nil {
		return appearance, fmt.Errorf("failed to load header font: %w", err)
	}

	appearance.Message.FontFace, err = bitmap.LoadFont(appearance.Message.FontName, appearance.Message.FontSize)
	if err != nil {
		return appearance, fmt.Errorf("failed to load message font: %w", err)
	}

	return appearance, nil
}

// getAppearance returns the appearance settings (now single appearance for all chat types)
func (w *TelegramWidget) getAppearance(_ tgclient.ChatType) *ChatAppearance {
	return &w.appearance
}

// Update handles widget state updates
func (w *TelegramWidget) Update() error {
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

	// Handle transition progress
	if w.transitionActive && w.currentMessage != nil {
		appearance := w.getAppearance(w.currentMessage.ChatType)
		transitionSpeed := appearance.Transitions.InSpeed
		if transitionSpeed <= 0 {
			transitionSpeed = 0.5
		}

		elapsed := now.Sub(w.transitionStartTime).Seconds()
		w.transitionProgress = elapsed / transitionSpeed
		if w.transitionProgress >= 1.0 {
			w.transitionProgress = 1.0
			w.transitionActive = false
			w.oldFrame = nil
		}
	} else if w.transitionActive && w.currentMessage == nil {
		// Cancel transition if message was cleared
		w.transitionActive = false
		w.oldFrame = nil
	}

	// Update scroll offsets
	if w.currentMessage != nil {
		elapsed := now.Sub(w.lastUpdateTime).Seconds()
		appearance := w.getAppearance(w.currentMessage.ChatType)

		// Update header scroll
		if appearance.Header.ScrollEnabled {
			w.headerScrollOffset += appearance.Header.ScrollSpeed * elapsed
		}

		// Update message scroll
		if appearance.Message.ScrollEnabled {
			w.messageScrollOffset += appearance.Message.ScrollSpeed * elapsed
		}
	}
	w.lastUpdateTime = now

	// Check message timeout
	if w.currentMessage != nil {
		appearance := w.getAppearance(w.currentMessage.ChatType)
		if appearance.Timeout > 0 {
			if time.Since(w.messageStartTime).Seconds() >= float64(appearance.Timeout) {
				// Remember dismissed message ID to prevent re-showing
				w.dismissedMessageID = w.currentMessage.ID
				w.currentMessage = nil
				w.headerScrollOffset = 0
				w.messageScrollOffset = 0
			}
		}
	}

	// Refresh messages from client
	if w.client.IsConnected() {
		w.messages = w.client.GetMessages()
		// If no current message, show latest (unless it was dismissed)
		if w.currentMessage == nil && len(w.messages) > 0 {
			// Skip dismissed message
			if w.messages[0].ID != w.dismissedMessageID {
				w.currentMessage = &w.messages[0]
				w.messageStartTime = time.Now()
				w.dismissedMessageID = 0 // Reset when showing new message
			}
		}
	}

	return nil
}

// startTransition initiates a transition to a new message
func (w *TelegramWidget) startTransition(chatType tgclient.ChatType) {
	appearance := w.getAppearance(chatType)
	if appearance.Transitions.In == "none" {
		return
	}

	// Capture current frame
	w.oldFrame = bitmap.NewGrayscaleImage(w.width, w.height, w.GetRenderBackgroundColor())
	if w.currentMessage != nil {
		w.renderMessage(w.oldFrame, *w.currentMessage)
	}

	// Set up transition
	w.transitionActive = true
	w.transitionProgress = 0.0
	w.transitionStartTime = time.Now()
	w.activeTransition = w.selectTransition(appearance.Transitions.In)

	// Pre-generate pixel order for dissolve_pixel
	if w.activeTransition == "dissolve_pixel" {
		w.pixelOrder = w.generatePixelOrder(w.width, w.height)
	}
}

// selectTransition returns the actual transition type (handles "random")
func (w *TelegramWidget) selectTransition(transitionType string) string {
	if transitionType != "random" {
		return transitionType
	}

	transitions := []string{
		"push_left", "push_right", "push_up", "push_down",
		"slide_left", "slide_right", "slide_up", "slide_down",
		"dissolve_fade", "dissolve_pixel", "dissolve_dither",
		"box_in", "box_out", "clock_wipe",
	}
	return transitions[rand.Intn(len(transitions))]
}

// generatePixelOrder creates a shuffled list of pixel indices for dissolve_pixel
func (w *TelegramWidget) generatePixelOrder(width, height int) []int {
	total := width * height
	order := make([]int, total)
	for i := 0; i < total; i++ {
		order[i] = i
	}
	for i := total - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	return order
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
		w.drawStatusText(img, "Connecting...")
	} else if w.connectionError != nil {
		w.renderError(img)
	} else if !w.client.IsConnected() {
		w.drawStatusText(img, "Disconnected")
	} else if w.currentMessage == nil {
		// No message to display - return empty/transparent widget
		// (don't show "No messages" - widget should disappear after timeout)
	} else {
		// Handle transition
		if w.transitionActive && w.oldFrame != nil {
			newFrame := bitmap.NewGrayscaleImage(w.width, w.height, w.GetRenderBackgroundColor())
			w.renderMessage(newFrame, *w.currentMessage)
			w.applyTransition(img, w.oldFrame, newFrame, w.transitionProgress, w.activeTransition, w.pixelOrder)
		} else {
			w.renderMessage(img, *w.currentMessage)
		}
	}

	// Draw border if enabled
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, nil
}

// renderError displays error message on two lines
func (w *TelegramWidget) renderError(img *image.Gray) {
	errMsg := w.connectionError.Error()
	maxLen := 22
	line1 := errMsg
	line2 := ""
	if len(errMsg) > maxLen {
		line1 = errMsg[:maxLen]
		if len(errMsg) > maxLen*2 {
			line2 = errMsg[maxLen : maxLen*2]
		} else {
			line2 = errMsg[maxLen:]
		}
	}
	w.drawStatusText(img, line1)
	if line2 != "" {
		// Draw second line below
		w.statusRenderer.DrawAt(img, line2, 2, w.height/2+2)
	}
}

// drawStatusText draws status text using internal font
func (w *TelegramWidget) drawStatusText(img *image.Gray, text string) {
	w.statusRenderer.DrawAt(img, text, 2, w.height/2-3)
}

// renderMessage renders a message with header, separator, and message text
func (w *TelegramWidget) renderMessage(img *image.Gray, msg tgclient.MessageInfo) {
	appearance := w.getAppearance(msg.ChatType)

	// Calculate layout based on what's enabled
	headerHeight := 0
	separatorY := 0
	messageY := 0

	// Calculate header height if enabled
	if appearance.Header.Enabled {
		_, textHeight := bitmap.SmartMeasureText("Ag", appearance.Header.FontFace, appearance.Header.FontName)
		if textHeight == 0 {
			textHeight = 16 // fallback if font measurement fails
		}
		headerHeight = textHeight + 2
	}

	// Calculate separator and message Y positions
	if appearance.Header.Enabled {
		separatorY = headerHeight
		// Add separator space only if separator is enabled (color >= 0 and thickness > 0)
		if appearance.Separator.Color >= 0 && appearance.Separator.Thickness > 0 && appearance.Message.Enabled {
			messageY = separatorY + appearance.Separator.Thickness + 1
		} else {
			messageY = separatorY
		}
	}

	// Render header
	if appearance.Header.Enabled {
		headerText := w.formatHeader(msg)
		if headerText != "" {
			// Apply blink effect
			if appearance.Header.Blink && !w.blinkState {
				// Skip rendering when blinking off
			} else {
				w.renderScrollingText(img, headerText, appearance.Header, w.headerScrollOffset, 0, 0, w.width, headerHeight)
			}
		}
	}

	// Render separator
	if appearance.Separator.Color >= 0 && appearance.Separator.Thickness > 0 && appearance.Header.Enabled && appearance.Message.Enabled {
		for dy := 0; dy < appearance.Separator.Thickness; dy++ {
			for x := 0; x < w.width; x++ {
				img.SetGray(x, separatorY+dy, color.Gray{Y: uint8(appearance.Separator.Color)})
			}
		}
	}

	// Render message area
	msgHeight := w.height - messageY
	if msgHeight > 0 {
		var messageText string
		if appearance.Message.Enabled {
			messageText = msg.Text
			// If text is empty but there's media, show media type as placeholder
			if messageText == "" && msg.MediaType != "" {
				messageText = "[" + msg.MediaType + "]"
			}
		} else {
			// Show placeholder when message display is disabled
			messageText = "You have a new message"
		}

		// Apply blink effect
		if appearance.Message.Blink && !w.blinkState {
			// Skip rendering when blinking off
		} else {
			w.renderMultiLineText(img, messageText, appearance.Message, w.messageScrollOffset, 0, messageY, w.width, msgHeight)
		}
	}
}

// wrapText wraps text into lines that fit within the given width
// wordBreak: "normal" (break on spaces/newlines) or "break-all" (break anywhere)
func (w *TelegramWidget) wrapText(text string, elem ElementAppearance, maxWidth int) []string {
	if text == "" {
		return nil
	}

	var lines []string
	var currentLine string

	// Helper to measure text width
	measureWidth := func(s string) int {
		width, _ := bitmap.SmartMeasureText(s, elem.FontFace, elem.FontName)
		return width
	}

	// Helper to add a word to current line or start new line
	addWord := func(word string) {
		if currentLine == "" {
			// Check if word itself fits
			if measureWidth(word) <= maxWidth {
				currentLine = word
			} else {
				// Word doesn't fit - break it character by character
				for _, r := range word {
					ch := string(r)
					testLine := currentLine + ch
					if measureWidth(testLine) <= maxWidth {
						currentLine = testLine
					} else {
						if currentLine != "" {
							lines = append(lines, currentLine)
						}
						currentLine = ch
					}
				}
			}
		} else {
			testLine := currentLine + " " + word
			if measureWidth(testLine) <= maxWidth {
				currentLine = testLine
			} else {
				// Word doesn't fit on current line
				lines = append(lines, currentLine)
				// Check if word fits on new line
				if measureWidth(word) <= maxWidth {
					currentLine = word
				} else {
					// Break the word character by character
					currentLine = ""
					for _, r := range word {
						ch := string(r)
						testLine := currentLine + ch
						if measureWidth(testLine) <= maxWidth {
							currentLine = testLine
						} else {
							if currentLine != "" {
								lines = append(lines, currentLine)
							}
							currentLine = ch
						}
					}
				}
			}
		}
	}

	if elem.WordBreak == "break-all" {
		// Break at any character
		for _, r := range text {
			if r == '\n' {
				lines = append(lines, currentLine)
				currentLine = ""
				continue
			}
			ch := string(r)
			testLine := currentLine + ch
			if measureWidth(testLine) <= maxWidth {
				currentLine = testLine
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = ch
			}
		}
	} else {
		// Normal word break - break on spaces and newlines
		// Split by newlines first
		paragraphs := splitByNewlines(text)
		for i, para := range paragraphs {
			if i > 0 {
				// Add previous line before starting new paragraph
				if currentLine != "" {
					lines = append(lines, currentLine)
					currentLine = ""
				}
			}
			// Split paragraph into words
			words := splitIntoWords(para)
			for _, word := range words {
				if word != "" {
					addWord(word)
				}
			}
		}
	}

	// Don't forget the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// splitByNewlines splits text by newline characters
func splitByNewlines(text string) []string {
	var result []string
	current := ""
	for _, r := range text {
		if r == '\n' || r == '\r' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	result = append(result, current)
	return result
}

// splitIntoWords splits text into words by spaces
func splitIntoWords(text string) []string {
	var words []string
	current := ""
	for _, r := range text {
		if r == ' ' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// truncateWithEllipsis truncates lines that don't fit and adds "..." to the last visible line
func (w *TelegramWidget) truncateWithEllipsis(lines []string, elem ElementAppearance, maxWidth, maxHeight int) []string {
	if len(lines) == 0 {
		return lines
	}

	_, lineHeight := bitmap.SmartMeasureText("Ag", elem.FontFace, elem.FontName)
	if lineHeight == 0 {
		lineHeight = 16
	}

	maxLines := maxHeight / lineHeight
	if maxLines <= 0 {
		maxLines = 1
	}

	if len(lines) <= maxLines {
		return lines
	}

	// Truncate to maxLines and add ellipsis to last line
	result := lines[:maxLines]
	lastLine := result[maxLines-1]

	ellipsis := "..."
	ellipsisWidth, _ := bitmap.SmartMeasureText(ellipsis, elem.FontFace, elem.FontName)

	// Remove characters from end until ellipsis fits
	for len(lastLine) > 0 {
		lineWidth, _ := bitmap.SmartMeasureText(lastLine+ellipsis, elem.FontFace, elem.FontName)
		if lineWidth <= maxWidth {
			break
		}
		// Remove last rune
		runes := []rune(lastLine)
		lastLine = string(runes[:len(runes)-1])
	}

	// If the line is too short, try without removing characters
	if lastLine == "" {
		testWidth, _ := bitmap.SmartMeasureText(ellipsis, elem.FontFace, elem.FontName)
		if testWidth <= maxWidth {
			lastLine = ""
		}
	}

	result[maxLines-1] = lastLine + ellipsis

	// Handle edge case where ellipsis alone doesn't fit
	if ellipsisWidth > maxWidth && maxLines > 1 {
		result = result[:maxLines-1]
	}

	return result
}

// renderMultiLineText renders wrapped text with optional vertical scrolling
func (w *TelegramWidget) renderMultiLineText(img *image.Gray, text string, elem ElementAppearance, scrollOffset float64, x, y, width, height int) {
	// Wrap text into lines
	lines := w.wrapText(text, elem, width)
	if len(lines) == 0 {
		return
	}

	_, lineHeight := bitmap.SmartMeasureText("Ag", elem.FontFace, elem.FontName)
	if lineHeight == 0 {
		lineHeight = 16
	}

	totalTextHeight := len(lines) * lineHeight

	// Check if scrolling is needed
	if totalTextHeight <= height || !elem.ScrollEnabled {
		// No scrolling - truncate with ellipsis if needed
		if totalTextHeight > height {
			lines = w.truncateWithEllipsis(lines, elem, width, height)
		}

		// Render lines
		currentY := y
		for _, line := range lines {
			if currentY+lineHeight > y+height {
				break
			}
			w.renderSingleLine(img, line, elem, x, currentY, width, lineHeight)
			currentY += lineHeight
		}
		return
	}

	// Scrolling enabled - vertical scroll through lines
	totalScrollHeight := totalTextHeight + elem.ScrollGap

	switch elem.ScrollMode {
	case "continuous":
		offset := int(scrollOffset) % totalScrollHeight
		startY := y - offset

		// Draw lines with wrapping
		for i := 0; i < 2; i++ { // Draw twice for seamless loop
			lineY := startY + i*totalScrollHeight
			for _, line := range lines {
				if lineY+lineHeight > y && lineY < y+height {
					// Line is visible
					w.renderSingleLine(img, line, elem, x, lineY, width, lineHeight)
				}
				lineY += lineHeight
			}
		}

	case "bounce":
		maxOffset := float64(totalTextHeight - height)
		if maxOffset <= 0 {
			// Fits without scrolling
			currentY := y
			for _, line := range lines {
				w.renderSingleLine(img, line, elem, x, currentY, width, lineHeight)
				currentY += lineHeight
			}
			return
		}

		offset := scrollOffset
		cycle := int(offset / maxOffset)
		progress := offset - float64(cycle)*maxOffset
		if cycle%2 == 1 {
			progress = maxOffset - progress
		}
		startY := y - int(progress)

		for _, line := range lines {
			if startY+lineHeight > y && startY < y+height {
				w.renderSingleLine(img, line, elem, x, startY, width, lineHeight)
			}
			startY += lineHeight
		}

	case "pause_ends":
		maxOffset := float64(totalTextHeight - height)
		if maxOffset <= 0 {
			currentY := y
			for _, line := range lines {
				w.renderSingleLine(img, line, elem, x, currentY, width, lineHeight)
				currentY += lineHeight
			}
			return
		}

		pausePixels := 100
		offset := int(scrollOffset) % (int(maxOffset) + pausePixels)
		if offset > int(maxOffset) {
			offset = int(maxOffset)
		}
		startY := y - offset

		for _, line := range lines {
			if startY+lineHeight > y && startY < y+height {
				w.renderSingleLine(img, line, elem, x, startY, width, lineHeight)
			}
			startY += lineHeight
		}

	default:
		// No scroll mode - just render what fits
		currentY := y
		for _, line := range lines {
			if currentY+lineHeight > y+height {
				break
			}
			w.renderSingleLine(img, line, elem, x, currentY, width, lineHeight)
			currentY += lineHeight
		}
	}
}

// renderSingleLine renders a single line of text
func (w *TelegramWidget) renderSingleLine(img *image.Gray, text string, elem ElementAppearance, x, y, width, height int) {
	// Calculate position with alignment
	textX, textY := bitmap.SmartCalculateTextPosition(text, elem.FontFace, elem.FontName, x, y, width, height, elem.HorizAlign, elem.VertAlign)
	bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, textX, textY, x, y, width, height)
}

// renderScrollingText renders text with scrolling support (single line, horizontal scroll)
func (w *TelegramWidget) renderScrollingText(img *image.Gray, text string, elem ElementAppearance, scrollOffset float64, x, y, width, height int) {
	textWidth, _ := bitmap.SmartMeasureText(text, elem.FontFace, elem.FontName)

	// Calculate base position using SmartCalculateTextPosition which handles TTF baseline correctly
	textX, textY := bitmap.SmartCalculateTextPosition(text, elem.FontFace, elem.FontName, x, y, width, height, elem.HorizAlign, elem.VertAlign)

	// If text fits or scrolling disabled, draw normally
	if textWidth <= width || !elem.ScrollEnabled {
		bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, textX, textY, x, y, width, height)
		return
	}

	// Handle scrolling - text is wider than container
	totalWidth := textWidth + elem.ScrollGap

	switch elem.ScrollMode {
	case "continuous":
		// Wrap scroll offset
		offset := int(scrollOffset) % totalWidth
		// Start at x position and scroll left
		scrollX := x - offset

		// Draw first copy
		bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, scrollX, textY, x, y, width, height)

		// Draw second copy for seamless loop
		scrollX2 := scrollX + totalWidth
		if scrollX2 < x+width {
			bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, scrollX2, textY, x, y, width, height)
		}

	case "bounce":
		maxOffset := float64(textWidth - width)
		if maxOffset <= 0 {
			// Text fits, no bouncing needed
			bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, textX, textY, x, y, width, height)
			return
		}
		offset := scrollOffset
		// Bounce back and forth
		cycle := int(offset / maxOffset)
		progress := offset - float64(cycle)*maxOffset
		if cycle%2 == 1 {
			progress = maxOffset - progress
		}
		scrollX := x - int(progress)
		bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, scrollX, textY, x, y, width, height)

	case "pause_ends":
		maxOffset := float64(textWidth - width)
		if maxOffset <= 0 {
			// Text fits, no scrolling needed
			bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, textX, textY, x, y, width, height)
			return
		}
		pausePixels := 100
		offset := int(scrollOffset) % (int(maxOffset) + pausePixels)
		if offset > int(maxOffset) {
			offset = int(maxOffset) // Pause at end
		}
		scrollX := x - offset
		bitmap.SmartDrawTextAtPosition(img, text, elem.FontFace, elem.FontName, scrollX, textY, x, y, width, height)
	}
}

// formatHeader creates the header string for a message
func (w *TelegramWidget) formatHeader(msg tgclient.MessageInfo) string {
	switch msg.ChatType {
	case tgclient.ChatTypePrivate:
		if msg.SenderName != "" {
			return msg.SenderName
		}
		return fmt.Sprintf("User %d", msg.ChatID)
	case tgclient.ChatTypeGroup, tgclient.ChatTypeChannel:
		if msg.ChatTitle != "" {
			return msg.ChatTitle
		}
		return fmt.Sprintf("Chat %d", msg.ChatID)
	default:
		return ""
	}
}

// applyTransition composites old and new frames based on transition type and progress
func (w *TelegramWidget) applyTransition(dst, oldFrame, newFrame *image.Gray, progress float64, transitionType string, pixelOrder []int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	switch transitionType {
	case "none":
		if progress < 0.5 {
			shared.CopyGrayImage(dst, oldFrame)
		} else {
			shared.CopyGrayImage(dst, newFrame)
		}

	case "push_left":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, -1, 0)
	case "push_right":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, 1, 0)
	case "push_up":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, 0, -1)
	case "push_down":
		w.applyPushTransition(dst, oldFrame, newFrame, progress, 0, 1)

	case "slide_left":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, -1, 0)
	case "slide_right":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, 1, 0)
	case "slide_up":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, 0, -1)
	case "slide_down":
		w.applySlideTransition(dst, oldFrame, newFrame, progress, 0, 1)

	case "dissolve_fade":
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				oldVal := oldFrame.GrayAt(x, y).Y
				newVal := newFrame.GrayAt(x, y).Y
				blended := uint8(float64(oldVal)*(1-progress) + float64(newVal)*progress)
				dst.SetGray(x, y, color.Gray{Y: blended})
			}
		}

	case "dissolve_pixel":
		shared.CopyGrayImage(dst, oldFrame)
		pixelsToShow := int(float64(len(pixelOrder)) * progress)
		for i := 0; i < pixelsToShow && i < len(pixelOrder); i++ {
			idx := pixelOrder[i]
			x := idx % width
			y := idx / width
			dst.SetGray(x, y, newFrame.GrayAt(x, y))
		}

	case "dissolve_dither":
		threshold := uint8(progress * 255)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				ditherVal := uint8(((x ^ y) * 17) % 256)
				if ditherVal < threshold {
					dst.SetGray(x, y, newFrame.GrayAt(x, y))
				} else {
					dst.SetGray(x, y, oldFrame.GrayAt(x, y))
				}
			}
		}

	case "box_in":
		centerX, centerY := width/2, height/2
		maxDist := centerX
		if centerY > maxDist {
			maxDist = centerY
		}
		revealDist := int(float64(maxDist) * progress)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dx := telegramAbs(x - centerX)
				dy := telegramAbs(y - centerY)
				dist := dx
				if dy > dist {
					dist = dy
				}
				if dist <= revealDist {
					dst.SetGray(x, y, newFrame.GrayAt(x, y))
				} else {
					dst.SetGray(x, y, oldFrame.GrayAt(x, y))
				}
			}
		}

	case "box_out":
		centerX, centerY := width/2, height/2
		maxDist := centerX
		if centerY > maxDist {
			maxDist = centerY
		}
		hideDist := int(float64(maxDist) * progress)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dx := telegramAbs(x - centerX)
				dy := telegramAbs(y - centerY)
				dist := dx
				if dy > dist {
					dist = dy
				}
				if dist >= maxDist-hideDist {
					dst.SetGray(x, y, newFrame.GrayAt(x, y))
				} else {
					dst.SetGray(x, y, oldFrame.GrayAt(x, y))
				}
			}
		}

	case "clock_wipe":
		pi := 3.14159265358979323846
		centerX, centerY := float64(width)/2, float64(height)/2
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dx := float64(x) - centerX
				dy := float64(y) - centerY
				angle := atan2(dy, dx) + pi
				normalizedAngle := angle / (2 * pi)
				if normalizedAngle <= progress {
					dst.SetGray(x, y, newFrame.GrayAt(x, y))
				} else {
					dst.SetGray(x, y, oldFrame.GrayAt(x, y))
				}
			}
		}

	default:
		shared.CopyGrayImage(dst, newFrame)
	}
}

// atan2 calculates arctangent of y/x
func atan2(y, x float64) float64 {
	if x > 0 {
		return atan(y / x)
	} else if x < 0 {
		if y >= 0 {
			return atan(y/x) + 3.14159265358979323846
		}
		return atan(y/x) - 3.14159265358979323846
	} else {
		if y > 0 {
			return 3.14159265358979323846 / 2
		} else if y < 0 {
			return -3.14159265358979323846 / 2
		}
		return 0
	}
}

// atan calculates arctangent using Taylor series
func atan(x float64) float64 {
	if x > 1 {
		return 3.14159265358979323846/2 - atan(1/x)
	} else if x < -1 {
		return -3.14159265358979323846/2 - atan(1/x)
	}
	result := x
	term := x
	for i := 1; i < 20; i++ {
		term *= -x * x
		result += term / float64(2*i+1)
	}
	return result
}

// applyPushTransition applies push transition
func (w *TelegramWidget) applyPushTransition(dst, oldFrame, newFrame *image.Gray, progress float64, dirX, dirY int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	offsetX := int(float64(width) * progress * float64(dirX))
	offsetY := int(float64(height) * progress * float64(dirY))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldX := x - offsetX
			oldY := y - offsetY
			newX := x - offsetX - width*dirX
			newY := y - offsetY - height*dirY

			if oldX >= 0 && oldX < width && oldY >= 0 && oldY < height {
				dst.SetGray(x, y, oldFrame.GrayAt(oldX, oldY))
			} else if newX >= 0 && newX < width && newY >= 0 && newY < height {
				dst.SetGray(x, y, newFrame.GrayAt(newX, newY))
			}
		}
	}
}

// applySlideTransition applies slide transition (new slides over old)
func (w *TelegramWidget) applySlideTransition(dst, oldFrame, newFrame *image.Gray, progress float64, dirX, dirY int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// First, copy old frame
	shared.CopyGrayImage(dst, oldFrame)

	// Calculate how much of new frame to show
	revealX := int(float64(width) * progress)
	revealY := int(float64(height) * progress)

	// Draw new frame sliding in
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var newX, newY int
			var inRange bool

			switch {
			case dirX < 0: // slide left
				newX = x + width - revealX
				newY = y
				inRange = x < revealX
			case dirX > 0: // slide right
				newX = x - (width - revealX)
				newY = y
				inRange = x >= width-revealX
			case dirY < 0: // slide up
				newX = x
				newY = y + height - revealY
				inRange = y < revealY
			case dirY > 0: // slide down
				newX = x
				newY = y - (height - revealY)
				inRange = y >= height-revealY
			}

			if inRange && newX >= 0 && newX < width && newY >= 0 && newY < height {
				dst.SetGray(x, y, newFrame.GrayAt(newX, newY))
			}
		}
	}
}

func telegramAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Stop cleans up resources
func (w *TelegramWidget) Stop() {
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
