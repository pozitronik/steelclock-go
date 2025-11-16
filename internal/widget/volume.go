package widget

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// VolumeWidget displays system volume level
type VolumeWidget struct {
	*BaseWidget
	displayMode       string
	fillColor         uint8
	barBorder         bool
	autoHide          bool
	autoHideTimeout   float64
	gaugeColor        uint8
	gaugeNeedleColor  uint8
	triangleFillColor uint8
	triangleBorder    bool

	mu               sync.RWMutex
	volume           float64 // 0-100
	isMuted          bool
	lastVolumeChange time.Time
	lastVolume       float64
	lastGoodVolume   float64 // Last successfully read volume (fallback for errors)
	lastGoodMuted    bool    // Last successfully read mute status
	face             font.Face
}

// NewVolumeWidget creates a new volume widget
func NewVolumeWidget(cfg config.WidgetConfig) (*VolumeWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Properties.DisplayMode
	if displayMode == "" {
		displayMode = "bar_horizontal"
	}

	fillColor := cfg.Properties.FillColor
	if fillColor == 0 {
		fillColor = 255
	}

	gaugeColor := cfg.Properties.GaugeColor
	if gaugeColor == 0 {
		gaugeColor = 200
	}

	gaugeNeedleColor := cfg.Properties.GaugeNeedleColor
	if gaugeNeedleColor == 0 {
		gaugeNeedleColor = 255
	}

	triangleFillColor := cfg.Properties.TriangleFillColor
	if triangleFillColor == 0 {
		triangleFillColor = 255
	}

	autoHideTimeout := cfg.Properties.AutoHideTimeout
	if autoHideTimeout == 0 {
		autoHideTimeout = 2.0 // 2 seconds default
	}

	// Load font for text mode
	var fontFace font.Face
	if displayMode == "text" {
		fontSize := cfg.Properties.FontSize
		if fontSize == 0 {
			fontSize = 10
		}
		face, err := bitmap.LoadFont(cfg.Properties.Font, fontSize)
		if err == nil {
			fontFace = face
		}
	}

	return &VolumeWidget{
		BaseWidget:        base,
		displayMode:       displayMode,
		fillColor:         uint8(fillColor),
		barBorder:         cfg.Properties.BarBorder,
		autoHide:          cfg.Properties.AutoHide,
		autoHideTimeout:   autoHideTimeout,
		gaugeColor:        uint8(gaugeColor),
		gaugeNeedleColor:  uint8(gaugeNeedleColor),
		triangleFillColor: uint8(triangleFillColor),
		triangleBorder:    cfg.Properties.TriangleBorder,
		lastVolumeChange:  time.Now(),
		face:              fontFace,
	}, nil
}

// Update updates the volume information
func (w *VolumeWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Get current volume
	volume, muted, err := getSystemVolume()
	if err != nil {
		// If error reading volume, use last good values
		// This prevents flickering when volume API has temporary issues
		volume = w.lastGoodVolume
		muted = w.lastGoodMuted
	} else {
		// Successful read - save as last good values
		w.lastGoodVolume = volume
		w.lastGoodMuted = muted
	}

	// Check if volume changed
	if volume != w.lastVolume || muted != w.isMuted {
		w.lastVolumeChange = time.Now()
		w.lastVolume = volume
	}

	w.volume = volume
	w.isMuted = muted

	return nil
}

// Render renders the volume widget
func (w *VolumeWidget) Render() (image.Image, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	pos := w.GetPosition()
	style := w.GetStyle()

	// Check if we should hide the widget (auto-hide mode)
	if w.autoHide {
		timeSinceChange := time.Since(w.lastVolumeChange).Seconds()
		if timeSinceChange > w.autoHideTimeout {
			// Return transparent/empty image
			img := bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))
			return img, nil
		}
	}

	// Create base image
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))

	// Draw border if enabled
	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	// Render based on display mode
	switch w.displayMode {
	case "text":
		w.renderText(img, pos, style)
	case "bar_horizontal":
		w.renderBarHorizontal(img, pos, style)
	case "bar_vertical":
		w.renderBarVertical(img, pos, style)
	case "gauge":
		w.renderGauge(img, pos, style)
	case "triangle":
		w.renderTriangle(img, pos, style)
	default:
		w.renderBarHorizontal(img, pos, style)
	}

	return img, nil
}

// renderText renders volume as text
func (w *VolumeWidget) renderText(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	if w.face == nil {
		return
	}

	text := fmt.Sprintf("%.0f%%", w.volume)
	if w.isMuted {
		text = "MUTE"
	}

	// Center the text (draws in white by default)
	bitmap.DrawAlignedText(img, text, w.face, "center", "center", 0)
}

// renderBarHorizontal renders volume as horizontal bar
func (w *VolumeWidget) renderBarHorizontal(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border {
		padding = 3
	}

	x := padding
	y := padding
	width := pos.W - (padding * 2)
	height := pos.H - (padding * 2)

	if width <= 0 || height <= 0 {
		return
	}

	// Draw bar background
	barImg := bitmap.NewGrayscaleImage(width, height, uint8(style.BackgroundColor))

	// Draw fill based on volume
	fillColor := color.Gray{Y: w.fillColor}
	fillWidth := int(float64(width) * (w.volume / 100.0))
	if fillWidth > 0 {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < fillWidth; dx++ {
				barImg.Set(dx, dy, fillColor)
			}
		}
	}

	// Draw bar border if enabled
	if w.barBorder {
		bitmap.DrawBorder(barImg, w.fillColor)
	}

	// Copy to main image
	for dy := 0; dy < height && y+dy < pos.H; dy++ {
		for dx := 0; dx < width && x+dx < pos.W; dx++ {
			img.Set(x+dx, y+dy, barImg.GrayAt(dx, dy))
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderBarVertical renders volume as vertical bar
func (w *VolumeWidget) renderBarVertical(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border {
		padding = 3
	}

	x := padding
	y := padding
	width := pos.W - (padding * 2)
	height := pos.H - (padding * 2)

	if width <= 0 || height <= 0 {
		return
	}

	// Draw bar background
	barImg := bitmap.NewGrayscaleImage(width, height, uint8(style.BackgroundColor))

	// Draw fill based on volume (from bottom)
	fillColor := color.Gray{Y: w.fillColor}
	fillHeight := int(float64(height) * (w.volume / 100.0))
	startY := height - fillHeight

	if fillHeight > 0 {
		for dy := startY; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				barImg.Set(dx, dy, fillColor)
			}
		}
	}

	// Draw bar border if enabled
	if w.barBorder {
		bitmap.DrawBorder(barImg, w.fillColor)
	}

	// Copy to main image
	for dy := 0; dy < height && y+dy < pos.H; dy++ {
		for dx := 0; dx < width && x+dx < pos.W; dx++ {
			img.Set(x+dx, y+dy, barImg.GrayAt(dx, dy))
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderGauge renders volume as an old-fashioned gauge with needle
func (w *VolumeWidget) renderGauge(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	centerX := pos.W / 2
	centerY := pos.H - 3 // Near bottom

	// Radius of the gauge arc
	radius := pos.H - 6
	if pos.W/2 < radius {
		radius = pos.W/2 - 3
	}

	if radius <= 0 {
		return
	}

	gaugeColor := color.Gray{Y: w.gaugeColor}
	needleColor := color.Gray{Y: w.gaugeNeedleColor}

	// Draw gauge arc (semicircle from 180° to 0°)
	for angle := 180.0; angle >= 0; angle -= 2.0 {
		rad := angle * math.Pi / 180.0
		x := centerX + int(float64(radius)*math.Cos(rad))
		y := centerY - int(float64(radius)*math.Sin(rad))

		if x >= 0 && x < pos.W && y >= 0 && y < pos.H {
			img.Set(x, y, gaugeColor)
		}
	}

	// Draw tick marks
	for tick := 0; tick <= 10; tick++ {
		angle := 180.0 - float64(tick)*18.0 // 0-180 degrees in 10 steps
		rad := angle * math.Pi / 180.0

		// Outer point
		x1 := centerX + int(float64(radius)*math.Cos(rad))
		y1 := centerY - int(float64(radius)*math.Sin(rad))

		// Inner point
		tickLen := 3
		if tick%5 == 0 {
			tickLen = 5 // Longer ticks at 0%, 50%, 100%
		}
		x2 := centerX + int(float64(radius-tickLen)*math.Cos(rad))
		y2 := centerY - int(float64(radius-tickLen)*math.Sin(rad))

		drawLine(img, x1, y1, x2, y2, gaugeColor)
	}

	// Draw needle based on volume
	needleAngle := 180.0 - (w.volume / 100.0 * 180.0)
	needleRad := needleAngle * math.Pi / 180.0
	needleLen := radius - 2

	needleX := centerX + int(float64(needleLen)*math.Cos(needleRad))
	needleY := centerY - int(float64(needleLen)*math.Sin(needleRad))

	drawLine(img, centerX, centerY, needleX, needleY, needleColor)

	// Draw center point
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if centerX+dx >= 0 && centerX+dx < pos.W && centerY+dy >= 0 && centerY+dy < pos.H {
				img.Set(centerX+dx, centerY+dy, needleColor)
			}
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// renderTriangle renders volume as a right-angled triangle that fills from left to right
// Right angle at bottom-right, hypotenuse from bottom-left to top-right
func (w *VolumeWidget) renderTriangle(img *image.Gray, pos config.PositionConfig, style config.StyleConfig) {
	padding := 2
	if style.Border {
		padding = 3
	}

	availWidth := pos.W - (padding * 2)
	availHeight := pos.H - (padding * 2)

	if availWidth <= 0 || availHeight <= 0 {
		return
	}

	fillColor := color.Gray{Y: w.triangleFillColor}
	borderColor := color.Gray{Y: w.triangleFillColor / 2}

	// Calculate how much of the triangle to fill based on volume (0-100%)
	fillWidth := int(float64(availWidth) * (w.volume / 100.0))

	// Draw right-angled triangle from left to right
	// Right angle at bottom-right corner
	for col := 0; col < availWidth; col++ {
		// Calculate height at this column
		// Height increases linearly from left (0) to right (availHeight)
		colHeight := ((col + 1) * availHeight) / availWidth
		if colHeight < 1 {
			colHeight = 1
		}

		// Start from bottom
		startY := padding + availHeight - colHeight
		x := padding + col

		// Determine if this column should be filled
		shouldFill := col < fillWidth

		// Draw the column from top to bottom
		for y := 0; y < colHeight; y++ {
			actualY := startY + y
			if x >= 0 && x < pos.W && actualY >= 0 && actualY < pos.H {
				if shouldFill {
					img.Set(x, actualY, fillColor)
				} else if w.triangleBorder {
					// Draw border outline for unfilled area
					if y == 0 || y == colHeight-1 {
						img.Set(x, actualY, borderColor)
					}
				}
			}
		}
	}

	// Draw mute indicator
	if w.isMuted {
		w.drawMuteIndicator(img, pos)
	}
}

// drawMuteIndicator draws a mute indicator (X or slash)
func (w *VolumeWidget) drawMuteIndicator(img *image.Gray, pos config.PositionConfig) {
	// Draw a visible mute overlay - semi-transparent effect with pattern
	// Use alternating pattern instead of solid X for better visibility
	muteColor := color.Gray{Y: 128} // Mid-gray for visibility on any background

	// Draw thicker diagonal stripes
	thickness := 2
	for i := 0; i < pos.W && i < pos.H; i++ {
		for t := -thickness; t <= thickness; t++ {
			// Diagonal \
			y1 := i + t
			if y1 >= 0 && y1 < pos.H && i >= 0 && i < pos.W {
				img.Set(i, y1, muteColor)
			}
			// Diagonal /
			y2 := pos.H - i - 1 + t
			if y2 >= 0 && y2 < pos.H && i >= 0 && i < pos.W {
				img.Set(i, y2, muteColor)
			}
		}
	}
}

// getSystemVolume returns the current system volume (0-100) and mute status
// This is a platform-specific function
func getSystemVolume() (volume float64, muted bool, err error) {
	// Get actual system volume from platform-specific implementation
	return getSystemVolumeImpl()
}

// drawLine draws a line between two points using Bresenham's algorithm
func drawLine(img *image.Gray, x0, y0, x1, y1 int, c color.Gray) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		img.Set(x0, y0, c)

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
