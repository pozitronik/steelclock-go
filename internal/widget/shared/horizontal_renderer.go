package shared

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// HorizontalTextRendererConfig holds configuration for HorizontalTextRenderer
type HorizontalTextRendererConfig struct {
	FontFace      font.Face
	FontName      string
	HorizAlign    config.HAlign
	VertAlign     config.VAlign
	ScrollMode    ScrollMode
	ScrollGap     int
	ScrollEnabled bool
}

// HorizontalTextRenderer handles rendering of single-line text with optional horizontal scrolling
type HorizontalTextRenderer struct {
	fontFace      font.Face
	fontName      string
	horizAlign    config.HAlign
	vertAlign     config.VAlign
	scrollMode    ScrollMode
	scrollGap     int
	scrollEnabled bool
}

// NewHorizontalTextRenderer creates a new HorizontalTextRenderer with the specified configuration
func NewHorizontalTextRenderer(cfg HorizontalTextRendererConfig) *HorizontalTextRenderer {
	return &HorizontalTextRenderer{
		fontFace:      cfg.FontFace,
		fontName:      cfg.FontName,
		horizAlign:    cfg.HorizAlign,
		vertAlign:     cfg.VertAlign,
		scrollMode:    cfg.ScrollMode,
		scrollGap:     cfg.ScrollGap,
		scrollEnabled: cfg.ScrollEnabled,
	}
}

// MeasureTextWidth returns the width of text
func (r *HorizontalTextRenderer) MeasureTextWidth(text string) int {
	width, _ := bitmap.SmartMeasureText(text, r.fontFace, r.fontName)
	return width
}

// Render draws single-line text with optional horizontal scrolling
// scrollOffset is the current scroll position in pixels
func (r *HorizontalTextRenderer) Render(img *image.Gray, text string, scrollOffset float64, bounds image.Rectangle) {
	x, y := bounds.Min.X, bounds.Min.Y
	width, height := bounds.Dx(), bounds.Dy()

	textWidth := r.MeasureTextWidth(text)

	// Calculate base position
	textX, textY := bitmap.SmartCalculateTextPosition(text, r.fontFace, r.fontName, x, y, width, height, r.horizAlign, r.vertAlign)

	// If text fits or scrolling disabled, draw normally
	if textWidth <= width || !r.scrollEnabled {
		bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, textX, textY, x, y, width, height)
		return
	}

	// Handle scrolling - text is wider than container
	totalWidth := textWidth + r.scrollGap

	switch r.scrollMode {
	case ScrollContinuous:
		r.renderContinuousScroll(img, text, textY, scrollOffset, totalWidth, x, y, width, height)

	case ScrollBounce:
		r.renderBounceScroll(img, text, textX, textY, scrollOffset, textWidth, x, y, width, height)

	case ScrollPauseEnds:
		r.renderPauseEndsScroll(img, text, textX, textY, scrollOffset, textWidth, x, y, width, height)

	default:
		// No scrolling, draw at calculated position
		bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, textX, textY, x, y, width, height)
	}
}

// renderContinuousScroll renders with continuous looping scroll
func (r *HorizontalTextRenderer) renderContinuousScroll(img *image.Gray, text string, textY int, scrollOffset float64, totalWidth, x, y, width, height int) {
	offset := int(scrollOffset) % totalWidth
	scrollX := x - offset

	// Draw first copy
	bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, scrollX, textY, x, y, width, height)

	// Draw second copy for seamless loop
	scrollX2 := scrollX + totalWidth
	if scrollX2 < x+width {
		bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, scrollX2, textY, x, y, width, height)
	}
}

// renderBounceScroll renders with bounce-back scroll
func (r *HorizontalTextRenderer) renderBounceScroll(img *image.Gray, text string, textX, textY int, scrollOffset float64, textWidth, x, y, width, height int) {
	maxOffset := float64(textWidth - width)
	if maxOffset <= 0 {
		bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, textX, textY, x, y, width, height)
		return
	}

	offset := scrollOffset
	cycle := int(offset / maxOffset)
	progress := offset - float64(cycle)*maxOffset
	if cycle%2 == 1 {
		progress = maxOffset - progress
	}
	scrollX := x - int(progress)
	bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, scrollX, textY, x, y, width, height)
}

// renderPauseEndsScroll renders with pause at ends scroll
func (r *HorizontalTextRenderer) renderPauseEndsScroll(img *image.Gray, text string, textX, textY int, scrollOffset float64, textWidth, x, y, width, height int) {
	maxOffset := float64(textWidth - width)
	if maxOffset <= 0 {
		bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, textX, textY, x, y, width, height)
		return
	}

	pausePixels := 100
	offset := int(scrollOffset) % (int(maxOffset) + pausePixels)
	if offset > int(maxOffset) {
		offset = int(maxOffset)
	}
	scrollX := x - offset
	bitmap.SmartDrawTextAtPosition(img, text, r.fontFace, r.fontName, scrollX, textY, x, y, width, height)
}
