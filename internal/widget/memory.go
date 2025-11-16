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
	barBorder        bool
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

	gaugeColor := cfg.Properties.GaugeColor
	if gaugeColor == 0 {
		gaugeColor = 200
	}

	gaugeNeedleColor := cfg.Properties.GaugeNeedleColor
	if gaugeNeedleColor == 0 {
		gaugeNeedleColor = 255
	}

	historyLen := cfg.Properties.HistoryLength
	if historyLen == 0 {
		historyLen = 30
	}

	// Load font for text mode
	var fontFace font.Face
	var err error
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(cfg.Properties.Font, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &MemoryWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		fontSize:         fontSize,
		fontName:         cfg.Properties.Font,
		horizAlign:       horizAlign,
		vertAlign:        vertAlign,
		padding:          cfg.Properties.Padding,
		barBorder:        cfg.Properties.BarBorder,
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
	w.mu.RLock()
	switch w.displayMode {
	case "text":
		w.renderText(img)
	case "bar_horizontal":
		bitmap.DrawHorizontalBar(img, contentX, contentY, contentW, contentH, w.currentUsage, w.fillColor, w.barBorder)
	case "bar_vertical":
		bitmap.DrawVerticalBar(img, contentX, contentY, contentW, contentH, w.currentUsage, w.fillColor, w.barBorder)
	case "graph":
		bitmap.DrawGraph(img, contentX, contentY, contentW, contentH, w.history, w.historyLen, w.fillColor)
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
