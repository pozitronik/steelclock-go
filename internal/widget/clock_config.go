package widget

import "golang.org/x/image/font"

// ClockDisplayMode represents available display modes for clock widgets
type ClockDisplayMode string

const (
	ClockModeText    ClockDisplayMode = "text"
	ClockModeAnalog  ClockDisplayMode = "analog"
	ClockModeBinary  ClockDisplayMode = "binary"
	ClockModeSegment ClockDisplayMode = "segment"
)

// ClockTextConfig holds configuration for text mode clock rendering
type ClockTextConfig struct {
	FontFace   font.Face
	FontName   string
	HorizAlign string
	VertAlign  string
	Padding    int
	Format     string // Go time format string (e.g., "15:04:05")
}

// ClockAnalogConfig holds configuration for analog clock rendering
type ClockAnalogConfig struct {
	HorizAlign  string
	VertAlign   string
	Padding     int
	ShowSeconds bool
	ShowTicks   bool
	FaceColor   int // -1 = transparent, 0-255 = color
	HourColor   int // -1 = transparent, 0-255 = color
	MinuteColor int // -1 = transparent, 0-255 = color
	SecondColor int // -1 = transparent, 0-255 = color
}

// ClockBinaryConfig holds configuration for binary clock rendering
type ClockBinaryConfig struct {
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
}

// ClockSegmentConfig holds configuration for 7-segment clock rendering
type ClockSegmentConfig struct {
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
}

// NewClockBinaryConfig creates a ClockBinaryConfig with default values
func NewClockBinaryConfig() ClockBinaryConfig {
	return ClockBinaryConfig{
		Style:      "bcd",
		Layout:     "vertical",
		Format:     "%H:%M:%S",
		ShowLabels: false,
		ShowHint:   false,
		DotSize:    4,
		DotSpacing: 2,
		DotStyle:   "circle",
		OnColor:    255,
		OffColor:   40,
	}
}

// NewClockSegmentConfig creates a ClockSegmentConfig with default values
func NewClockSegmentConfig() ClockSegmentConfig {
	return ClockSegmentConfig{
		Format:           "%H:%M:%S",
		DigitHeight:      0, // auto-fit
		SegmentThickness: 2,
		SegmentStyle:     "rectangle",
		DigitSpacing:     2,
		ColonStyle:       "dots",
		ColonBlink:       true,
		OnColor:          255,
		OffColor:         30,
		FlipStyle:        "none",
		FlipSpeed:        0.15,
	}
}
