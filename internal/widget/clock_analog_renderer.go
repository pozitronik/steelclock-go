package widget

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

// ClockAnalogRenderer renders clock in analog (clock face) mode
type ClockAnalogRenderer struct {
	config ClockAnalogConfig
}

// NewClockAnalogRenderer creates a new analog mode clock renderer
func NewClockAnalogRenderer(cfg ClockAnalogConfig) *ClockAnalogRenderer {
	return &ClockAnalogRenderer{
		config: cfg,
	}
}

// Render draws the clock as an analog clock face with hands
//
//nolint:gocyclo // Complex geometric calculations for clock face rendering
func (r *ClockAnalogRenderer) Render(img *image.Gray, t time.Time, x, y, w, h int) error {
	// Calculate maximum radius that fits within bounds
	maxRadius := h / 2
	if w/2 < maxRadius {
		maxRadius = w / 2
	}
	radius := maxRadius - r.config.Padding - 2 // Account for padding and edge margin

	if radius < 5 {
		return nil // Too small to draw
	}

	// Calculate center position based on alignment
	var centerX, centerY int

	// Horizontal alignment
	switch r.config.HorizAlign {
	case "left":
		centerX = x + radius + r.config.Padding + 2
	case "right":
		centerX = x + w - radius - r.config.Padding - 2
	default: // "center"
		centerX = x + w/2
	}

	// Vertical alignment
	switch r.config.VertAlign {
	case "top":
		centerY = y + radius + r.config.Padding + 2
	case "bottom":
		centerY = y + h - radius - r.config.Padding - 2
	default: // "center"
		centerY = y + h/2
	}

	// Draw clock face circle (if faceColor is not transparent)
	if r.config.FaceColor >= 0 {
		faceC := color.Gray{Y: uint8(r.config.FaceColor)}
		bitmap.DrawCircle(img, centerX, centerY, radius, faceC)

		// Draw hour markers if enabled
		if r.config.ShowTicks {
			for hour := 0; hour < 12; hour++ {
				angle := float64(hour) * 30.0 // 30 degrees per hour
				rad := (angle - 90.0) * math.Pi / 180.0

				// Outer point on circle
				x1 := centerX + int(float64(radius)*math.Cos(rad))
				y1 := centerY + int(float64(radius)*math.Sin(rad))

				// Inner point (tick mark length)
				tickLen := 2
				if hour%3 == 0 {
					tickLen = 4 // Longer ticks at 12, 3, 6, 9
				}
				x2 := centerX + int(float64(radius-tickLen)*math.Cos(rad))
				y2 := centerY + int(float64(radius-tickLen)*math.Sin(rad))

				bitmap.DrawLine(img, x1, y1, x2, y2, faceC)
			}
		}

		// Draw center dot
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if centerX+dx >= 0 && centerX+dx < x+w && centerY+dy >= 0 && centerY+dy < y+h {
					img.Set(centerX+dx, centerY+dy, faceC)
				}
			}
		}
	}

	// Get current time components
	hour := t.Hour() % 12
	minute := t.Minute()
	second := t.Second()

	// Calculate hand angles (in degrees, 0 = 12 o'clock, clockwise)
	// Subtract 90 to convert from 0=3 o'clock to 0=12 o'clock
	hourAngle := (float64(hour)*30.0 + float64(minute)*0.5 - 90.0) * math.Pi / 180.0
	minuteAngle := (float64(minute)*6.0 + float64(second)*0.1 - 90.0) * math.Pi / 180.0
	secondAngle := (float64(second)*6.0 - 90.0) * math.Pi / 180.0

	// Draw hour hand (short and thick) if not transparent
	if r.config.HourColor >= 0 {
		hourC := color.Gray{Y: uint8(r.config.HourColor)}
		hourLen := int(float64(radius) * 0.5)
		hourX := centerX + int(float64(hourLen)*math.Cos(hourAngle))
		hourY := centerY + int(float64(hourLen)*math.Sin(hourAngle))
		bitmap.DrawLine(img, centerX, centerY, hourX, hourY, hourC)
	}

	// Draw minute hand (medium length) if not transparent
	if r.config.MinuteColor >= 0 {
		minuteC := color.Gray{Y: uint8(r.config.MinuteColor)}
		minuteLen := int(float64(radius) * 0.75)
		minuteX := centerX + int(float64(minuteLen)*math.Cos(minuteAngle))
		minuteY := centerY + int(float64(minuteLen)*math.Sin(minuteAngle))
		bitmap.DrawLine(img, centerX, centerY, minuteX, minuteY, minuteC)
	}

	// Draw second hand (long and thin) if enabled and not transparent
	if r.config.ShowSeconds && r.config.SecondColor >= 0 {
		secondC := color.Gray{Y: uint8(r.config.SecondColor)}
		secondLen := int(float64(radius) * 0.9)
		secondX := centerX + int(float64(secondLen)*math.Cos(secondAngle))
		secondY := centerY + int(float64(secondLen)*math.Sin(secondAngle))
		bitmap.DrawLine(img, centerX, centerY, secondX, secondY, secondC)
	}

	return nil
}

// NeedsUpdate returns false as analog mode has no animations
func (r *ClockAnalogRenderer) NeedsUpdate() bool {
	return false
}
