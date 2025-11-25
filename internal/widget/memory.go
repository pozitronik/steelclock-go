package widget

import (
	"fmt"
	"image"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/shirou/gopsutil/v4/mem"
	"golang.org/x/image/font"
)

// MemoryWidget displays RAM usage
type MemoryWidget struct {
	*BaseWidget
	displayMode      string
	fontSize         int
	fontName         string
	horizAlign       string
	vertAlign        string
	padding          int
	barDirection     string
	barBorder        bool
	graphFilled      bool
	fillColor        uint8
	gaugeColor       uint8
	gaugeNeedleColor uint8
	historyLen       int
	currentUsage     float64
	history          []float64
	fontFace         font.Face
	mu               sync.RWMutex // Protects currentUsage and history
}

// NewMemoryWidget creates a new memory widget
func NewMemoryWidget(cfg config.WidgetConfig) (*MemoryWidget, error) {
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
		if cfg.Gauge != nil && cfg.Gauge.Colors != nil {
			if cfg.Gauge.Colors.Fill != nil {
				fillColor = *cfg.Gauge.Colors.Fill
			}
			if cfg.Gauge.Colors.Arc != nil {
				gaugeColor = *cfg.Gauge.Colors.Arc
			}
			if cfg.Gauge.Colors.Needle != nil {
				gaugeNeedleColor = *cfg.Gauge.Colors.Needle
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

	// Extract bar settings
	barDirection := "horizontal"
	barBorder := false
	if cfg.Bar != nil {
		if cfg.Bar.Direction != "" {
			barDirection = cfg.Bar.Direction
		}
		barBorder = cfg.Bar.Border
	}

	// Load font for text mode
	var fontFace font.Face
	var err error
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(fontName, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &MemoryWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		fontSize:         fontSize,
		fontName:         fontName,
		horizAlign:       horizAlign,
		vertAlign:        vertAlign,
		padding:          padding,
		barDirection:     barDirection,
		barBorder:        barBorder,
		graphFilled:      graphFilled,
		fillColor:        uint8(fillColor),
		gaugeColor:       uint8(gaugeColor),
		gaugeNeedleColor: uint8(gaugeNeedleColor),
		historyLen:       historyLen,
		history:          make([]float64, 0, historyLen),
		fontFace:         fontFace,
	}, nil
}

// Update updates the memory usage
func (w *MemoryWidget) Update() error {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.currentUsage = vmem.UsedPercent

	// Clamp to 0-100
	if w.currentUsage < 0 {
		w.currentUsage = 0
	}
	if w.currentUsage > 100 {
		w.currentUsage = 100
	}

	// Add to history for graph mode
	if w.displayMode == "graph" {
		w.history = append(w.history, w.currentUsage)
		if len(w.history) > w.historyLen {
			w.history = w.history[1:]
		}
	}
	w.mu.Unlock()

	return nil
}

// Render creates an image of the memory widget
func (w *MemoryWidget) Render() (image.Image, error) {
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
	w.mu.RLock()
	switch w.displayMode {
	case "text":
		w.renderText(img)
	case "bar":
		if w.barDirection == "vertical" {
			bitmap.DrawVerticalBar(img, contentX, contentY, contentW, contentH, w.currentUsage, w.fillColor, w.barBorder)
		} else {
			bitmap.DrawHorizontalBar(img, contentX, contentY, contentW, contentH, w.currentUsage, w.fillColor, w.barBorder)
		}
	case "graph":
		bitmap.DrawGraph(img, contentX, contentY, contentW, contentH, w.history, w.historyLen, w.fillColor, w.graphFilled)
	case "gauge":
		w.renderGauge(img, pos)
	}
	w.mu.RUnlock()

	return img, nil
}

func (w *MemoryWidget) renderText(img *image.Gray) {
	// Note: caller must hold read lock
	text := fmt.Sprintf("%.0f", w.currentUsage)
	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
}

func (w *MemoryWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	// Note: caller must hold read lock
	// Use shared gauge drawing function
	bitmap.DrawGauge(img, pos, w.currentUsage, w.gaugeColor, w.gaugeNeedleColor)
}
