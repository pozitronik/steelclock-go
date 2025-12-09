package shared

import (
	"fmt"
	"image"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// Dual strategy implementations are stateless singletons.
var (
	dualTextStrategy  DualMetricDisplayStrategy = &DualTextDisplayStrategy{}
	dualBarStrategy   DualMetricDisplayStrategy = &DualBarDisplayStrategy{}
	dualGraphStrategy DualMetricDisplayStrategy = &DualGraphDisplayStrategy{}
	dualGaugeStrategy DualMetricDisplayStrategy = &DualGaugeDisplayStrategy{}
)

// GetDualMetricStrategy returns the appropriate dual strategy for a display mode.
// Returns dualTextStrategy as fallback for unknown modes.
func GetDualMetricStrategy(mode DisplayMode) DualMetricDisplayStrategy {
	switch mode {
	case DisplayModeText:
		return dualTextStrategy
	case DisplayModeBar:
		return dualBarStrategy
	case DisplayModeGraph:
		return dualGraphStrategy
	case DisplayModeGauge:
		return dualGaugeStrategy
	default:
		return dualTextStrategy
	}
}

// DualTextDisplayStrategy renders dual metrics as formatted text.
type DualTextDisplayStrategy struct{}

// Render formats both values and draws them as text.
// If FormattedText is provided, it is used directly.
// Otherwise, TextFormat is used with PrimaryValue and SecondaryValue.
func (s *DualTextDisplayStrategy) Render(img *image.Gray, data DualMetricData, renderer *DualMetricRenderer) {
	text := data.FormattedText
	if text == "" {
		format := data.TextFormat
		if format == "" {
			format = "%.0f/%.0f"
		}
		text = fmt.Sprintf(format, data.PrimaryValue, data.SecondaryValue)
	}
	renderer.RenderText(img, text)
}

// DualBarDisplayStrategy renders dual metrics as stacked bars.
type DualBarDisplayStrategy struct{}

// Render draws dual progress bars representing both values.
func (s *DualBarDisplayStrategy) Render(img *image.Gray, data DualMetricData, renderer *DualMetricRenderer) {
	r := data.ContentArea
	renderer.RenderBar(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), data.PrimaryValue, data.SecondaryValue)
}

// DualGraphDisplayStrategy renders dual metrics as overlapping line graphs.
type DualGraphDisplayStrategy struct{}

// Render draws overlapping line graphs from historical values.
func (s *DualGraphDisplayStrategy) Render(img *image.Gray, data DualMetricData, renderer *DualMetricRenderer) {
	r := data.ContentArea
	renderer.RenderGraph(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), data.PrimaryHistory, data.SecondaryHistory)
}

// DualGaugeDisplayStrategy renders dual metrics as nested gauges.
type DualGaugeDisplayStrategy struct{}

// Render draws nested gauge dials representing both values.
// Does nothing if SupportsGauge is false.
func (s *DualGaugeDisplayStrategy) Render(img *image.Gray, data DualMetricData, renderer *DualMetricRenderer) {
	if !data.SupportsGauge {
		return
	}
	pos := config.PositionConfig{
		X: data.GaugeArea.Min.X,
		Y: data.GaugeArea.Min.Y,
		W: data.WidgetWidth,
		H: data.WidgetHeight,
	}
	renderer.RenderGauge(img, pos, data.PrimaryValue, data.SecondaryValue)
}
