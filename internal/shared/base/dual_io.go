package base

import (
	"fmt"
	"image"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
)

// WidgetBase defines the interface for base widget functionality
// needed by DualIOWidget
type WidgetBase interface {
	GetPosition() config.PositionConfig
	GetStyle() config.StyleConfig
	GetRenderBackgroundColor() uint8
	CreateCanvas() *image.Gray
	ApplyBorder(img *image.Gray)
}

// DualIOTextConfig holds text formatting configuration for dual I/O widgets
type DualIOTextConfig struct {
	PrimaryPrefix   string // e.g., "↓" for network RX, "R" for disk read
	SecondaryPrefix string // e.g., "↑" for network TX, "W" for disk write
}

// DualIOWidget contains common fields and methods for dual-metric I/O widgets
// (network rx/tx, disk read/write, etc.)
type DualIOWidget struct {
	Base          WidgetBase
	DisplayMode   render.DisplayMode
	Padding       int
	MaxSpeedBps   float64 // Max speed in bytes per second (-1 for auto-scale)
	Unit          string  // "auto", "Mbps", "MB/s", etc.
	ShowUnit      bool    // Show unit suffix in text mode
	SupportsGauge bool    // Whether this widget supports gauge mode
	TextConfig    DualIOTextConfig
	Converter     *util.ByteRateConverter
	Renderer      *render.DualMetricRenderer
	Strategy      render.DualMetricDisplayStrategy

	// Runtime state - current values in bytes per second
	PrimaryValue   float64
	SecondaryValue float64

	// History for graph mode
	PrimaryHistory   *util.RingBuffer[float64]
	SecondaryHistory *util.RingBuffer[float64]

	Mu sync.RWMutex
}

// DualIOConfig holds configuration for creating a DualIOWidget
type DualIOConfig struct {
	Base          WidgetBase
	DisplayMode   render.DisplayMode
	Padding       int
	MaxSpeedBps   float64
	Unit          string
	ShowUnit      bool
	SupportsGauge bool
	TextConfig    DualIOTextConfig
	Converter     *util.ByteRateConverter
	Renderer      *render.DualMetricRenderer
	HistoryLen    int
}

// NewDualIOWidget creates a new DualIOWidget with the given configuration
func NewDualIOWidget(cfg DualIOConfig) *DualIOWidget {
	return &DualIOWidget{
		Base:             cfg.Base,
		DisplayMode:      cfg.DisplayMode,
		Padding:          cfg.Padding,
		MaxSpeedBps:      cfg.MaxSpeedBps,
		Unit:             cfg.Unit,
		ShowUnit:         cfg.ShowUnit,
		SupportsGauge:    cfg.SupportsGauge,
		TextConfig:       cfg.TextConfig,
		Converter:        cfg.Converter,
		Renderer:         cfg.Renderer,
		Strategy:         render.GetDualMetricStrategy(cfg.DisplayMode),
		PrimaryHistory:   util.NewRingBuffer[float64](cfg.HistoryLen),
		SecondaryHistory: util.NewRingBuffer[float64](cfg.HistoryLen),
	}
}

// SetValues updates current primary and secondary values (thread-safe)
func (w *DualIOWidget) SetValues(primary, secondary float64) {
	w.Mu.Lock()
	w.PrimaryValue = primary
	w.SecondaryValue = secondary
	w.Mu.Unlock()
}

// AddToHistory adds values to history buffers (thread-safe)
// Should only be called when DisplayMode is 'graph'
func (w *DualIOWidget) AddToHistory(primary, secondary float64) {
	w.Mu.Lock()
	w.PrimaryHistory.Push(primary)
	w.SecondaryHistory.Push(secondary)
	w.Mu.Unlock()
}

// SetValuesAndHistory updates values and optionally adds to history (thread-safe)
func (w *DualIOWidget) SetValuesAndHistory(primary, secondary float64, addHistory bool) {
	w.Mu.Lock()
	w.PrimaryValue = primary
	w.SecondaryValue = secondary
	if addHistory {
		w.PrimaryHistory.Push(primary)
		w.SecondaryHistory.Push(secondary)
	}
	w.Mu.Unlock()
}

// IsGraphMode returns true if the widget is in graph display mode
func (w *DualIOWidget) IsGraphMode() bool {
	return w.DisplayMode == render.DisplayModeGraph
}

// Render creates an image of the dual I/O widget
func (w *DualIOWidget) Render() (image.Image, error) {
	pos := w.Base.GetPosition()

	// Create canvas with background using base widget helper
	img := w.Base.CreateCanvas()

	// Draw border if enabled using base widget helper
	w.Base.ApplyBorder(img)

	// Calculate content area
	contentX := w.Padding
	contentY := w.Padding
	contentW := pos.W - w.Padding*2
	contentH := pos.H - w.Padding*2

	w.Mu.RLock()
	defer w.Mu.RUnlock()

	// Prepare data based on display mode
	primaryPct, secondaryPct := w.calculatePercentages()
	primaryHist, secondaryHist := w.normalizeHistory()

	data := render.DualMetricData{
		PrimaryValue:     primaryPct,
		SecondaryValue:   secondaryPct,
		PrimaryHistory:   primaryHist,
		SecondaryHistory: secondaryHist,
		FormattedText:    w.formatText(),
		ContentArea:      image.Rect(contentX, contentY, contentX+contentW, contentY+contentH),
		GaugeArea:        image.Rect(0, 0, pos.W, pos.H),
		WidgetWidth:      pos.W,
		WidgetHeight:     pos.H,
		SupportsGauge:    w.SupportsGauge,
	}

	w.Strategy.Render(img, data, w.Renderer)

	return img, nil
}

// formatText formats the text output with unit conversion
func (w *DualIOWidget) formatText() string {
	if w.Unit == "auto" {
		// Auto-scale each value independently
		primaryVal, primaryUnit := w.Converter.AutoScale(w.PrimaryValue)
		secondaryVal, secondaryUnit := w.Converter.AutoScale(w.SecondaryValue)
		return fmt.Sprintf("%s%s%s %s%s%s",
			w.TextConfig.PrimaryPrefix, FormatDualIOValue(primaryVal), primaryUnit,
			w.TextConfig.SecondaryPrefix, FormatDualIOValue(secondaryVal), secondaryUnit)
	}

	// Fixed unit for both values
	primaryVal, unitName := w.Converter.Convert(w.PrimaryValue, w.Unit)
	secondaryVal, _ := w.Converter.Convert(w.SecondaryValue, w.Unit)

	if w.ShowUnit {
		return fmt.Sprintf("%s%s %s%s %s",
			w.TextConfig.PrimaryPrefix, FormatDualIOValue(primaryVal),
			w.TextConfig.SecondaryPrefix, FormatDualIOValue(secondaryVal), unitName)
	}
	return fmt.Sprintf("%s%s %s%s",
		w.TextConfig.PrimaryPrefix, FormatDualIOValue(primaryVal),
		w.TextConfig.SecondaryPrefix, FormatDualIOValue(secondaryVal))
}

// calculatePercentages calculates primary/secondary percentages based on max speed
func (w *DualIOWidget) calculatePercentages() (primaryPct, secondaryPct float64) {
	maxSpeed := w.MaxSpeedBps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.PrimaryValue, w.SecondaryValue)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	primaryPct = ClampValue((w.PrimaryValue/maxSpeed)*100, 0, 100)
	secondaryPct = ClampValue((w.SecondaryValue/maxSpeed)*100, 0, 100)
	return
}

// normalizeHistory normalizes history data to 0-100 scale
func (w *DualIOWidget) normalizeHistory() (primaryPct, secondaryPct []float64) {
	if w.PrimaryHistory.Len() < 2 {
		return nil, nil
	}

	// Get history slices
	primaryData := w.PrimaryHistory.ToSlice()
	secondaryData := w.SecondaryHistory.ToSlice()

	// Determine max speed for normalization
	maxSpeed := w.MaxSpeedBps
	if maxSpeed < 0 {
		// Find max in history
		maxSpeed = 1.0
		for _, v := range primaryData {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
		for _, v := range secondaryData {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
	}

	// Normalize to percentages
	primaryPct = make([]float64, len(primaryData))
	secondaryPct = make([]float64, len(secondaryData))

	for i := range primaryData {
		primaryPct[i] = (primaryData[i] / maxSpeed) * 100
	}
	for i := range secondaryData {
		secondaryPct[i] = (secondaryData[i] / maxSpeed) * 100
	}

	return
}

// FormatDualIOValue formats a value with appropriate precision
func FormatDualIOValue(value float64) string {
	if value >= 100 {
		return fmt.Sprintf("%.0f", value)
	} else if value >= 10 {
		return fmt.Sprintf("%.1f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

// ClampValue restricts a value to the given range
func ClampValue(value, minVal, maxVal float64) float64 {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
