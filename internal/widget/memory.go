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
	fillColor        int // -1 = no fill, 0-255 = fill color
	lineColor        int // 0-255 = line color
	gaugeColor       uint8
	gaugeNeedleColor uint8
	gaugeShowTicks   bool
	gaugeTicksColor  uint8
	historyLen       int
	currentUsage     float64
	history          *RingBuffer[float64] // Ring buffer for graph history - O(1) push with zero allocations
	fontFace         font.Face
	mu               sync.RWMutex // Protects currentUsage and history
}

// NewMemoryWidget creates a new memory widget
func NewMemoryWidget(cfg config.WidgetConfig) (*MemoryWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := helper.GetDisplayMode("text")
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()
	gaugeSettings := helper.GetGaugeSettings()

	// Load font for text mode
	fontFace, err := helper.LoadFontForTextMode(displayMode)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &MemoryWidget{
		BaseWidget:       base,
		displayMode:      displayMode,
		fontSize:         textSettings.FontSize,
		fontName:         textSettings.FontName,
		horizAlign:       textSettings.HorizAlign,
		vertAlign:        textSettings.VertAlign,
		padding:          padding,
		barDirection:     barSettings.Direction,
		barBorder:        barSettings.Border,
		fillColor:        graphSettings.FillColor,
		lineColor:        graphSettings.LineColor,
		gaugeColor:       uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor: uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:   gaugeSettings.ShowTicks,
		gaugeTicksColor:  uint8(gaugeSettings.TicksColor),
		historyLen:       graphSettings.HistoryLen,
		history:          NewRingBuffer[float64](graphSettings.HistoryLen),
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

	// Add to history for graph mode (ring buffer handles capacity automatically)
	if w.displayMode == "graph" {
		w.history.Push(w.currentUsage)
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
		barColor := uint8(255)
		if w.fillColor >= 0 && w.fillColor <= 255 {
			barColor = uint8(w.fillColor)
		}
		if w.barDirection == "vertical" {
			bitmap.DrawVerticalBar(img, contentX, contentY, contentW, contentH, w.currentUsage, barColor, w.barBorder)
		} else {
			bitmap.DrawHorizontalBar(img, contentX, contentY, contentW, contentH, w.currentUsage, barColor, w.barBorder)
		}
	case "graph":
		bitmap.DrawGraph(img, contentX, contentY, contentW, contentH, w.history.ToSlice(), w.historyLen, w.fillColor, w.lineColor)
	case "gauge":
		w.renderGauge(img, pos)
	}
	w.mu.RUnlock()

	return img, nil
}

func (w *MemoryWidget) renderText(img *image.Gray) {
	// Note: caller must hold read lock
	text := fmt.Sprintf("%.0f", w.currentUsage)
	bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
}

func (w *MemoryWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	// Note: caller must hold read lock
	// Use shared gauge drawing function
	bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, w.currentUsage, w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)
}
