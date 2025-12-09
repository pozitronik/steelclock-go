package config

// Display modes for widgets
const (
	ModeText  = "text"
	ModeBar   = "bar"
	ModeGraph = "graph"
	ModeGauge = "gauge"
)

// Compound display modes (mode + direction)
const (
	ModeBarHorizontal = "bar_horizontal"
	ModeBarVertical   = "bar_vertical"
)

// Directions for bars and layouts
const (
	DirectionHorizontal = "horizontal"
	DirectionVertical   = "vertical"
)

// Horizontal alignments
const (
	AlignLeft   = "left"
	AlignCenter = "center"
	AlignRight  = "right"
)

// Vertical alignments
const (
	AlignTop    = "top"
	AlignBottom = "bottom"
	// AlignCenter is shared with horizontal alignment
)

// BlinkMode defines how blinking behaves
type BlinkMode string

const (
	// BlinkNever disables blinking
	BlinkNever BlinkMode = "never"
	// BlinkAlways blinks at a constant rate
	BlinkAlways BlinkMode = "always"
	// BlinkProgressive scales blink speed based on intensity (e.g., message count)
	BlinkProgressive BlinkMode = "progressive"
)
