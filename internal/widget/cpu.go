package widget

import (
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"github.com/shirou/gopsutil/v4/cpu"
	"golang.org/x/image/font"
)

func init() {
	Register("cpu", func(cfg config.WidgetConfig) (Widget, error) {
		return NewCPUWidget(cfg)
	})
}

// CPUWidget displays CPU usage
type CPUWidget struct {
	*BaseWidget
	displayMode shared.DisplayMode
	perCore     bool
	padding     int
	coreBorder  bool
	coreMargin  int
	fillColor   int // -1 = no fill, 0-255 = fill color (used for per-core border color)
	historyLen  int

	// MetricRenderer for single-value (non-perCore) rendering
	renderer *shared.MetricRenderer

	// Separate typed fields instead of interface{} to avoid runtime type assertions
	currentUsageSingle  float64   // Aggregate CPU usage (when perCore=false)
	currentUsagePerCore []float64 // Per-core CPU usage (when perCore=true)
	// Ring buffers for graph history - O(1) push with zero allocations
	historySingle  *shared.RingBuffer[float64]   // Aggregate history (when perCore=false)
	historyPerCore *shared.RingBuffer[[]float64] // Per-core history (when perCore=true)
	hasData        bool                          // Indicates if currentUsage has been set
	coreCount      int
	fontFace       font.Face    // Kept for per-core text rendering
	fontName       string       // Kept for per-core text rendering
	mu             sync.RWMutex // Protects currentUsage and history
}

// NewCPUWidget creates a new CPU widget
func NewCPUWidget(cfg config.WidgetConfig) (*CPUWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := shared.DisplayMode(helper.GetDisplayMode("text"))
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()
	gaugeSettings := helper.GetGaugeSettings()
	perCore, coreBorder, coreMargin := helper.GetPerCoreSettings()

	// Get core count
	cores, err := cpu.Counts(true)
	if err != nil || cores == 0 {
		cores = 1
	}

	// Load font for text mode
	fontFace, err := helper.LoadFontForTextMode(string(displayMode))
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Determine bar color
	barColor := uint8(255)
	if graphSettings.FillColor >= 0 && graphSettings.FillColor <= 255 {
		barColor = uint8(graphSettings.FillColor)
	}

	// Create metric renderer for single-value mode
	renderer := shared.NewMetricRenderer(
		shared.BarConfig{
			Direction: barSettings.Direction,
			Border:    barSettings.Border,
			Color:     barColor,
		},
		shared.GraphConfig{
			FillColor:  graphSettings.FillColor,
			LineColor:  graphSettings.LineColor,
			HistoryLen: graphSettings.HistoryLen,
		},
		shared.GaugeConfig{
			ArcColor:    uint8(gaugeSettings.ArcColor),
			NeedleColor: uint8(gaugeSettings.NeedleColor),
			ShowTicks:   gaugeSettings.ShowTicks,
			TicksColor:  uint8(gaugeSettings.TicksColor),
		},
		shared.TextConfig{
			FontFace:   fontFace,
			FontName:   textSettings.FontName,
			HorizAlign: textSettings.HorizAlign,
			VertAlign:  textSettings.VertAlign,
			Padding:    padding,
		},
	)

	return &CPUWidget{
		BaseWidget:     base,
		displayMode:    displayMode,
		perCore:        perCore,
		padding:        padding,
		coreBorder:     coreBorder,
		coreMargin:     coreMargin,
		fillColor:      graphSettings.FillColor,
		historyLen:     graphSettings.HistoryLen,
		renderer:       renderer,
		historySingle:  shared.NewRingBuffer[float64](graphSettings.HistoryLen),
		historyPerCore: shared.NewRingBuffer[[]float64](graphSettings.HistoryLen),
		coreCount:      cores,
		fontFace:       fontFace,
		fontName:       textSettings.FontName,
	}, nil
}

// Update updates the CPU usage
func (w *CPUWidget) Update() error {
	if w.perCore {
		// Per-core usage
		percentages, err := cpu.Percent(100*time.Millisecond, true)
		if err != nil {
			return err
		}

		// Clamp to 0-100
		for i := range percentages {
			if percentages[i] < 0 {
				percentages[i] = 0
			}
			if percentages[i] > 100 {
				percentages[i] = 100
			}
		}

		w.mu.Lock()
		w.currentUsagePerCore = percentages
		w.hasData = true

		// Add to history (ring buffer handles capacity automatically)
		if w.displayMode == shared.DisplayModeGraph {
			w.historyPerCore.Push(percentages)
		}
		w.mu.Unlock()
	} else {
		// Aggregate usage
		percentages, err := cpu.Percent(100*time.Millisecond, false)
		if err != nil {
			return err
		}

		usage := 0.0
		if len(percentages) > 0 {
			usage = percentages[0]
		}

		// Clamp to 0-100
		if usage < 0 {
			usage = 0
		}
		if usage > 100 {
			usage = 100
		}

		w.mu.Lock()
		w.currentUsageSingle = usage
		w.hasData = true

		// Add to history (ring buffer handles capacity automatically)
		if w.displayMode == shared.DisplayModeGraph {
			w.historySingle.Push(usage)
		}
		w.mu.Unlock()
	}

	return nil
}

// Render creates an image of the CPU widget
func (w *CPUWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw border if enabled (border >= 0 means enabled with that color)
	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	// Calculate content area
	contentX := w.padding
	contentY := w.padding
	contentW := pos.W - w.padding*2
	contentH := pos.H - w.padding*2

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return img, nil
	}

	// For perCore mode, use specialized grid renderers
	if w.perCore {
		switch w.displayMode {
		case shared.DisplayModeText:
			w.renderTextGrid(img, w.currentUsagePerCore)
		case shared.DisplayModeBar:
			w.renderBarGrid(img, contentX, contentY, contentW, contentH)
		case shared.DisplayModeGraph:
			w.renderGraphGrid(img, contentX, contentY, contentW, contentH)
		case shared.DisplayModeGauge:
			w.renderGaugeGrid(img, pos)
		}
		return img, nil
	}

	// For single-value mode, use MetricRenderer
	switch w.displayMode {
	case shared.DisplayModeText:
		text := fmt.Sprintf("%.0f", w.currentUsageSingle)
		w.renderer.RenderText(img, text)
	case shared.DisplayModeBar:
		w.renderer.RenderBar(img, contentX, contentY, contentW, contentH, w.currentUsageSingle)
	case shared.DisplayModeGraph:
		w.renderer.RenderGraph(img, contentX, contentY, contentW, contentH, w.historySingle.ToSlice())
	case shared.DisplayModeGauge:
		w.renderer.RenderGauge(img, 0, 0, pos.W, pos.H, w.currentUsageSingle)
	}

	return img, nil
}

// renderTextGrid renders CPU usage for each core in a grid layout
func (w *CPUWidget) renderTextGrid(img *image.Gray, cores []float64) {
	pos := w.GetPosition()
	numCores := len(cores)
	if numCores == 0 {
		return
	}

	// Calculate grid dimensions
	// Try to make it roughly square, preferring more columns than rows
	cols := int(math.Ceil(math.Sqrt(float64(numCores))))
	rows := int(math.Ceil(float64(numCores) / float64(cols)))

	// Calculate cell dimensions with margins
	totalMarginWidth := (cols - 1) * w.coreMargin
	totalMarginHeight := (rows - 1) * w.coreMargin
	cellWidth := (pos.W - totalMarginWidth) / cols
	cellHeight := (pos.H - totalMarginHeight) / rows

	borderColor := uint8(255)
	if w.fillColor >= 0 && w.fillColor <= 255 {
		borderColor = uint8(w.fillColor)
	}

	// Draw each core value in its grid cell
	for i, usage := range cores {
		row := i / cols
		col := i % cols

		cellX := col * (cellWidth + w.coreMargin)
		cellY := row * (cellHeight + w.coreMargin)

		// Draw border if enabled
		if w.coreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, borderColor)
		}

		// Format: just the percentage value
		text := fmt.Sprintf("%.0f", usage)

		// Draw text centered in the cell using explicit coordinates
		bitmap.SmartDrawTextInRect(img, text, w.fontFace, w.fontName, cellX, cellY, cellWidth, cellHeight, "center", "center", 0)
	}
}

// renderBarGrid renders CPU usage bars for each core
func (w *CPUWidget) renderBarGrid(img *image.Gray, x, y, width, height int) {
	cores := w.currentUsagePerCore
	barColor := w.renderer.Bar.Color
	border := w.renderer.Bar.Border || w.coreBorder

	if w.renderer.Bar.Direction == "vertical" {
		coreWidth := (width - (len(cores)-1)*w.coreMargin) / len(cores)
		for i, usage := range cores {
			coreX := x + i*(coreWidth+w.coreMargin)
			bitmap.DrawVerticalBar(img, coreX, y, coreWidth, height, usage, barColor, border)
		}
	} else {
		coreHeight := (height - (len(cores)-1)*w.coreMargin) / len(cores)
		for i, usage := range cores {
			coreY := y + i*(coreHeight+w.coreMargin)
			bitmap.DrawHorizontalBar(img, x, coreY, width, coreHeight, usage, barColor, border)
		}
	}
}

// renderGraphGrid renders CPU usage graphs for each core in a grid layout
func (w *CPUWidget) renderGraphGrid(img *image.Gray, x, y, width, height int) {
	if w.historyPerCore.Len() < 2 {
		return
	}

	// Get core count from first history entry
	firstEntry := w.historyPerCore.Get(0)
	numCores := len(firstEntry)

	// Calculate grid dimensions
	cols := int(math.Ceil(math.Sqrt(float64(numCores))))
	rows := int(math.Ceil(float64(numCores) / float64(cols)))

	// Calculate cell dimensions with margins
	totalMarginWidth := (cols - 1) * w.coreMargin
	totalMarginHeight := (rows - 1) * w.coreMargin
	cellWidth := (width - totalMarginWidth) / cols
	cellHeight := (height - totalMarginHeight) / rows

	// Transpose history: convert from [time][core] to [core][time]
	historySlice := w.historyPerCore.ToSlice()
	coreHistories := make([][]float64, numCores)
	for i := 0; i < numCores; i++ {
		coreHistories[i] = make([]float64, len(historySlice))
		for t, cores := range historySlice {
			if i < len(cores) {
				coreHistories[i][t] = cores[i]
			}
		}
	}

	// Draw a graph for each core
	borderColor := w.renderer.Bar.Color
	for i := 0; i < numCores; i++ {
		row := i / cols
		col := i % cols

		cellX := x + col*(cellWidth+w.coreMargin)
		cellY := y + row*(cellHeight+w.coreMargin)

		// Draw border if enabled
		if w.coreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, borderColor)
		}

		bitmap.DrawGraph(img, cellX, cellY, cellWidth, cellHeight, coreHistories[i],
			w.renderer.Graph.HistoryLen, w.renderer.Graph.FillColor, w.renderer.Graph.LineColor)
	}
}

// renderGaugeGrid renders CPU usage gauges for each core in a grid layout
func (w *CPUWidget) renderGaugeGrid(img *image.Gray, pos config.PositionConfig) {
	cores := w.currentUsagePerCore
	numCores := len(cores)

	// Calculate grid dimensions
	cols := int(math.Ceil(math.Sqrt(float64(numCores))))
	rows := int(math.Ceil(float64(numCores) / float64(cols)))

	// Calculate cell dimensions with margins
	totalMarginWidth := (cols - 1) * w.coreMargin
	totalMarginHeight := (rows - 1) * w.coreMargin
	cellWidth := (pos.W - totalMarginWidth) / cols
	cellHeight := (pos.H - totalMarginHeight) / rows

	borderColor := w.renderer.Bar.Color

	// Draw a gauge for each core
	for i, usage := range cores {
		row := i / cols
		col := i % cols

		cellX := col * (cellWidth + w.coreMargin)
		cellY := row * (cellHeight + w.coreMargin)

		// Draw border if enabled
		if w.coreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, borderColor)
		}

		bitmap.DrawGauge(img, cellX, cellY, cellWidth, cellHeight, usage,
			w.renderer.Gauge.ArcColor, w.renderer.Gauge.NeedleColor,
			w.renderer.Gauge.ShowTicks, w.renderer.Gauge.TicksColor)
	}
}
