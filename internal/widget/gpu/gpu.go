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
var supportedMetrics = map[string]bool{
	MetricUtilization:       true,
	MetricUtilization3D:     true,
	MetricUtilizationCopy:   true,
	MetricUtilizationEncode: true,
	MetricUtilizationDecode: true,
	MetricMemoryDedicated:   true,
	MetricMemoryShared:      true,
}

// AdapterInfo contains information about a GPU adapter.
// Index is a sequential number (0, 1, 2, ...) assigned during discovery.
// On Windows, LUID (Locally Unique Identifier) is the primary key used to
// distinguish adapters, since all GPUs report phys_0 in PDH instance names.
type AdapterInfo struct {
	Index                int
	Name                 string
	LUID                 string // Locally Unique Identifier from PDH instance names (Windows only)
	DedicatedVideoMemory uint64 // Total dedicated VRAM in bytes (from DXGI, Windows only)
	SharedSystemMemory   uint64 // Total shared system memory in bytes (from DXGI, Windows only)
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
	adapter    int    // GPU adapter index
	metric     string // Metric to display
	textFormat string // Format string for bar text overlay (from text.format)

	// GPU metrics reader
	reader       Reader
	readerFailed bool // True if reader initialization failed

	// Current value and history
	currentValue float64
	history      *util.RingBuffer[float64]
	hasData      bool
	mu           sync.RWMutex

	// totalMemoryGB is the adapter's total memory for the configured metric
	// (dedicated or shared), in gibibytes. Zero for non-memory metrics, where
	// a used/total GB breakdown doesn't apply. Set once at New() since adapter
	// memory capacity doesn't change at runtime.
	totalMemoryGB float64
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

	// Extract text format for bar overlay
	textFormat := ""
	if cfg.Text != nil {
		textFormat = cfg.Text.Format
	}

	// Validate metric
	if !supportedMetrics[metric] {
		return nil, fmt.Errorf("unsupported GPU metric: %q (supported: utilization, utilization_3d, utilization_copy, utilization_video_encode, utilization_video_decode, memory_dedicated, memory_shared)", metric)
	}

	// Initialize reader (platform-specific)
	reader, readerErr := newReader()
	readerFailed := false
	var totalMemoryGB float64
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
			for _, a := range adapters {
				if a.Index != adapter {
					continue
				}
				switch metric {
				case MetricMemoryDedicated:
					totalMemoryGB = float64(a.DedicatedVideoMemory) / gibibyte
				case MetricMemoryShared:
					totalMemoryGB = float64(a.SharedSystemMemory) / gibibyte
				}
			}
		}
	}

	return &Widget{
		BaseWidget:    base,
		displayMode:   mr.DisplayMode,
		historyLen:    mr.HistoryLen,
		strategy:      mr.Strategy,
		Renderer:      mr.Renderer,
		adapter:       adapter,
		metric:        metric,
		textFormat:    textFormat,
		reader:        reader,
		readerFailed:  readerFailed,
		history:       util.NewRingBuffer[float64](mr.HistoryLen),
		totalMemoryGB: totalMemoryGB,
	}, nil
}

// gibibyte is the byte count of one gibibyte (1024^3), matching the unit
// Windows Task Manager and most OS memory displays label "GB".
const gibibyte = 1024 * 1024 * 1024

// Update updates the GPU metrics
func (w *Widget) Update() error {
	if w.readerFailed || w.reader == nil {
		return nil // Render shows "GPU N/A"; no point logging every tick
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

	// Determine text format: use configured format for text mode, default for others
	textFmt := "%.0f"
	if w.textFormat != "" {
		textFmt = w.textFormat
	}

	// In text mode, {used}/{total}/{percent} tokens render a GB breakdown for
	// memory metrics (e.g. "V {used}GB {percent}%"); plain printf formats keep
	// rendering just the percentage through the strategy below.
	if w.displayMode == render.DisplayModeText && render.IsUsageTokenFormat(textFmt) {
		w.Renderer.RenderText(img, w.usageText(textFmt))
		return img, nil
	}

	// Use strategy pattern for rendering
	w.strategy.Render(img, render.MetricData{
		Value:       w.currentValue,
		History:     w.history.ToSlice(),
		TextFormat:  textFmt,
		ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
		GaugeArea:   image.Rect(0, 0, pos.W, pos.H),
	}, w.Renderer)

	return img, nil
}

// usageText renders a token text format from the current values. Used memory
// is derived from the usage percentage and the adapter's total memory.
// Callers must hold w.mu.
func (w *Widget) usageText(format string) string {
	usedGB := w.currentValue / 100 * w.totalMemoryGB
	return render.FormatUsageTokens(format, usedGB, w.totalMemoryGB, w.currentValue)
}

// Stop releases resources
func (w *Widget) Stop() {
	if w.reader != nil {
		w.reader.Close()
	}
}
