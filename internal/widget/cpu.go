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
	currentUsage     interface{} // float64 or []float64
	history          []interface{}
	coreCount        int
	fontFace         font.Face
	mu               sync.RWMutex // Protects currentUsage and history
}

// NewCPUWidget creates a new CPU widget
func NewCPUWidget(cfg config.WidgetConfig) (*CPUWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Mode
	if displayMode == "" {
		displayMode = "text"
	}

	// Extract text settings
	fontSize := 10
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
		fontName = cfg.Text.Font
		if cfg.Text.Align != nil {
			if cfg.Text.Align.H != "" {
				horizAlign = cfg.Text.Align.H
			}
			if cfg.Text.Align.V != "" {
				vertAlign = cfg.Text.Align.V
			}
		}
	}

	// Extract padding from style
	if cfg.Style != nil {
		padding = cfg.Style.Padding
	}

	// Extract colors from mode-specific configs
	fillColor := 255
	gaugeColor := 200
	gaugeNeedleColor := 255
	gaugeShowTicks := true
	gaugeTicksColor := 150

	switch displayMode {
	case "bar":
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Fill != nil {
				fillColor = *cfg.Bar.Colors.Fill
			}
		}
	case "graph":
		if cfg.Graph != nil && cfg.Graph.Colors != nil {
			if cfg.Graph.Colors.Fill != nil {
				fillColor = *cfg.Graph.Colors.Fill
			}
		}
	case "gauge":
		if cfg.Gauge != nil {
			if cfg.Gauge.ShowTicks != nil {
				gaugeShowTicks = *cfg.Gauge.ShowTicks
			}
			if cfg.Gauge.Colors != nil {
				if cfg.Gauge.Colors.Fill != nil {
					fillColor = *cfg.Gauge.Colors.Fill
				}
				if cfg.Gauge.Colors.Arc != nil {
					gaugeColor = *cfg.Gauge.Colors.Arc
				}
				if cfg.Gauge.Colors.Needle != nil {
					gaugeNeedleColor = *cfg.Gauge.Colors.Needle
				}
				if cfg.Gauge.Colors.Ticks != nil {
					gaugeTicksColor = *cfg.Gauge.Colors.Ticks
				}
			}
		}
	}

	// Extract graph settings
	historyLen := 30
	graphFilled := true // Default to filled
	if cfg.Graph != nil {
		if cfg.Graph.History > 0 {
			historyLen = cfg.Graph.History
		}
		if cfg.Graph.Filled != nil {
			graphFilled = *cfg.Graph.Filled
		}
	}

	// Extract per-core settings
	perCore := false
	coreBorder := false
	coreMargin := 0
	if cfg.PerCore != nil {
		perCore = cfg.PerCore.Enabled
		coreBorder = cfg.PerCore.Border
		coreMargin = cfg.PerCore.Margin
	}

	// Extract bar settings
	barDirection := "horizontal"
	barBorder := false
	if cfg.Bar != nil {
		if cfg.Bar.Direction != "" {
			barDirection = cfg.Bar.Direction
		}
		barBorder = cfg.Bar.Border
	}

	// Get core count
	cores, err := cpu.Counts(true)
	if err != nil || cores == 0 {
		cores = 1
	}

	// Load font for text mode
	var fontFace font.Face
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(fontName, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &CPUWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		perCore:          perCore,
		fontSize:         fontSize,
		fontName:         fontName,
		horizAlign:       horizAlign,
		vertAlign:        vertAlign,
		padding:          padding,
		coreBorder:       coreBorder,
		coreMargin:       coreMargin,
		barDirection:     barDirection,
		barBorder:        barBorder,
		graphFilled:      graphFilled,
		fillColor:        uint8(fillColor),
		gaugeColor:       uint8(gaugeColor),
		gaugeNeedleColor: uint8(gaugeNeedleColor),
		gaugeShowTicks:   gaugeShowTicks,
		gaugeTicksColor:  uint8(gaugeTicksColor),
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
			bitmap.DrawHorizontalBar(img, x, coreY, width, coreHeight, usage, w.fillColor, w.barBorder || w.coreBorder)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawHorizontalBar(img, x, y, width, height, usage, w.fillColor, w.barBorder)
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
			bitmap.DrawVerticalBar(img, coreX, y, coreWidth, height, usage, w.fillColor, w.barBorder || w.coreBorder)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawVerticalBar(img, x, y, width, height, usage, w.fillColor, w.barBorder)
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

			bitmap.DrawGraph(img, cellX, cellY, cellWidth, cellHeight, coreHistories[i], w.historyLen, w.fillColor, w.graphFilled)
		}
	} else {
		// Single value history
		history := make([]float64, len(w.history))
		for i, item := range w.history {
			history[i] = item.(float64)
		}
		bitmap.DrawGraph(img, x, y, width, height, history, w.historyLen, w.fillColor, w.graphFilled)
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

			bitmap.DrawGauge(img, cellX, cellY, cellWidth, cellHeight, usage, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, usage, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)
	}
}
