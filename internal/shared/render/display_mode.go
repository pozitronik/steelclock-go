package render

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// DisplayMode represents available display modes for metric widgets
type DisplayMode string

const (
	DisplayModeText  = DisplayMode(config.ModeText)
	DisplayModeBar   = DisplayMode(config.ModeBar)
	DisplayModeGraph = DisplayMode(config.ModeGraph)
	DisplayModeGauge = DisplayMode(config.ModeGauge)
)

// BarConfig holds configuration for bar rendering
type BarConfig struct {
	Direction string // "horizontal" or "vertical"
	Border    bool
	Color     uint8
}

// GraphConfig holds configuration for graph rendering
type GraphConfig struct {
	FillColor  int // -1 = no fill, 0-255 = fill color
	LineColor  int // 0-255 = line color
	HistoryLen int
}

// GaugeConfig holds configuration for gauge rendering
type GaugeConfig struct {
	ArcColor    uint8
	NeedleColor uint8
	ShowTicks   bool
	TicksColor  uint8
}

// TextConfig holds configuration for text rendering
type TextConfig struct {
	FontFace   font.Face
	FontName   string
	HorizAlign config.HAlign
	VertAlign  config.VAlign
	Padding    int
}

// MetricRenderer handles rendering of single-value metrics (0-100 percentage)
type MetricRenderer struct {
	Bar   BarConfig
	Graph GraphConfig
	Gauge GaugeConfig
	Text  TextConfig
}

// NewMetricRenderer creates a new MetricRenderer with the given configuration
func NewMetricRenderer(bar BarConfig, graph GraphConfig, gauge GaugeConfig, text TextConfig) *MetricRenderer {
	return &MetricRenderer{
		Bar:   bar,
		Graph: graph,
		Gauge: gauge,
		Text:  text,
	}
}

// RenderBar renders a single-value bar (horizontal or vertical)
func (r *MetricRenderer) RenderBar(img *image.Gray, x, y, w, h int, value float64) {
	if r.Bar.Direction == config.DirectionVertical {
		bitmap.DrawVerticalBar(img, x, y, w, h, value, r.Bar.Color, r.Bar.Border)
	} else {
		bitmap.DrawHorizontalBar(img, x, y, w, h, value, r.Bar.Color, r.Bar.Border)
	}
}

// RenderGraph renders a graph from history data
func (r *MetricRenderer) RenderGraph(img *image.Gray, x, y, w, h int, history []float64) {
	bitmap.DrawGraph(img, x, y, w, h, history, r.Graph.HistoryLen, r.Graph.FillColor, r.Graph.LineColor)
}

// RenderGauge renders a gauge for a single value
func (r *MetricRenderer) RenderGauge(img *image.Gray, x, y, w, h int, value float64) {
	bitmap.DrawGauge(img, x, y, w, h, value, r.Gauge.ArcColor, r.Gauge.NeedleColor, r.Gauge.ShowTicks, r.Gauge.TicksColor)
}

// RenderText renders aligned text
func (r *MetricRenderer) RenderText(img *image.Gray, text string) {
	bitmap.SmartDrawAlignedText(img, text, r.Text.FontFace, r.Text.FontName, r.Text.HorizAlign, r.Text.VertAlign, r.Text.Padding)
}

// Render dispatches to the appropriate render method based on display mode
// For text mode, the caller should format the text and call RenderText directly
func (r *MetricRenderer) Render(img *image.Gray, mode DisplayMode, x, y, w, h int, value float64, history []float64) {
	switch mode {
	case DisplayModeBar:
		r.RenderBar(img, x, y, w, h, value)
	case DisplayModeGraph:
		r.RenderGraph(img, x, y, w, h, history)
	case DisplayModeGauge:
		r.RenderGauge(img, x, y, w, h, value)
		// Text mode requires custom formatting, so it's not handled here
	}
}

// DualBarConfig holds configuration for dual-value bar rendering
type DualBarConfig struct {
	Direction      string // "horizontal" or "vertical"
	Border         bool
	PrimaryColor   int // 0-255
	SecondaryColor int // 0-255
}

// DualGraphConfig holds configuration for dual graph rendering
type DualGraphConfig struct {
	HistoryLen    int
	PrimaryFill   int // -1 = no fill, 0-255 = fill color
	PrimaryLine   int // 0-255 = line color
	SecondaryFill int
	SecondaryLine int
}

// DualGaugeConfig holds configuration for dual gauge rendering
type DualGaugeConfig struct {
	PrimaryArcColor      uint8
	PrimaryNeedleColor   uint8
	SecondaryArcColor    uint8
	SecondaryNeedleColor uint8
}

// DualMetricRenderer handles rendering of dual-value metrics (e.g., rx/tx, read/write)
type DualMetricRenderer struct {
	Bar   DualBarConfig
	Graph DualGraphConfig
	Gauge DualGaugeConfig
	Text  TextConfig
}

// NewDualMetricRenderer creates a new DualMetricRenderer
func NewDualMetricRenderer(bar DualBarConfig, graph DualGraphConfig, gauge DualGaugeConfig, text TextConfig) *DualMetricRenderer {
	return &DualMetricRenderer{
		Bar:   bar,
		Graph: graph,
		Gauge: gauge,
		Text:  text,
	}
}

// RenderBar renders a dual-value bar
func (r *DualMetricRenderer) RenderBar(img *image.Gray, x, y, w, h int, primary, secondary float64) {
	if r.Bar.Direction == config.DirectionVertical {
		bitmap.DrawDualVerticalBar(img, x, y, w, h, primary, secondary, r.Bar.PrimaryColor, r.Bar.SecondaryColor, r.Bar.Border)
	} else {
		bitmap.DrawDualHorizontalBar(img, x, y, w, h, primary, secondary, r.Bar.PrimaryColor, r.Bar.SecondaryColor, r.Bar.Border)
	}
}

// RenderGraph renders dual overlapping graphs
func (r *DualMetricRenderer) RenderGraph(img *image.Gray, x, y, w, h int, primaryHistory, secondaryHistory []float64) {
	bitmap.DrawDualGraph(img, x, y, w, h, primaryHistory, secondaryHistory, r.Graph.HistoryLen,
		r.Graph.PrimaryFill, r.Graph.PrimaryLine, r.Graph.SecondaryFill, r.Graph.SecondaryLine)
}

// RenderGauge renders dual gauges (outer and inner)
func (r *DualMetricRenderer) RenderGauge(img *image.Gray, pos config.PositionConfig, primary, secondary float64) {
	bitmap.DrawDualGauge(img, pos, primary, secondary,
		r.Gauge.PrimaryArcColor, r.Gauge.PrimaryNeedleColor,
		r.Gauge.SecondaryArcColor, r.Gauge.SecondaryNeedleColor)
}

// RenderText renders aligned text
func (r *DualMetricRenderer) RenderText(img *image.Gray, text string) {
	bitmap.SmartDrawAlignedText(img, text, r.Text.FontFace, r.Text.FontName, r.Text.HorizAlign, r.Text.VertAlign, r.Text.Padding)
}
