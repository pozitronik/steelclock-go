package hwmon

import (
	"fmt"
	"image"
	"log"
	"strings"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
)

const defaultLHMURL = "http://localhost:8085"

func init() {
	widget.Register("hwmon", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Widget displays hardware sensor data from LibreHardwareMonitor/OpenHardwareMonitor
type Widget struct {
	*widget.BaseWidget
	displayMode render.DisplayMode
	perCore     bool
	padding     int
	coreBorder  bool
	coreMargin  int
	fillColor   int
	historyLen  int

	// Sensor selection config
	sensorID     string // exact sensor path
	sensorType   string // type filter
	sensorFilter string // substring filter on ID or name
	minVal       float64
	maxVal       float64

	// Strategy pattern for rendering
	strategy     render.MetricDisplayStrategy
	gridStrategy render.GridMetricDisplayStrategy
	Renderer     *render.MetricRenderer

	// Provider
	hwmonProvider metrics.HWMonProvider

	// Current values
	currentNorm  float64   // Normalized single value (0-100)
	currentNorms []float64 // Normalized per-sensor values (per_core mode)
	rawValue     float64   // Raw single value (for text display)
	rawValues    []float64 // Raw per-sensor values
	rawUnit      string    // Unit from matched sensor(s)

	// History for graph mode
	historySingle  *util.RingBuffer[float64]
	historyPerCore *util.RingBuffer[[]float64]

	hasData        bool
	unavailable    bool
	unavailableMsg string
	sensorCount    int
	userTextFormat string // user-provided text.format override
	fontFace       font.Face
	fontName       string
	mu             sync.RWMutex
}

// New creates a new Hardware Monitor widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	mr, err := helper.BuildMetricRenderer()
	if err != nil {
		return nil, err
	}

	perCore, coreBorder, coreMargin := helper.GetPerCoreSettings()

	// HWMon-specific settings
	url := defaultLHMURL
	sensorID := ""
	sensorType := ""
	sensorFilter := ""
	minVal := 0.0
	maxVal := 100.0

	userTextFormat := ""
	if cfg.Text != nil && cfg.Text.Format != "" {
		userTextFormat = cfg.Text.Format
	}

	if cfg.HWMon != nil {
		if cfg.HWMon.URL != "" {
			url = cfg.HWMon.URL
		}
		sensorID = cfg.HWMon.SensorID
		sensorType = cfg.HWMon.SensorType
		sensorFilter = cfg.HWMon.SensorFilter
		if cfg.HWMon.Min != 0 || cfg.HWMon.Max != 0 {
			minVal = cfg.HWMon.Min
		}
		if cfg.HWMon.Max > 0 {
			maxVal = cfg.HWMon.Max
		}
	}

	return &Widget{
		BaseWidget:     base,
		displayMode:    mr.DisplayMode,
		perCore:        perCore,
		padding:        mr.Padding,
		coreBorder:     coreBorder,
		coreMargin:     coreMargin,
		fillColor:      mr.FillColor,
		historyLen:     mr.HistoryLen,
		sensorID:       sensorID,
		sensorType:     sensorType,
		sensorFilter:   sensorFilter,
		minVal:         minVal,
		maxVal:         maxVal,
		strategy:       mr.Strategy,
		gridStrategy:   render.GetGridMetricStrategy(mr.DisplayMode),
		Renderer:       mr.Renderer,
		hwmonProvider:  metrics.NewLHMHTTPProvider(url),
		historySingle:  util.NewRingBuffer[float64](mr.HistoryLen),
		historyPerCore: util.NewRingBuffer[[]float64](mr.HistoryLen),
		userTextFormat: userTextFormat,
		fontFace:       mr.FontFace,
		fontName:       mr.FontName,
	}, nil
}

// normalize converts a raw value to a 0-100 scale based on configured min/max.
func (w *Widget) normalize(value float64) float64 {
	if w.maxVal <= w.minVal {
		return 0
	}
	normalized := ((value - w.minVal) / (w.maxVal - w.minVal)) * 100.0
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 100 {
		normalized = 100
	}
	return normalized
}

// filterSensors returns sensors matching the configured filters.
// Priority: sensor_id (exact) > sensor_type + sensor_filter > sensor_type > sensor_filter.
func (w *Widget) filterSensors(stats []metrics.HWMonStat) []metrics.HWMonStat {
	// Exact sensor_id match
	if w.sensorID != "" {
		for _, s := range stats {
			if s.SensorID == w.sensorID {
				return []metrics.HWMonStat{s}
			}
		}
		return nil
	}

	filtered := stats

	// Filter by sensor type
	if w.sensorType != "" {
		var byType []metrics.HWMonStat
		for _, s := range filtered {
			if strings.EqualFold(s.Type, w.sensorType) {
				byType = append(byType, s)
			}
		}
		filtered = byType
	}

	// Filter by substring in sensor ID or name
	if w.sensorFilter != "" {
		var byFilter []metrics.HWMonStat
		lowerFilter := strings.ToLower(w.sensorFilter)
		for _, s := range filtered {
			if strings.Contains(strings.ToLower(s.SensorID), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Name), lowerFilter) {
				byFilter = append(byFilter, s)
			}
		}
		filtered = byFilter
	}

	return filtered
}

// formatValue formats a sensor value for text display.
func (w *Widget) formatValue(value float64, unit string) string {
	switch unit {
	case "°C":
		return fmt.Sprintf("%.0f°C", value)
	case "%":
		return fmt.Sprintf("%.0f%%", value)
	case "W":
		return fmt.Sprintf("%.1fW", value)
	case "MHz":
		return fmt.Sprintf("%.0fMHz", value)
	case "V":
		return fmt.Sprintf("%.2fV", value)
	case "A":
		return fmt.Sprintf("%.2fA", value)
	case "":
		return fmt.Sprintf("%.1f", value)
	default:
		return fmt.Sprintf("%.1f%s", value, unit)
	}
}

// Update reads current sensor data from LHM/OHM
func (w *Widget) Update() error {
	w.mu.RLock()
	alreadyUnavailable := w.unavailable
	w.mu.RUnlock()

	if alreadyUnavailable {
		return nil
	}

	stats, err := w.hwmonProvider.Sensors()
	if err != nil {
		log.Printf("hwmon: sensors unavailable: %v", err)
		w.mu.Lock()
		w.unavailable = true
		w.unavailableMsg = "No sensors"
		w.mu.Unlock()
		return nil
	}

	filtered := w.filterSensors(stats)
	if len(filtered) == 0 {
		return nil
	}

	// Determine unit from the first matched sensor
	unit := filtered[0].Unit

	if w.perCore {
		normalized := make([]float64, len(filtered))
		raw := make([]float64, len(filtered))
		for i, s := range filtered {
			raw[i] = s.Value
			normalized[i] = w.normalize(s.Value)
		}

		w.mu.Lock()
		w.currentNorms = normalized
		w.rawValues = raw
		w.rawUnit = unit
		w.sensorCount = len(filtered)
		w.hasData = true
		if w.displayMode == render.DisplayModeGraph {
			w.historyPerCore.Push(normalized)
		}
		w.mu.Unlock()
	} else {
		// Aggregate: average of all filtered sensors
		sum := 0.0
		for _, s := range filtered {
			sum += s.Value
		}
		rawAvg := sum / float64(len(filtered))
		normalized := w.normalize(rawAvg)

		w.mu.Lock()
		w.currentNorm = normalized
		w.rawValue = rawAvg
		w.rawUnit = unit
		w.hasData = true
		if w.displayMode == render.DisplayModeGraph {
			w.historySingle.Push(normalized)
		}
		w.mu.Unlock()
	}

	return nil
}

// Render creates an image of the Hardware Monitor widget
func (w *Widget) Render() (image.Image, error) {
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	content := w.GetContentArea()
	pos := w.GetPosition()

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.unavailable {
		bitmap.SmartDrawAlignedText(img, w.unavailableMsg, nil, bitmap.FontNamePixel5x7, "center", "center", 0)
		return img, nil
	}

	if !w.hasData {
		return img, nil
	}

	// Per-sensor grid mode
	if w.perCore {
		borderColor := uint8(255)
		if w.fillColor >= 0 && w.fillColor <= 255 {
			borderColor = uint8(w.fillColor)
		}

		gridData := render.GridMetricData{
			Values:      w.currentNorms,
			ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
			Position:    pos,
			CoreBorder:  w.coreBorder,
			CoreMargin:  w.coreMargin,
			BorderColor: borderColor,
			FontFace:    w.fontFace,
			FontName:    w.fontName,
		}

		if w.displayMode == render.DisplayModeGraph && w.historyPerCore.Len() >= 2 {
			historySlice := w.historyPerCore.ToSlice()
			numSensors := len(historySlice[0])
			sensorHistories := make([][]float64, numSensors)
			for i := 0; i < numSensors; i++ {
				sensorHistories[i] = make([]float64, len(historySlice))
				for t, sensors := range historySlice {
					if i < len(sensors) {
						sensorHistories[i][t] = sensors[i]
					}
				}
			}
			gridData.History = sensorHistories
		}

		w.gridStrategy.Render(img, gridData, w.Renderer)
		return img, nil
	}

	// Single-value mode
	value := w.currentNorm
	textFormat := "%.0f"
	if w.displayMode == render.DisplayModeText {
		value = w.rawValue
		if w.userTextFormat != "" {
			textFormat = w.userTextFormat
		} else {
			textFormat = w.textFormatString()
		}
	}

	w.strategy.Render(img, render.MetricData{
		Value:       value,
		History:     w.historySingle.ToSlice(),
		TextFormat:  textFormat,
		ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
		GaugeArea:   image.Rect(0, 0, pos.W, pos.H),
	}, w.Renderer)

	return img, nil
}

// textFormatString returns a printf-style format string for the current sensor unit.
func (w *Widget) textFormatString() string {
	switch w.rawUnit {
	case "°C":
		return "%.0f°C"
	case "%":
		return "%.0f%%"
	case "W":
		return "%.1fW"
	case "MHz":
		return "%.0fMHz"
	case "V":
		return "%.2fV"
	case "A":
		return "%.2fA"
	case "":
		return "%.1f"
	default:
		return "%.1f" + w.rawUnit
	}
}
