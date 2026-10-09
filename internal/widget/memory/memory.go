package memory

import (
	"image"
	"sync"

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
	usedGB         float64
	totalGB        float64
	history        *util.RingBuffer[float64]
	textFormat     string
	memoryProvider metrics.MemoryProvider
}

// New creates a new memory widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Build common metric renderer (shared with CPU widget)
	mr, err := helper.BuildMetricRenderer()
	if err != nil {
		return nil, err
	}

	textFormat := "%.0f"
	if cfg.Text != nil && cfg.Text.Format != "" {
		textFormat = cfg.Text.Format
	}

	return &Widget{
		BaseWidget:     base,
		strategy:       mr.Strategy,
		Renderer:       mr.Renderer,
		displayMode:    mr.DisplayMode,
		history:        util.NewRingBuffer[float64](mr.HistoryLen),
		textFormat:     textFormat,
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

	// Best-effort: GB figures are a display nicety, not worth failing Update() over.
	usedGB, totalGB, gbErr := w.memoryProvider.UsedGB()

	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentValue = percent
	if gbErr == nil {
		w.usedGB = usedGB
		w.totalGB = totalGB
	}
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

	// In text mode, {used}/{total}/{percent} tokens render a GB breakdown
	// (e.g. "R {used}GB {percent}%"); plain printf formats keep rendering
	// just the percentage through the strategy below.
	if w.displayMode == render.DisplayModeText && render.IsUsageTokenFormat(w.textFormat) {
		w.Renderer.RenderText(img, w.usageText())
		return img, nil
	}

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

// usageText renders the token text format from the current values.
// Callers must hold w.mu.
func (w *Widget) usageText() string {
	return render.FormatUsageTokens(w.textFormat, w.usedGB, w.totalGB, w.currentValue)
}
