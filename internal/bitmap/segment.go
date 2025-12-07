package bitmap

import (
	"image"
	"image/color"
)

// SegmentStyle defines the visual style for seven-segment display segments
type SegmentStyle string

const (
	// SegmentStyleRectangle draws simple rectangular segments
	SegmentStyleRectangle SegmentStyle = "rectangle"
	// SegmentStyleHexagon draws segments with angled/pointed ends (classic LCD style)
	SegmentStyleHexagon SegmentStyle = "hexagon"
	// SegmentStyleRounded draws segments with rounded/semicircular ends
	SegmentStyleRounded SegmentStyle = "rounded"
)

// SegmentPatterns contains the bit patterns for digits 0-9
// Bit order: gfedcba (bit 0 = segment a, bit 6 = segment g)
//
//	Segment layout:
//	 aaa
//	f   b
//	 ggg
//	e   c
//	 ddd
var SegmentPatterns = [10]byte{
	0b0111111, // 0: abcdef
	0b0000110, // 1: bc
	0b1011011, // 2: abdeg
	0b1001111, // 3: abcdg
	0b1100110, // 4: bcfg
	0b1101101, // 5: acdfg
	0b1111101, // 6: acdefg
	0b0000111, // 7: abc
	0b1111111, // 8: all
	0b1101111, // 9: abcdfg
}

// DrawSegmentDigit draws a seven-segment digit at the specified position
// digit: 0-9 (values outside this range will display nothing)
// style: segment shape style (rectangle, hexagon, rounded)
// thickness: segment thickness in pixels
// onColor: color for lit segments
// offColor: color for unlit segments (use same as background to hide)
func DrawSegmentDigit(img *image.Gray, x, y, width, height int, digit int, style SegmentStyle, thickness int, onColor, offColor uint8) {
	if digit < 0 || digit > 9 {
		return
	}

	pattern := SegmentPatterns[digit]

	// Calculate middle Y position first to properly center the middle segment
	middleY := y + (height-thickness)/2

	// Calculate segment lengths
	upperVSegLen := middleY - y - thickness
	lowerVSegLen := height - thickness - (middleY - y) - thickness
	hSegLen := width - 2*thickness + 2 // +2 for 1px overlap on each side

	// Decode segment pattern
	segA := pattern&0x01 != 0
	segB := pattern&0x02 != 0
	segC := pattern&0x04 != 0
	segD := pattern&0x08 != 0
	segE := pattern&0x10 != 0
	segF := pattern&0x20 != 0
	segG := pattern&0x40 != 0

	// Draw horizontal segments (with 1px overlap into vertical segment area)
	hStartX := x + thickness - 1
	drawHSegment(img, hStartX, y, hSegLen, thickness, segA, onColor, offColor, style)
	drawHSegment(img, hStartX, middleY, hSegLen, thickness, segG, onColor, offColor, style)
	drawHSegment(img, hStartX, y+height-thickness, hSegLen, thickness, segD, onColor, offColor, style)

	// Draw vertical segments
	drawVSegment(img, x+width-thickness, y+thickness, upperVSegLen, thickness, segB, onColor, offColor, style)
	drawVSegment(img, x+width-thickness, middleY+thickness, lowerVSegLen, thickness, segC, onColor, offColor, style)
	drawVSegment(img, x, y+thickness, upperVSegLen, thickness, segF, onColor, offColor, style)
	drawVSegment(img, x, middleY+thickness, lowerVSegLen, thickness, segE, onColor, offColor, style)
}

// DrawSegmentDigitAnimated draws a seven-segment digit with animation support
// animProgress: 0.0-1.0 for fade-in effect (multiplied with onColor)
func DrawSegmentDigitAnimated(img *image.Gray, x, y, width, height int, digit int, style SegmentStyle, thickness int, onColor, offColor uint8, animProgress float64) {
	adjustedOnColor := uint8(float64(onColor) * animProgress)
	DrawSegmentDigit(img, x, y, width, height, digit, style, thickness, adjustedOnColor, offColor)
}

// drawHSegment draws a horizontal segment with the specified style
func drawHSegment(img *image.Gray, x, y, length, thickness int, on bool, onColor, offColor uint8, style SegmentStyle) {
	c := offColor
	if on {
		c = onColor
	}
	col := color.Gray{Y: c}

	switch style {
	case SegmentStyleHexagon:
		drawHSegmentHexagon(img, x, y, length, thickness, col)
	case SegmentStyleRounded:
		drawHSegmentRounded(img, x, y, length, thickness, col)
	default:
		drawHSegmentRectangle(img, x, y, length, thickness, col)
	}
}

// drawVSegment draws a vertical segment with the specified style
func drawVSegment(img *image.Gray, x, y, length, thickness int, on bool, onColor, offColor uint8, style SegmentStyle) {
	c := offColor
	if on {
		c = onColor
	}
	col := color.Gray{Y: c}

	switch style {
	case SegmentStyleHexagon:
		drawVSegmentHexagon(img, x, y, length, thickness, col)
	case SegmentStyleRounded:
		drawVSegmentRounded(img, x, y, length, thickness, col)
	default:
		drawVSegmentRectangle(img, x, y, length, thickness, col)
	}
}

// drawHSegmentRectangle draws a simple rectangular horizontal segment
func drawHSegmentRectangle(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	for dy := 0; dy < thickness; dy++ {
		for dx := 0; dx < length; dx++ {
			img.SetGray(x+dx, y+dy, col)
		}
	}
}

// drawVSegmentRectangle draws a simple rectangular vertical segment
func drawVSegmentRectangle(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	for dy := 0; dy < length; dy++ {
		for dx := 0; dx < thickness; dx++ {
			img.SetGray(x+dx, y+dy, col)
		}
	}
}

// drawHSegmentHexagon draws a horizontal segment with angled/pointed ends (classic LCD style)
// Shape: <======>
func drawHSegmentHexagon(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2

	for dy := 0; dy < thickness; dy++ {
		distFromCenter := dy
		if dy > halfThick {
			distFromCenter = thickness - 1 - dy
		}
		taper := halfThick - distFromCenter

		startX := x + taper
		endX := x + length - taper

		for dx := startX; dx < endX; dx++ {
			img.SetGray(dx, y+dy, col)
		}
	}
}

// drawVSegmentHexagon draws a vertical segment with angled/pointed ends (classic LCD style)
func drawVSegmentHexagon(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2

	for dy := 0; dy < length; dy++ {
		var taper int
		if dy < halfThick {
			taper = halfThick - dy
		} else if dy >= length-halfThick {
			taper = halfThick - (length - 1 - dy)
		} else {
			taper = 0
		}

		startX := x + taper
		endX := x + thickness - taper

		for dx := startX; dx < endX; dx++ {
			img.SetGray(dx, y+dy, col)
		}
	}
}

// drawHSegmentRounded draws a horizontal segment with rounded/semicircular ends
func drawHSegmentRounded(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2
	radiusSq := halfThick * halfThick

	for dy := 0; dy < thickness; dy++ {
		distY := dy - halfThick

		for dx := 0; dx < length; dx++ {
			draw := false

			if dx < halfThick {
				distX := dx - halfThick
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else if dx >= length-halfThick {
				distX := dx - (length - halfThick - 1)
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else {
				draw = true
			}

			if draw {
				img.SetGray(x+dx, y+dy, col)
			}
		}
	}
}

// drawVSegmentRounded draws a vertical segment with rounded/semicircular ends
func drawVSegmentRounded(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2
	radiusSq := halfThick * halfThick

	for dy := 0; dy < length; dy++ {
		for dx := 0; dx < thickness; dx++ {
			draw := false
			distX := dx - halfThick

			if dy < halfThick {
				distY := dy - halfThick
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else if dy >= length-halfThick {
				distY := dy - (length - halfThick - 1)
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else {
				draw = true
			}

			if draw {
				img.SetGray(x+dx, y+dy, col)
			}
		}
	}
}

// ColonStyle defines the visual style for the colon separator
type ColonStyle string

const (
	// ColonStyleDots draws two circular dots
	ColonStyleDots ColonStyle = "dots"
	// ColonStyleBar draws a vertical bar
	ColonStyleBar ColonStyle = "bar"
	// ColonStyleNone draws nothing
	ColonStyleNone ColonStyle = "none"
)

// DrawSegmentColon draws a colon separator for segment displays
// style: dots, bar, or none
// thickness: segment thickness (used for dot radius or bar width)
// visible: whether to draw (for blinking support)
func DrawSegmentColon(img *image.Gray, x, y, width, height int, style ColonStyle, thickness int, onColor uint8, visible bool) {
	if style == ColonStyleNone || !visible {
		return
	}

	col := color.Gray{Y: onColor}
	centerX := x + width/2
	dotY1 := y + height/3
	dotY2 := y + height*2/3

	if style == ColonStyleBar {
		// Draw vertical bar
		for dy := dotY1; dy <= dotY2; dy++ {
			for dx := -thickness / 2; dx <= thickness/2; dx++ {
				img.SetGray(centerX+dx, dy, col)
			}
		}
	} else {
		// Draw dots (default)
		DrawFilledCircle(img, centerX, dotY1, thickness, col)
		DrawFilledCircle(img, centerX, dotY2, thickness, col)
	}
}
