package memory

import (
	"fmt"
	"image"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("memory", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Widget displays RAM usage
type Widget struct {
	*widget.BaseWidget
	mu             sync.RWMutex
	strategy       render.MetricDisplayStrategy
	Renderer       *render.MetricRenderer
	displayMode    render.DisplayMode
	currentValue   float64
	history        *util.RingBuffer[float64]
	textFormat     string
	memoryProvider metrics.MemoryProvider
}

// New creates a new memory widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := render.DisplayMode(helper.GetDisplayMode(config.ModeText))
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()
	gaugeSettings := helper.GetGaugeSettings()

	// Load font for text mode
	fontFace, err := bitmap.LoadFontForTextMode(string(displayMode), textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Determine bar color
	barColor := uint8(255)
	if graphSettings.FillColor >= 0 && graphSettings.FillColor <= 255 {
		barColor = uint8(graphSettings.FillColor)
	}

	// Create metric renderer
	renderer := render.NewMetricRenderer(
		render.BarConfig{
			Direction: barSettings.Direction,
			Border:    barSettings.Border,
			Color:     barColor,
		},
		render.GraphConfig{
			FillColor:  graphSettings.FillColor,
			LineColor:  graphSettings.LineColor,
			HistoryLen: graphSettings.HistoryLen,
		},
		render.GaugeConfig{
			ArcColor:    uint8(gaugeSettings.ArcColor),
			NeedleColor: uint8(gaugeSettings.NeedleColor),
			ShowTicks:   gaugeSettings.ShowTicks,
			TicksColor:  uint8(gaugeSettings.TicksColor),
		},
		render.TextConfig{
			FontFace:   fontFace,
			FontName:   textSettings.FontName,
			HorizAlign: textSettings.HorizAlign,
			VertAlign:  textSettings.VertAlign,
			Padding:    padding,
		},
	)

	return &Widget{
		BaseWidget:     base,
		strategy:       render.GetMetricStrategy(displayMode),
		Renderer:       renderer,
		displayMode:    displayMode,
		history:        util.NewRingBuffer[float64](graphSettings.HistoryLen),
		textFormat:     "%.0f",
		memoryProvider: metrics.DefaultMemory,
	}, nil
}

// Update updates the memory usage
func (w *Widget) Update() error {
	percent, err := w.memoryProvider.UsedPercent()
	if err != nil {
		return err
	}

	// Clamp to 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentValue = percent
	if w.displayMode == render.DisplayModeGraph {
		w.history.Push(percent)
	}

	return nil
}

// GetValue returns the current memory usage percentage (thread-safe)
func (w *Widget) GetValue() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentValue
}

// Render creates an image of the memory widget
func (w *Widget) Render() (image.Image, error) {
	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Get content area (adjusted for padding) and full bounds for gauge
	content := w.GetContentArea()
	pos := w.GetPosition()

	w.mu.RLock()
	defer w.mu.RUnlock()

	// Delegate rendering to strategy
	w.strategy.Render(img, render.MetricData{
		Value:       w.currentValue,
		History:     w.history.ToSlice(),
		TextFormat:  w.textFormat,
		ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
		GaugeArea:   image.Rect(0, 0, pos.W, pos.H),
	}, w.Renderer)

	return img, nil
}
