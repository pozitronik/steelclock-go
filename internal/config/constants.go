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

// HAlign defines horizontal alignment
type HAlign string

// Horizontal alignments
const (
	AlignLeft   HAlign = "left"
	AlignCenter HAlign = "center"
	AlignRight  HAlign = "right"
)

// VAlign defines vertical alignment
type VAlign string

// Vertical alignments
const (
	AlignTop    VAlign = "top"
	AlignMiddle VAlign = "center" // same value as AlignCenter for compatibility
	AlignBottom VAlign = "bottom"
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

// ScrollMode defines the scrolling behavior
type ScrollMode string

const (
	// ScrollContinuous scrolls continuously in one direction (marquee style)
	ScrollContinuous ScrollMode = "continuous"
	// ScrollBounce bounces between ends
	ScrollBounce ScrollMode = "bounce"
	// ScrollPauseEnds pauses at the end before resetting
	ScrollPauseEnds ScrollMode = "pause_ends"
)

// ScrollDirection defines the scroll direction
type ScrollDirection string

const (
	ScrollLeft  ScrollDirection = "left"
	ScrollRight ScrollDirection = "right"
	ScrollUp    ScrollDirection = "up"
	ScrollDown  ScrollDirection = "down"
)
