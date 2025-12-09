package shared

import (
	"fmt"
	"image"
	"math"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// GridMetricData holds data for grid-based multi-value metric rendering (e.g., per-core CPU).
type GridMetricData struct {
	Values      []float64   // Current values for each cell (e.g., per-core CPU percentages)
	History     [][]float64 // Historical values per cell [cell][time] for graph mode
	ContentArea image.Rectangle
	Position    config.PositionConfig
	CoreBorder  bool
	CoreMargin  int
	BorderColor uint8
	FontFace    font.Face
	FontName    string
}

// GridMetricDisplayStrategy defines the interface for rendering grid-based metrics.
type GridMetricDisplayStrategy interface {
	Render(img *image.Gray, data GridMetricData, renderer *MetricRenderer)
}

// Grid strategy implementations are stateless singletons.
var (
	gridTextStrategy  GridMetricDisplayStrategy = &GridTextDisplayStrategy{}
	gridBarStrategy   GridMetricDisplayStrategy = &GridBarDisplayStrategy{}
	gridGraphStrategy GridMetricDisplayStrategy = &GridGraphDisplayStrategy{}
	gridGaugeStrategy GridMetricDisplayStrategy = &GridGaugeDisplayStrategy{}
)

// GetGridMetricStrategy returns the appropriate grid strategy for a display mode.
func GetGridMetricStrategy(mode DisplayMode) GridMetricDisplayStrategy {
	switch mode {
	case DisplayModeText:
		return gridTextStrategy
	case DisplayModeBar:
		return gridBarStrategy
	case DisplayModeGraph:
		return gridGraphStrategy
	case DisplayModeGauge:
		return gridGaugeStrategy
	default:
		return gridTextStrategy
	}
}

// calculateGridLayout calculates grid dimensions for a given number of cells.
// Returns cols, rows, cellWidth, cellHeight.
func calculateGridLayout(numCells, totalWidth, totalHeight, margin int) (cols, rows, cellWidth, cellHeight int) {
	if numCells == 0 {
		return 0, 0, 0, 0
	}

	// Try to make it roughly square, preferring more columns than rows
	cols = int(math.Ceil(math.Sqrt(float64(numCells))))
	rows = int(math.Ceil(float64(numCells) / float64(cols)))

	// Calculate cell dimensions with margins
	totalMarginWidth := (cols - 1) * margin
	totalMarginHeight := (rows - 1) * margin
	cellWidth = (totalWidth - totalMarginWidth) / cols
	cellHeight = (totalHeight - totalMarginHeight) / rows

	return
}

// GridTextDisplayStrategy renders per-core metrics as text in a grid.
type GridTextDisplayStrategy struct{}

func (s *GridTextDisplayStrategy) Render(img *image.Gray, data GridMetricData, _ *MetricRenderer) {
	numCells := len(data.Values)
	if numCells == 0 {
		return
	}

	cols, rows, cellWidth, cellHeight := calculateGridLayout(
		numCells, data.Position.W, data.Position.H, data.CoreMargin)

	for i, value := range data.Values {
		row := i / cols
		col := i % cols

		cellX := col * (cellWidth + data.CoreMargin)
		cellY := row * (cellHeight + data.CoreMargin)

		// Draw border if enabled
		if data.CoreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, data.BorderColor)
		}

		// Format and draw text centered in cell
		text := fmt.Sprintf("%.0f", value)
		bitmap.SmartDrawTextInRect(img, text, data.FontFace, data.FontName,
			cellX, cellY, cellWidth, cellHeight, "center", "center", 0)
	}

	_ = rows // suppress unused warning
}

// GridBarDisplayStrategy renders per-core metrics as bars in a grid.
type GridBarDisplayStrategy struct{}

func (s *GridBarDisplayStrategy) Render(img *image.Gray, data GridMetricData, renderer *MetricRenderer) {
	numCells := len(data.Values)
	if numCells == 0 {
		return
	}

	r := data.ContentArea
	barColor := renderer.Bar.Color
	border := renderer.Bar.Border || data.CoreBorder

	if renderer.Bar.Direction == config.DirectionVertical {
		coreWidth := (r.Dx() - (numCells-1)*data.CoreMargin) / numCells
		for i, value := range data.Values {
			coreX := r.Min.X + i*(coreWidth+data.CoreMargin)
			bitmap.DrawVerticalBar(img, coreX, r.Min.Y, coreWidth, r.Dy(), value, barColor, border)
		}
	} else {
		coreHeight := (r.Dy() - (numCells-1)*data.CoreMargin) / numCells
		for i, value := range data.Values {
			coreY := r.Min.Y + i*(coreHeight+data.CoreMargin)
			bitmap.DrawHorizontalBar(img, r.Min.X, coreY, r.Dx(), coreHeight, value, barColor, border)
		}
	}
}

// GridGraphDisplayStrategy renders per-core metrics as graphs in a grid.
type GridGraphDisplayStrategy struct{}

func (s *GridGraphDisplayStrategy) Render(img *image.Gray, data GridMetricData, renderer *MetricRenderer) {
	numCells := len(data.History)
	if numCells == 0 || len(data.History[0]) < 2 {
		return
	}

	r := data.ContentArea
	cols, rows, cellWidth, cellHeight := calculateGridLayout(
		numCells, r.Dx(), r.Dy(), data.CoreMargin)

	for i := 0; i < numCells; i++ {
		row := i / cols
		col := i % cols

		cellX := r.Min.X + col*(cellWidth+data.CoreMargin)
		cellY := r.Min.Y + row*(cellHeight+data.CoreMargin)

		// Draw border if enabled
		if data.CoreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, data.BorderColor)
		}

		bitmap.DrawGraph(img, cellX, cellY, cellWidth, cellHeight, data.History[i],
			renderer.Graph.HistoryLen, renderer.Graph.FillColor, renderer.Graph.LineColor)
	}

	_ = rows // suppress unused warning
}

// GridGaugeDisplayStrategy renders per-core metrics as gauges in a grid.
type GridGaugeDisplayStrategy struct{}

func (s *GridGaugeDisplayStrategy) Render(img *image.Gray, data GridMetricData, renderer *MetricRenderer) {
	numCells := len(data.Values)
	if numCells == 0 {
		return
	}

	cols, rows, cellWidth, cellHeight := calculateGridLayout(
		numCells, data.Position.W, data.Position.H, data.CoreMargin)

	for i, value := range data.Values {
		row := i / cols
		col := i % cols

		cellX := col * (cellWidth + data.CoreMargin)
		cellY := row * (cellHeight + data.CoreMargin)

		// Draw border if enabled
		if data.CoreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, data.BorderColor)
		}

		bitmap.DrawGauge(img, cellX, cellY, cellWidth, cellHeight, value,
			renderer.Gauge.ArcColor, renderer.Gauge.NeedleColor,
			renderer.Gauge.ShowTicks, renderer.Gauge.TicksColor)
	}

	_ = rows // suppress unused warning
}
