package widget

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// Spawn delay constants for matrix column respawn timing
const (
	// minSpawnDelay is the minimum frames before a column respawns
	minSpawnDelay = 10
	// spawnDelayRange is the random range added to minSpawnDelay
	spawnDelayRange = 40
	// densityDelayRange is additional delay range when density check fails
	densityDelayRange = 60
)

// MatrixColumn represents a single falling column of characters
type MatrixColumn struct {
	x          int     // X position
	y          float64 // Current Y position (float for smooth movement)
	speed      float64 // Fall speed in pixels per update
	length     int     // Number of characters in the trail
	chars      []rune  // Characters in this column
	brightness []uint8 // Brightness for each character (fades down)
	active     bool    // Whether this column is currently falling
	spawnDelay int     // Frames until spawn
}

// MatrixWidget displays the classic Matrix "digital rain" effect
type MatrixWidget struct {
	*BaseWidget
	mu sync.Mutex

	// Configuration
	charset        []rune  // Character set to use
	charsetName    string  // Name of charset for reference
	density        float64 // Column density (0.0-1.0)
	minSpeed       float64 // Minimum fall speed
	maxSpeed       float64 // Maximum fall speed
	minLength      int     // Minimum trail length
	maxLength      int     // Maximum trail length
	headColor      uint8   // Brightness of leading character
	trailFade      float64 // How quickly trail fades (0.0-1.0)
	charChangeRate float64 // Probability of character changing per frame

	// State
	columns    []*MatrixColumn
	glyphSet   *glyphs.GlyphSet
	charWidth  int
	charHeight int
	numColumns int
	lastUpdate time.Time
	rng        *rand.Rand
}

// Default character sets
var (
	// Katakana-like characters (classic Matrix look)
	matrixKatakana = []rune("アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン")
	// ASCII characters
	matrixASCII = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%&*")
	// Binary
	matrixBinary = []rune("01")
	// Digits only
	matrixDigits = []rune("0123456789")
	// Hex
	matrixHex = []rune("0123456789ABCDEF")
)

// NewMatrixWidget creates a new Matrix digital rain widget
func NewMatrixWidget(cfg config.WidgetConfig) (*MatrixWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	// Get matrix-specific configuration
	density := 0.4
	minSpeed := 0.5
	maxSpeed := 2.0
	minLength := 4
	maxLength := 15
	headColor := uint8(255)
	trailFade := 0.85
	charChangeRate := 0.02
	charsetName := "ascii" // Default to ASCII since it's guaranteed to work

	if cfg.Matrix != nil {
		if cfg.Matrix.Density > 0 {
			density = cfg.Matrix.Density
		}
		if cfg.Matrix.MinSpeed > 0 {
			minSpeed = cfg.Matrix.MinSpeed
		}
		if cfg.Matrix.MaxSpeed > 0 {
			maxSpeed = cfg.Matrix.MaxSpeed
		}
		if cfg.Matrix.MinLength > 0 {
			minLength = cfg.Matrix.MinLength
		}
		if cfg.Matrix.MaxLength > 0 {
			maxLength = cfg.Matrix.MaxLength
		}
		if cfg.Matrix.HeadColor > 0 {
			headColor = uint8(cfg.Matrix.HeadColor)
		}
		if cfg.Matrix.TrailFade > 0 {
			trailFade = cfg.Matrix.TrailFade
		}
		if cfg.Matrix.CharChangeRate > 0 {
			charChangeRate = cfg.Matrix.CharChangeRate
		}
		if cfg.Matrix.Charset != "" {
			charsetName = cfg.Matrix.Charset
		}
	}

	// Select character set
	charset := matrixASCII
	switch charsetName {
	case "katakana":
		charset = matrixKatakana
	case "binary":
		charset = matrixBinary
	case "digits":
		charset = matrixDigits
	case "hex":
		charset = matrixHex
	case "ascii":
		charset = matrixASCII
	}

	// Determine font size: "small" (3x5), "large" (5x7), or "auto" (default)
	fontSizeOption := "auto"
	if cfg.Matrix != nil && cfg.Matrix.FontSize != "" {
		fontSizeOption = cfg.Matrix.FontSize
	}

	glyphSet := glyphs.Font3x5
	charWidth := 4  // 3 + 1 spacing
	charHeight := 6 // 5 + 1 spacing

	switch fontSizeOption {
	case "small":
		// Use 3x5 font (already set as default)
	case "large":
		glyphSet = glyphs.Font5x7
		charWidth = 6  // 5 + 1 spacing
		charHeight = 8 // 7 + 1 spacing
	default: // "auto" or unrecognized
		// Auto-select based on display height
		if pos.H >= 30 {
			glyphSet = glyphs.Font5x7
			charWidth = 6
			charHeight = 8
		}
	}

	// Calculate number of columns
	numColumns := pos.W / charWidth

	// Initialize random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	w := &MatrixWidget{
		BaseWidget:     base,
		charset:        charset,
		charsetName:    charsetName,
		density:        density,
		minSpeed:       minSpeed,
		maxSpeed:       maxSpeed,
		minLength:      minLength,
		maxLength:      maxLength,
		headColor:      headColor,
		trailFade:      trailFade,
		charChangeRate: charChangeRate,
		glyphSet:       glyphSet,
		charWidth:      charWidth,
		charHeight:     charHeight,
		numColumns:     numColumns,
		lastUpdate:     time.Now(),
		rng:            rng,
	}

	// Initialize columns
	w.initColumns()

	return w, nil
}

// initColumns creates initial column state
func (w *MatrixWidget) initColumns() {
	pos := w.GetPosition()
	w.columns = make([]*MatrixColumn, w.numColumns)

	for i := 0; i < w.numColumns; i++ {
		w.columns[i] = &MatrixColumn{
			x:          i * w.charWidth,
			y:          float64(-w.rng.Intn(pos.H)), // Start above screen
			speed:      w.minSpeed + w.rng.Float64()*(w.maxSpeed-w.minSpeed),
			length:     w.minLength + w.rng.Intn(w.maxLength-w.minLength+1),
			chars:      make([]rune, w.maxLength+5),
			brightness: make([]uint8, w.maxLength+5),
			active:     w.rng.Float64() < w.density,
			spawnDelay: w.rng.Intn(30),
		}

		// Initialize characters
		for j := range w.columns[i].chars {
			w.columns[i].chars[j] = w.randomChar()
		}

		// Initialize brightness (fading trail)
		for j := range w.columns[i].brightness {
			if j == 0 {
				w.columns[i].brightness[j] = w.headColor
			} else {
				w.columns[i].brightness[j] = uint8(float64(w.headColor) * math.Pow(w.trailFade, float64(j)))
			}
		}
	}
}

// randomChar returns a random character from the charset
func (w *MatrixWidget) randomChar() rune {
	return w.charset[w.rng.Intn(len(w.charset))]
}

// Update advances the matrix animation
func (w *MatrixWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	pos := w.GetPosition()
	now := time.Now()
	dt := now.Sub(w.lastUpdate).Seconds()
	w.lastUpdate = now

	// Scale movement by delta time (target ~30fps)
	timeScale := dt * 30.0

	for _, col := range w.columns {
		if !col.active {
			col.spawnDelay--
			if col.spawnDelay <= 0 {
				col.active = true
				col.y = float64(-w.charHeight)
				col.speed = w.minSpeed + w.rng.Float64()*(w.maxSpeed-w.minSpeed)
				col.length = w.minLength + w.rng.Intn(w.maxLength-w.minLength+1)
			}
			continue
		}

		// Move column down
		col.y += col.speed * timeScale

		// Randomly change characters in trail
		if w.rng.Float64() < w.charChangeRate {
			idx := w.rng.Intn(len(col.chars))
			col.chars[idx] = w.randomChar()
		}

		// Check if column has fallen off the screen
		if int(col.y)-col.length*w.charHeight > pos.H {
			col.active = false
			col.spawnDelay = w.rng.Intn(spawnDelayRange) + minSpawnDelay
			// Decide if this column should respawn based on density
			if w.rng.Float64() > w.density {
				col.spawnDelay += w.rng.Intn(densityDelayRange)
			}
		}
	}

	return nil
}

// Render draws the matrix effect
func (w *MatrixWidget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw each active column
	for _, col := range w.columns {
		if !col.active {
			continue
		}

		// Draw characters in trail
		for i := 0; i < col.length; i++ {
			charY := int(col.y) - i*w.charHeight

			// Skip if off the screen
			if charY < -w.charHeight || charY >= pos.H {
				continue
			}

			// Get character and brightness
			charIdx := i % len(col.chars)
			char := col.chars[charIdx]
			brightness := col.brightness[i]

			// Skip very dim characters
			if brightness < 20 {
				continue
			}

			// Draw the character
			w.drawChar(img, col.x, charY, char, brightness)
		}
	}

	// Draw border if enabled
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	return img, nil
}

// drawChar draws a single character at the specified position
func (w *MatrixWidget) drawChar(img *image.Gray, x, y int, char rune, brightness uint8) {
	glyph := glyphs.GetGlyph(w.glyphSet, char)
	if glyph == nil {
		return
	}

	c := color.Gray{Y: brightness}
	glyphs.DrawGlyph(img, glyph, x, y, c)
}
