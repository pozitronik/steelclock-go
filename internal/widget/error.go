package widget

import (
	"image"
	"image/color"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// ErrorWidget displays error messages with warning symbols
type ErrorWidget struct {
	*BaseWidget
	message     string
	flashState  bool
	lastFlash   time.Time
	flashPeriod time.Duration
}

// NewErrorWidget creates a new error widget
func NewErrorWidget(displayWidth, displayHeight int, message string) *ErrorWidget {
	cfg := config.WidgetConfig{
		Type:    "error",
		ID:      "error_display",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: displayWidth,
			H: displayHeight,
		},
		Style: config.StyleConfig{
			BackgroundColor: 0,
			Border:          false,
			BorderColor:     255,
		},
	}

	return &ErrorWidget{
		BaseWidget:  NewBaseWidget(cfg),
		message:     message,
		flashState:  true,
		lastFlash:   time.Now(),
		flashPeriod: 500 * time.Millisecond, // Flash every 500ms
	}
}

// Update toggles flash state
func (w *ErrorWidget) Update() error {
	now := time.Now()
	if now.Sub(w.lastFlash) >= w.flashPeriod {
		w.flashState = !w.flashState
		w.lastFlash = now
	}
	return nil
}

// Render draws the error display with warning triangles
func (w *ErrorWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Only draw if flash state is on
	if !w.flashState {
		return img, nil
	}

	c := color.Gray{Y: uint8(style.BorderColor)}

	// Draw warning triangles on left and right
	// Triangle size
	triangleSize := 10
	if pos.H < 20 {
		triangleSize = pos.H / 2
	}

	// Left triangle
	leftX := 5
	centerY := pos.H / 2
	bitmap.DrawWarningTriangle(img, leftX, centerY, triangleSize, c)

	// Right triangle
	rightX := pos.W - 5 - triangleSize
	bitmap.DrawWarningTriangle(img, rightX, centerY, triangleSize, c)

	// Draw message text centered between triangles
	availableX := leftX + triangleSize + 5
	availableW := (rightX) - (leftX + triangleSize + 5)

	// Calculate text width (6 pixels per character including space)
	charWidth := 6
	textWidth := len(w.message) * charWidth

	// Center text in available space
	textX := availableX + (availableW-textWidth)/2
	if textX < availableX {
		textX = availableX // Don't go past left boundary
	}

	// Draw text character by character using a simple 5x7 bitmap font
	drawErrorText(img, w.message, textX, centerY-3, c)

	return img, nil
}

// drawErrorText draws text using a simple bitmap font
func drawErrorText(img *image.Gray, text string, x, y int, c color.Gray) {
	charWidth := 6 // 5 pixels + 1 space
	charHeight := 7

	currentX := x
	bounds := img.Bounds()

	for _, ch := range text {
		// Get character bitmap
		charBitmap := getCharBitmap(ch)

		// Draw character
		for dy := 0; dy < charHeight && y+dy >= 0 && y+dy < bounds.Max.Y; dy++ {
			for dx := 0; dx < 5 && currentX+dx >= 0 && currentX+dx < bounds.Max.X; dx++ {
				if charBitmap[dy][dx] {
					img.Set(currentX+dx, y+dy, c)
				}
			}
		}

		currentX += charWidth
	}
}

// charBitmaps is a map of character bitmaps for the 5x7 font
var charBitmaps = map[rune][7][5]bool{
	'A': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
	},
	'B': {
		{true, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, false},
	},
	'C': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'D': {
		{true, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, false},
	},
	'E': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, true},
	},
	'F': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
	},
	'G': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, false},
		{true, false, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, true},
	},
	'H': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
	},
	'I': {
		{false, true, true, true, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, true, true, true, false},
	},
	'K': {
		{true, false, false, false, true},
		{true, false, false, true, false},
		{true, false, true, false, false},
		{true, true, false, false, false},
		{true, false, true, false, false},
		{true, false, false, true, false},
		{true, false, false, false, true},
	},
	'L': {
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, true},
	},
	'M': {
		{true, false, false, false, true},
		{true, true, false, true, true},
		{true, false, true, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
	},
	'N': {
		{true, false, false, false, true},
		{true, true, false, false, true},
		{true, false, true, false, true},
		{true, false, false, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
	},
	'O': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'P': {
		{true, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, false, false, false, false},
	},
	'R': {
		{true, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, false},
		{true, false, true, false, false},
		{true, false, false, true, false},
		{true, false, false, false, true},
	},
	'S': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, false},
		{false, true, true, true, false},
		{false, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'T': {
		{true, true, true, true, true},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
	},
	'U': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'V': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, false, true, false},
		{false, true, false, true, false},
		{false, false, true, false, false},
	},
	'W': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, true, false, true},
		{true, false, true, false, true},
		{true, true, false, true, true},
		{true, false, false, false, true},
	},
	'Y': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, false, true, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
	},
	' ': {
		{false, false, false, false, false},
		{false, false, false, false, false},
		{false, false, false, false, false},
		{false, false, false, false, false},
		{false, false, false, false, false},
		{false, false, false, false, false},
		{false, false, false, false, false},
	},
	'!': {
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, false, false, false},
		{false, false, true, false, false},
	},
}

// defaultCharBitmap is used for unknown characters
var defaultCharBitmap = [7][5]bool{
	{true, true, true, true, true},
	{true, false, false, false, true},
	{true, false, false, false, true},
	{true, false, false, false, true},
	{true, false, false, false, true},
	{true, false, false, false, true},
	{true, true, true, true, true},
}

// getCharBitmap returns a 5x7 bitmap for common characters
func getCharBitmap(ch rune) [7][5]bool {
	if btmp, ok := charBitmaps[ch]; ok {
		return btmp
	}
	return defaultCharBitmap
}
