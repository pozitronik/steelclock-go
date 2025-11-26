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
	width         int
	height        int
	bgColor       uint8
	widgets       []widget.Widget
	sortedWidgets []widget.Widget // Pre-sorted by z-order (cached to avoid sorting every frame)
}

// NewManager creates a new layout manager
func NewManager(display config.DisplayConfig, widgets []widget.Widget) *Manager {
	// Pre-sort widgets by z-order once during initialization
	// Z-order only changes on config reload, which creates a new Manager
	sortedWidgets := make([]widget.Widget, len(widgets))
	copy(sortedWidgets, widgets)
	sort.Slice(sortedWidgets, func(i, j int) bool {
		return sortedWidgets[i].GetPosition().Z < sortedWidgets[j].GetPosition().Z
	})

	return &Manager{
		width:         display.Width,
		height:        display.Height,
		bgColor:       uint8(display.Background),
		widgets:       widgets,
		sortedWidgets: sortedWidgets,
	}
}

// Composite renders all widgets onto a single canvas
func (m *Manager) Composite() (image.Image, error) {
	// Create canvas
	canvas := bitmap.NewGrayscaleImage(m.width, m.height, m.bgColor)

	// Use pre-sorted widgets (sorted once in NewManager)
	// Render and composite each widget
	for _, w := range m.sortedWidgets {
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

		// Check if widget has transparent background (background = -1)
		transparentBg := style.Background == -1

		// Composite widget onto canvas
		if transparentBg {
			// Transparent background: only copy non-background pixels
			compositeWithTransparency(canvas, widgetImg, pos, style.Background)
		} else {
			// Opaque background: draw all pixels
			destRect := image.Rect(pos.X, pos.Y, pos.X+pos.W, pos.Y+pos.H)
			draw.Draw(canvas, destRect, widgetImg, image.Point{}, draw.Over)
		}
	}

	return canvas, nil
}

// compositeWithTransparency composites a widget image onto canvas, skipping background pixels.
// Optimized version using direct slice access instead of GrayAt/SetGray calls.
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

	// Pre-calculate bounds and clipping region
	srcBounds := grayWidget.Bounds()
	dstBounds := canvas.Bounds()

	// Calculate the visible region (intersection of widget and canvas)
	startX := pos.X
	startY := pos.Y
	endX := pos.X + srcBounds.Dx()
	endY := pos.Y + srcBounds.Dy()

	// Clip to canvas bounds
	if startX < dstBounds.Min.X {
		startX = dstBounds.Min.X
	}
	if startY < dstBounds.Min.Y {
		startY = dstBounds.Min.Y
	}
	if endX > dstBounds.Max.X {
		endX = dstBounds.Max.X
	}
	if endY > dstBounds.Max.Y {
		endY = dstBounds.Max.Y
	}

	// Nothing to draw if clipped completely
	if startX >= endX || startY >= endY {
		return
	}

	// Get direct access to underlying pixel slices
	srcPix := grayWidget.Pix
	srcStride := grayWidget.Stride
	dstPix := canvas.Pix
	dstStride := canvas.Stride

	// Source offset (widget may start at non-zero position)
	srcOffsetX := startX - pos.X - srcBounds.Min.X
	srcOffsetY := startY - pos.Y - srcBounds.Min.Y

	// Copy non-transparent pixels using direct slice access
	for y := startY; y < endY; y++ {
		srcY := srcOffsetY + (y - startY)
		srcRowStart := srcY*srcStride + srcOffsetX
		dstRowStart := (y-dstBounds.Min.Y)*dstStride + (startX - dstBounds.Min.X)

		for x := 0; x < endX-startX; x++ {
			pixelValue := srcPix[srcRowStart+x]
			if pixelValue != transparentValue {
				dstPix[dstRowStart+x] = pixelValue
			}
		}
	}
}
