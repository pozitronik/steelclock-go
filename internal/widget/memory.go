package widget

import (
	"fmt"
	"image"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"github.com/shirou/gopsutil/v4/mem"
)

func init() {
	Register("memory", func(cfg config.WidgetConfig) (Widget, error) {
		return NewMemoryWidget(cfg)
	})
}

// MemoryWidget displays RAM usage
type MemoryWidget struct {
	*BaseWidget
	displayMode  shared.DisplayMode
	padding      int
	renderer     *shared.MetricRenderer
	currentUsage float64
	history      *shared.RingBuffer[float64]
	mu           sync.RWMutex
}

// NewMemoryWidget creates a new memory widget
func NewMemoryWidget(cfg config.WidgetConfig) (*MemoryWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := shared.DisplayMode(helper.GetDisplayMode("text"))
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()
	gaugeSettings := helper.GetGaugeSettings()

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

	// Create metric renderer
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

	return &MemoryWidget{
		BaseWidget:  base,
		displayMode: displayMode,
		padding:     padding,
		renderer:    renderer,
		history:     shared.NewRingBuffer[float64](graphSettings.HistoryLen),
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
	if w.displayMode == shared.DisplayModeGraph {
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
	defer w.mu.RUnlock()

	switch w.displayMode {
	case shared.DisplayModeText:
		text := fmt.Sprintf("%.0f", w.currentUsage)
		w.renderer.RenderText(img, text)
	case shared.DisplayModeBar:
		w.renderer.RenderBar(img, contentX, contentY, contentW, contentH, w.currentUsage)
	case shared.DisplayModeGraph:
		w.renderer.RenderGraph(img, contentX, contentY, contentW, contentH, w.history.ToSlice())
	case shared.DisplayModeGauge:
		w.renderer.RenderGauge(img, 0, 0, pos.W, pos.H, w.currentUsage)
	}

	return img, nil
}
