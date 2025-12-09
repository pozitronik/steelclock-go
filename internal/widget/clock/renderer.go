package clock

import (
	"image"
	"time"
)

// Renderer defines the interface for clock display mode renderers
type Renderer interface {
	// Render draws the clock to the image at the specified position
	// t is the current time to display
	// x, y are the top-left coordinates of the rendering area
	// w, h are the dimensions of the rendering area
	Render(img *image.Gray, t time.Time, x, y, w, h int) error

	// NeedsUpdate returns true if the renderer needs faster refresh
	// (e.g., during animations like segment flip effects)
	// This allows the compositor to know when to increase update frequency
	NeedsUpdate() bool
}
