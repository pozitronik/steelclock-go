package clock

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// BinaryRenderer renders clock in binary (BCD or true binary) mode
type BinaryRenderer struct {
	config BinaryConfig
}

// NewBinaryRenderer creates a new binary mode clock renderer
func NewBinaryRenderer(cfg BinaryConfig) *BinaryRenderer {
	return &BinaryRenderer{
		config: cfg,
	}
}

// Render draws the clock as binary dots (BCD or true binary style)
func (r *BinaryRenderer) Render(img *image.Gray, t time.Time, x, y, w, h int) error {
	components := parseBinaryFormat(r.config.Format)

	if r.config.Style == binaryStyleTrue {
		r.renderTrueBinaryClock(img, t, x, y, w, h, components)
	} else {
		r.renderBCDClock(img, t, x, y, w, h, components)
	}
	return nil
}

// NeedsUpdate returns false as binary mode has no animations
func (r *BinaryRenderer) NeedsUpdate() bool {
	return false
}

// renderBCDClock renders Binary-Coded Decimal clock
func (r *BinaryRenderer) renderBCDClock(img *image.Gray, t time.Time, x, y, w, h int, components binaryTimeComponents) {
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()

	// Convert to 12-hour format if enabled
	isPM := false
	if r.config.Use12h {
		hour, isPM = convert24to12(hour)
	}

	// Build list of digit pairs based on format
	var pairs []digitPair

	if components.showHours {
		pairs = append(pairs, digitPair{hour / 10, hour % 10, "H"})
	}
	if components.showMinutes {
		pairs = append(pairs, digitPair{minute / 10, minute % 10, "M"})
	}
	if components.showSeconds {
		pairs = append(pairs, digitPair{second / 10, second % 10, "S"})
	}

	if len(pairs) == 0 {
		return
	}

	dotUnit := r.config.DotSize + r.config.DotSpacing
	colonSpace := r.config.DotSize + r.config.DotSpacing

	// Calculate dimensions based on layout
	var totalWidth, totalHeight int
	numDigitCols := len(pairs) * 2
	numColons := len(pairs) - 1

	// Reserve space for labels if enabled
	labelSpace := 0
	if r.config.ShowLabels {
		labelSpace = 8 // pixels for label
	}

	// Reserve space for hint if enabled
	hintSpace := 0
	if r.config.ShowHint {
		hintSpace = 12 // pixels for decimal hint
	}

	// Reserve space for AM/PM indicator if enabled
	ampmSpace := 0
	if r.config.Use12h && r.config.ShowAmPm {
		ampmSpace = dotUnit + r.config.DotSpacing
	}

	if r.config.Layout == config.DirectionHorizontal {
		// Horizontal: bits go left to right, digits stack vertically
		totalWidth = 4*dotUnit + labelSpace + hintSpace
		totalHeight = numDigitCols*dotUnit + numColons*colonSpace/2 + ampmSpace
	} else {
		// Vertical (default): bits go top to bottom, digits go left to right
		totalWidth = numDigitCols*dotUnit + numColons*colonSpace + labelSpace + hintSpace + ampmSpace
		totalHeight = 4 * dotUnit
	}

	startX := x + (w-totalWidth)/2
	startY := y + (h-totalHeight)/2

	onColor := color.Gray{Y: uint8(r.config.OnColor)}
	offColor := color.Gray{Y: uint8(r.config.OffColor)}

	if r.config.Layout == config.DirectionHorizontal {
		r.renderBCDHorizontal(img, pairs, startX, startY, dotUnit, colonSpace, labelSpace, onColor, offColor, isPM)
	} else {
		r.renderBCDVertical(img, pairs, startX, startY, dotUnit, colonSpace, labelSpace, onColor, offColor, isPM)
	}
}

// renderBCDVertical renders BCD clock with bits stacked vertically (columns for digits)
func (r *BinaryRenderer) renderBCDVertical(img *image.Gray, pairs []digitPair, startX, startY, dotUnit, colonSpace, labelSpace int, onColor, offColor color.Gray, isPM bool) {
	xPos := startX

	// Draw labels at top if enabled
	if r.config.ShowLabels && labelSpace > 0 {
		startY += labelSpace
	}

	for pairIdx, pair := range pairs {
		digits := [2]int{pair.d1, pair.d2}

		// Draw label above pair
		if r.config.ShowLabels {
			labelX := xPos + dotUnit - 2
			labelY := startY - labelSpace + 2
			drawSmallChar(img, pair.label, labelX, labelY, onColor)
		}

		// Draw two digit columns
		for d := 0; d < 2; d++ {
			digit := digits[d]
			for row := 0; row < 4; row++ {
				bitValue := 1 << (3 - row)
				isOn := (digit & bitValue) != 0

				// Skip drawing if color is transparent
				if isOn && r.config.OnColor < 0 {
					continue
				}
				if !isOn && r.config.OffColor < 0 {
					continue
				}

				c := offColor
				if isOn {
					c = onColor
				}

				cx := xPos + r.config.DotSize/2
				cy := startY + row*dotUnit + r.config.DotSize/2
				drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
			}
			xPos += dotUnit
		}

		// Draw colon after pair (except last)
		if pairIdx < len(pairs)-1 {
			colonX := xPos + colonSpace/2
			colonY1 := startY + 1*dotUnit + r.config.DotSize/2
			colonY2 := startY + 2*dotUnit + r.config.DotSize/2
			drawDot(img, colonX, colonY1, r.config.DotSize, r.config.DotStyle, onColor)
			drawDot(img, colonX, colonY2, r.config.DotSize, r.config.DotStyle, onColor)
			xPos += colonSpace
		}
	}

	// Draw AM/PM indicator bit if enabled
	if r.config.Use12h && r.config.ShowAmPm {
		// Draw a single dot at the bottom: filled = PM, outline = AM
		cx := xPos + r.config.DotSpacing + r.config.DotSize/2
		cy := startY + 3*dotUnit + r.config.DotSize/2 // Bottom position
		c := offColor
		if isPM {
			c = onColor
		}
		drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
	}

	// Draw hint (decimal time) if enabled
	if r.config.ShowHint {
		hintX := xPos + 4
		if r.config.Use12h && r.config.ShowAmPm {
			hintX += dotUnit
		}
		for pairIdx, pair := range pairs {
			hintY := startY + pairIdx*12
			value := pair.d1*10 + pair.d2
			hintStr := fmt.Sprintf("%02d", value)
			drawSmallText(img, hintStr, hintX, hintY, onColor)
		}
	}
}

// renderBCDHorizontal renders BCD clock with bits arranged horizontally
func (r *BinaryRenderer) renderBCDHorizontal(img *image.Gray, pairs []digitPair, startX, startY, dotUnit, colonSpace, labelSpace int, onColor, offColor color.Gray, isPM bool) {
	yPos := startY

	// Adjust starting X for labels
	dotStartX := startX
	if r.config.ShowLabels && labelSpace > 0 {
		dotStartX += labelSpace
	}

	for pairIdx, pair := range pairs {
		digits := [2]int{pair.d1, pair.d2}

		// Draw label on the left if enabled
		if r.config.ShowLabels {
			labelX := startX
			labelY := yPos + dotUnit/2 - 2
			drawSmallChar(img, pair.label, labelX, labelY, onColor)
		}

		// Draw two digit rows (each digit is 4 bits horizontal)
		for d := 0; d < 2; d++ {
			digit := digits[d]
			for bit := 0; bit < 4; bit++ {
				bitValue := 1 << (3 - bit)
				isOn := (digit & bitValue) != 0

				// Skip drawing if color is transparent
				if isOn && r.config.OnColor < 0 {
					continue
				}
				if !isOn && r.config.OffColor < 0 {
					continue
				}

				c := offColor
				if isOn {
					c = onColor
				}

				cx := dotStartX + bit*dotUnit + r.config.DotSize/2
				cy := yPos + r.config.DotSize/2
				drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
			}
			yPos += dotUnit
		}

		// Draw hint on the right if enabled
		if r.config.ShowHint {
			value := pair.d1*10 + pair.d2
			hintStr := fmt.Sprintf("%02d", value)
			hintX := dotStartX + 4*dotUnit + 2
			hintY := yPos - 2*dotUnit + dotUnit/2 - 2
			drawSmallText(img, hintStr, hintX, hintY, onColor)
		}

		// Add spacing after pair (colon area, except last)
		if pairIdx < len(pairs)-1 {
			yPos += colonSpace / 2
		}
	}

	// Draw AM/PM indicator bit if enabled
	if r.config.Use12h && r.config.ShowAmPm {
		// Draw a single dot at the bottom right: filled = PM, outline = AM
		cx := dotStartX + 3*dotUnit + r.config.DotSize/2 // Right position
		cy := yPos + r.config.DotSpacing + r.config.DotSize/2
		c := offColor
		if isPM {
			c = onColor
		}
		drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
	}
}

// renderTrueBinaryClock renders true binary clock (rows for H, M, S as binary numbers)
func (r *BinaryRenderer) renderTrueBinaryClock(img *image.Gray, t time.Time, x, y, w, h int, components binaryTimeComponents) {
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()

	// Convert to 12-hour format if enabled
	isPM := false
	if r.config.Use12h {
		hour, isPM = convert24to12(hour)
	}

	// Build list of values based on format
	type binaryValue struct {
		value    int
		bits     int
		label    string
		decValue int
	}
	var values []binaryValue

	if components.showHours {
		// In 12-hour mode, hours range 1-12, which needs 4 bits
		// In 24-hour mode, hours range 0-23, which needs 5 bits
		hourBits := 5
		if r.config.Use12h {
			hourBits = 4
		}
		values = append(values, binaryValue{hour, hourBits, "H", hour})
	}
	if components.showMinutes {
		values = append(values, binaryValue{minute, 6, "M", minute})
	}
	if components.showSeconds {
		values = append(values, binaryValue{second, 6, "S", second})
	}

	if len(values) == 0 {
		return
	}

	dotUnit := r.config.DotSize + r.config.DotSpacing
	maxBits := 6

	labelSpace := 0
	if r.config.ShowLabels {
		labelSpace = 8
	}
	hintSpace := 0
	if r.config.ShowHint {
		hintSpace = 12
	}

	// Reserve space for AM/PM indicator if enabled
	ampmSpace := 0
	if r.config.Use12h && r.config.ShowAmPm {
		ampmSpace = dotUnit
	}

	var totalWidth, totalHeight int

	if r.config.Layout == config.DirectionHorizontal {
		// Horizontal: each value is a row of bits
		totalWidth = maxBits*dotUnit + labelSpace + hintSpace
		totalHeight = len(values)*dotUnit + ampmSpace
	} else {
		// Vertical: each value is a column of bits
		totalWidth = len(values)*dotUnit + labelSpace + hintSpace + ampmSpace
		totalHeight = maxBits * dotUnit
	}

	startX := x + (w-totalWidth)/2
	startY := y + (h-totalHeight)/2

	onColor := color.Gray{Y: uint8(r.config.OnColor)}
	offColor := color.Gray{Y: uint8(r.config.OffColor)}

	if r.config.Layout == config.DirectionHorizontal {
		// Each value on its own row
		dotStartX := startX
		if r.config.ShowLabels {
			dotStartX += labelSpace
		}

		for row, v := range values {
			// Draw label
			if r.config.ShowLabels {
				labelY := startY + row*dotUnit + r.config.DotSize/2 - 2
				drawSmallChar(img, v.label, startX, labelY, onColor)
			}

			// Right-align bits
			offsetX := (maxBits - v.bits) * dotUnit

			for bit := 0; bit < v.bits; bit++ {
				bitValue := 1 << (v.bits - 1 - bit)
				isOn := (v.value & bitValue) != 0

				// Skip drawing if color is transparent
				if isOn && r.config.OnColor < 0 {
					continue
				}
				if !isOn && r.config.OffColor < 0 {
					continue
				}

				c := offColor
				if isOn {
					c = onColor
				}

				cx := dotStartX + offsetX + bit*dotUnit + r.config.DotSize/2
				cy := startY + row*dotUnit + r.config.DotSize/2
				drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
			}

			// Draw hint
			if r.config.ShowHint {
				hintX := dotStartX + maxBits*dotUnit + 2
				hintY := startY + row*dotUnit + r.config.DotSize/2 - 2
				hintStr := fmt.Sprintf("%02d", v.decValue)
				drawSmallText(img, hintStr, hintX, hintY, onColor)
			}
		}

		// Draw AM/PM indicator bit if enabled (horizontal layout)
		if r.config.Use12h && r.config.ShowAmPm {
			// Draw a single dot at the bottom: filled = PM, outline = AM
			cx := dotStartX + (maxBits-1)*dotUnit + r.config.DotSize/2
			cy := startY + len(values)*dotUnit + r.config.DotSize/2
			c := offColor
			if isPM {
				c = onColor
			}
			drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
		}
	} else {
		// Vertical: each value in its own column
		dotStartX := startX
		if r.config.ShowLabels {
			dotStartX += labelSpace
		}

		for col, v := range values {
			// Draw label at top
			if r.config.ShowLabels {
				labelX := dotStartX + col*dotUnit + r.config.DotSize/2 - 2
				drawSmallChar(img, v.label, labelX, startY-labelSpace+2, onColor)
			}

			// Top-align bits
			offsetY := (maxBits - v.bits) * dotUnit

			for bit := 0; bit < v.bits; bit++ {
				bitValue := 1 << (v.bits - 1 - bit)
				isOn := (v.value & bitValue) != 0

				// Skip drawing if color is transparent
				if isOn && r.config.OnColor < 0 {
					continue
				}
				if !isOn && r.config.OffColor < 0 {
					continue
				}

				c := offColor
				if isOn {
					c = onColor
				}

				cx := dotStartX + col*dotUnit + r.config.DotSize/2
				cy := startY + offsetY + bit*dotUnit + r.config.DotSize/2
				drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
			}
		}

		// Draw hints at bottom
		if r.config.ShowHint {
			for col, v := range values {
				hintX := dotStartX + col*dotUnit
				hintY := startY + maxBits*dotUnit + 2
				hintStr := fmt.Sprintf("%02d", v.decValue)
				drawSmallText(img, hintStr, hintX, hintY, onColor)
			}
		}

		// Draw AM/PM indicator bit if enabled (vertical layout)
		if r.config.Use12h && r.config.ShowAmPm {
			// Draw a single dot to the right of the values: filled = PM, outline = AM
			cx := dotStartX + len(values)*dotUnit + r.config.DotSize/2
			cy := startY + (maxBits-1)*dotUnit + r.config.DotSize/2
			c := offColor
			if isPM {
				c = onColor
			}
			drawDot(img, cx, cy, r.config.DotSize, r.config.DotStyle, c)
		}
	}
}
