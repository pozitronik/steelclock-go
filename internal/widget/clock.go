package widget

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// ClockWidget displays current time
type ClockWidget struct {
	*BaseWidget
	format      string
	fontSize    int
	fontName    string
	horizAlign  string
	vertAlign   string
	padding     int
	displayMode string
	currentTime string
	mu          sync.RWMutex // Protects currentTime field
	fontFace    font.Face
	// Analog mode settings
	showSeconds bool
	showTicks   bool
	// Colors for analog mode
	faceColor   int
	hourColor   int
	minuteColor int
	secondColor int
	// Binary mode settings
	binaryFormat string // format string for binary mode
	binaryStyle  string // "bcd" or "true"
	binaryLayout string // "vertical" or "horizontal"
	showLabels   bool
	showHint     bool
	dotSize      int
	dotSpacing   int
	dotStyle     string // "circle" or "square"
	dotOnColor   int
	dotOffColor  int
	// Segment mode settings
	segmentFormat    string // format string for segment mode
	digitHeight      int
	segmentThickness int
	segmentStyle     string // "rectangle", "hexagon", "rounded"
	digitSpacing     int
	colonStyle       string
	colonBlink       bool
	segmentOnColor   int
	segmentOffColor  int
	flipStyle        string
	flipSpeed        float64
	// Animation state for segment mode
	lastDigits     [8]int // Support up to 8 digits
	digitAnimStart [8]time.Time
	digitAnimating [8]bool
}

// NewClockWidget creates a new clock widget
func NewClockWidget(cfg config.WidgetConfig) (*ClockWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := helper.GetDisplayMode("text")
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Clock-specific: time format (defaults to 12 for clock, override helper's default of 10)
	format := "15:04:05" // Default Go time format (HH:MM:SS)
	fontSize := 12
	fontName := textSettings.FontName
	if cfg.Text != nil {
		if cfg.Text.Format != "" {
			format = convertStrftimeToGo(cfg.Text.Format)
		}
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
	}

	// Analog mode settings
	showSeconds := true
	showTicks := true
	if cfg.Analog != nil {
		showSeconds = cfg.Analog.ShowSeconds
		showTicks = cfg.Analog.ShowTicks
	}

	// Colors for analog mode (defaults to white)
	faceColor := 255
	hourColor := 255
	minuteColor := 255
	secondColor := 255
	if cfg.Analog != nil && cfg.Analog.Colors != nil {
		if cfg.Analog.Colors.Face != nil {
			faceColor = *cfg.Analog.Colors.Face
		}
		if cfg.Analog.Colors.Hour != nil {
			hourColor = *cfg.Analog.Colors.Hour
		}
		if cfg.Analog.Colors.Minute != nil {
			minuteColor = *cfg.Analog.Colors.Minute
		}
		if cfg.Analog.Colors.Second != nil {
			secondColor = *cfg.Analog.Colors.Second
		}
	}

	// Binary mode settings (defaults)
	binaryFormat := "%H:%M:%S"
	binaryStyle := "bcd"
	binaryLayout := "vertical"
	showLabels := false
	showHint := false
	dotSize := 4
	dotSpacing := 2
	dotStyle := "circle"
	dotOnColor := 255
	dotOffColor := 40
	if cfg.Binary != nil {
		if cfg.Binary.Format != "" {
			binaryFormat = cfg.Binary.Format
		}
		if cfg.Binary.Style != "" {
			binaryStyle = cfg.Binary.Style
		}
		if cfg.Binary.Layout != "" {
			binaryLayout = cfg.Binary.Layout
		}
		showLabels = cfg.Binary.ShowLabels
		showHint = cfg.Binary.ShowHint
		if cfg.Binary.DotSize > 0 {
			dotSize = cfg.Binary.DotSize
		}
		if cfg.Binary.DotSpacing >= 0 {
			dotSpacing = cfg.Binary.DotSpacing
		}
		if cfg.Binary.DotStyle != "" {
			dotStyle = cfg.Binary.DotStyle
		}
		if cfg.Binary.OnColor != nil {
			dotOnColor = *cfg.Binary.OnColor
		}
		if cfg.Binary.OffColor != nil {
			dotOffColor = *cfg.Binary.OffColor
		}
	}

	// Segment mode settings (defaults)
	segmentFormat := "%H:%M:%S"
	digitHeight := 0 // 0 = auto-fit
	segmentThickness := 2
	segmentStyle := "rectangle" // "rectangle", "hexagon", "rounded"
	digitSpacing := 2
	colonStyle := "dots"
	colonBlink := true
	segmentOnColor := 255
	segmentOffColor := 30
	flipStyle := "none" // "none" = disabled, "fade" = fade animation
	flipSpeed := 0.15
	if cfg.Segment != nil {
		if cfg.Segment.Format != "" {
			segmentFormat = cfg.Segment.Format
		}
		if cfg.Segment.DigitHeight > 0 {
			digitHeight = cfg.Segment.DigitHeight
		}
		if cfg.Segment.SegmentThickness > 0 {
			segmentThickness = cfg.Segment.SegmentThickness
		}
		if cfg.Segment.SegmentStyle != "" {
			segmentStyle = cfg.Segment.SegmentStyle
		}
		if cfg.Segment.DigitSpacing >= 0 {
			digitSpacing = cfg.Segment.DigitSpacing
		}
		if cfg.Segment.ColonStyle != "" {
			colonStyle = cfg.Segment.ColonStyle
		}
		if cfg.Segment.ColonBlink != nil {
			colonBlink = *cfg.Segment.ColonBlink
		}
		if cfg.Segment.OnColor != nil {
			segmentOnColor = *cfg.Segment.OnColor
		}
		if cfg.Segment.OffColor != nil {
			segmentOffColor = *cfg.Segment.OffColor
		}
		if cfg.Segment.Flip != nil {
			if cfg.Segment.Flip.Style != "" {
				flipStyle = cfg.Segment.Flip.Style
			}
			if cfg.Segment.Flip.Speed > 0 {
				flipSpeed = cfg.Segment.Flip.Speed
			}
		}
	}

	// Load font only for text mode
	var fontFace font.Face
	var err error
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(fontName, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &ClockWidget{
		BaseWidget:       base,
		format:           format,
		fontSize:         fontSize,
		fontName:         fontName,
		horizAlign:       textSettings.HorizAlign,
		vertAlign:        textSettings.VertAlign,
		padding:          padding,
		displayMode:      displayMode,
		fontFace:         fontFace,
		showSeconds:      showSeconds,
		showTicks:        showTicks,
		faceColor:        faceColor,
		hourColor:        hourColor,
		minuteColor:      minuteColor,
		secondColor:      secondColor,
		binaryFormat:     binaryFormat,
		binaryStyle:      binaryStyle,
		binaryLayout:     binaryLayout,
		showLabels:       showLabels,
		showHint:         showHint,
		dotSize:          dotSize,
		dotSpacing:       dotSpacing,
		dotStyle:         dotStyle,
		dotOnColor:       dotOnColor,
		dotOffColor:      dotOffColor,
		segmentFormat:    segmentFormat,
		digitHeight:      digitHeight,
		segmentThickness: segmentThickness,
		segmentStyle:     segmentStyle,
		digitSpacing:     digitSpacing,
		colonStyle:       colonStyle,
		colonBlink:       colonBlink,
		segmentOnColor:   segmentOnColor,
		segmentOffColor:  segmentOffColor,
		flipStyle:        flipStyle,
		flipSpeed:        flipSpeed,
		lastDigits:       [8]int{-1, -1, -1, -1, -1, -1, -1, -1}, // Initialize to invalid
	}, nil
}

// Update updates the current time
func (w *ClockWidget) Update() error {
	w.mu.Lock()
	w.currentTime = time.Now().Format(w.format)
	w.mu.Unlock()
	return nil
}

// Render creates an image of the clock
func (w *ClockWidget) Render() (image.Image, error) {
	// Check if time needs to be updated
	w.mu.RLock()
	isEmpty := w.currentTime == ""
	w.mu.RUnlock()

	// Update time if not set
	if isEmpty {
		if err := w.Update(); err != nil {
			return nil, fmt.Errorf("failed to update clock: %w", err)
		}
	}

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw border if enabled (border >= 0 means enabled with that color)
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	// Render based on display mode
	switch w.displayMode {
	case "analog", "clock_face":
		w.renderClockFace(img)
	case "binary":
		w.renderBinaryClock(img)
	case "segment":
		w.renderSegmentClock(img)
	default: // "text"
		// Get current time with read lock and copy to local variable
		w.mu.RLock()
		timeStr := w.currentTime
		w.mu.RUnlock()
		bitmap.SmartDrawAlignedText(img, timeStr, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}

	return img, nil
}

// renderClockFace draws an analog clock face with hour, minute, and second hands
//
//nolint:gocyclo // Complex geometric calculations for clock face rendering
func (w *ClockWidget) renderClockFace(img *image.Gray) {
	pos := w.GetPosition()

	// Calculate maximum radius that fits within widget bounds
	maxRadius := pos.H / 2
	if pos.W/2 < maxRadius {
		maxRadius = pos.W / 2
	}
	radius := maxRadius - w.padding - 2 // Account for padding and edge margin

	if radius < 5 {
		return // Too small to draw
	}

	// Calculate center position based on alignment
	var centerX, centerY int

	// Horizontal alignment
	switch w.horizAlign {
	case "left":
		centerX = radius + w.padding + 2
	case "right":
		centerX = pos.W - radius - w.padding - 2
	default: // "center"
		centerX = pos.W / 2
	}

	// Vertical alignment
	switch w.vertAlign {
	case "top":
		centerY = radius + w.padding + 2
	case "bottom":
		centerY = pos.H - radius - w.padding - 2
	default: // "center"
		centerY = pos.H / 2
	}

	// Draw clock face circle (if faceColor is not transparent)
	if w.faceColor >= 0 {
		faceC := color.Gray{Y: uint8(w.faceColor)}
		bitmap.DrawCircle(img, centerX, centerY, radius, faceC)

		// Draw hour markers if enabled
		if w.showTicks {
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
				if centerX+dx >= 0 && centerX+dx < pos.W && centerY+dy >= 0 && centerY+dy < pos.H {
					img.Set(centerX+dx, centerY+dy, faceC)
				}
			}
		}
	}

	// Get current time
	now := time.Now()
	hour := now.Hour() % 12
	minute := now.Minute()
	second := now.Second()

	// Calculate hand angles (in degrees, 0 = 12 o'clock, clockwise)
	// Subtract 90 to convert from 0=3 o'clock to 0=12 o'clock
	hourAngle := (float64(hour)*30.0 + float64(minute)*0.5 - 90.0) * math.Pi / 180.0
	minuteAngle := (float64(minute)*6.0 + float64(second)*0.1 - 90.0) * math.Pi / 180.0
	secondAngle := (float64(second)*6.0 - 90.0) * math.Pi / 180.0

	// Draw hour hand (short and thick) if not transparent
	if w.hourColor >= 0 {
		hourC := color.Gray{Y: uint8(w.hourColor)}
		hourLen := int(float64(radius) * 0.5)
		hourX := centerX + int(float64(hourLen)*math.Cos(hourAngle))
		hourY := centerY + int(float64(hourLen)*math.Sin(hourAngle))
		bitmap.DrawLine(img, centerX, centerY, hourX, hourY, hourC)
	}

	// Draw minute hand (medium length) if not transparent
	if w.minuteColor >= 0 {
		minuteC := color.Gray{Y: uint8(w.minuteColor)}
		minuteLen := int(float64(radius) * 0.75)
		minuteX := centerX + int(float64(minuteLen)*math.Cos(minuteAngle))
		minuteY := centerY + int(float64(minuteLen)*math.Sin(minuteAngle))
		bitmap.DrawLine(img, centerX, centerY, minuteX, minuteY, minuteC)
	}

	// Draw second hand (long and thin) if enabled and not transparent
	if w.showSeconds && w.secondColor >= 0 {
		secondC := color.Gray{Y: uint8(w.secondColor)}
		secondLen := int(float64(radius) * 0.9)
		secondX := centerX + int(float64(secondLen)*math.Cos(secondAngle))
		secondY := centerY + int(float64(secondLen)*math.Sin(secondAngle))
		bitmap.DrawLine(img, centerX, centerY, secondX, secondY, secondC)
	}
}

// convertStrftimeToGo converts Python strftime format to Go time format
// This is a simplified converter for common formats
func convertStrftimeToGo(strftime string) string {
	// Map common strftime patterns to Go format
	replacements := map[string]string{
		"%H:%M:%S": "15:04:05",
		"%H:%M":    "15:04",
		"%Y-%m-%d": "2006-01-02",
		"%d.%m.%Y": "02.01.2006",
		"%Y":       "2006",
		"%m":       "01",
		"%d":       "02",
		"%H":       "15",
		"%M":       "04",
		"%S":       "05",
	}

	result := strftime
	for old, goFmt := range replacements {
		if result == old {
			return goFmt
		}
	}

	// If no exact match found, try common patterns
	// For more complex formats, users should use Go format directly
	return result
}

// binaryTimeComponents holds parsed time components for binary display
type binaryTimeComponents struct {
	showHours   bool
	showMinutes bool
	showSeconds bool
	labels      []string // Labels for each component
}

// parseBinaryFormat parses format string to determine which components to show
func parseBinaryFormat(format string) binaryTimeComponents {
	result := binaryTimeComponents{}

	// Check for hour markers
	if strings.Contains(format, "%H") || strings.Contains(format, "%I") {
		result.showHours = true
		result.labels = append(result.labels, "H")
	}

	// Check for minute markers
	if strings.Contains(format, "%M") {
		result.showMinutes = true
		result.labels = append(result.labels, "M")
	}

	// Check for second markers
	if strings.Contains(format, "%S") {
		result.showSeconds = true
		result.labels = append(result.labels, "S")
	}

	// Default to full time if nothing specified
	if !result.showHours && !result.showMinutes && !result.showSeconds {
		result.showHours = true
		result.showMinutes = true
		result.showSeconds = true
		result.labels = []string{"H", "M", "S"}
	}

	return result
}

// renderBinaryClock draws a binary clock (BCD or true binary style)
func (w *ClockWidget) renderBinaryClock(img *image.Gray) {
	pos := w.GetPosition()
	now := time.Now()

	components := parseBinaryFormat(w.binaryFormat)

	if w.binaryStyle == "true" {
		w.renderTrueBinaryClock(img, now, pos, components)
	} else {
		w.renderBCDClock(img, now, pos, components)
	}
}

// digitPair holds a pair of BCD digits with label
type digitPair struct {
	d1, d2 int
	label  string
}

// renderBCDClock renders Binary-Coded Decimal clock
func (w *ClockWidget) renderBCDClock(img *image.Gray, now time.Time, pos config.PositionConfig, components binaryTimeComponents) {
	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()

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

	dotUnit := w.dotSize + w.dotSpacing
	colonSpace := w.dotSize + w.dotSpacing

	// Calculate dimensions based on layout
	var totalWidth, totalHeight int
	numDigitCols := len(pairs) * 2
	numColons := len(pairs) - 1

	// Reserve space for labels if enabled
	labelSpace := 0
	if w.showLabels {
		labelSpace = 8 // pixels for label
	}

	// Reserve space for hint if enabled
	hintSpace := 0
	if w.showHint {
		hintSpace = 12 // pixels for decimal hint
	}

	if w.binaryLayout == "horizontal" {
		// Horizontal: bits go left to right, digits stack vertically
		// 4 bits wide, N*2 digits tall (per pair)
		totalWidth = 4*dotUnit + labelSpace + hintSpace
		totalHeight = numDigitCols*dotUnit + numColons*colonSpace/2
	} else {
		// Vertical (default): bits go top to bottom, digits go left to right
		// N*2 digits wide, 4 bits tall
		totalWidth = numDigitCols*dotUnit + numColons*colonSpace + labelSpace + hintSpace
		totalHeight = 4 * dotUnit
	}

	startX := (pos.W - totalWidth) / 2
	startY := (pos.H - totalHeight) / 2

	onColor := color.Gray{Y: uint8(w.dotOnColor)}
	offColor := color.Gray{Y: uint8(w.dotOffColor)}

	if w.binaryLayout == "horizontal" {
		w.renderBCDHorizontal(img, pairs, startX, startY, dotUnit, colonSpace, labelSpace, onColor, offColor)
	} else {
		w.renderBCDVertical(img, pairs, startX, startY, dotUnit, colonSpace, labelSpace, onColor, offColor)
	}
}

// renderBCDVertical renders BCD clock with bits stacked vertically (columns for digits)
func (w *ClockWidget) renderBCDVertical(img *image.Gray, pairs []digitPair, startX, startY, dotUnit, colonSpace, labelSpace int, onColor, offColor color.Gray) {
	x := startX

	// Draw labels at top if enabled
	if w.showLabels && labelSpace > 0 {
		startY += labelSpace
	}

	for pairIdx, pair := range pairs {
		digits := [2]int{pair.d1, pair.d2}

		// Draw label above pair
		if w.showLabels {
			labelX := x + dotUnit - 2
			labelY := startY - labelSpace + 2
			w.drawSmallChar(img, pair.label, labelX, labelY, onColor)
		}

		// Draw two digit columns
		for d := 0; d < 2; d++ {
			digit := digits[d]
			for row := 0; row < 4; row++ {
				bitValue := 1 << (3 - row)
				isOn := (digit & bitValue) != 0

				c := offColor
				if isOn {
					c = onColor
				}

				cx := x + w.dotSize/2
				cy := startY + row*dotUnit + w.dotSize/2
				w.drawDot(img, cx, cy, c)
			}
			x += dotUnit
		}

		// Draw colon after pair (except last)
		if pairIdx < len(pairs)-1 {
			colonX := x + colonSpace/2
			colonY1 := startY + 1*dotUnit + w.dotSize/2
			colonY2 := startY + 2*dotUnit + w.dotSize/2
			w.drawDot(img, colonX, colonY1, onColor)
			w.drawDot(img, colonX, colonY2, onColor)
			x += colonSpace
		}
	}

	// Draw hint (decimal time) if enabled
	if w.showHint {
		hintX := x + 4
		for pairIdx, pair := range pairs {
			hintY := startY + pairIdx*12
			value := pair.d1*10 + pair.d2
			hintStr := fmt.Sprintf("%02d", value)
			w.drawSmallText(img, hintStr, hintX, hintY, onColor)
		}
	}
}

// renderBCDHorizontal renders BCD clock with bits arranged horizontally
func (w *ClockWidget) renderBCDHorizontal(img *image.Gray, pairs []digitPair, startX, startY, dotUnit, colonSpace, labelSpace int, onColor, offColor color.Gray) {
	y := startY

	// Adjust starting X for labels
	dotStartX := startX
	if w.showLabels && labelSpace > 0 {
		dotStartX += labelSpace
	}

	for pairIdx, pair := range pairs {
		digits := [2]int{pair.d1, pair.d2}

		// Draw label on the left if enabled
		if w.showLabels {
			labelX := startX
			labelY := y + dotUnit/2 - 2
			w.drawSmallChar(img, pair.label, labelX, labelY, onColor)
		}

		// Draw two digit rows (each digit is 4 bits horizontal)
		for d := 0; d < 2; d++ {
			digit := digits[d]
			for bit := 0; bit < 4; bit++ {
				bitValue := 1 << (3 - bit)
				isOn := (digit & bitValue) != 0

				c := offColor
				if isOn {
					c = onColor
				}

				cx := dotStartX + bit*dotUnit + w.dotSize/2
				cy := y + w.dotSize/2
				w.drawDot(img, cx, cy, c)
			}
			y += dotUnit
		}

		// Draw hint on the right if enabled
		if w.showHint {
			value := pair.d1*10 + pair.d2
			hintStr := fmt.Sprintf("%02d", value)
			hintX := dotStartX + 4*dotUnit + 2
			hintY := y - 2*dotUnit + dotUnit/2 - 2
			w.drawSmallText(img, hintStr, hintX, hintY, onColor)
		}

		// Add spacing after pair (colon area, except last)
		if pairIdx < len(pairs)-1 {
			y += colonSpace / 2
		}
	}
}

// renderTrueBinaryClock renders true binary clock (rows for H, M, S as binary numbers)
func (w *ClockWidget) renderTrueBinaryClock(img *image.Gray, now time.Time, pos config.PositionConfig, components binaryTimeComponents) {
	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()

	// Build list of values based on format
	type binaryValue struct {
		value    int
		bits     int
		label    string
		decValue int
	}
	var values []binaryValue

	if components.showHours {
		values = append(values, binaryValue{hour, 5, "H", hour})
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

	dotUnit := w.dotSize + w.dotSpacing
	maxBits := 6

	labelSpace := 0
	if w.showLabels {
		labelSpace = 8
	}
	hintSpace := 0
	if w.showHint {
		hintSpace = 12
	}

	var totalWidth, totalHeight int

	if w.binaryLayout == "horizontal" {
		// Horizontal: each value is a row of bits
		totalWidth = maxBits*dotUnit + labelSpace + hintSpace
		totalHeight = len(values) * dotUnit
	} else {
		// Vertical: each value is a column of bits
		totalWidth = len(values)*dotUnit + labelSpace + hintSpace
		totalHeight = maxBits * dotUnit
	}

	startX := (pos.W - totalWidth) / 2
	startY := (pos.H - totalHeight) / 2

	onColor := color.Gray{Y: uint8(w.dotOnColor)}
	offColor := color.Gray{Y: uint8(w.dotOffColor)}

	if w.binaryLayout == "horizontal" {
		// Each value on its own row
		dotStartX := startX
		if w.showLabels {
			dotStartX += labelSpace
		}

		for row, v := range values {
			// Draw label
			if w.showLabels {
				labelY := startY + row*dotUnit + w.dotSize/2 - 2
				w.drawSmallChar(img, v.label, startX, labelY, onColor)
			}

			// Right-align bits
			offsetX := (maxBits - v.bits) * dotUnit

			for bit := 0; bit < v.bits; bit++ {
				bitValue := 1 << (v.bits - 1 - bit)
				isOn := (v.value & bitValue) != 0

				c := offColor
				if isOn {
					c = onColor
				}

				cx := dotStartX + offsetX + bit*dotUnit + w.dotSize/2
				cy := startY + row*dotUnit + w.dotSize/2
				w.drawDot(img, cx, cy, c)
			}

			// Draw hint
			if w.showHint {
				hintX := dotStartX + maxBits*dotUnit + 2
				hintY := startY + row*dotUnit + w.dotSize/2 - 2
				hintStr := fmt.Sprintf("%02d", v.decValue)
				w.drawSmallText(img, hintStr, hintX, hintY, onColor)
			}
		}
	} else {
		// Vertical: each value in its own column
		dotStartX := startX
		if w.showLabels {
			dotStartX += labelSpace
		}

		for col, v := range values {
			// Draw label at top
			if w.showLabels {
				labelX := dotStartX + col*dotUnit + w.dotSize/2 - 2
				w.drawSmallChar(img, v.label, labelX, startY-labelSpace+2, onColor)
			}

			// Top-align bits
			offsetY := (maxBits - v.bits) * dotUnit

			for bit := 0; bit < v.bits; bit++ {
				bitValue := 1 << (v.bits - 1 - bit)
				isOn := (v.value & bitValue) != 0

				c := offColor
				if isOn {
					c = onColor
				}

				cx := dotStartX + col*dotUnit + w.dotSize/2
				cy := startY + offsetY + bit*dotUnit + w.dotSize/2
				w.drawDot(img, cx, cy, c)
			}
		}

		// Draw hints at bottom
		if w.showHint {
			for col, v := range values {
				hintX := dotStartX + col*dotUnit
				hintY := startY + maxBits*dotUnit + 2
				hintStr := fmt.Sprintf("%02d", v.decValue)
				w.drawSmallText(img, hintStr, hintX, hintY, onColor)
			}
		}
	}
}

// drawSmallChar draws a single small character (for labels)
func (w *ClockWidget) drawSmallChar(img *image.Gray, ch string, x, y int, c color.Gray) {
	// Simple 3x5 font for H, M, S
	patterns := map[string][]uint8{
		"H": {0b101, 0b101, 0b111, 0b101, 0b101},
		"M": {0b101, 0b111, 0b111, 0b101, 0b101},
		"S": {0b111, 0b100, 0b111, 0b001, 0b111},
	}

	pattern, ok := patterns[ch]
	if !ok {
		return
	}

	for row, bits := range pattern {
		for col := 0; col < 3; col++ {
			if bits&(1<<(2-col)) != 0 {
				img.SetGray(x+col, y+row, c)
			}
		}
	}
}

// drawSmallText draws small text for hints (2-digit numbers)
func (w *ClockWidget) drawSmallText(img *image.Gray, text string, x, y int, c color.Gray) {
	// Simple 3x5 digit patterns
	digitPatterns := map[rune][]uint8{
		'0': {0b111, 0b101, 0b101, 0b101, 0b111},
		'1': {0b010, 0b110, 0b010, 0b010, 0b111},
		'2': {0b111, 0b001, 0b111, 0b100, 0b111},
		'3': {0b111, 0b001, 0b111, 0b001, 0b111},
		'4': {0b101, 0b101, 0b111, 0b001, 0b001},
		'5': {0b111, 0b100, 0b111, 0b001, 0b111},
		'6': {0b111, 0b100, 0b111, 0b101, 0b111},
		'7': {0b111, 0b001, 0b001, 0b001, 0b001},
		'8': {0b111, 0b101, 0b111, 0b101, 0b111},
		'9': {0b111, 0b101, 0b111, 0b001, 0b111},
	}

	offsetX := 0
	for _, ch := range text {
		pattern, ok := digitPatterns[ch]
		if !ok {
			offsetX += 4
			continue
		}

		for row, bits := range pattern {
			for col := 0; col < 3; col++ {
				if bits&(1<<(2-col)) != 0 {
					img.SetGray(x+offsetX+col, y+row, c)
				}
			}
		}
		offsetX += 4 // 3 pixels + 1 spacing
	}
}

// drawDot draws a single dot (circle or square)
func (w *ClockWidget) drawDot(img *image.Gray, cx, cy int, c color.Gray) {
	if w.dotStyle == "square" {
		// Draw filled square
		half := w.dotSize / 2
		for dy := -half; dy <= half; dy++ {
			for dx := -half; dx <= half; dx++ {
				img.SetGray(cx+dx, cy+dy, c)
			}
		}
	} else {
		// Draw filled circle
		bitmap.DrawFilledCircle(img, cx, cy, w.dotSize/2, c)
	}
}

// Segment patterns for digits 0-9 (bits: gfedcba)
var segmentPatterns = [10]byte{
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

// segmentDigitSource represents a digit source for segment display
type segmentDigitSource struct {
	isLiteral    bool // true = use literalValue, false = use time component
	literalValue int  // 0-9 literal digit
	timeType     byte // 'H', 'M', 'S' for time component
	isFirst      bool // true = tens digit, false = units digit
}

// parseSegmentFormatAdvanced parses format string to build a list of digit sources
// Supports: %H (hours), %M (minutes), %S (seconds), and literal digits 0-9
// Colons and other separators are tracked for colon placement
func parseSegmentFormatAdvanced(format string) ([]segmentDigitSource, []int) {
	var digits []segmentDigitSource
	var colonPositions []int // positions after which to draw colon

	i := 0
	for i < len(format) {
		ch := format[i]

		if ch == '%' && i+1 < len(format) {
			// Format specifier
			spec := format[i+1]
			switch spec {
			case 'H', 'I':
				digits = append(digits, segmentDigitSource{timeType: 'H', isFirst: true})
				digits = append(digits, segmentDigitSource{timeType: 'H', isFirst: false})
			case 'M':
				digits = append(digits, segmentDigitSource{timeType: 'M', isFirst: true})
				digits = append(digits, segmentDigitSource{timeType: 'M', isFirst: false})
			case 'S':
				digits = append(digits, segmentDigitSource{timeType: 'S', isFirst: true})
				digits = append(digits, segmentDigitSource{timeType: 'S', isFirst: false})
			}
			i += 2
		} else if ch >= '0' && ch <= '9' {
			// Literal digit
			digits = append(digits, segmentDigitSource{isLiteral: true, literalValue: int(ch - '0')})
			i++
		} else if ch == ':' {
			// Colon - mark position after last digit
			if len(digits) > 0 {
				colonPositions = append(colonPositions, len(digits)-1)
			}
			i++
		} else {
			// Skip other characters
			i++
		}
	}

	// Default to %H:%M:%S if nothing parsed
	if len(digits) == 0 {
		digits = []segmentDigitSource{
			{timeType: 'H', isFirst: true},
			{timeType: 'H', isFirst: false},
			{timeType: 'M', isFirst: true},
			{timeType: 'M', isFirst: false},
			{timeType: 'S', isFirst: true},
			{timeType: 'S', isFirst: false},
		}
		colonPositions = []int{1, 3}
	}

	return digits, colonPositions
}

// renderSegmentClock draws a seven-segment display clock
func (w *ClockWidget) renderSegmentClock(img *image.Gray) {
	pos := w.GetPosition()
	now := time.Now()

	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()

	// Parse format to get digit sources and colon positions
	sources, colonPositions := parseSegmentFormatAdvanced(w.segmentFormat)

	if len(sources) == 0 {
		return
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
	digitH := w.digitHeight
	if digitH == 0 {
		// Auto-fit: use most of available height
		digitH = pos.H - 2*w.padding - 4
	}
	if digitH < 10 {
		digitH = 10
	}

	// Width is ~60% of height for 7-segment look, minus 1px for tighter fit
	digitW := digitH*6/10 - 1
	colonW := w.segmentThickness * 4

	// Each colon has digitSpacing on both sides (before from digit, after added explicitly)
	totalWidth := len(digitInfos)*digitW + (len(digitInfos)-1)*w.digitSpacing + numColons*(colonW+w.digitSpacing)
	startX := (pos.W - totalWidth) / 2
	startY := (pos.H - digitH) / 2

	// Check if flip animation is enabled
	flipEnabled := w.flipStyle != "" && w.flipStyle != "none"

	// Update animation state
	w.mu.Lock()
	for i, di := range digitInfos {
		if i < len(w.lastDigits) && w.lastDigits[i] != di.value {
			if w.lastDigits[i] != -1 && flipEnabled {
				w.digitAnimStart[i] = now
				w.digitAnimating[i] = true
			}
			w.lastDigits[i] = di.value
		}
	}
	w.mu.Unlock()

	// Draw digits
	x := startX
	for i, di := range digitInfos {
		// Check for animation
		animProgress := 1.0
		if i < len(w.digitAnimating) {
			w.mu.RLock()
			if w.digitAnimating[i] {
				elapsed := now.Sub(w.digitAnimStart[i]).Seconds()
				animProgress = elapsed / w.flipSpeed
				if animProgress >= 1.0 {
					animProgress = 1.0
					w.mu.RUnlock()
					w.mu.Lock()
					w.digitAnimating[i] = false
					w.mu.Unlock()
					w.mu.RLock()
				}
			}
			w.mu.RUnlock()
		}

		w.drawSegmentDigit(img, x, startY, digitW, digitH, di.value, animProgress)
		x += digitW + w.digitSpacing

		// Draw colon after this digit if needed (with equal spacing on both sides)
		if di.colonAfter {
			w.drawColon(img, x, startY, colonW, digitH, now)
			x += colonW + w.digitSpacing
		}
	}
}

// drawSegmentDigit draws a single seven-segment digit
func (w *ClockWidget) drawSegmentDigit(img *image.Gray, x, y, width, height int, digit int, animProgress float64) {
	pattern := segmentPatterns[digit]
	thickness := w.segmentThickness

	// Calculate middle Y position first to properly center the middle segment
	// Using (height-thickness)/2 instead of height/2-thickness/2 avoids integer truncation asymmetry
	middleY := y + (height-thickness)/2

	// Calculate segment lengths
	// Upper and lower vertical segments may have different lengths due to integer division
	upperVSegLen := middleY - y - thickness                        // from bottom of segment A to top of segment G
	lowerVSegLen := height - thickness - (middleY - y) - thickness // from bottom of segment G to top of segment D
	// Horizontal segments span between vertical segments (with slight overlap for corners)
	hSegLen := width - 2*thickness + 2 // +2 for 1px overlap on each side

	// Calculate colors based on animation progress
	onColor := uint8(float64(w.segmentOnColor) * animProgress)
	offColor := uint8(w.segmentOffColor)

	// Segment positions:
	//  aaa
	// f   b
	//  ggg
	// e   c
	//  ddd

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
	w.drawHSegment(img, hStartX, y, hSegLen, thickness, segA, onColor, offColor)                  // a (top)
	w.drawHSegment(img, hStartX, middleY, hSegLen, thickness, segG, onColor, offColor)            // g (middle)
	w.drawHSegment(img, hStartX, y+height-thickness, hSegLen, thickness, segD, onColor, offColor) // d (bottom)

	// Draw vertical segments (upper use upperVSegLen, lower use lowerVSegLen)
	w.drawVSegment(img, x+width-thickness, y+thickness, upperVSegLen, thickness, segB, onColor, offColor)       // b (top-right)
	w.drawVSegment(img, x+width-thickness, middleY+thickness, lowerVSegLen, thickness, segC, onColor, offColor) // c (bottom-right)
	w.drawVSegment(img, x, y+thickness, upperVSegLen, thickness, segF, onColor, offColor)                       // f (top-left)
	w.drawVSegment(img, x, middleY+thickness, lowerVSegLen, thickness, segE, onColor, offColor)                 // e (bottom-left)
}

// drawHSegment draws a horizontal segment with the configured style
func (w *ClockWidget) drawHSegment(img *image.Gray, x, y, length, thickness int, on bool, onColor, offColor uint8) {
	c := offColor
	if on {
		c = onColor
	}
	col := color.Gray{Y: c}

	switch w.segmentStyle {
	case "hexagon":
		w.drawHSegmentHexagon(img, x, y, length, thickness, col)
	case "rounded":
		w.drawHSegmentRounded(img, x, y, length, thickness, col)
	default: // "rectangle"
		w.drawHSegmentRectangle(img, x, y, length, thickness, col)
	}
}

// drawVSegment draws a vertical segment with the configured style
func (w *ClockWidget) drawVSegment(img *image.Gray, x, y, length, thickness int, on bool, onColor, offColor uint8) {
	c := offColor
	if on {
		c = onColor
	}
	col := color.Gray{Y: c}

	switch w.segmentStyle {
	case "hexagon":
		w.drawVSegmentHexagon(img, x, y, length, thickness, col)
	case "rounded":
		w.drawVSegmentRounded(img, x, y, length, thickness, col)
	default: // "rectangle"
		w.drawVSegmentRectangle(img, x, y, length, thickness, col)
	}
}

// drawHSegmentRectangle draws a simple rectangular horizontal segment
func (w *ClockWidget) drawHSegmentRectangle(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	for dy := 0; dy < thickness; dy++ {
		for dx := 0; dx < length; dx++ {
			img.SetGray(x+dx, y+dy, col)
		}
	}
}

// drawVSegmentRectangle draws a simple rectangular vertical segment
func (w *ClockWidget) drawVSegmentRectangle(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	for dy := 0; dy < length; dy++ {
		for dx := 0; dx < thickness; dx++ {
			img.SetGray(x+dx, y+dy, col)
		}
	}
}

// drawHSegmentHexagon draws a horizontal segment with angled/pointed ends (classic LCD style)
// Shape: <======>
func (w *ClockWidget) drawHSegmentHexagon(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2

	for dy := 0; dy < thickness; dy++ {
		// Calculate taper at ends based on distance from center
		distFromCenter := dy
		if dy > halfThick {
			distFromCenter = thickness - 1 - dy
		}
		// Taper amount: 0 at center row, increases toward top/bottom
		taper := halfThick - distFromCenter

		startX := x + taper
		endX := x + length - taper

		for dx := startX; dx < endX; dx++ {
			img.SetGray(dx, y+dy, col)
		}
	}
}

// drawVSegmentHexagon draws a vertical segment with angled/pointed ends (classic LCD style)
// Shape: pointed at top and bottom
func (w *ClockWidget) drawVSegmentHexagon(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2

	for dy := 0; dy < length; dy++ {
		// Calculate taper at ends based on distance from top/bottom
		var taper int
		if dy < halfThick {
			// Top taper
			taper = halfThick - dy
		} else if dy >= length-halfThick {
			// Bottom taper
			taper = halfThick - (length - 1 - dy)
		} else {
			// Middle section - full width
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
func (w *ClockWidget) drawHSegmentRounded(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2
	radiusSq := halfThick * halfThick

	for dy := 0; dy < thickness; dy++ {
		// Distance from center line
		distY := dy - halfThick

		for dx := 0; dx < length; dx++ {
			draw := false

			if dx < halfThick {
				// Left cap - check if within semicircle
				distX := dx - halfThick
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else if dx >= length-halfThick {
				// Right cap - check if within semicircle
				distX := dx - (length - halfThick - 1)
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else {
				// Middle section - always draw
				draw = true
			}

			if draw {
				img.SetGray(x+dx, y+dy, col)
			}
		}
	}
}

// drawVSegmentRounded draws a vertical segment with rounded/semicircular ends
func (w *ClockWidget) drawVSegmentRounded(img *image.Gray, x, y, length, thickness int, col color.Gray) {
	halfThick := thickness / 2
	radiusSq := halfThick * halfThick

	for dy := 0; dy < length; dy++ {
		for dx := 0; dx < thickness; dx++ {
			draw := false
			// Distance from center line
			distX := dx - halfThick

			if dy < halfThick {
				// Top cap - check if within semicircle
				distY := dy - halfThick
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else if dy >= length-halfThick {
				// Bottom cap - check if within semicircle
				distY := dy - (length - halfThick - 1)
				if distX*distX+distY*distY <= radiusSq {
					draw = true
				}
			} else {
				// Middle section - always draw
				draw = true
			}

			if draw {
				img.SetGray(x+dx, y+dy, col)
			}
		}
	}
}

// drawColon draws the colon separator between digit pairs
func (w *ClockWidget) drawColon(img *image.Gray, x, y, width, height int, now time.Time) {
	if w.colonStyle == "none" {
		return
	}

	// Determine if colon should be visible (blinking)
	visible := true
	if w.colonBlink {
		visible = now.Second()%2 == 0
	}

	if !visible {
		return
	}

	onColor := color.Gray{Y: uint8(w.segmentOnColor)}
	centerX := x + width/2
	dotY1 := y + height/3
	dotY2 := y + height*2/3

	if w.colonStyle == "bar" {
		// Draw vertical bar
		for dy := dotY1; dy <= dotY2; dy++ {
			for dx := -w.segmentThickness / 2; dx <= w.segmentThickness/2; dx++ {
				img.SetGray(centerX+dx, dy, onColor)
			}
		}
	} else {
		// Draw dots (default)
		dotRadius := w.segmentThickness
		bitmap.DrawFilledCircle(img, centerX, dotY1, dotRadius, onColor)
		bitmap.DrawFilledCircle(img, centerX, dotY2, dotRadius, onColor)
	}
}
