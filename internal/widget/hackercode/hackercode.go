// Package hackercode provides a widget that displays procedurally generated code.
package hackercode

import (
	"image"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
)

func init() {
	widget.Register("hacker_code", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Code style constants.
const (
	StyleC     = "c"
	StyleAsm   = "asm"
	StyleMixed = "mixed"
)

// Default configuration values.
const (
	defaultStyle         = StyleC
	defaultTypingSpeed   = 50  // characters per second
	defaultLineDelay     = 200 // milliseconds
	defaultShowCursor    = true
	defaultCursorBlinkMs = 500
	defaultIndentSize    = 2
)

// Widget displays procedurally generated code being "typed" in real-time.
type Widget struct {
	*widget.BaseWidget
	mu sync.Mutex

	// Configuration
	style          string // "c", "asm", "mixed"
	typingSpeedMin int    // minimum characters per second
	typingSpeedMax int    // maximum characters per second
	currentSpeed   int    // current typing speed for this line
	lineDelay      int    // milliseconds pause at end of line
	showCursor     bool
	cursorBlinkMs  int
	indentSize     int

	// Text rendering
	fontFace        font.Face        // TTF font face (nil for internal fonts)
	fontName        string           // Font name for smart rendering functions
	glyphSet        *glyphs.GlyphSet // Only used for internal fonts (nil for TTF)
	charWidth       int              // Character width (average for TTF)
	charHeight      int              // Character height
	maxCharsPerLine int              // Maximum characters that fit on one line
	contentWidth    int              // Content area width in pixels

	// State
	lines         []string  // Completed lines (may exceed maxLines, scroll handled in Render)
	maxLines      int       // Maximum lines that fit on screen
	currentLine   string    // Full line to type
	typedChars    int       // Characters typed so far on current line
	lineComplete  bool      // True when line is complete, waiting for delay
	lineDelayEnd  time.Time // When line delay ends
	cursorVisible bool      // Cursor blink state
	lastCharTime  time.Time // When last character was typed

	// Generators
	generator    CodeGenerator
	cGenerator   *CGenerator
	asmGenerator *AsmGenerator
	rng          *rand.Rand
}

// New creates a new hacker code widget.
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	pos := base.GetPosition()
	helper := shared.NewConfigHelper(cfg)

	// Parse hacker_code specific configuration
	style := defaultStyle
	typingSpeedMin := defaultTypingSpeed
	typingSpeedMax := defaultTypingSpeed
	lineDelay := defaultLineDelay
	showCursor := defaultShowCursor
	cursorBlinkMs := defaultCursorBlinkMs
	indentSize := defaultIndentSize

	if cfg.HackerCode != nil {
		if cfg.HackerCode.Style != "" {
			style = cfg.HackerCode.Style
		}
		if cfg.HackerCode.TypingSpeed != nil {
			if cfg.HackerCode.TypingSpeed.Min > 0 {
				typingSpeedMin = cfg.HackerCode.TypingSpeed.Min
			}
			if cfg.HackerCode.TypingSpeed.Max > 0 {
				typingSpeedMax = cfg.HackerCode.TypingSpeed.Max
			}
			// Ensure min <= max
			if typingSpeedMin > typingSpeedMax {
				typingSpeedMin, typingSpeedMax = typingSpeedMax, typingSpeedMin
			}
		}
		if cfg.HackerCode.LineDelay >= 0 {
			lineDelay = cfg.HackerCode.LineDelay
		}
		if cfg.HackerCode.ShowCursor != nil {
			showCursor = *cfg.HackerCode.ShowCursor
		}
		if cfg.HackerCode.CursorBlinkMs > 0 {
			cursorBlinkMs = cfg.HackerCode.CursorBlinkMs
		}
		if cfg.HackerCode.IndentSize > 0 {
			indentSize = cfg.HackerCode.IndentSize
		}
	}

	// Determine font from standard text settings
	textSettings := helper.GetTextSettings()
	fontName := textSettings.FontName
	fontSize := textSettings.FontSize

	// Load font - returns font.Face for TTF, nil for internal fonts
	fontFace, _ := bitmap.LoadFont(fontName, fontSize)

	var glyphSet *glyphs.GlyphSet
	var charWidth, charHeight int

	if fontFace == nil && bitmap.IsInternalFont(fontName) {
		// Internal bitmap font
		glyphSet = bitmap.GetInternalFontByName(fontName)
		if glyphSet == nil {
			glyphSet = glyphs.Font5x7 // Fallback
		}
		charWidth = glyphSet.GlyphWidth + 1   // glyph width + spacing
		charHeight = glyphSet.GlyphHeight + 1 // glyph height + spacing
	} else if fontFace != nil {
		// TTF font - measure character dimensions
		// Use 'M' as reference for width (common monospace reference)
		charWidth, charHeight = bitmap.MeasureText("M", fontFace)
		// Add spacing
		charWidth++
		charHeight++
	} else {
		// Fallback to default internal font (font: null with no TTF available)
		glyphSet = glyphs.Font5x7
		charWidth = glyphSet.GlyphWidth + 1
		charHeight = glyphSet.GlyphHeight + 1
	}

	// Calculate content area dimensions (approximation using position, actual may differ slightly with borders/padding)
	contentWidth := pos.W
	maxCharsPerLine := contentWidth / charWidth
	if maxCharsPerLine < 1 {
		maxCharsPerLine = 1
	}

	// Calculate max lines that fit on screen
	maxLines := pos.H / charHeight
	if maxLines < 1 {
		maxLines = 1
	}

	// Initialize random number generator
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))

	// Create generators
	cGen := NewCGenerator(seed)
	asmGen := NewAsmGenerator(seed + 1)

	// Select initial generator
	var generator CodeGenerator
	switch style {
	case StyleAsm:
		generator = asmGen
	case StyleMixed:
		// Start with C, will alternate
		generator = cGen
	default:
		generator = cGen
	}

	now := time.Now()
	w := &Widget{
		BaseWidget:      base,
		style:           style,
		typingSpeedMin:  typingSpeedMin,
		typingSpeedMax:  typingSpeedMax,
		lineDelay:       lineDelay,
		showCursor:      showCursor,
		cursorBlinkMs:   cursorBlinkMs,
		indentSize:      indentSize,
		fontFace:        fontFace,
		fontName:        fontName,
		glyphSet:        glyphSet,
		charWidth:       charWidth,
		charHeight:      charHeight,
		maxCharsPerLine: maxCharsPerLine,
		contentWidth:    contentWidth,
		lines:           make([]string, 0, maxLines+1),
		maxLines:        maxLines,
		cursorVisible:   true,
		lastCharTime:    now,
		generator:       generator,
		cGenerator:      cGen,
		asmGenerator:    asmGen,
		rng:             rng,
	}

	// Pick initial typing speed
	w.currentSpeed = w.pickTypingSpeed()

	// Get first line to type
	w.currentLine = w.generator.NextLine()

	return w, nil
}

// pickTypingSpeed returns a random typing speed between min and max.
func (w *Widget) pickTypingSpeed() int {
	if w.typingSpeedMin == w.typingSpeedMax {
		return w.typingSpeedMin
	}
	return w.typingSpeedMin + w.rng.Intn(w.typingSpeedMax-w.typingSpeedMin+1)
}

// wrapLine breaks a line into segments that fit within maxCharsPerLine.
// For TTF fonts, it measures actual text width; for internal fonts, uses character count.
func (w *Widget) wrapLine(line string) []string {
	if len(line) == 0 {
		return []string{""}
	}

	var segments []string
	runes := []rune(line)

	if w.fontFace != nil {
		// TTF font: measure actual width
		start := 0
		for start < len(runes) {
			// Find how many characters fit
			end := start + 1
			for end <= len(runes) {
				segment := string(runes[start:end])
				width, _ := bitmap.MeasureText(segment, w.fontFace)
				if width > w.contentWidth {
					// This character doesn't fit, use previous position
					if end > start+1 {
						end--
					}
					break
				}
				end++
			}
			if end > len(runes) {
				end = len(runes)
			}
			segments = append(segments, string(runes[start:end]))
			start = end
		}
	} else {
		// Internal font: use character count
		for i := 0; i < len(runes); i += w.maxCharsPerLine {
			end := i + w.maxCharsPerLine
			if end > len(runes) {
				end = len(runes)
			}
			segments = append(segments, string(runes[i:end]))
		}
	}

	if len(segments) == 0 {
		return []string{""}
	}
	return segments
}

// Update advances the typing animation.
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()

	// Handle cursor blink
	if w.showCursor {
		elapsed := now.Sub(w.lastCharTime)
		// Toggle cursor based on blink interval
		w.cursorVisible = (int(elapsed.Milliseconds())/w.cursorBlinkMs)%2 == 0
	}

	// Handle line delay (pause at end of line)
	if w.lineComplete {
		if now.After(w.lineDelayEnd) {
			w.startNextLine()
		}
		return nil
	}

	// Calculate how many characters should be typed based on elapsed time
	// This handles both slow and fast typing speeds correctly
	elapsed := now.Sub(w.lastCharTime)
	charInterval := time.Second / time.Duration(w.currentSpeed)

	// Type characters one by one until we've caught up with elapsed time
	charsTypedThisUpdate := 0
	for elapsed >= charInterval && w.typedChars < len(w.currentLine) {
		w.typedChars++
		charsTypedThisUpdate++
		elapsed -= charInterval
	}

	// Update lastCharTime only if we typed something
	if charsTypedThisUpdate > 0 {
		w.lastCharTime = now.Add(-elapsed) // Account for remainder
	}

	// Check if line is complete
	if w.typedChars >= len(w.currentLine) {
		w.lineComplete = true
		w.lineDelayEnd = now.Add(time.Duration(w.lineDelay) * time.Millisecond)
	}

	return nil
}

// startNextLine completes the current line and prepares for the next one.
// Scrolling is NOT done here - it's handled in Render when we need to display
// a line that would be off-screen.
func (w *Widget) startNextLine() {
	// Add completed line to buffer
	w.lines = append(w.lines, w.currentLine)

	// In mixed mode, occasionally switch generators
	if w.style == StyleMixed && w.rng.Float64() < 0.1 {
		if w.generator == w.cGenerator {
			w.generator = w.asmGenerator
			w.asmGenerator.Reset()
		} else {
			w.generator = w.cGenerator
			w.cGenerator.Reset()
		}
	}

	// Get next line with new random typing speed
	w.currentLine = w.generator.NextLine()
	w.typedChars = 0
	w.lineComplete = false
	w.lastCharTime = time.Now()
	w.currentSpeed = w.pickTypingSpeed()
}

// Render draws the code display.
func (w *Widget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create canvas with background
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Get content area
	contentArea := w.GetContentArea()

	// Build list of all visual lines (wrapped completed lines + wrapped current line)
	var allVisualLines []string
	for _, line := range w.lines {
		wrapped := w.wrapLine(line)
		allVisualLines = append(allVisualLines, wrapped...)
	}

	// Handle current line being typed
	var currentLineSegments []string
	var cursorSegmentIdx int
	var cursorPosInSegment int

	if w.typedChars > 0 {
		partialLine := w.currentLine[:w.typedChars]
		currentLineSegments = w.wrapLine(partialLine)

		// Find cursor position: it's at the end of the last segment
		cursorSegmentIdx = len(currentLineSegments) - 1
		if cursorSegmentIdx >= 0 {
			cursorPosInSegment = len([]rune(currentLineSegments[cursorSegmentIdx]))
		}
	} else {
		// No chars typed yet, cursor at start of first segment
		currentLineSegments = []string{""}
		cursorSegmentIdx = 0
		cursorPosInSegment = 0
	}

	// Calculate total visual lines
	totalVisualLines := len(allVisualLines) + len(currentLineSegments)

	// Determine scroll offset
	scrollOffset := 0
	if totalVisualLines > w.maxLines {
		scrollOffset = totalVisualLines - w.maxLines
	}

	// Draw visual lines with scroll offset
	y := contentArea.Y
	visualLineIdx := 0

	// Draw completed lines (wrapped)
	for _, line := range allVisualLines {
		if visualLineIdx >= scrollOffset && y < contentArea.Y+contentArea.Height {
			w.drawText(img, contentArea.X, y, line)
			y += w.charHeight
		}
		visualLineIdx++
	}

	// Draw current line segments being typed
	for segIdx, segment := range currentLineSegments {
		if visualLineIdx >= scrollOffset && y < contentArea.Y+contentArea.Height {
			w.drawText(img, contentArea.X, y, segment)

			// Draw cursor on this segment if it's the cursor segment
			if w.showCursor && w.cursorVisible && segIdx == cursorSegmentIdx {
				var cursorX int
				if w.fontFace != nil {
					// TTF font: measure actual text width
					textWidth, _ := bitmap.MeasureText(segment, w.fontFace)
					cursorX = contentArea.X + textWidth
				} else {
					// Internal font: use fixed character width
					cursorX = contentArea.X + cursorPosInSegment*w.charWidth
				}
				w.drawCursor(img, cursorX, y)
			}
			y += w.charHeight
		}
		visualLineIdx++
	}

	return img, nil
}

// drawText draws a string at the specified position.
func (w *Widget) drawText(img *image.Gray, x, y int, text string) {
	if w.fontFace != nil {
		// TTF font rendering
		bitmap.DrawTextAt(img, text, w.fontFace, x, y)
	} else if w.glyphSet != nil {
		// Internal font rendering
		col := color.Gray{Y: 255}
		for _, ch := range text {
			glyph := glyphs.GetGlyph(w.glyphSet, ch)
			if glyph != nil {
				glyphs.DrawGlyph(img, glyph, x, y, col)
			}
			x += w.charWidth
		}
	}
}

// drawCursor draws the cursor at the specified position.
func (w *Widget) drawCursor(img *image.Gray, x, y int) {
	// Draw a block cursor
	cursorWidth := w.charWidth - 1
	cursorHeight := w.charHeight - 1

	bounds := img.Bounds()
	for dy := 0; dy < cursorHeight; dy++ {
		for dx := 0; dx < cursorWidth; dx++ {
			px := x + dx
			py := y + dy
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				img.SetGray(px, py, color.Gray{Y: 255})
			}
		}
	}
}
