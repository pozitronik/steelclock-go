package layout

import (
	"fmt"
	"image"
	"image/draw"
	"sort"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// Manager handles widget positioning and compositing
type Manager struct {
	width   int
	height  int
	bgColor uint8
	widgets []widget.Widget
}

// NewManager creates a new layout manager
func NewManager(display config.DisplayConfig, widgets []widget.Widget) *Manager {
	return &Manager{
		width:   display.Width,
		height:  display.Height,
		bgColor: uint8(display.BackgroundColor),
		widgets: widgets,
	}
}

// Composite renders all widgets onto a single canvas
func (m *Manager) Composite() (image.Image, error) {
	// Create canvas
	canvas := bitmap.NewGrayscaleImage(m.width, m.height, m.bgColor)

	// Sort widgets by z-order
	sortedWidgets := make([]widget.Widget, len(m.widgets))
	copy(sortedWidgets, m.widgets)
	sort.Slice(sortedWidgets, func(i, j int) bool {
		return sortedWidgets[i].GetPosition().ZOrder < sortedWidgets[j].GetPosition().ZOrder
	})

	// Render and composite each widget
	for _, w := range sortedWidgets {
		// Render widget
		// NOTE: We do NOT call Update() here because widgets have dedicated
		// update loops running in background goroutines (see compositor.widgetUpdateLoop).
		// Calling Update() here would create a race condition.
		widgetImg, err := w.Render()
		if err != nil {
			return nil, fmt.Errorf("failed to render widget %s: %w", w.Name(), err)
		}

		// Skip if widget returned nil (hidden, e.g., auto-hide)
		if widgetImg == nil {
			continue
		}

		// Get widget style and position
		style := w.GetStyle()
		pos := w.GetPosition()

		// Check if widget has transparent background (background_color = -1)
		transparentBg := style.BackgroundColor == -1

		// Composite widget onto canvas
		if transparentBg {
			// Transparent background: only copy non-background pixels
			compositeWithTransparency(canvas, widgetImg, pos, style.BackgroundColor)
		} else {
			// Opaque background: draw all pixels
			destRect := image.Rect(pos.X, pos.Y, pos.X+pos.W, pos.Y+pos.H)
			draw.Draw(canvas, destRect, widgetImg, image.Point{}, draw.Over)
		}
	}

	return canvas, nil
}

// compositeWithTransparency composites a widget image onto canvas, skipping background pixels
func compositeWithTransparency(canvas *image.Gray, widgetImg image.Image, pos config.PositionConfig, bgColor int) {
	// Convert widget image to Gray if needed
	grayWidget, ok := widgetImg.(*image.Gray)
	if !ok {
		return
	}

	// Determine which color value to treat as transparent
	// If bgColor is -1, we skip black (0) pixels
	transparentValue := uint8(0)
	if bgColor >= 0 && bgColor <= 255 {
		transparentValue = uint8(bgColor)
	}

	// Copy only non-transparent pixels
	bounds := grayWidget.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			pixelValue := grayWidget.GrayAt(x, y).Y

			// Skip transparent pixels
			if pixelValue == transparentValue {
				continue
			}

			// Calculate destination coordinates
			destX := pos.X + x
			destY := pos.Y + y

			// Check bounds
			canvasBounds := canvas.Bounds()
			if destX >= canvasBounds.Min.X && destX < canvasBounds.Max.X &&
				destY >= canvasBounds.Min.Y && destY < canvasBounds.Max.Y {
				canvas.SetGray(destX, destY, grayWidget.GrayAt(x, y))
			}
		}
	}
}
