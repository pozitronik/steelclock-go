package widget

import (
	"bufio"
	"image"
	"image/color"
	"os"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// DebugWidget displays a pixel pattern from a text file for debugging display output.
// File format: each line represents a pixel row, '0' = black, any other character = white.
// Lines can be any length; missing pixels are black.
type DebugWidget struct {
	*BaseWidget
	filePath string
	pattern  [][]bool // Cached pattern: true = white (255), false = black (0)
}

// NewDebugWidget creates a new debug widget
func NewDebugWidget(cfg config.WidgetConfig) (*DebugWidget, error) {
	filePath := ""
	if cfg.Debug != nil && cfg.Debug.File != "" {
		filePath = cfg.Debug.File
	}

	w := &DebugWidget{
		BaseWidget: NewBaseWidget(cfg),
		filePath:   filePath,
	}

	// Load pattern initially
	if filePath != "" {
		w.loadPattern()
	}

	return w, nil
}

// loadPattern reads the pattern file and parses it into a boolean matrix
func (w *DebugWidget) loadPattern() {
	if w.filePath == "" {
		w.pattern = nil
		return
	}

	file, err := os.Open(w.filePath)
	if err != nil {
		w.pattern = nil
		return
	}
	defer file.Close()

	var pattern [][]bool
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		row := make([]bool, len(line))
		for i, ch := range line {
			// '0' = black (false), anything else = white (true)
			row[i] = ch != '0'
		}
		pattern = append(pattern, row)
	}

	w.pattern = pattern
}

// Update reloads the pattern file on each update interval
func (w *DebugWidget) Update() error {
	w.loadPattern()
	return nil
}

// Render draws the pattern to an image
func (w *DebugWidget) Render() (image.Image, error) {
	pos := w.GetPosition()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if w.pattern == nil {
		return img, nil
	}

	// Draw the pattern
	for y, row := range w.pattern {
		if y >= pos.H {
			break
		}
		for x, pixel := range row {
			if x >= pos.W {
				break
			}
			if pixel {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
			// false = black, which is already the background (or leave as-is)
		}
	}

	return img, nil
}
