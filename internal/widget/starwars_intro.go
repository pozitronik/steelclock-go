package widget

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// Animation phases
const (
	PhasePreIntroFadeIn = iota
	PhasePreIntroHold
	PhasePreIntroFadeOut
	PhaseLogoHold
	PhaseLogoShrink
	PhaseCrawl
	PhasePauseEnd
)

// BackgroundStar represents a static background star
type BackgroundStar struct {
	x, y       int
	brightness uint8
}

// StarWarsIntroWidget displays the iconic Star Wars opening crawl effect
// with pre-intro text, shrinking logo, and perspective text crawl
type StarWarsIntroWidget struct {
	*BaseWidget
	mu sync.Mutex

	// Pre-intro configuration
	preIntroEnabled bool
	preIntroText    string
	preIntroLines   []string // Split on \n for multi-line support
	preIntroColor   int
	preIntroFadeIn  float64
	preIntroHold    float64
	preIntroFadeOut float64

	// Logo configuration
	logoEnabled        bool
	logoText           string
	logoLines          []string // Split logo text
	logoColor          int
	logoHoldBefore     float64
	logoShrinkDuration float64
	logoFinalScale     float64

	// Stars configuration
	starsEnabled    bool
	starsCount      int
	starsBrightness int
	stars           []BackgroundStar

	// Crawl configuration
	lines       []string // Text lines to display
	scrollSpeed float64  // Pixels per frame
	perspective float64  // Perspective strength (0.0 = none, 1.0 = strong)
	slant       float64  // Text slant angle in degrees
	fadeTop     float64  // Fade start position (0.0 = top, 1.0 = bottom)
	textColor   uint8    // Text brightness
	lineSpacing int      // Pixels between lines
	charWidth   int      // Character width for built-in font

	// General settings
	loop       bool    // Loop when sequence completes
	pauseAtEnd float64 // Seconds to pause at end before looping

	// State
	phase        int       // Current animation phase
	phaseStart   time.Time // When current phase started
	scrollOffset float64   // Current scroll position (increases as text scrolls up)
	totalHeight  float64   // Total height of all text content
	logoScale    float64   // Current logo scale (1.0 = full, 0.0 = gone)

	// Font
	glyphSet *glyphs.GlyphSet // Internal font for rendering

	// Display dimensions
	width  int
	height int
}

// NewStarWarsIntroWidget creates a new Star Wars intro widget
func NewStarWarsIntroWidget(cfg config.WidgetConfig) (*StarWarsIntroWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	// Default crawl text
	defaultLines := []string{
		"Episode IV",
		"A NEW HOPE",
		"",
		"It is a period of civil war.",
		"Rebel spaceships, striking",
		"from a hidden base, have won",
		"their first victory against",
		"the evil Galactic Empire.",
	}

	// Default configuration
	lines := defaultLines
	scrollSpeed := 0.5
	perspective := 0.7
	slant := 60.0 // Degrees - high value needed for visible radial slant on small fonts
	fadeTop := 0.3
	textColor := uint8(255)
	lineSpacing := 8
	charWidth := 6
	loop := true
	pauseAtEnd := 3.0

	// Pre-intro defaults
	preIntroEnabled := true
	preIntroText := "A long time ago in a galaxy far, far away...."
	preIntroColor := 80
	preIntroFadeIn := 2.0
	preIntroHold := 2.0
	preIntroFadeOut := 1.0

	// Logo defaults
	logoEnabled := true
	logoText := "STAR\nWARS"
	logoColor := 255
	logoHoldBefore := 0.5
	logoShrinkDuration := 4.0
	logoFinalScale := 0.1

	// Stars defaults
	starsEnabled := true
	starsCount := 50
	starsBrightness := 200

	if cfg.StarWarsIntro != nil {
		// Pre-intro settings
		if cfg.StarWarsIntro.PreIntro != nil {
			pi := cfg.StarWarsIntro.PreIntro
			if pi.Enabled != nil {
				preIntroEnabled = *pi.Enabled
			}
			if pi.Text != "" {
				preIntroText = pi.Text
			}
			if pi.Color > 0 {
				preIntroColor = pi.Color
			}
			if pi.FadeIn > 0 {
				preIntroFadeIn = pi.FadeIn
			}
			if pi.Hold > 0 {
				preIntroHold = pi.Hold
			}
			if pi.FadeOut > 0 {
				preIntroFadeOut = pi.FadeOut
			}
		}

		// Logo settings
		if cfg.StarWarsIntro.Logo != nil {
			l := cfg.StarWarsIntro.Logo
			if l.Enabled != nil {
				logoEnabled = *l.Enabled
			}
			if l.Text != "" {
				logoText = l.Text
			}
			if l.Color > 0 {
				logoColor = l.Color
			}
			if l.HoldBefore > 0 {
				logoHoldBefore = l.HoldBefore
			}
			if l.ShrinkDuration > 0 {
				logoShrinkDuration = l.ShrinkDuration
			}
			if l.FinalScale > 0 {
				logoFinalScale = l.FinalScale
			}
		}

		// Stars settings
		if cfg.StarWarsIntro.Stars != nil {
			s := cfg.StarWarsIntro.Stars
			if s.Enabled != nil {
				starsEnabled = *s.Enabled
			}
			if s.Count > 0 {
				starsCount = s.Count
			}
			if s.Brightness > 0 {
				starsBrightness = s.Brightness
			}
		}

		// Crawl settings
		if len(cfg.StarWarsIntro.Text) > 0 {
			lines = cfg.StarWarsIntro.Text
		}
		if cfg.StarWarsIntro.ScrollSpeed > 0 {
			scrollSpeed = cfg.StarWarsIntro.ScrollSpeed
		}
		if cfg.StarWarsIntro.Perspective > 0 {
			perspective = cfg.StarWarsIntro.Perspective
		}
		if cfg.StarWarsIntro.Slant != 0 {
			slant = cfg.StarWarsIntro.Slant
		}
		if cfg.StarWarsIntro.FadeTop > 0 {
			fadeTop = cfg.StarWarsIntro.FadeTop
		}
		if cfg.StarWarsIntro.TextColor > 0 {
			textColor = uint8(cfg.StarWarsIntro.TextColor)
		}
		if cfg.StarWarsIntro.LineSpacing > 0 {
			lineSpacing = cfg.StarWarsIntro.LineSpacing
		}
		if cfg.StarWarsIntro.Loop != nil {
			loop = *cfg.StarWarsIntro.Loop
		}
		if cfg.StarWarsIntro.PauseAtEnd > 0 {
			pauseAtEnd = cfg.StarWarsIntro.PauseAtEnd
		}
	}

	// Calculate total content height for crawl
	totalHeight := float64(len(lines) * lineSpacing)

	// Get internal font (use 5x7 for better readability)
	glyphSet := bitmap.GetInternalFontByName("5x7")
	if glyphSet == nil {
		glyphSet = glyphs.Font5x7 // Fallback
	}

	// Split logo text into lines
	logoLines := strings.Split(logoText, "\n")

	// Split pre-intro text into lines
	preIntroLines := strings.Split(preIntroText, "\n")

	// Initialize background stars
	stars := make([]BackgroundStar, starsCount)
	for i := range stars {
		stars[i] = BackgroundStar{
			x:          rand.Intn(pos.W),
			y:          rand.Intn(pos.H),
			brightness: uint8(rand.Intn(starsBrightness/2) + starsBrightness/2),
		}
	}

	// Determine starting phase
	startPhase := PhasePreIntroFadeIn
	if !preIntroEnabled {
		if logoEnabled {
			startPhase = PhaseLogoHold
		} else {
			startPhase = PhaseCrawl
		}
	}

	w := &StarWarsIntroWidget{
		BaseWidget: base,

		preIntroEnabled: preIntroEnabled,
		preIntroText:    preIntroText,
		preIntroLines:   preIntroLines,
		preIntroColor:   preIntroColor,
		preIntroFadeIn:  preIntroFadeIn,
		preIntroHold:    preIntroHold,
		preIntroFadeOut: preIntroFadeOut,

		logoEnabled:        logoEnabled,
		logoText:           logoText,
		logoLines:          logoLines,
		logoColor:          logoColor,
		logoHoldBefore:     logoHoldBefore,
		logoShrinkDuration: logoShrinkDuration,
		logoFinalScale:     logoFinalScale,
		logoScale:          1.0,

		starsEnabled:    starsEnabled,
		starsCount:      starsCount,
		starsBrightness: starsBrightness,
		stars:           stars,

		lines:        lines,
		scrollSpeed:  scrollSpeed,
		perspective:  perspective,
		slant:        slant,
		fadeTop:      fadeTop,
		textColor:    textColor,
		lineSpacing:  lineSpacing,
		charWidth:    charWidth,
		scrollOffset: 0,
		totalHeight:  totalHeight,

		loop:       loop,
		pauseAtEnd: pauseAtEnd,

		phase:      startPhase,
		phaseStart: time.Now(),

		glyphSet: glyphSet,
		width:    pos.W,
		height:   pos.H,
	}

	return w, nil
}

// Update advances the animation
func (w *StarWarsIntroWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(w.phaseStart).Seconds()

	switch w.phase {
	case PhasePreIntroFadeIn:
		if elapsed >= w.preIntroFadeIn {
			w.phase = PhasePreIntroHold
			w.phaseStart = now
		}

	case PhasePreIntroHold:
		if elapsed >= w.preIntroHold {
			w.phase = PhasePreIntroFadeOut
			w.phaseStart = now
		}

	case PhasePreIntroFadeOut:
		if elapsed >= w.preIntroFadeOut {
			if w.logoEnabled {
				w.phase = PhaseLogoHold
				w.logoScale = 1.0
			} else {
				w.phase = PhaseCrawl
				w.scrollOffset = 0
			}
			w.phaseStart = now
		}

	case PhaseLogoHold:
		if elapsed >= w.logoHoldBefore {
			w.phase = PhaseLogoShrink
			w.phaseStart = now
		}

	case PhaseLogoShrink:
		progress := elapsed / w.logoShrinkDuration
		if progress >= 1.0 {
			w.phase = PhaseCrawl
			w.scrollOffset = 0
			w.phaseStart = now
		} else {
			// Ease-out for smooth deceleration toward vanishing point
			eased := 1.0 - math.Pow(1.0-progress, 2)
			w.logoScale = 1.0 - eased*(1.0-w.logoFinalScale)
		}

	case PhaseCrawl:
		w.scrollOffset += w.scrollSpeed
		// Check if all text has scrolled off
		if w.scrollOffset > w.totalHeight+float64(w.height) {
			if w.loop {
				w.phase = PhasePauseEnd
				w.phaseStart = now
			}
		}

	case PhasePauseEnd:
		if elapsed >= w.pauseAtEnd {
			// Reset to beginning
			w.scrollOffset = 0
			w.logoScale = 1.0
			if w.preIntroEnabled {
				w.phase = PhasePreIntroFadeIn
			} else if w.logoEnabled {
				w.phase = PhaseLogoHold
			} else {
				w.phase = PhaseCrawl
			}
			w.phaseStart = now
		}
	}

	return nil
}

// Render draws the current animation frame
func (w *StarWarsIntroWidget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	elapsed := time.Since(w.phaseStart).Seconds()

	switch w.phase {
	case PhasePreIntroFadeIn, PhasePreIntroHold, PhasePreIntroFadeOut:
		w.renderPreIntro(img, elapsed)

	case PhaseLogoHold, PhaseLogoShrink:
		w.renderStars(img)
		w.renderLogo(img)

	case PhaseCrawl:
		w.renderStars(img)
		w.renderCrawl(img)

	case PhasePauseEnd:
		w.renderStars(img)
	}

	// Draw border if enabled
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, nil
}

// renderPreIntro draws the pre-intro text with fade effect
func (w *StarWarsIntroWidget) renderPreIntro(img *image.Gray, elapsed float64) {
	var brightness float64

	switch w.phase {
	case PhasePreIntroFadeIn:
		brightness = elapsed / w.preIntroFadeIn
		if brightness > 1.0 {
			brightness = 1.0
		}
	case PhasePreIntroHold:
		brightness = 1.0
	case PhasePreIntroFadeOut:
		brightness = 1.0 - elapsed/w.preIntroFadeOut
		if brightness < 0 {
			brightness = 0
		}
	}

	c := uint8(float64(w.preIntroColor) * brightness)
	if c < 5 {
		return
	}

	// Calculate total height of all lines
	lineHeight := 8 // Use standard line height for pre-intro
	totalHeight := len(w.preIntroLines) * lineHeight

	// Start Y position to center all lines vertically
	startY := (w.height - totalHeight) / 2

	// Left padding
	leftPadding := 2

	// Draw each line left-aligned with compact spacing
	for i, line := range w.preIntroLines {
		y := startY + i*lineHeight
		w.drawTextCompact(img, line, leftPadding, y, c)
	}
}

// renderStars draws the background starfield
func (w *StarWarsIntroWidget) renderStars(img *image.Gray) {
	if !w.starsEnabled {
		return
	}

	for _, star := range w.stars {
		if star.x >= 0 && star.x < w.width && star.y >= 0 && star.y < w.height {
			img.SetGray(star.x, star.y, color.Gray{Y: star.brightness})
		}
	}
}

// renderLogo draws the shrinking logo
func (w *StarWarsIntroWidget) renderLogo(img *image.Gray) {
	if w.logoScale < w.logoFinalScale {
		return
	}

	// Calculate scaled font size based on logo scale
	// At scale 1.0, use large text that fills width
	// At smaller scales, reduce size proportionally

	brightness := uint8(float64(w.logoColor) * w.logoScale)
	if brightness < 10 {
		return
	}

	// Calculate total logo dimensions
	maxLineWidth := 0
	for _, line := range w.logoLines {
		lineWidth := len(line) * w.charWidth
		if lineWidth > maxLineWidth {
			maxLineWidth = lineWidth
		}
	}

	// Get glyph height for accurate vertical calculation
	glyphHeight := 7 // Default for 5x7 font
	if w.glyphSet != nil {
		glyphHeight = w.glyphSet.GlyphHeight
	}

	// Calculate logo height: lines * (glyph height + spacing between lines)
	// Last line doesn't need spacing after it
	numLines := len(w.logoLines)
	logoHeight := numLines*glyphHeight + (numLines-1)*w.lineSpacing

	// Scale factor to fit width (with small margin)
	scaleByWidth := float64(w.width-4) / float64(maxLineWidth)

	// Scale factor to fit height (with small margin)
	scaleByHeight := float64(w.height-4) / float64(logoHeight)

	// Use the smaller scale to ensure logo fits both dimensions
	baseScale := scaleByWidth
	if scaleByHeight < baseScale {
		baseScale = scaleByHeight
	}

	scale := baseScale * w.logoScale
	if scale < 0.3 {
		scale = 0.3
	}

	// Calculate vertical position - stays centered while shrinking
	// Each line takes glyphHeight, with lineSpacing between lines
	lineStep := float64(glyphHeight + w.lineSpacing)
	totalTextHeight := float64(numLines)*float64(glyphHeight)*scale + float64(numLines-1)*float64(w.lineSpacing)*scale

	// Logo shrinks toward screen center
	centerY := float64(w.height) / 2
	baseY := centerY - totalTextHeight/2

	// Draw each line of the logo
	for i, line := range w.logoLines {
		scaledLineWidth := float64(len(line)*w.charWidth) * scale
		x := int((float64(w.width) - scaledLineWidth) / 2)
		y := int(baseY + float64(i)*lineStep*scale)

		w.drawTextScaled(img, line, x, y, brightness, scale)
	}
}

// renderCrawl draws the text crawl with perspective and slant
func (w *StarWarsIntroWidget) renderCrawl(img *image.Gray) {
	for i, line := range w.lines {
		w.drawPerspectiveLine(img, line, i)
	}
}

// drawPerspectiveLine draws a single line with perspective, fade, and radial slant effects
func (w *StarWarsIntroWidget) drawPerspectiveLine(img *image.Gray, text string, lineIndex int) {
	// Calculate base Y position for this line
	// Text starts at bottom of screen and scrolls up
	baseY := float64(w.height) + float64(lineIndex*w.lineSpacing) - w.scrollOffset

	// Skip if completely off the screen
	if baseY < -float64(w.lineSpacing*2) || baseY > float64(w.height)+float64(w.lineSpacing) {
		return
	}

	// Calculate perspective scale based on Y position
	// At bottom (y = height): scale = 1.0 (full size)
	// At top (y = 0): scale = 1.0 - perspective (smaller)
	normalizedY := baseY / float64(w.height)
	if normalizedY < 0 {
		normalizedY = 0
	}
	if normalizedY > 1 {
		normalizedY = 1
	}

	// Perspective: text gets smaller toward the top
	scale := (1.0 - w.perspective) + w.perspective*normalizedY

	// Calculate fade based on Y position
	var fadeFactor float64 = 1.0
	if normalizedY < w.fadeTop {
		fadeFactor = normalizedY / w.fadeTop
	}

	// Calculate brightness
	brightness := uint8(float64(w.textColor) * fadeFactor)
	if brightness < 10 {
		return
	}

	// Calculate text width at this scale
	// Each line fills the full width at the bottom, narrower at top
	textWidth := float64(len(text) * w.charWidth)
	scaledWidth := textWidth * scale

	// Center horizontally
	startX := (float64(w.width) - scaledWidth) / 2

	// Screen center X for radial slant calculation
	centerX := float64(w.width) / 2

	// Draw each character with radial slant toward center
	for i, ch := range text {
		if ch == ' ' {
			continue
		}

		// Character position with perspective scaling
		charOffset := float64(i) * float64(w.charWidth) * scale
		charX := startX + charOffset
		charY := int(baseY)

		// Calculate character center position
		charCenterX := charX + float64(w.charWidth)*scale/2

		// Calculate radial slant: characters lean toward the center
		// Left of center: positive slant (lean right toward center)
		// Right of center: negative slant (lean left toward center)
		// The magnitude is proportional to distance from center
		distFromCenter := charCenterX - centerX
		// Normalize by half-width to get -1 to +1 range
		normalizedDist := distFromCenter / (float64(w.width) / 2)
		// Invert: left side (negative dist) should lean right (positive slant)
		// Apply slant strength and Y-based reduction (less slant near top)
		charSlant := -normalizedDist * w.slant * normalizedY

		if int(charX) < -w.charWidth || int(charX) >= w.width || charY < 0 || charY >= w.height {
			continue
		}

		w.drawCharSlanted(img, int(charX), charY, ch, brightness, scale, charSlant)
	}
}

// drawText draws text at the specified position
func (w *StarWarsIntroWidget) drawText(img *image.Gray, text string, x, y int, brightness uint8, scale float64, slant float64) {
	for i, ch := range text {
		if ch == ' ' {
			continue
		}
		charX := x + int(float64(i*w.charWidth)*scale)
		w.drawCharSlanted(img, charX, y, ch, brightness, scale, slant)
	}
}

// drawTextCompact draws text with half-width spaces and narrow punctuation
func (w *StarWarsIntroWidget) drawTextCompact(img *image.Gray, text string, x, y int, brightness uint8) {
	halfWidth := w.charWidth / 2
	cursorX := x
	for _, ch := range text {
		switch ch {
		case ' ':
			cursorX += halfWidth
		case ',', '.', '!', ':', ';', '\'', '"':
			w.drawChar(img, cursorX, y, ch, brightness, 1.0)
			cursorX += halfWidth
		default:
			w.drawChar(img, cursorX, y, ch, brightness, 1.0)
			cursorX += w.charWidth
		}
	}
}

// drawTextScaled draws text at a specific scale (for logo)
func (w *StarWarsIntroWidget) drawTextScaled(img *image.Gray, text string, x, y int, brightness uint8, scale float64) {
	for i, ch := range text {
		if ch == ' ' {
			continue
		}
		charX := x + int(float64(i*w.charWidth)*scale)
		w.drawChar(img, charX, y, ch, brightness, scale)
	}
}

// drawChar draws a single character at the specified position with scaling
// When scale > 1, fills rectangles to avoid gaps
func (w *StarWarsIntroWidget) drawChar(img *image.Gray, x, y int, ch rune, brightness uint8, scale float64) {
	glyph := glyphs.GetGlyph(w.glyphSet, ch)
	if glyph == nil {
		return
	}

	c := color.Gray{Y: brightness}
	glyphHeight := glyph.Height
	glyphWidth := glyph.Width

	// Calculate pixel size for filling (at least 1, more when scaled up)
	pixelSize := int(math.Ceil(scale))
	if pixelSize < 1 {
		pixelSize = 1
	}

	for row := 0; row < glyphHeight && row < len(glyph.Data); row++ {
		for col := 0; col < glyphWidth && col < len(glyph.Data[row]); col++ {
			if glyph.Data[row][col] {
				px := x + int(float64(col)*scale)
				py := y + int(float64(row)*scale)

				// Fill a rectangle to avoid gaps when scaled up
				for dy := 0; dy < pixelSize; dy++ {
					for dx := 0; dx < pixelSize; dx++ {
						fx, fy := px+dx, py+dy
						if fx >= 0 && fx < w.width && fy >= 0 && fy < w.height {
							img.Set(fx, fy, c)
						}
					}
				}
			}
		}
	}
}

// drawCharSlanted draws a character with italic/slant effect
// The slant creates a shear transformation where top of character shifts right
func (w *StarWarsIntroWidget) drawCharSlanted(img *image.Gray, x, y int, ch rune, brightness uint8, scale float64, slantDegrees float64) {
	glyph := glyphs.GetGlyph(w.glyphSet, ch)
	if glyph == nil {
		return
	}

	c := color.Gray{Y: brightness}
	glyphHeight := glyph.Height
	glyphWidth := glyph.Width

	// Calculate pixel size for filling
	pixelSize := int(math.Ceil(scale))
	if pixelSize < 1 {
		pixelSize = 1
	}

	// Slant factor: how many pixels shift horizontally per vertical pixel
	// tan(angle) gives the ratio, but we need to amplify for small fonts
	slantRad := slantDegrees * math.Pi / 180.0
	slantFactor := math.Tan(slantRad)

	// Calculate the total rendered height of the character
	renderedHeight := float64(glyphHeight) * scale

	for row := 0; row < glyphHeight && row < len(glyph.Data); row++ {
		// Calculate the actual Y position on screen for this row
		screenY := float64(row) * scale

		// Slant: top of character shifts RIGHT, bottom stays put
		// Calculate how far from the BOTTOM this row is (in screen pixels)
		distFromBottom := renderedHeight - screenY
		// Apply slant: shift right based on distance from bottom
		slantOffset := distFromBottom * slantFactor

		for col := 0; col < glyphWidth && col < len(glyph.Data[row]); col++ {
			if glyph.Data[row][col] {
				px := x + int(float64(col)*scale+slantOffset)
				py := y + int(screenY)

				// Fill rectangle to avoid gaps
				for dy := 0; dy < pixelSize; dy++ {
					for dx := 0; dx < pixelSize; dx++ {
						fx, fy := px+dx, py+dy
						if fx >= 0 && fx < w.width && fy >= 0 && fy < w.height {
							img.Set(fx, fy, c)
						}
					}
				}
			}
		}
	}
}

// getCharWidth returns the width of a character in the internal font
func (w *StarWarsIntroWidget) getCharWidth(ch rune) int {
	glyph := glyphs.GetGlyph(w.glyphSet, ch)
	if glyph == nil {
		return w.charWidth
	}
	return glyph.Width
}
