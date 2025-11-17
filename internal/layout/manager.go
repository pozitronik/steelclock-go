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
		widgetImg, err := w.Render()
		if err != nil {
			return nil, fmt.Errorf("failed to render widget %s: %w", w.Name(), err)
		}

		// Skip if widget returned nil (hidden, e.g., auto-hide)
		if widgetImg == nil {
			continue
		}

		// Get widget position
		pos := w.GetPosition()

		// Draw widget on canvas at its position
		destRect := image.Rect(pos.X, pos.Y, pos.X+pos.W, pos.Y+pos.H)
		draw.Draw(canvas, destRect, widgetImg, image.Point{}, draw.Over)
	}

	return canvas, nil
}
