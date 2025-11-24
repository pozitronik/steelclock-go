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
	fillColor        uint8
	gaugeColor       uint8
	gaugeNeedleColor uint8
	historyLen       int
	currentUsage     interface{} // float64 or []float64
	history          []interface{}
	coreCount        int
	fontFace         font.Face
	mu               sync.RWMutex // Protects currentUsage and history
}

// NewCPUWidget creates a new CPU widget
func NewCPUWidget(cfg config.WidgetConfig) (*CPUWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Properties.DisplayMode
	if displayMode == "" {
		displayMode = "text"
	}

	fontSize := cfg.Properties.FontSize
	if fontSize == 0 {
		fontSize = 10
	}

	horizAlign := cfg.Properties.HorizontalAlign
	if horizAlign == "" {
		horizAlign = "center"
	}

	vertAlign := cfg.Properties.VerticalAlign
	if vertAlign == "" {
		vertAlign = "center"
	}

	fillColor := 255
	if cfg.Properties.FillColor != nil {
		fillColor = *cfg.Properties.FillColor
	}

	gaugeColor := 200
	if cfg.Properties.GaugeColor != nil {
		gaugeColor = *cfg.Properties.GaugeColor
	}

	gaugeNeedleColor := 255
	if cfg.Properties.GaugeNeedleColor != nil {
		gaugeNeedleColor = *cfg.Properties.GaugeNeedleColor
	}

	historyLen := cfg.Properties.HistoryLength
	if historyLen == 0 {
		historyLen = 30
	}

	// Get core count
	cores, err := cpu.Counts(true)
	if err != nil || cores == 0 {
		cores = 1
	}

	// Load font for text mode
	var fontFace font.Face
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(cfg.Properties.Font, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &CPUWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		perCore:          cfg.Properties.PerCore,
		fontSize:         fontSize,
		fontName:         cfg.Properties.Font,
		horizAlign:       horizAlign,
		vertAlign:        vertAlign,
		padding:          cfg.Properties.Padding,
		coreBorder:       cfg.Properties.CoreBorder,
		coreMargin:       cfg.Properties.CoreMargin,
		fillColor:        uint8(fillColor),
		gaugeColor:       uint8(gaugeColor),
		gaugeNeedleColor: uint8(gaugeNeedleColor),
		historyLen:       historyLen,
		history:          make([]interface{}, 0, historyLen),
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
		w.currentUsage = percentages

		// Add to history
		if w.displayMode == "graph" {
			w.history = append(w.history, percentages)
			if len(w.history) > w.historyLen {
				w.history = w.history[1:]
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
		w.currentUsage = usage

		// Add to history
		if w.displayMode == "graph" {
			w.history = append(w.history, usage)
			if len(w.history) > w.historyLen {
				w.history = w.history[1:]
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

	// Draw border if enabled
	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
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
	case "bar_horizontal":
		w.renderBarHorizontal(img, contentX, contentY, contentW, contentH)
	case "bar_vertical":
		w.renderBarVertical(img, contentX, contentY, contentW, contentH)
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

	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
		w.renderTextGrid(img, cores)
	} else {
		usage := w.currentUsage.(float64)
		text := fmt.Sprintf("%.0f", usage)
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

	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
		coreHeight := (height - (len(cores)-1)*w.coreMargin) / len(cores)

		for i, usage := range cores {
			coreY := y + i*(coreHeight+w.coreMargin)
			bitmap.DrawHorizontalBar(img, x, coreY, width, coreHeight, usage, w.fillColor, w.coreBorder)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawHorizontalBar(img, x, y, width, height, usage, w.fillColor, w.coreBorder)
	}
}

func (w *CPUWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
		coreWidth := (width - (len(cores)-1)*w.coreMargin) / len(cores)

		for i, usage := range cores {
			coreX := x + i*(coreWidth+w.coreMargin)
			bitmap.DrawVerticalBar(img, coreX, y, coreWidth, height, usage, w.fillColor, w.coreBorder)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawVerticalBar(img, x, y, width, height, usage, w.fillColor, w.coreBorder)
	}
}

func (w *CPUWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.history) < 2 {
		return
	}

	if w.perCore {
		// Get core count from first history entry
		firstEntry := w.history[0].([]float64)
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
			coreHistories[i] = make([]float64, len(w.history))
			for t, item := range w.history {
				cores := item.([]float64)
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

			bitmap.DrawGraph(img, cellX, cellY, cellWidth, cellHeight, coreHistories[i], w.historyLen, w.fillColor)
		}
	} else {
		// Single value history
		history := make([]float64, len(w.history))
		for i, item := range w.history {
			history[i] = item.(float64)
		}
		bitmap.DrawGraph(img, x, y, width, height, history, w.historyLen, w.fillColor)
	}
}

func (w *CPUWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
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

			bitmap.DrawGaugeAt(img, cellX, cellY, cellWidth, cellHeight, usage, w.gaugeColor, w.gaugeNeedleColor)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawGauge(img, pos, usage, w.gaugeColor, w.gaugeNeedleColor)
	}
}
