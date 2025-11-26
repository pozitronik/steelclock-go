package widget

import (
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/shirou/gopsutil/v4/cpu"
	"golang.org/x/image/font"
)

// CPUWidget displays CPU usage
type CPUWidget struct {
	*BaseWidget
	displayMode      string
	perCore          bool
	fontSize         int
	fontName         string
	horizAlign       string
	vertAlign        string
	padding          int
	coreBorder       bool
	coreMargin       int
	barDirection     string
	barBorder        bool
	graphFilled      bool
	fillColor        uint8
	gaugeColor       uint8
	gaugeNeedleColor uint8
	gaugeShowTicks   bool
	gaugeTicksColor  uint8
	historyLen       int
	// Separate typed fields instead of interface{} to avoid runtime type assertions
	currentUsageSingle  float64     // Aggregate CPU usage (when perCore=false)
	currentUsagePerCore []float64   // Per-core CPU usage (when perCore=true)
	historySingle       []float64   // Aggregate history (when perCore=false)
	historyPerCore      [][]float64 // Per-core history (when perCore=true)
	hasData             bool        // Indicates if currentUsage has been set
	coreCount           int
	fontFace            font.Face
	mu                  sync.RWMutex // Protects currentUsage and history
}

// NewCPUWidget creates a new CPU widget
func NewCPUWidget(cfg config.WidgetConfig) (*CPUWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := helper.GetDisplayMode("text")
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()
	gaugeSettings := helper.GetGaugeSettings()
	fillColor := helper.GetFillColorForMode(displayMode)
	perCore, coreBorder, coreMargin := helper.GetPerCoreSettings()

	// Get core count
	cores, err := cpu.Counts(true)
	if err != nil || cores == 0 {
		cores = 1
	}

	// Load font for text mode
	fontFace, err := helper.LoadFontForTextMode(displayMode)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &CPUWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		perCore:          perCore,
		fontSize:         textSettings.FontSize,
		fontName:         textSettings.FontName,
		horizAlign:       textSettings.HorizAlign,
		vertAlign:        textSettings.VertAlign,
		padding:          padding,
		coreBorder:       coreBorder,
		coreMargin:       coreMargin,
		barDirection:     barSettings.Direction,
		barBorder:        barSettings.Border,
		graphFilled:      graphSettings.Filled,
		fillColor:        uint8(fillColor),
		gaugeColor:       uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor: uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:   gaugeSettings.ShowTicks,
		gaugeTicksColor:  uint8(gaugeSettings.TicksColor),
		historyLen:       graphSettings.HistoryLen,
		historySingle:    make([]float64, 0, graphSettings.HistoryLen),
		historyPerCore:   make([][]float64, 0, graphSettings.HistoryLen),
		coreCount:        cores,
		fontFace:         fontFace,
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

		// Add to history
		// FIXME: Consider using a ring buffer instead of slice append/trim.
		// Current approach causes slice growth followed by trimming, which may
		// lead to unnecessary allocations. A ring buffer would avoid this overhead.
		if w.displayMode == "graph" {
			w.historyPerCore = append(w.historyPerCore, percentages)
			if len(w.historyPerCore) > w.historyLen {
				w.historyPerCore = w.historyPerCore[1:]
			}
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

		// Add to history
		// FIXME: Consider using a ring buffer instead of slice append/trim.
		// See comment above for per-core history.
		if w.displayMode == "graph" {
			w.historySingle = append(w.historySingle, usage)
			if len(w.historySingle) > w.historyLen {
				w.historySingle = w.historySingle[1:]
			}
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

	// Render based on display mode
	switch w.displayMode {
	case "text":
		w.renderText(img)
	case "bar":
		if w.barDirection == "vertical" {
			w.renderBarVertical(img, contentX, contentY, contentW, contentH)
		} else {
			w.renderBarHorizontal(img, contentX, contentY, contentW, contentH)
		}
	case "graph":
		w.renderGraph(img, contentX, contentY, contentW, contentH)
	case "gauge":
		w.renderGauge(img, pos)
	}

	return img, nil
}

func (w *CPUWidget) renderText(img *image.Gray) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return
	}

	if w.perCore {
		w.renderTextGrid(img, w.currentUsagePerCore)
	} else {
		text := fmt.Sprintf("%.0f", w.currentUsageSingle)
		bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
	}
}

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

	// Draw each core value in its grid cell
	for i, usage := range cores {
		row := i / cols
		col := i % cols

		cellX := col * (cellWidth + w.coreMargin)
		cellY := row * (cellHeight + w.coreMargin)

		// Draw border if enabled
		if w.coreBorder {
			bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, w.fillColor)
		}

		// Format: just the percentage value
		text := fmt.Sprintf("%.0f", usage)

		// Draw text centered in the cell using explicit coordinates
		bitmap.DrawTextInRect(img, text, w.fontFace, cellX, cellY, cellWidth, cellHeight, "center", "center", 0)
	}
}

func (w *CPUWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return
	}

	if w.perCore {
		cores := w.currentUsagePerCore
		coreHeight := (height - (len(cores)-1)*w.coreMargin) / len(cores)

		for i, usage := range cores {
			coreY := y + i*(coreHeight+w.coreMargin)
			bitmap.DrawHorizontalBar(img, x, coreY, width, coreHeight, usage, w.fillColor, w.barBorder || w.coreBorder)
		}
	} else {
		bitmap.DrawHorizontalBar(img, x, y, width, height, w.currentUsageSingle, w.fillColor, w.barBorder)
	}
}

func (w *CPUWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return
	}

	if w.perCore {
		cores := w.currentUsagePerCore
		coreWidth := (width - (len(cores)-1)*w.coreMargin) / len(cores)

		for i, usage := range cores {
			coreX := x + i*(coreWidth+w.coreMargin)
			bitmap.DrawVerticalBar(img, coreX, y, coreWidth, height, usage, w.fillColor, w.barBorder || w.coreBorder)
		}
	} else {
		bitmap.DrawVerticalBar(img, x, y, width, height, w.currentUsageSingle, w.fillColor, w.barBorder)
	}
}

func (w *CPUWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.perCore {
		if len(w.historyPerCore) < 2 {
			return
		}

		// Get core count from first history entry
		firstEntry := w.historyPerCore[0]
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
		coreHistories := make([][]float64, numCores)
		for i := 0; i < numCores; i++ {
			coreHistories[i] = make([]float64, len(w.historyPerCore))
			for t, cores := range w.historyPerCore {
				if i < len(cores) {
					coreHistories[i][t] = cores[i]
				}
			}
		}

		// Draw a graph for each core
		for i := 0; i < numCores; i++ {
			row := i / cols
			col := i % cols

			cellX := x + col*(cellWidth+w.coreMargin)
			cellY := y + row*(cellHeight+w.coreMargin)

			// Draw border if enabled
			if w.coreBorder {
				bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, w.fillColor)
			}

			bitmap.DrawGraph(img, cellX, cellY, cellWidth, cellHeight, coreHistories[i], w.historyLen, w.fillColor, w.graphFilled)
		}
	} else {
		if len(w.historySingle) < 2 {
			return
		}
		bitmap.DrawGraph(img, x, y, width, height, w.historySingle, w.historyLen, w.fillColor, w.graphFilled)
	}
}

func (w *CPUWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return
	}

	if w.perCore {
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

		// Draw a gauge for each core
		for i, usage := range cores {
			row := i / cols
			col := i % cols

			cellX := col * (cellWidth + w.coreMargin)
			cellY := row * (cellHeight + w.coreMargin)

			// Draw border if enabled
			if w.coreBorder {
				bitmap.DrawRectangle(img, cellX, cellY, cellWidth, cellHeight, w.fillColor)
			}

			bitmap.DrawGauge(img, cellX, cellY, cellWidth, cellHeight, usage, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)
		}
	} else {
		bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, w.currentUsageSingle, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)
	}
}
