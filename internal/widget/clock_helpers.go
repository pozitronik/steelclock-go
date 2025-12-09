package widget

import (
	"image"
	"image/color"
	"strings"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

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

// digitPair holds a pair of BCD digits with label
type digitPair struct {
	d1, d2 int
	label  string
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

// Small character patterns for binary clock labels (3x5 font)
var smallCharPatterns = map[string][]uint8{
	"H": {0b101, 0b101, 0b111, 0b101, 0b101},
	"M": {0b101, 0b111, 0b111, 0b101, 0b101},
	"S": {0b111, 0b100, 0b111, 0b001, 0b111},
}

// Small digit patterns for binary clock hints (3x5 font)
var smallDigitPatterns = map[rune][]uint8{
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

// drawSmallChar draws a single small character (for labels)
func drawSmallChar(img *image.Gray, ch string, x, y int, c color.Gray) {
	pattern, ok := smallCharPatterns[ch]
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
func drawSmallText(img *image.Gray, text string, x, y int, c color.Gray) {
	offsetX := 0
	for _, ch := range text {
		pattern, ok := smallDigitPatterns[ch]
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

// drawDot draws a single dot (circle or square) for binary clock
func drawDot(img *image.Gray, cx, cy int, dotSize int, dotStyle string, c color.Gray) {
	if dotStyle == dotStyleSquare {
		// Draw filled square
		half := dotSize / 2
		for dy := -half; dy <= half; dy++ {
			for dx := -half; dx <= half; dx++ {
				img.SetGray(cx+dx, cy+dy, c)
			}
		}
	} else {
		// Draw filled circle
		bitmap.DrawFilledCircle(img, cx, cy, dotSize/2, c)
	}
}
