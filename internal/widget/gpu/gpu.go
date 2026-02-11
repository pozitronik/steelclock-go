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

// GPU metric constants
const (
	MetricUtilization       = "utilization"
	MetricUtilization3D     = "utilization_3d"
	MetricUtilizationCopy   = "utilization_copy"
	MetricUtilizationEncode = "utilization_video_encode"
	MetricUtilizationDecode = "utilization_video_decode"
	MetricMemoryDedicated   = "memory_dedicated"
	MetricMemoryShared      = "memory_shared"
)

// supportedMetrics lists metrics that are currently functional.
// Memory metrics (memory_dedicated, memory_shared) are defined as constants but not yet
// supported because PDH doesn't provide total VRAM counters needed for percentage calculation.
var supportedMetrics = map[string]bool{
	MetricUtilization:       true,
	MetricUtilization3D:     true,
	MetricUtilizationCopy:   true,
	MetricUtilizationEncode: true,
	MetricUtilizationDecode: true,
}

// AdapterInfo contains information about a GPU adapter
type AdapterInfo struct {
	Index int
	Name  string
	LUID  string // Locally Unique Identifier (Windows)
}

// Reader is the interface for reading GPU metrics
type Reader interface {
	// GetMetric returns the current value for the specified metric and adapter.
	// Returns value in percentage (0-100) for utilization metrics.
	GetMetric(adapter int, metric string) (float64, error)
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

	// Build common metric renderer
	mr, err := helper.BuildMetricRenderer()
	if err != nil {
		return nil, err
	}

	// Extract GPU-specific settings
	adapter := 0
	metric := MetricUtilization
	if cfg.GPU != nil {
		adapter = cfg.GPU.Adapter
		if cfg.GPU.Metric != "" {
			metric = cfg.GPU.Metric
		}
	}

	// Validate metric
	if !supportedMetrics[metric] {
		return nil, fmt.Errorf("unsupported GPU metric: %q (supported: utilization, utilization_3d, utilization_copy, utilization_video_encode, utilization_video_decode)", metric)
	}

	// Initialize reader (platform-specific)
	reader, readerErr := newReader()
	readerFailed := false
	if readerErr != nil {
		log.Printf("[GPU] Failed to initialize reader: %v", readerErr)
		readerFailed = true
	} else {
		// Log available adapters
		adapters, listErr := reader.ListAdapters()
		if listErr != nil {
			log.Printf("[GPU] Failed to list adapters: %v", listErr)
		} else {
			log.Printf("[GPU] Found %d adapter(s):", len(adapters))
			for _, a := range adapters {
				log.Printf("[GPU]   %d: %s", a.Index, a.Name)
			}
		}
	}

	return &Widget{
		BaseWidget:   base,
		displayMode:  mr.DisplayMode,
		historyLen:   mr.HistoryLen,
		strategy:     mr.Strategy,
		Renderer:     mr.Renderer,
		adapter:      adapter,
		metric:       metric,
		reader:       reader,
		readerFailed: readerFailed,
		history:      util.NewRingBuffer[float64](mr.HistoryLen),
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

	// Clamp to 0-100 (utilization metrics are already in percentage)
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
