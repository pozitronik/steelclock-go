package widget

import (
	"fmt"
	"image"
	"image/color"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// KeyboardWidget displays lock key status
type KeyboardWidget struct {
	*BaseWidget
	fontSize      int
	horizAlign    string
	vertAlign     string
	padding       int
	spacing       int
	separator     string
	capsUseIcon   bool // true if caps lock uses icon, false if text
	numUseIcon    bool // true if num lock uses icon, false if text
	scrollUseIcon bool // true if scroll lock uses icon, false if text
	capsLockOn    string
	capsLockOff   string
	numLockOn     string
	numLockOff    string
	scrollLockOn  string
	scrollLockOff string
	colorOn       uint8
	colorOff      uint8
	capsState     bool
	numState      bool
	scrollState   bool
	fontFace      font.Face
}

// NewKeyboardWidget creates a new keyboard widget
func NewKeyboardWidget(cfg config.WidgetConfig) (*KeyboardWidget, error) {
	base := NewBaseWidget(cfg)

	// Extract text settings
	fontSize := 10
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
		fontName = cfg.Text.Font
		if cfg.Text.Align != nil {
			if cfg.Text.Align.H != "" {
				horizAlign = cfg.Text.Align.H
			}
			if cfg.Text.Align.V != "" {
				vertAlign = cfg.Text.Align.V
			}
		}
	}

	// Extract padding from style
	if cfg.Style != nil {
		padding = cfg.Style.Padding
	}

	// Color handling - only apply defaults when not explicitly set (nil)
	// Allow 0 as valid value (black/invisible)
	colorOn := 255
	colorOff := 100
	if cfg.Colors != nil {
		if cfg.Colors.On != nil {
			colorOn = *cfg.Colors.On
		}
		if cfg.Colors.Off != nil {
			colorOff = *cfg.Colors.Off
		}
	}

	// Layout settings - separator and spacing
	separator := ""
	spacing := 2
	if cfg.Layout != nil {
		separator = cfg.Layout.Separator
		if cfg.Layout.Spacing > 0 {
			spacing = cfg.Layout.Spacing
		}
	}

	// Lock indicator symbols from indicators config
	// Defaults
	capsOn := "C"
	capsOff := "c"
	numOn := "N"
	numOff := "n"
	scrollOn := "S"
	scrollOff := "s"

	// Per-indicator mode detection: if BOTH on and off are nil, use icon mode
	// Otherwise, use text mode (even if only one is defined)
	capsUseIcon := true
	numUseIcon := true
	scrollUseIcon := true

	if cfg.Indicators != nil {
		if cfg.Indicators.Caps != nil {
			capsUseIcon = false // Text mode if config is present
			if cfg.Indicators.Caps.On != "" {
				capsOn = cfg.Indicators.Caps.On
			}
			if cfg.Indicators.Caps.Off != "" {
				capsOff = cfg.Indicators.Caps.Off
			}
		}
		if cfg.Indicators.Num != nil {
			numUseIcon = false
			if cfg.Indicators.Num.On != "" {
				numOn = cfg.Indicators.Num.On
			}
			if cfg.Indicators.Num.Off != "" {
				numOff = cfg.Indicators.Num.Off
			}
		}
		if cfg.Indicators.Scroll != nil {
			scrollUseIcon = false
			if cfg.Indicators.Scroll.On != "" {
				scrollOn = cfg.Indicators.Scroll.On
			}
			if cfg.Indicators.Scroll.Off != "" {
				scrollOff = cfg.Indicators.Scroll.Off
			}
		}
	}

	// Load font (needed when any indicator uses text mode)
	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &KeyboardWidget{
		BaseWidget:    base,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       padding,
		spacing:       spacing,
		separator:     separator,
		capsUseIcon:   capsUseIcon,
		numUseIcon:    numUseIcon,
		scrollUseIcon: scrollUseIcon,
		capsLockOn:    capsOn,
		capsLockOff:   capsOff,
		numLockOn:     numOn,
		numLockOff:    numOff,
		scrollLockOn:  scrollOn,
		scrollLockOff: scrollOff,
		colorOn:       uint8(colorOn),
		colorOff:      uint8(colorOff),
		fontFace:      fontFace,
	}, nil
}

// Render creates an image of the keyboard widget
func (w *KeyboardWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	// Determine rendering mode based on indicator configurations
	allIcons := w.capsUseIcon && w.numUseIcon && w.scrollUseIcon
	allText := !w.capsUseIcon && !w.numUseIcon && !w.scrollUseIcon

	if allIcons {
		w.renderIcons(img)
	} else if allText {
		w.renderText(img)
	} else {
		w.renderMixed(img)
	}

	return img, nil
}

// renderText renders keyboard indicators as text
func (w *KeyboardWidget) renderText(img *image.Gray) {
	// Build indicator text
	indicators := []struct {
		state bool
		on    string
		off   string
	}{
		{w.capsState, w.capsLockOn, w.capsLockOff},
		{w.numState, w.numLockOn, w.numLockOff},
		{w.scrollState, w.scrollLockOn, w.scrollLockOff},
	}

	text := ""
	for i, ind := range indicators {
		if i > 0 {
			text += w.separator
		}
		if ind.state {
			text += ind.on
		} else {
			text += ind.off
		}
	}

	// Draw text
	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
}

// renderIcons renders keyboard indicators as icons
func (w *KeyboardWidget) renderIcons(img *image.Gray) {
	// Get image bounds
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Define indicators with their icons
	indicators := []struct {
		state    bool
		iconType string
	}{
		{w.capsState, "arrow_up"},     // Caps Lock - up arrow (uppercase)
		{w.numState, "lock"},          // Num Lock - lock icon
		{w.scrollState, "arrow_down"}, // Scroll Lock - down arrow
	}
	iconCount := len(indicators)

	// Auto-calculate icon size based on available space
	// Try sizes in descending order: 16, 12, 8
	availableWidth := imgWidth - (w.padding * 2)
	availableHeight := imgHeight - (w.padding * 2)

	var iconSet *glyphs.GlyphSet
	var iconSize int
	var actualSpacing int

	// Try each size and pick the largest that fits
	// Consider spacing only if it fits; otherwise icons can touch
	for _, size := range []int{16, 12, 8} {
		// Check if icons fit in height
		if size > availableHeight {
			continue
		}

		// Calculate required width for icons alone (without spacing)
		iconsWidth := iconCount * size

		// Check if icons fit in width
		if iconsWidth <= availableWidth {
			iconSize = size

			// Calculate actual spacing that fits (may be less than configured)
			availableForSpacing := availableWidth - iconsWidth
			spacingSlots := iconCount - 1
			if spacingSlots > 0 {
				actualSpacing = availableForSpacing / spacingSlots
				// Don't exceed configured spacing
				if actualSpacing > w.spacing {
					actualSpacing = w.spacing
				}
			}
			break
		}
	}

	// Fallback to the smallest size if nothing fits
	if iconSize == 0 {
		iconSize = 8
		actualSpacing = 0
	}

	// Select icon set based on calculated size
	switch iconSize {
	case 16:
		iconSet = glyphs.KeyboardIcons16x16
	case 12:
		iconSet = glyphs.KeyboardIcons12x12
	default:
		iconSet = glyphs.KeyboardIcons8x8
	}

	// Calculate total width needed (using actual spacing that fits)
	totalWidth := iconCount*iconSize + (iconCount-1)*actualSpacing

	// Calculate horizontal position based on alignment
	var startX int
	switch w.horizAlign {
	case "left":
		startX = w.padding
	case "right":
		startX = imgWidth - w.padding - totalWidth
	default: // "center"
		startX = (imgWidth - totalWidth) / 2
	}

	// Ensure startX is within bounds
	if startX < w.padding {
		startX = w.padding
	}
	if startX+totalWidth > imgWidth-w.padding {
		startX = imgWidth - w.padding - totalWidth
	}

	// Calculate vertical position based on alignment
	var baseY int
	switch w.vertAlign {
	case "top":
		baseY = w.padding
	case "bottom":
		baseY = imgHeight - w.padding - iconSize
	default: // "center"
		baseY = (imgHeight - iconSize) / 2
	}

	// Draw each indicator icon
	currentX := startX
	for _, ind := range indicators {
		var iconName string
		var c color.Gray

		// Select icon based on state
		if ind.state {
			c = color.Gray{Y: w.colorOn}
			// For lock type, use closed lock when ON
			if ind.iconType == "lock" {
				iconName = "lock_closed"
			} else {
				iconName = ind.iconType
			}
		} else {
			c = color.Gray{Y: w.colorOff}
			// For lock type, use open lock when OFF
			if ind.iconType == "lock" {
				iconName = "lock_open"
			} else {
				// For arrows, still show but dimmed
				iconName = ind.iconType
			}
		}

		icon := glyphs.GetIcon(iconSet, iconName)
		if icon != nil {
			glyphs.DrawGlyph(img, icon, currentX, baseY, c)
		}

		currentX += iconSize + actualSpacing
	}
}

// renderMixed renders keyboard indicators in mixed mode (some text, some icons)
func (w *KeyboardWidget) renderMixed(img *image.Gray) {
	// Get image bounds
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	availableWidth := imgWidth - (w.padding * 2)
	availableHeight := imgHeight - (w.padding * 2)

	// Define indicators with their mode and content
	type indicator struct {
		state    bool
		useIcon  bool
		iconType string
		textOn   string
		textOff  string
	}

	indicators := []indicator{
		{w.capsState, w.capsUseIcon, "arrow_up", w.capsLockOn, w.capsLockOff},
		{w.numState, w.numUseIcon, "lock", w.numLockOn, w.numLockOff},
		{w.scrollState, w.scrollUseIcon, "arrow_down", w.scrollLockOn, w.scrollLockOff},
	}

	// Determine icon size for icon-mode indicators (same logic as renderIcons)
	var iconSize int
	hasAnyIcon := w.capsUseIcon || w.numUseIcon || w.scrollUseIcon
	if hasAnyIcon {
		iconCount := 0
		if w.capsUseIcon {
			iconCount++
		}
		if w.numUseIcon {
			iconCount++
		}
		if w.scrollUseIcon {
			iconCount++
		}

		// Try each size and pick the largest that fits
		// Consider spacing only if it fits; otherwise icons can touch
		for _, size := range []int{16, 12, 8} {
			// Check if icons fit in height
			if size > availableHeight {
				continue
			}

			// Calculate required width for icons alone (without spacing)
			iconsWidth := iconCount * size

			// Check if icons fit in available width (rough estimate for mixed mode)
			// Note: In mixed mode, we also have text, so this is approximate
			if iconsWidth <= availableWidth {
				iconSize = size
				break
			}
		}

		// Fallback to the smallest size if nothing fits
		if iconSize == 0 {
			iconSize = 8
		}
	}

	// Select icon set based on calculated size
	var iconSet *glyphs.GlyphSet
	if hasAnyIcon {
		switch iconSize {
		case 16:
			iconSet = glyphs.KeyboardIcons16x16
		case 12:
			iconSet = glyphs.KeyboardIcons12x12
		default:
			iconSet = glyphs.KeyboardIcons8x8
		}
	}

	// Calculate width for each indicator
	type indicatorLayout struct {
		ind   indicator
		width int
	}
	var layouts []indicatorLayout
	totalWidth := 0

	for i, ind := range indicators {
		var width int
		if ind.useIcon {
			width = iconSize
		} else {
			// Measure text width
			text := ind.textOff
			if ind.state {
				text = ind.textOn
			}
			drawer := &font.Drawer{Face: w.fontFace}
			advance := drawer.MeasureString(text)
			width = advance.Ceil()
		}

		layouts = append(layouts, indicatorLayout{ind: ind, width: width})
		totalWidth += width

		// Add spacing between indicators (except after last one)
		if i < len(indicators)-1 {
			totalWidth += w.spacing
		}
	}

	// Calculate horizontal start position based on alignment
	var startX int
	switch w.horizAlign {
	case "left":
		startX = w.padding
	case "right":
		startX = imgWidth - w.padding - totalWidth
	default: // "center"
		startX = (imgWidth - totalWidth) / 2
	}

	// Ensure startX is within bounds
	if startX < w.padding {
		startX = w.padding
	}
	if startX+totalWidth > imgWidth-w.padding {
		startX = imgWidth - w.padding - totalWidth
	}

	// Render each indicator
	currentX := startX
	for _, layout := range layouts {
		ind := layout.ind

		if ind.useIcon {
			// Render as icon
			var iconName string
			var c color.Gray

			if ind.state {
				c = color.Gray{Y: w.colorOn}
				if ind.iconType == "lock" {
					iconName = "lock_closed"
				} else {
					iconName = ind.iconType
				}
			} else {
				c = color.Gray{Y: w.colorOff}
				if ind.iconType == "lock" {
					iconName = "lock_open"
				} else {
					iconName = ind.iconType
				}
			}

			// Calculate vertical position for icon
			var baseY int
			switch w.vertAlign {
			case "top":
				baseY = w.padding
			case "bottom":
				baseY = imgHeight - w.padding - iconSize
			default: // "center"
				baseY = (imgHeight - iconSize) / 2
			}

			icon := glyphs.GetIcon(iconSet, iconName)
			if icon != nil {
				glyphs.DrawGlyph(img, icon, currentX, baseY, c)
			}

			currentX += iconSize
		} else {
			// Render as text
			text := ind.textOff
			if ind.state {
				text = ind.textOn
			}

			// Draw text at current position with alignment
			// Note: DrawAlignedText is for the whole image, so we need to draw manually
			drawer := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(color.Gray{Y: 255}),
				Face: w.fontFace,
			}

			// Calculate vertical position for text
			metrics := w.fontFace.Metrics()
			textHeight := (metrics.Ascent + metrics.Descent).Ceil()
			var baseY int
			switch w.vertAlign {
			case "top":
				baseY = w.padding + metrics.Ascent.Ceil()
			case "bottom":
				baseY = imgHeight - w.padding - textHeight + metrics.Ascent.Ceil()
			default: // "center"
				baseY = (imgHeight-textHeight)/2 + metrics.Ascent.Ceil()
			}

			drawer.Dot = fixed.Point26_6{
				X: fixed.I(currentX),
				Y: fixed.I(baseY),
			}
			drawer.DrawString(text)

			currentX += layout.width
		}

		// Add spacing for next indicator
		currentX += w.spacing
	}
}
