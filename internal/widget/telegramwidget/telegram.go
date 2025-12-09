package telegramwidget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	tgclient "github.com/pozitronik/steelclock-go/internal/telegram"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"golang.org/x/image/font"
)

func init() {
	widget.Register("telegram", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Error display constants
const (
	// maxErrorLineLength is the maximum characters per line when displaying error messages
	maxErrorLineLength = 22
)

// ElementAppearance holds processed appearance settings for header or message
type ElementAppearance struct {
	Enabled    bool
	Blink      bool
	FontFace   font.Face
	FontName   string
	FontSize   int
	HorizAlign config.HAlign
	VertAlign  config.VAlign
	// Format string with tokens: {sender}, {chat}, {type}, {time}, {date}, {forwarded}
	Format string
	// Scroll settings
	ScrollEnabled   bool
	ScrollDirection shared.ScrollDirection
	ScrollSpeed     float64
	ScrollMode      shared.ScrollMode
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

// Widget displays Telegram notifications
type Widget struct {
	*widget.BaseWidget
	mu sync.RWMutex

	// Telegram client (shared via registry)
	client  *tgclient.Client
	authCfg *config.TelegramAuthConfig // stored for releasing client

	// Appearance (single appearance for all chat types now)
	appearance ChatAppearance

	// State
	messages           []tgclient.MessageInfo
	currentMessage     *tgclient.MessageInfo
	messageStartTime   time.Time
	dismissedMessageID int // Track dismissed message to prevent re-showing after timeout

	// Connection manager (shared module)
	connection *shared.ConnectionManager

	// Scroll state (per element)
	headerScroller  *shared.TextScroller
	messageScroller *shared.TextScroller

	// Blink animator
	blink *shared.BlinkAnimator

	// Transition manager
	transition *shared.TransitionManager

	// Display dimensions
	width  int
	height int

	// Status text renderer for connection/error messages
	statusRenderer *shared.StatusRenderer
}

// New creates a new Telegram notification widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
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

	// Create scrollers from appearance config
	headerScrollerCfg := shared.ScrollerConfig{
		Speed:     appearance.Header.ScrollSpeed,
		Mode:      appearance.Header.ScrollMode,
		Direction: appearance.Header.ScrollDirection,
		Gap:       appearance.Header.ScrollGap,
	}
	messageScrollerCfg := shared.ScrollerConfig{
		Speed:     appearance.Message.ScrollSpeed,
		Mode:      appearance.Message.ScrollMode,
		Direction: appearance.Message.ScrollDirection,
		Gap:       appearance.Message.ScrollGap,
	}

	w := &Widget{
		BaseWidget:      base,
		client:          client,
		authCfg:         cfg.Auth,
		appearance:      appearance,
		messages:        make([]tgclient.MessageInfo, 0),
		connection:      shared.NewConnectionManager(client, 30*time.Second, 60*time.Second),
		width:           pos.W,
		height:          pos.H,
		statusRenderer:  statusRenderer,
		headerScroller:  shared.NewTextScroller(headerScrollerCfg),
		messageScroller: shared.NewTextScroller(messageScrollerCfg),
		blink:           shared.NewBlinkAnimator(shared.BlinkAlways, 500*time.Millisecond),
		transition:      shared.NewTransitionManager(pos.W, pos.H),
	}

	// Add message callback (using Add instead of Set for proper multi-widget support)
	client.AddMessageCallback(func(msg tgclient.MessageInfo) {
		w.mu.Lock()
		defer w.mu.Unlock()

		// Update messages list
		w.messages = w.client.GetMessages()

		// Start transition for new message
		// This works for both:
		// - Transitioning between messages (currentMessage != nil)
		// - First message appearance (currentMessage == nil, transitions from empty)
		w.startTransition(msg.ChatType)

		// Set new current message
		msgCopy := msg
		w.currentMessage = &msgCopy
		w.messageStartTime = time.Now()
		w.dismissedMessageID = 0 // Reset dismissed ID when new message arrives

		// Reset scroll offsets for new message
		w.headerScroller.Reset()
		w.messageScroller.Reset()

		// Trigger auto-hide timer (widget becomes visible when message arrives)
		w.TriggerAutoHide()
	})

	// Add error callback (using Add instead of Set for proper multi-widget support)
	// Note: ConnectionManager handles connection errors internally,
	// but we keep this for client-level errors (e.g., message fetch failures)
	client.AddErrorCallback(func(err error) {
		// Connection manager handles errors internally
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
			HorizAlign:      config.AlignLeft,
			VertAlign:       config.AlignTop,
			Format:          "", // empty = auto format based on chat type
			ScrollEnabled:   true,
			ScrollDirection: shared.ScrollLeft,
			ScrollSpeed:     30,
			ScrollMode:      shared.ScrollContinuous,
			ScrollGap:       20,
			WordBreak:       "normal",
		},
		Message: ElementAppearance{
			Enabled:         true,
			Blink:           false,
			FontName:        "",
			FontSize:        16,
			HorizAlign:      config.AlignLeft,
			VertAlign:       config.AlignTop,
			Format:          "", // not used for message
			ScrollEnabled:   true,
			ScrollDirection: shared.ScrollLeft,
			ScrollSpeed:     30,
			ScrollMode:      shared.ScrollContinuous,
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
			if app.Header.Text.Format != "" {
				appearance.Header.Format = app.Header.Text.Format
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
func (w *Widget) getAppearance(_ tgclient.ChatType) *ChatAppearance {
	return &w.appearance
}

// Update handles widget state updates
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update blink state (pass message count for potential progressive blinking)
	w.blink.Update(len(w.messages))

	// Handle connection via shared manager
	w.connection.Update()

	// Handle transition progress
	if w.transition.IsActive() {
		if w.currentMessage != nil {
			w.transition.Update()
		} else {
			// Cancel transition if message was cleared
			w.transition.Cancel()
		}
	}

	// Update scroll offsets via scrollers
	// Note: scrollers manage their own timing internally
	// We pass 0 for content/container size here as the actual dimensions
	// are determined during rendering. This just advances the offset.
	if w.currentMessage != nil {
		appearance := w.getAppearance(w.currentMessage.ChatType)

		// Update header scroll
		// Pass large content size to prevent scroller's internal wrap-around
		// The rendering code handles wrap via modulo with actual content size
		if appearance.Header.ScrollEnabled {
			w.headerScroller.Update(1000000, w.width)
		}

		// Update message scroll - same approach
		// Actual wrap-around happens in renderMultiLineText based on real content height
		if appearance.Message.ScrollEnabled {
			w.messageScroller.Update(1000000, w.height)
		}
	}

	// Check message timeout
	if w.currentMessage != nil {
		appearance := w.getAppearance(w.currentMessage.ChatType)
		if appearance.Timeout > 0 {
			if time.Since(w.messageStartTime).Seconds() >= float64(appearance.Timeout) {
				// Remember dismissed message ID to prevent re-showing
				w.dismissedMessageID = w.currentMessage.ID
				w.currentMessage = nil
				w.headerScroller.Reset()
				w.messageScroller.Reset()
			}
		}
	}

	// Refresh messages from client
	if w.connection.IsConnected() {
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
func (w *Widget) startTransition(chatType tgclient.ChatType) {
	appearance := w.getAppearance(chatType)
	if appearance.Transitions.In == "none" {
		return
	}

	// Capture current frame
	oldFrame := bitmap.NewGrayscaleImage(w.width, w.height, w.GetRenderBackgroundColor())
	if w.currentMessage != nil {
		w.renderMessage(oldFrame, *w.currentMessage)
	}

	// Get transition duration
	transitionSpeed := appearance.Transitions.InSpeed
	if transitionSpeed <= 0 {
		transitionSpeed = 0.5
	}

	// Start transition via manager
	w.transition.Start(shared.TransitionType(appearance.Transitions.In), transitionSpeed, oldFrame)
}

// Render draws the widget
func (w *Widget) Render() (image.Image, error) {
	// Check if widget should be hidden (auto-hide mode)
	if w.ShouldHide() {
		return nil, nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	// Create canvas with background
	img := w.CreateCanvas()

	// Draw based on state
	if w.connection.IsConnecting() {
		w.drawStatusText(img, "Connecting...")
	} else if w.connection.GetError() != nil {
		w.renderError(img)
	} else if !w.connection.IsConnected() {
		// Show "Connecting..." on initial state (before first connection attempt)
		// to avoid brief "Disconnected" flash
		if w.connection.IsInitialState() {
			w.drawStatusText(img, "Connecting...")
		} else {
			w.drawStatusText(img, "Disconnected")
		}
	} else if w.currentMessage == nil {
		// No message to display - return empty/transparent widget
		// (don't show "No messages" - widget should disappear after timeout)
	} else {
		// Handle transition - use Live methods for accurate timing regardless of Update() frequency
		if w.transition.IsActiveLive() {
			newFrame := bitmap.NewGrayscaleImage(w.width, w.height, w.GetRenderBackgroundColor())
			w.renderMessage(newFrame, *w.currentMessage)
			w.transition.ApplyLive(img, newFrame)
		} else {
			w.renderMessage(img, *w.currentMessage)
		}
	}

	// Draw border if enabled
	w.ApplyBorder(img)

	return img, nil
}

// renderError displays error message on two lines
func (w *Widget) renderError(img *image.Gray) {
	connErr := w.connection.GetError()
	if connErr == nil {
		return
	}
	errMsg := connErr.Error()
	line1 := errMsg
	line2 := ""
	if len(errMsg) > maxErrorLineLength {
		line1 = errMsg[:maxErrorLineLength]
		if len(errMsg) > maxErrorLineLength*2 {
			line2 = errMsg[maxErrorLineLength : maxErrorLineLength*2]
		} else {
			line2 = errMsg[maxErrorLineLength:]
		}
	}
	w.drawStatusText(img, line1)
	if line2 != "" {
		// Draw second line below
		w.statusRenderer.DrawAt(img, line2, 2, w.height/2+2)
	}
}

// drawStatusText draws status text using internal font
func (w *Widget) drawStatusText(img *image.Gray, text string) {
	w.statusRenderer.DrawAt(img, text, 2, w.height/2-3)
}

// renderMessage renders a message with header, separator, and message text
// Each region (header, message) is rendered to a sub-image for proper clipping
func (w *Widget) renderMessage(img *image.Gray, msg tgclient.MessageInfo) {
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

	// Render header to sub-image (provides natural clipping)
	if appearance.Header.Enabled && headerHeight > 0 {
		headerText := w.formatHeader(msg)
		if headerText != "" {
			// Apply blink effect
			if appearance.Header.Blink && !w.blink.ShouldRender() {
				// Skip rendering when blinking off
			} else {
				// Create sub-image for header region
				headerImg := image.NewGray(image.Rect(0, 0, w.width, headerHeight))
				// Fill with background color
				bgColor := w.GetRenderBackgroundColor()
				for i := range headerImg.Pix {
					headerImg.Pix[i] = bgColor
				}
				// Render header (coordinates relative to sub-image: 0,0)
				w.renderScrollingText(headerImg, headerText, appearance.Header, w.headerScroller.GetOffset(), 0, 0, w.width, headerHeight)
				// Copy to main image at (0, 0)
				bitmap.CopyGrayRegion(img, headerImg, 0, 0)
			}
		}
	}

	// Render separator directly (no scrolling, no clipping needed)
	if appearance.Separator.Color >= 0 && appearance.Separator.Thickness > 0 && appearance.Header.Enabled && appearance.Message.Enabled {
		bitmap.DrawFilledRectangle(img, 0, separatorY, w.width, appearance.Separator.Thickness, uint8(appearance.Separator.Color))
	}

	// Render message area to sub-image (provides natural clipping)
	msgHeight := w.height - messageY
	if msgHeight > 0 {
		var messageText string
		if appearance.Message.Enabled {
			// Build message text: media placeholder(s) + caption
			if msg.MediaType != "" {
				messageText = "[" + msg.MediaType + "]"
				// Add caption on new line if present
				if msg.Text != "" {
					messageText += "\n" + msg.Text
				}
			} else {
				messageText = msg.Text
			}
		} else {
			// Show placeholder when message display is disabled
			messageText = "You have a new message"
		}

		// Apply blink effect
		if appearance.Message.Blink && !w.blink.ShouldRender() {
			// Skip rendering when blinking off
		} else {
			// Create sub-image for message region
			msgImg := image.NewGray(image.Rect(0, 0, w.width, msgHeight))
			// Fill with background color
			bgColor := w.GetRenderBackgroundColor()
			for i := range msgImg.Pix {
				msgImg.Pix[i] = bgColor
			}
			// Render message (coordinates relative to sub-image: 0,0)
			w.renderMultiLineText(msgImg, messageText, appearance.Message, w.messageScroller.GetOffset(), 0, 0, w.width, msgHeight)
			// Copy to main image at (0, messageY)
			bitmap.CopyGrayRegion(img, msgImg, 0, messageY)
		}
	}
}

// renderMultiLineText renders wrapped text with optional vertical scrolling
func (w *Widget) renderMultiLineText(img *image.Gray, text string, elem ElementAppearance, scrollOffset float64, x, y, width, height int) {
	renderer := shared.NewMultiLineRenderer(shared.MultiLineRendererConfig{
		FontFace:      elem.FontFace,
		FontName:      elem.FontName,
		HorizAlign:    elem.HorizAlign,
		VertAlign:     elem.VertAlign,
		ScrollMode:    elem.ScrollMode,
		ScrollGap:     elem.ScrollGap,
		ScrollEnabled: elem.ScrollEnabled,
		WordBreak:     elem.WordBreak,
	}, width)

	bounds := image.Rect(x, y, x+width, y+height)
	renderer.Render(img, text, scrollOffset, bounds)
}

// renderScrollingText renders text with scrolling support (single line, horizontal scroll)
func (w *Widget) renderScrollingText(img *image.Gray, text string, elem ElementAppearance, scrollOffset float64, x, y, width, height int) {
	renderer := shared.NewHorizontalTextRenderer(shared.HorizontalTextRendererConfig{
		FontFace:      elem.FontFace,
		FontName:      elem.FontName,
		HorizAlign:    elem.HorizAlign,
		VertAlign:     elem.VertAlign,
		ScrollMode:    elem.ScrollMode,
		ScrollGap:     elem.ScrollGap,
		ScrollEnabled: elem.ScrollEnabled,
	})

	bounds := image.Rect(x, y, x+width, y+height)
	renderer.Render(img, text, scrollOffset, bounds)
}

// formatHeader creates the header string for a message
// Supports format tokens: {sender}, {chat}, {type}, {time}, {date}, {forwarded}
// If format is empty, uses auto format based on chat type
func (w *Widget) formatHeader(msg tgclient.MessageInfo) string {
	appearance := w.getAppearance(msg.ChatType)

	// Get sender name with fallback for private chats
	senderName := msg.SenderName
	if senderName == "" {
		// For private chats, use ChatTitle as sender (it's the other person's name)
		if msg.ChatType == tgclient.ChatTypePrivate {
			senderName = msg.ChatTitle
		}
		// Final fallback to ID
		if senderName == "" {
			senderName = fmt.Sprintf("User %d", msg.ChatID)
		}
	}

	// Get chat title with fallback
	chatTitle := msg.ChatTitle
	if chatTitle == "" {
		chatTitle = fmt.Sprintf("Chat %d", msg.ChatID)
	}

	// Get chat type string
	chatTypeStr := ""
	switch msg.ChatType {
	case tgclient.ChatTypePrivate:
		chatTypeStr = "private"
	case tgclient.ChatTypeGroup:
		chatTypeStr = "group"
	case tgclient.ChatTypeChannel:
		chatTypeStr = "channel"
	}

	// Build forwarded info string
	forwardedStr := ""
	if msg.IsForwarded {
		if msg.ForwardedFrom != "" {
			forwardedStr = "Fwd: " + msg.ForwardedFrom
		} else {
			forwardedStr = "Forwarded"
		}
	}

	// If no custom format, use auto format based on chat type
	format := appearance.Header.Format
	if format == "" {
		header := ""
		switch msg.ChatType {
		case tgclient.ChatTypePrivate:
			header = senderName
		case tgclient.ChatTypeGroup, tgclient.ChatTypeChannel:
			header = chatTitle
		}
		// Append forwarded info for default format
		if msg.IsForwarded && forwardedStr != "" {
			if header != "" {
				header += " | " + forwardedStr
			} else {
				header = forwardedStr
			}
		}
		return header
	}

	// Apply format tokens using TokenFormatter
	formatter := shared.NewTokenFormatter().
		Set("sender", senderName).
		Set("chat", chatTitle).
		Set("type", chatTypeStr).
		Set("time", msg.Time.Format("15:04")).
		Set("date", msg.Time.Format("Jan 2")).
		Set("forwarded", forwardedStr)

	return formatter.Format(format)
}

// Stop cleans up resources
func (w *Widget) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Release client via registry (will disconnect when ref count reaches 0)
	if w.authCfg != nil {
		tgclient.ReleaseClient(w.authCfg)
		w.authCfg = nil
	}
}
