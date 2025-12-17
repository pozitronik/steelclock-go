// Package clipboard provides a widget that displays clipboard content or content type.
// It supports auto-show mode for notifications when clipboard content changes.
package clipboard

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/anim"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("clipboard", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// ContentType represents the type of content in the clipboard.
type ContentType int

const (
	// TypeEmpty indicates the clipboard is empty.
	TypeEmpty ContentType = iota
	// TypeText indicates plain text content.
	TypeText
	// TypeImage indicates image content.
	TypeImage
	// TypeFiles indicates file/path content.
	TypeFiles
	// TypeHTML indicates HTML content.
	TypeHTML
	// TypeUnknown indicates unrecognized content type.
	TypeUnknown
)

// String returns the human-readable name of the content type.
func (t ContentType) String() string {
	switch t {
	case TypeEmpty:
		return "Empty"
	case TypeText:
		return "Text"
	case TypeImage:
		return "Image"
	case TypeFiles:
		return "Files"
	case TypeHTML:
		return "HTML"
	default:
		return "Unknown"
	}
}

// Reader is the interface for platform-specific clipboard access.
type Reader interface {
	// Read returns current clipboard content and its type.
	// For text: returns the text content.
	// For image: returns metadata like "[Image: WxH]".
	// For files: returns first filename + count like "file.txt (+2 more)".
	Read() (content string, contentType ContentType, err error)

	// HasChanged returns true if clipboard changed since last check.
	HasChanged() bool

	// Close releases any resources held by the reader.
	Close() error
}

// Config holds clipboard-specific configuration.
type Config struct {
	// MaxLength is the maximum number of characters to display (default: 100).
	MaxLength int
	// ShowType shows content type prefix (default: true).
	ShowType bool
	// ScrollLongText enables horizontal scrolling for long text (default: true).
	ScrollLongText bool
	// PollIntervalMs is the clipboard check interval in milliseconds (default: 500).
	PollIntervalMs int
	// TextFormat is the format string with tokens (default: "{content}").
	// Supported tokens: {content}, {type}, {length}, {preview}
	TextFormat string
	// ShowInvisible shows invisible characters as escape sequences (default: false).
	// \n -> \n, \r -> \r, \t -> \t (literal backslash sequences)
	ShowInvisible bool
}

// Widget displays clipboard content with optional auto-show on change.
type Widget struct {
	*widget.BaseWidget
	cfg Config

	// Reader is the platform-specific clipboard reader
	reader Reader

	// Rendering
	textRenderer *render.HorizontalTextRenderer
	scroller     *anim.TextScroller

	// State
	content     string      // Current clipboard content to display
	contentType ContentType // Current content type
	mu          sync.RWMutex

	// Polling
	stopCh   chan struct{}
	pollOnce sync.Once
}

// New creates a new clipboard widget.
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Parse clipboard-specific config
	clipCfg := parseConfig(cfg)

	// Create text renderer
	textSettings := helper.GetTextSettings()

	fontFace, err := bitmap.LoadFont(textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	textRenderer := render.NewHorizontalTextRenderer(render.HorizontalTextRendererConfig{
		FontFace:      fontFace,
		FontName:      textSettings.FontName,
		HorizAlign:    textSettings.HorizAlign,
		VertAlign:     textSettings.VertAlign,
		ScrollEnabled: clipCfg.ScrollLongText,
		ScrollMode:    anim.ScrollPauseEnds,
		ScrollGap:     20,
	})

	// Create platform-specific clipboard reader
	reader, err := newReader()
	if err != nil {
		return nil, fmt.Errorf("failed to create clipboard reader: %w", err)
	}

	w := &Widget{
		BaseWidget:   base,
		cfg:          clipCfg,
		reader:       reader,
		textRenderer: textRenderer,
		content:      "[Empty]",
		contentType:  TypeEmpty,
		stopCh:       make(chan struct{}),
	}

	// Create scroller if enabled
	if clipCfg.ScrollLongText {
		scrollCfg := anim.ScrollerConfig{
			Speed:     30,
			Direction: anim.ScrollLeft,
			Mode:      anim.ScrollPauseEnds,
			PauseMs:   1000,
			Gap:       20,
		}
		// Override with user config if provided
		if cfg.Scroll != nil {
			if cfg.Scroll.Speed > 0 {
				scrollCfg.Speed = cfg.Scroll.Speed
			}
			if cfg.Scroll.PauseMs > 0 {
				scrollCfg.PauseMs = cfg.Scroll.PauseMs
			}
			if cfg.Scroll.Gap > 0 {
				scrollCfg.Gap = cfg.Scroll.Gap
			}
		}
		w.scroller = anim.NewTextScroller(scrollCfg)
	}

	// Do initial clipboard read
	w.readClipboard()

	// Start polling goroutine
	go w.pollClipboard()

	return w, nil
}

// readClipboard reads current clipboard content and updates widget state.
func (w *Widget) readClipboard() {
	content, contentType, err := w.reader.Read()
	if err != nil {
		return
	}

	w.mu.Lock()
	w.content = w.formatContent(content, contentType)
	w.contentType = contentType
	w.mu.Unlock()
}

// parseConfig extracts clipboard-specific configuration.
func parseConfig(cfg config.WidgetConfig) Config {
	clipCfg := Config{
		MaxLength:      100,
		ShowType:       true,
		ScrollLongText: true,
		PollIntervalMs: 500,
		TextFormat:     "{content}",
		ShowInvisible:  false,
	}

	// Extract from cfg.Clipboard if provided
	if cfg.Clipboard != nil {
		if cfg.Clipboard.MaxLength > 0 {
			clipCfg.MaxLength = cfg.Clipboard.MaxLength
		}
		if cfg.Clipboard.ShowType != nil {
			clipCfg.ShowType = *cfg.Clipboard.ShowType
		}
		if cfg.Clipboard.ScrollLongText != nil {
			clipCfg.ScrollLongText = *cfg.Clipboard.ScrollLongText
		}
		if cfg.Clipboard.PollIntervalMs > 0 {
			clipCfg.PollIntervalMs = cfg.Clipboard.PollIntervalMs
		}
		clipCfg.ShowInvisible = cfg.Clipboard.ShowInvisible
	}

	// Use text.format if provided
	if cfg.Text != nil && cfg.Text.Format != "" {
		clipCfg.TextFormat = cfg.Text.Format
	}

	return clipCfg
}

// pollClipboard periodically checks the clipboard for changes.
func (w *Widget) pollClipboard() {
	ticker := time.NewTicker(time.Duration(w.cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.reader.HasChanged() {
				content, contentType, err := w.reader.Read()
				if err != nil {
					continue
				}

				w.mu.Lock()
				w.content = w.formatContent(content, contentType)
				w.contentType = contentType
				w.mu.Unlock()

				// Trigger auto-show if enabled
				w.TriggerAutoHide()

				// Reset scroller for new content
				if w.scroller != nil {
					w.scroller.Reset()
				}
			}
		}
	}
}

// formatContent formats the clipboard content according to settings.
func (w *Widget) formatContent(content string, contentType ContentType) string {
	// Handle empty
	if contentType == TypeEmpty {
		return "[Empty]"
	}

	// Store original length before any modifications
	originalLen := len(content)

	// Truncate if needed (before character replacement to count original chars)
	if len(content) > w.cfg.MaxLength {
		content = content[:w.cfg.MaxLength-3] + "..."
	}

	// Handle whitespace characters
	if w.cfg.ShowInvisible {
		// Show invisible characters as escape sequences (ASCII-compatible)
		content = strings.ReplaceAll(content, "\r\n", "\\n") // Windows line ending first
		content = strings.ReplaceAll(content, "\n", "\\n")   // Unix line ending
		content = strings.ReplaceAll(content, "\r", "\\r")   // Old Mac line ending
		content = strings.ReplaceAll(content, "\t", "\\t")   // Tab
		// Note: regular spaces are kept as-is for readability
	} else {
		// Clean up whitespace for display
		content = strings.ReplaceAll(content, "\r\n", " ")
		content = strings.ReplaceAll(content, "\n", " ")
		content = strings.ReplaceAll(content, "\r", "")
		content = strings.ReplaceAll(content, "\t", " ")
	}

	// Apply format template
	result := w.cfg.TextFormat
	result = strings.ReplaceAll(result, "{content}", content)
	result = strings.ReplaceAll(result, "{type}", contentType.String())
	result = strings.ReplaceAll(result, "{length}", fmt.Sprintf("%d", originalLen))

	// Preview is first 20 chars (from processed content)
	preview := content
	if len(preview) > 20 {
		preview = preview[:20] + "..."
	}
	result = strings.ReplaceAll(result, "{preview}", preview)

	return result
}

// Update fetches the current clipboard state.
func (w *Widget) Update() error {
	// Polling is done in background goroutine, nothing to do here
	return nil
}

// Render creates an image of the clipboard content.
func (w *Widget) Render() (image.Image, error) {
	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	// Create canvas
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Get current content
	w.mu.RLock()
	content := w.content
	w.mu.RUnlock()

	// Get content area (accounts for padding and border)
	contentArea := w.GetContentArea()
	bounds := image.Rect(
		contentArea.X,
		contentArea.Y,
		contentArea.X+contentArea.Width,
		contentArea.Y+contentArea.Height,
	)

	// Apply scrolling if enabled and text is too wide
	var scrollOffset float64
	if w.scroller != nil {
		textWidth := w.textRenderer.MeasureTextWidth(content)
		scrollOffset = w.scroller.Update(textWidth, contentArea.Width)
	}

	w.textRenderer.Render(img, content, scrollOffset, bounds)

	return img, nil
}

// Stop stops the clipboard polling goroutine.
func (w *Widget) Stop() {
	w.pollOnce.Do(func() {
		close(w.stopCh)
		if w.reader != nil {
			_ = w.reader.Close()
		}
	})
}
