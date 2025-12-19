package clock

import (
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// DisplayMode represents available display modes for clock widgets
type DisplayMode string

const (
	ModeText    DisplayMode = "text"
	ModeAnalog  DisplayMode = "analog"
	ModeBinary  DisplayMode = "binary"
	ModeSegment DisplayMode = "segment"
)

// Binary clock styles
const (
	binaryStyleBCD  = "bcd"
	binaryStyleTrue = "true"
)

// Dot styles for binary clock
const (
	dotStyleCircle = "circle"
	dotStyleSquare = "square"
)

// Segment styles for 7-segment clock
const (
	segmentStyleRectangle = "rectangle"
	segmentStyleHexagon   = "hexagon"
	segmentStyleRounded   = "rounded"
)

// Colon styles for segment clock
const (
	colonStyleDots = "dots"
	colonStyleBar  = "bar"
	colonStyleNone = "none"
)

// AM/PM indicator styles for segment clock
const (
	ampmStyleDot  = "dot"
	ampmStyleText = "text"
)

// Flip animation styles
const (
	flipStyleNone = "none"
)

// TextConfig holds configuration for text mode clock rendering
type TextConfig struct {
	FontFace   font.Face
	FontName   string
	HorizAlign config.HAlign
	VertAlign  config.VAlign
	Padding    int
	Format     string // Go time format string (e.g., "15:04:05")
	Use12h     bool   // Use 12-hour format
	ShowAmPm   bool   // Show AM/PM text when Use12h is true
}

// AnalogConfig holds configuration for analog clock rendering
type AnalogConfig struct {
	HorizAlign  config.HAlign
	VertAlign   config.VAlign
	Padding     int
	ShowSeconds bool
	ShowTicks   bool
	FaceColor   int // -1 = transparent, 0-255 = color
	HourColor   int // -1 = transparent, 0-255 = color
	MinuteColor int // -1 = transparent, 0-255 = color
	SecondColor int // -1 = transparent, 0-255 = color
}

// BinaryConfig holds configuration for binary clock rendering
type BinaryConfig struct {
	Style      string // "bcd" or "true"
	Layout     string // "vertical" or "horizontal"
	Format     string // "%H:%M:%S" style format string
	ShowLabels bool
	ShowHint   bool
	DotSize    int
	DotSpacing int
	DotStyle   string // "circle" or "square"
	OnColor    int    // 0-255
	OffColor   int    // 0-255
	Use12h     bool   // Use 12-hour format
	ShowAmPm   bool   // Show AM/PM indicator bit (1=PM, 0=AM)
}

// SegmentConfig holds configuration for 7-segment clock rendering
type SegmentConfig struct {
	Format           string // "%H:%M:%S" style format string
	DigitHeight      int    // 0 = auto-fit
	SegmentThickness int
	SegmentStyle     string // "rectangle", "hexagon", "rounded"
	DigitSpacing     int
	ColonStyle       string // "dots", "bar", "none"
	ColonBlink       bool
	OnColor          int // 0-255
	OffColor         int // 0-255
	FlipStyle        string
	FlipSpeed        float64
	Use12h           bool   // Use 12-hour format
	ShowAmPm         bool   // Show AM/PM indicator
	AmPmStyle        string // "dot" or "text"
}

// NewBinaryConfig creates a BinaryConfig with default values
func NewBinaryConfig() BinaryConfig {
	return BinaryConfig{
		Style:      binaryStyleBCD,
		Layout:     config.DirectionVertical,
		Format:     "%H:%M:%S",
		ShowLabels: false,
		ShowHint:   false,
		DotSize:    4,
		DotSpacing: 2,
		DotStyle:   dotStyleCircle,
		OnColor:    255,
		OffColor:   40,
		Use12h:     false,
		ShowAmPm:   false,
	}
}

// NewSegmentConfig creates a SegmentConfig with default values
func NewSegmentConfig() SegmentConfig {
	return SegmentConfig{
		Format:           "%H:%M:%S",
		DigitHeight:      0, // auto-fit
		SegmentThickness: 2,
		SegmentStyle:     segmentStyleRectangle,
		DigitSpacing:     2,
		ColonStyle:       colonStyleDots,
		ColonBlink:       true,
		OnColor:          255,
		OffColor:         30,
		FlipStyle:        flipStyleNone,
		FlipSpeed:        0.15,
		Use12h:           false,
		ShowAmPm:         false,
		AmPmStyle:        ampmStyleDot,
	}
}
