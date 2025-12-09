package shared

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// MultiLineRendererConfig holds configuration for MultiLineRenderer
type MultiLineRendererConfig struct {
	FontFace      font.Face
	FontName      string
	HorizAlign    config.HAlign
	VertAlign     config.VAlign
	ScrollMode    ScrollMode // ScrollContinuous, ScrollBounce, ScrollPauseEnds
	ScrollGap     int
	ScrollEnabled bool
	WordBreak     string // "normal" or "break-all"
}

// MultiLineRenderer handles rendering of wrapped text with optional vertical scrolling
type MultiLineRenderer struct {
	wrapper       *TextWrapper
	fontFace      font.Face
	fontName      string
	horizAlign    config.HAlign
	vertAlign     config.VAlign
	scrollMode    ScrollMode
	scrollGap     int
	scrollEnabled bool
}

// NewMultiLineRenderer creates a new MultiLineRenderer with the specified configuration
func NewMultiLineRenderer(cfg MultiLineRendererConfig, maxWidth int) *MultiLineRenderer {
	mode := WrapModeNormal
	if cfg.WordBreak == "break-all" {
		mode = WrapModeBreakAll
	}

	return &MultiLineRenderer{
		wrapper:       NewTextWrapper(cfg.FontFace, cfg.FontName, maxWidth, mode),
		fontFace:      cfg.FontFace,
		fontName:      cfg.FontName,
		horizAlign:    cfg.HorizAlign,
		vertAlign:     cfg.VertAlign,
		scrollMode:    cfg.ScrollMode,
		scrollGap:     cfg.ScrollGap,
		scrollEnabled: cfg.ScrollEnabled,
	}
}

// SetMaxWidth updates the maximum width for text wrapping
func (r *MultiLineRenderer) SetMaxWidth(maxWidth int) {
	r.wrapper.SetMaxWidth(maxWidth)
}

// MeasureLineHeight returns the height of a single line
func (r *MultiLineRenderer) MeasureLineHeight() int {
	return r.wrapper.MeasureLineHeight()
}

// MeasureTotalHeight measures the total height of wrapped text
func (r *MultiLineRenderer) MeasureTotalHeight(text string, maxWidth int) int {
	r.wrapper.SetMaxWidth(maxWidth)
	lines := r.wrapper.Wrap(text)
	lineHeight := r.MeasureLineHeight()
	return len(lines) * lineHeight
}

// WrapText wraps text into lines that fit within the configured width
func (r *MultiLineRenderer) WrapText(text string) []string {
	return r.wrapper.Wrap(text)
}

// Render draws wrapped text with optional vertical scrolling
// scrollOffset is the current scroll position in pixels
func (r *MultiLineRenderer) Render(img *image.Gray, text string, scrollOffset float64, bounds image.Rectangle) {
	x, y := bounds.Min.X, bounds.Min.Y
	width, height := bounds.Dx(), bounds.Dy()

	lines := r.wrapper.Wrap(text)
	if len(lines) == 0 {
		return
	}

	lineHeight := r.MeasureLineHeight()
	totalTextHeight := len(lines) * lineHeight

	// Check if scrolling is needed
	if totalTextHeight <= height || !r.scrollEnabled {
		// No scrolling - truncate with ellipsis if needed
		if totalTextHeight > height {
			lines = r.wrapper.TruncateWithEllipsis(lines, height)
		}

		// Render lines
		currentY := y
		for _, line := range lines {
			if currentY+lineHeight > y+height {
				break
			}
			r.renderSingleLine(img, line, x, currentY, width, lineHeight)
			currentY += lineHeight
		}
		return
	}

	// Scrolling enabled
	totalScrollHeight := totalTextHeight + r.scrollGap

	switch r.scrollMode {
	case ScrollContinuous:
		r.renderContinuousScroll(img, lines, lineHeight, scrollOffset, totalScrollHeight, x, y, width, height)

	case ScrollBounce:
		r.renderBounceScroll(img, lines, lineHeight, scrollOffset, totalTextHeight, x, y, width, height)

	case ScrollPauseEnds:
		r.renderPauseEndsScroll(img, lines, lineHeight, scrollOffset, totalTextHeight, x, y, width, height)

	default:
		// No scroll mode - just render what fits
		currentY := y
		for _, line := range lines {
			if currentY+lineHeight > y+height {
				break
			}
			r.renderSingleLine(img, line, x, currentY, width, lineHeight)
			currentY += lineHeight
		}
	}
}

// renderContinuousScroll renders with continuous looping scroll
func (r *MultiLineRenderer) renderContinuousScroll(img *image.Gray, lines []string, lineHeight int, scrollOffset float64, totalScrollHeight, x, y, width, height int) {
	offset := int(scrollOffset) % totalScrollHeight
	startY := y - offset

	// Draw lines with wrapping (twice for seamless loop)
	for i := 0; i < 2; i++ {
		lineY := startY + i*totalScrollHeight
		for _, line := range lines {
			if lineY+lineHeight > y && lineY < y+height {
				r.renderSingleLine(img, line, x, lineY, width, lineHeight)
			}
			lineY += lineHeight
		}
	}
}

// renderBounceScroll renders with bounce-back scroll
func (r *MultiLineRenderer) renderBounceScroll(img *image.Gray, lines []string, lineHeight int, scrollOffset float64, totalTextHeight, x, y, width, height int) {
	maxOffset := float64(totalTextHeight - height)
	if maxOffset <= 0 {
		// Fits without scrolling
		currentY := y
		for _, line := range lines {
			r.renderSingleLine(img, line, x, currentY, width, lineHeight)
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
			r.renderSingleLine(img, line, x, startY, width, lineHeight)
		}
		startY += lineHeight
	}
}

// renderPauseEndsScroll renders with pause at ends scroll
func (r *MultiLineRenderer) renderPauseEndsScroll(img *image.Gray, lines []string, lineHeight int, scrollOffset float64, totalTextHeight, x, y, width, height int) {
	maxOffset := float64(totalTextHeight - height)
	if maxOffset <= 0 {
		currentY := y
		for _, line := range lines {
			r.renderSingleLine(img, line, x, currentY, width, lineHeight)
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
			r.renderSingleLine(img, line, x, startY, width, lineHeight)
		}
		startY += lineHeight
	}
}

// renderSingleLine renders a single line of text with alignment
func (r *MultiLineRenderer) renderSingleLine(img *image.Gray, text string, x, y, width, height int) {
	textX, textY := bitmap.SmartCalculateTextPosition(text, r.fontFace, r.fontName, x, y, width, height, r.horizAlign, r.vertAlign)
	bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, textX, textY, x, y, width, height)
}

// RenderLines renders pre-wrapped lines without scrolling
func (r *MultiLineRenderer) RenderLines(img *image.Gray, lines []string, bounds image.Rectangle) {
	if len(lines) == 0 {
		return
	}

	x, y := bounds.Min.X, bounds.Min.Y
	width, height := bounds.Dx(), bounds.Dy()
	lineHeight := r.MeasureLineHeight()

	currentY := y
	for _, line := range lines {
		if currentY+lineHeight > y+height {
			break
		}
		r.renderSingleLine(img, line, x, currentY, width, lineHeight)
		currentY += lineHeight
	}
}
