package shared

import (
	"fmt"
	"image"
)

// Strategy implementations are stateless singletons.
// They can be safely shared across all widgets.
var (
	textStrategy  MetricDisplayStrategy = &TextDisplayStrategy{}
	barStrategy   MetricDisplayStrategy = &BarDisplayStrategy{}
	graphStrategy MetricDisplayStrategy = &GraphDisplayStrategy{}
	gaugeStrategy MetricDisplayStrategy = &GaugeDisplayStrategy{}
)

// GetMetricStrategy returns the appropriate strategy for a display mode.
// Returns textStrategy as fallback for unknown modes.
func GetMetricStrategy(mode DisplayMode) MetricDisplayStrategy {
	switch mode {
	case DisplayModeText:
		return textStrategy
	case DisplayModeBar:
		return barStrategy
	case DisplayModeGraph:
		return graphStrategy
	case DisplayModeGauge:
		return gaugeStrategy
	default:
		return textStrategy
	}
}

// TextDisplayStrategy renders metrics as formatted text.
type TextDisplayStrategy struct{}

// Render formats the value and draws it as text.
func (s *TextDisplayStrategy) Render(img *image.Gray, data MetricData, renderer *MetricRenderer) {
	format := data.TextFormat
	if format == "" {
		format = "%.0f"
	}
	text := fmt.Sprintf(format, data.Value)
	renderer.RenderText(img, text)
}

// BarDisplayStrategy renders metrics as a horizontal or vertical bar.
type BarDisplayStrategy struct{}

// Render draws a progress bar representing the value.
func (s *BarDisplayStrategy) Render(img *image.Gray, data MetricData, renderer *MetricRenderer) {
	r := data.ContentArea
	renderer.RenderBar(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), data.Value)
}

// GraphDisplayStrategy renders metrics as a line graph using history.
type GraphDisplayStrategy struct{}

// Render draws a line graph from historical values.
func (s *GraphDisplayStrategy) Render(img *image.Gray, data MetricData, renderer *MetricRenderer) {
	r := data.ContentArea
	renderer.RenderGraph(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), data.History)
}

// GaugeDisplayStrategy renders metrics as an analog gauge.
type GaugeDisplayStrategy struct{}

// Render draws a gauge dial representing the value.
func (s *GaugeDisplayStrategy) Render(img *image.Gray, data MetricData, renderer *MetricRenderer) {
	r := data.GaugeArea
	renderer.RenderGauge(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), data.Value)
}
