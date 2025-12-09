package widget

import (
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

// ClockSegmentRenderer renders clock in 7-segment display mode
// This renderer is stateful - it owns the animation state for digit transitions
type ClockSegmentRenderer struct {
	config ClockSegmentConfig

	// Animation state - owned by this renderer
	lastDigits     [8]int // Support up to 8 digits
	digitAnimStart [8]time.Time
	digitAnimating [8]bool
	mu             sync.RWMutex
}

// NewClockSegmentRenderer creates a new segment mode clock renderer
func NewClockSegmentRenderer(cfg ClockSegmentConfig) *ClockSegmentRenderer {
	return &ClockSegmentRenderer{
		config:     cfg,
		lastDigits: [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, // Initialize to invalid
	}
}

// Render draws the clock as a 7-segment display
func (r *ClockSegmentRenderer) Render(img *image.Gray, t time.Time, x, y, w, h int) error {
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()

	// Parse format to get digit sources and colon positions
	sources, colonPositions := parseSegmentFormatAdvanced(r.config.Format)

	if len(sources) == 0 {
		return nil
	}

	// Build digit list with values and colon flags
	type digitInfo struct {
		value      int
		colonAfter bool
	}
	var digitInfos []digitInfo

	// Create a set for quick colon position lookup
	colonSet := make(map[int]bool)
	for _, pos := range colonPositions {
		colonSet[pos] = true
	}

	for i, src := range sources {
		var value int
		if src.isLiteral {
			value = src.literalValue
		} else {
			// Get value from time
			var timeVal int
			switch src.timeType {
			case 'H':
				timeVal = hour
			case 'M':
				timeVal = minute
			case 'S':
				timeVal = second
			}
			if src.isFirst {
				value = timeVal / 10
			} else {
				value = timeVal % 10
			}
		}
		digitInfos = append(digitInfos, digitInfo{value: value, colonAfter: colonSet[i]})
	}

	// Count colons
	numColons := len(colonPositions)

	// Calculate digit dimensions
	padding := 1 // Default padding for segment mode
	digitH := r.config.DigitHeight
	if digitH == 0 {
		// Auto-fit: use most of available height
		digitH = h - 2*padding - 4
	}
	if digitH < 10 {
		digitH = 10
	}

	// Width is ~60% of height for 7-segment look, minus 1px for tighter fit
	digitW := digitH*6/10 - 1
	colonW := r.config.SegmentThickness * 4

	// Each colon has digitSpacing on both sides
	totalWidth := len(digitInfos)*digitW + (len(digitInfos)-1)*r.config.DigitSpacing + numColons*(colonW+r.config.DigitSpacing)
	startX := x + (w-totalWidth)/2
	startY := y + (h-digitH)/2

	// Check if flip animation is enabled
	flipEnabled := r.config.FlipStyle != "" && r.config.FlipStyle != flipStyleNone

	// Update animation state
	r.mu.Lock()
	for i, di := range digitInfos {
		if i < len(r.lastDigits) && r.lastDigits[i] != di.value {
			if r.lastDigits[i] != -1 && flipEnabled {
				r.digitAnimStart[i] = t
				r.digitAnimating[i] = true
			}
			r.lastDigits[i] = di.value
		}
	}
	r.mu.Unlock()

	// Draw digits
	xPos := startX
	for i, di := range digitInfos {
		// Check for animation
		animProgress := 1.0
		if i < len(r.digitAnimating) {
			r.mu.RLock()
			if r.digitAnimating[i] {
				elapsed := t.Sub(r.digitAnimStart[i]).Seconds()
				animProgress = elapsed / r.config.FlipSpeed
				if animProgress >= 1.0 {
					animProgress = 1.0
					r.mu.RUnlock()
					r.mu.Lock()
					r.digitAnimating[i] = false
					r.mu.Unlock()
					r.mu.RLock()
				}
			}
			r.mu.RUnlock()
		}

		r.drawSegmentDigit(img, xPos, startY, digitW, digitH, di.value, animProgress)
		xPos += digitW + r.config.DigitSpacing

		// Draw colon after this digit if needed
		if di.colonAfter {
			r.drawColon(img, xPos, startY, colonW, digitH, t)
			xPos += colonW + r.config.DigitSpacing
		}
	}

	return nil
}

// NeedsUpdate returns true if any digit is currently animating
func (r *ClockSegmentRenderer) NeedsUpdate() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, animating := range r.digitAnimating {
		if animating {
			return true
		}
	}
	return false
}

// drawSegmentDigit draws a single seven-segment digit
func (r *ClockSegmentRenderer) drawSegmentDigit(img *image.Gray, x, y, width, height int, digit int, animProgress float64) {
	style := bitmap.SegmentStyleRectangle
	switch r.config.SegmentStyle {
	case segmentStyleHexagon:
		style = bitmap.SegmentStyleHexagon
	case segmentStyleRounded:
		style = bitmap.SegmentStyleRounded
	}

	bitmap.DrawSegmentDigitAnimated(img, x, y, width, height, digit, style,
		r.config.SegmentThickness, uint8(r.config.OnColor), uint8(r.config.OffColor), animProgress)
}

// drawColon draws the colon separator between digit pairs
func (r *ClockSegmentRenderer) drawColon(img *image.Gray, x, y, width, height int, t time.Time) {
	// Determine if colon should be visible (blinking)
	visible := true
	if r.config.ColonBlink {
		visible = t.Second()%2 == 0
	}

	// Convert style string to bitmap.ColonStyle
	style := bitmap.ColonStyleDots
	switch r.config.ColonStyle {
	case colonStyleBar:
		style = bitmap.ColonStyleBar
	case colonStyleNone:
		style = bitmap.ColonStyleNone
	}

	bitmap.DrawSegmentColon(img, x, y, width, height, style, r.config.SegmentThickness, uint8(r.config.OnColor), visible)
}
