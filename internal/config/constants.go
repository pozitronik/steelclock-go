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
