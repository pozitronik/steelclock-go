package gpu

import (
	"fmt"
	"image"
	"log"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("gpu", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Supported GPU metrics
const (
	MetricUtilization       = "utilization"
	MetricUtilization3D     = "utilization_3d"
	MetricUtilizationCopy   = "utilization_copy"
	MetricUtilizationEncode = "utilization_video_encode"
	MetricUtilizationDecode = "utilization_video_decode"
	MetricMemoryDedicated   = "memory_dedicated"
	MetricMemoryShared      = "memory_shared"
)

// AdapterInfo contains information about a GPU adapter
type AdapterInfo struct {
	Index int
	Name  string
	LUID  string // Locally Unique Identifier (Windows)
}

// Reader is the interface for reading GPU metrics
type Reader interface {
	// GetMetric returns the current value for the specified metric and adapter
	// Returns value in percentage (0-100) for utilization metrics
	// Returns value in bytes for memory metrics
	GetMetric(adapter int, metric string) (float64, error)
	// GetMemoryTotal returns total memory in bytes for the specified adapter
	// Used to calculate percentage for memory metrics
	GetMemoryTotal(adapter int, metricType string) (uint64, error)
	// ListAdapters returns information about available GPU adapters
	ListAdapters() ([]AdapterInfo, error)
	// Close releases resources
	Close()
}

// Widget displays GPU metrics
type Widget struct {
	*widget.BaseWidget
	displayMode render.DisplayMode
	historyLen  int

	// Strategy pattern for rendering
	strategy render.MetricDisplayStrategy
	// MetricRenderer for rendering
	Renderer *render.MetricRenderer

	// GPU configuration
	adapter int    // GPU adapter index
	metric  string // Metric to display

	// GPU metrics reader
	reader       Reader
	readerFailed bool // True if reader initialization failed

	// Current value and history
	currentValue float64
	history      *util.RingBuffer[float64]
	hasData      bool
	mu           sync.RWMutex
}

// New creates a new GPU widget
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

	// Extract GPU-specific settings
	adapter := 0
	metric := MetricUtilization
	if cfg.GPU != nil {
		adapter = cfg.GPU.Adapter
		if cfg.GPU.Metric != "" {
			metric = cfg.GPU.Metric
		}
	}

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

	// Initialize reader (platform-specific)
	reader, readerErr := newReader()
	readerFailed := false
	if readerErr != nil {
		log.Printf("[GPU] Failed to initialize reader: %v", readerErr)
		readerFailed = true
	} else {
		// Log available adapters
		adapters, err := reader.ListAdapters()
		if err != nil {
			log.Printf("[GPU] Failed to list adapters: %v", err)
		} else {
			log.Printf("[GPU] Found %d adapter(s):", len(adapters))
			for _, a := range adapters {
				log.Printf("[GPU]   %d: %s", a.Index, a.Name)
			}
		}
	}

	return &Widget{
		BaseWidget:   base,
		displayMode:  displayMode,
		historyLen:   graphSettings.HistoryLen,
		strategy:     render.GetMetricStrategy(displayMode),
		Renderer:     renderer,
		adapter:      adapter,
		metric:       metric,
		reader:       reader,
		readerFailed: readerFailed,
		history:      util.NewRingBuffer[float64](graphSettings.HistoryLen),
	}, nil
}

// Update updates the GPU metrics
func (w *Widget) Update() error {
	if w.readerFailed || w.reader == nil {
		return fmt.Errorf("GPU reader not available")
	}

	value, err := w.reader.GetMetric(w.adapter, w.metric)
	if err != nil {
		return err
	}

	// For memory metrics, convert to percentage
	if w.metric == MetricMemoryDedicated || w.metric == MetricMemoryShared {
		total, err := w.reader.GetMemoryTotal(w.adapter, w.metric)
		if err == nil && total > 0 {
			value = (value / float64(total)) * 100
		}
	}

	// Clamp to 0-100
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}

	w.mu.Lock()
	w.currentValue = value
	w.hasData = true

	// Add to history for graph mode
	if w.displayMode == render.DisplayModeGraph {
		w.history.Push(value)
	}
	w.mu.Unlock()

	return nil
}

// Render creates an image of the GPU widget
func (w *Widget) Render() (image.Image, error) {
	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Get content area and position
	content := w.GetContentArea()
	pos := w.GetPosition()

	// If reader failed, show error message
	if w.readerFailed {
		bitmap.DrawAlignedInternalText(img, "GPU N/A", nil, "center", "center", 0)
		return img, nil
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return img, nil
	}

	// Use strategy pattern for rendering
	w.strategy.Render(img, render.MetricData{
		Value:       w.currentValue,
		History:     w.history.ToSlice(),
		TextFormat:  "%.0f",
		ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
		GaugeArea:   image.Rect(0, 0, pos.W, pos.H),
	}, w.Renderer)

	return img, nil
}

// Stop releases resources
func (w *Widget) Stop() {
	if w.reader != nil {
		w.reader.Close()
	}
}
