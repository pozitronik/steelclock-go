package render

import (
	"image"
)

// MetricData holds data for single-value metric rendering.
// It provides all the information a display strategy needs to render.
type MetricData struct {
	Value       float64         // Current metric value (0-100 for percentages)
	History     []float64       // Historical values for graph mode
	TextFormat  string          // Format string for text mode (e.g., "%.0f" or "%.1f%%")
	ContentArea image.Rectangle // Bounds for bar/graph rendering (respects padding)
	GaugeArea   image.Rectangle // Bounds for gauge rendering (typically full widget)
}

// MetricDisplayStrategy defines the interface for rendering metrics in a specific display mode.
// Each implementation handles one display mode (text, bar, graph, gauge).
// Strategies are stateless and can be shared across widgets.
type MetricDisplayStrategy interface {
	// Render draws the metric data onto the image using the provided renderer.
	// The strategy decides which renderer method to call and how to use the data.
	Render(img *image.Gray, data MetricData, renderer *MetricRenderer)
}

// DualMetricData holds data for dual-value metric rendering (e.g., network RX/TX, disk R/W).
type DualMetricData struct {
	PrimaryValue     float64         // First value (e.g., download speed, read bytes)
	SecondaryValue   float64         // Second value (e.g., upload speed, write bytes)
	PrimaryHistory   []float64       // Historical values for primary metric
	SecondaryHistory []float64       // Historical values for secondary metric
	TextFormat       string          // Format string for text mode (e.g., "%.0f/%.0f")
	FormattedText    string          // Pre-formatted text (if non-empty, used instead of TextFormat)
	ContentArea      image.Rectangle // Bounds for bar/graph rendering
	GaugeArea        image.Rectangle // Bounds for gauge rendering
	WidgetWidth      int             // Widget width for dual gauge positioning
	WidgetHeight     int             // Widget height for dual gauge positioning
	SupportsGauge    bool            // Whether gauge mode is supported for this widget
}

// DualMetricDisplayStrategy defines the interface for rendering dual metrics.
// Used by widgets that display two related values (Network, Disk).
type DualMetricDisplayStrategy interface {
	// Render draws both metric values onto the image using the provided renderer.
	Render(img *image.Gray, data DualMetricData, renderer *DualMetricRenderer)
}
