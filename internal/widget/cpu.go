package widget

import (
	"fmt"
	"image"
	"time"

	"github.com/pozitronik/steelclock/internal/bitmap"
	"github.com/pozitronik/steelclock/internal/config"
	"github.com/shirou/gopsutil/v4/cpu"
	"golang.org/x/image/font"
)

// CPUWidget displays CPU usage
type CPUWidget struct {
	*BaseWidget
	displayMode  string
	perCore      bool
	fontSize     int
	fontName     string
	horizAlign   string
	vertAlign    string
	padding      int
	barBorder    bool
	barMargin    int
	fillColor    uint8
	historyLen   int
	currentUsage interface{} // float64 or []float64
	history      []interface{}
	coreCount    int
	fontFace     font.Face
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

	fillColor := cfg.Properties.FillColor
	if fillColor == 0 {
		fillColor = 255
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
		BaseWidget:  base,
		displayMode: displayMode,
		perCore:     cfg.Properties.PerCore,
		fontSize:    fontSize,
		fontName:    cfg.Properties.Font,
		horizAlign:  horizAlign,
		vertAlign:   vertAlign,
		padding:     cfg.Properties.Padding,
		barBorder:   cfg.Properties.BarBorder,
		barMargin:   cfg.Properties.BarMargin,
		fillColor:   uint8(fillColor),
		historyLen:  historyLen,
		history:     make([]interface{}, 0, historyLen),
		coreCount:   cores,
		fontFace:    fontFace,
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

		w.currentUsage = percentages

		// Add to history
		if w.displayMode == "graph" {
			w.history = append(w.history, percentages)
			if len(w.history) > w.historyLen {
				w.history = w.history[1:]
			}
		}
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

		w.currentUsage = usage

		// Add to history
		if w.displayMode == "graph" {
			w.history = append(w.history, usage)
			if len(w.history) > w.historyLen {
				w.history = w.history[1:]
			}
		}
	}

	return nil
}

// Render creates an image of the CPU widget
func (w *CPUWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))

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
	}

	return img, nil
}

func (w *CPUWidget) renderText(img *image.Gray) {
	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
		avg := 0.0
		for _, c := range cores {
			avg += c
		}
		avg /= float64(len(cores))
		text := fmt.Sprintf("%.0f", avg)
		bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
	} else {
		usage := w.currentUsage.(float64)
		text := fmt.Sprintf("%.0f", usage)
		bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
	}
}

func (w *CPUWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
		coreHeight := (height - (len(cores)-1)*w.barMargin) / len(cores)

		for i, usage := range cores {
			coreY := y + i*(coreHeight+w.barMargin)
			bitmap.DrawHorizontalBar(img, x, coreY, width, coreHeight, usage, w.fillColor, w.barBorder)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawHorizontalBar(img, x, y, width, height, usage, w.fillColor, w.barBorder)
	}
}

func (w *CPUWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	if w.currentUsage == nil {
		return
	}

	if w.perCore {
		cores := w.currentUsage.([]float64)
		coreWidth := (width - (len(cores)-1)*w.barMargin) / len(cores)

		for i, usage := range cores {
			coreX := x + i*(coreWidth+w.barMargin)
			bitmap.DrawVerticalBar(img, coreX, y, coreWidth, height, usage, w.fillColor, w.barBorder)
		}
	} else {
		usage := w.currentUsage.(float64)
		bitmap.DrawVerticalBar(img, x, y, width, height, usage, w.fillColor, w.barBorder)
	}
}

func (w *CPUWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	if len(w.history) < 2 {
		return
	}

	if w.perCore {
		// For per-core, we'll show average for simplicity
		avgHistory := make([]float64, len(w.history))
		for i, item := range w.history {
			cores := item.([]float64)
			avg := 0.0
			for _, c := range cores {
				avg += c
			}
			avgHistory[i] = avg / float64(len(cores))
		}
		bitmap.DrawGraph(img, x, y, width, height, avgHistory, w.historyLen, w.fillColor)
	} else {
		// Single value history
		history := make([]float64, len(w.history))
		for i, item := range w.history {
			history[i] = item.(float64)
		}
		bitmap.DrawGraph(img, x, y, width, height, history, w.historyLen, w.fillColor)
	}
}
