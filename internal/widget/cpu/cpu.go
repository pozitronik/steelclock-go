package cpu

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
)

func init() {
	widget.Register("cpu", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Widget displays CPU usage
type Widget struct {
	*widget.BaseWidget
	displayMode render.DisplayMode
	perCore     bool
	padding     int
	coreBorder  bool
	coreMargin  int
	fillColor   int // -1 = no fill, 0-255 = fill color (used for per-core border color)
	historyLen  int

	// Strategy pattern for single-value mode rendering
	strategy render.MetricDisplayStrategy
	// Strategy pattern for per-core (grid) mode rendering
	gridStrategy render.GridMetricDisplayStrategy
	// MetricRenderer for rendering
	Renderer *render.MetricRenderer

	// Metrics provider (abstraction over gopsutil)
	cpuProvider metrics.CPUProvider

	// Separate typed fields instead of interface{} to avoid runtime type assertions
	currentUsageSingle  float64   // Aggregate CPU usage (when perCore=false)
	currentUsagePerCore []float64 // Per-core CPU usage (when perCore=true)
	// Ring buffers for graph history - O(1) push with zero allocations
	historySingle  *util.RingBuffer[float64]   // Aggregate history (when perCore=false)
	historyPerCore *util.RingBuffer[[]float64] // Per-core history (when perCore=true)
	hasData        bool                        // Indicates if currentUsage has been set
	coreCount      int
	fontFace       font.Face    // Kept for per-core text rendering
	fontName       string       // Kept for per-core text rendering
	mu             sync.RWMutex // Protects currentUsage and history
}

// New creates a new CPU widget
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
	perCore, coreBorder, coreMargin := helper.GetPerCoreSettings()

	// Use default CPU provider
	cpuProvider := metrics.DefaultCPU

	// Get core count
	cores, err := cpuProvider.Counts(true)
	if err != nil || cores == 0 {
		cores = 1
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

	// Create metric renderer for single-value mode
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
		displayMode:    displayMode,
		perCore:        perCore,
		padding:        padding,
		coreBorder:     coreBorder,
		coreMargin:     coreMargin,
		fillColor:      graphSettings.FillColor,
		historyLen:     graphSettings.HistoryLen,
		strategy:       render.GetMetricStrategy(displayMode),
		gridStrategy:   render.GetGridMetricStrategy(displayMode),
		Renderer:       renderer,
		cpuProvider:    cpuProvider,
		historySingle:  util.NewRingBuffer[float64](graphSettings.HistoryLen),
		historyPerCore: util.NewRingBuffer[[]float64](graphSettings.HistoryLen),
		coreCount:      cores,
		fontFace:       fontFace,
		fontName:       textSettings.FontName,
	}, nil
}

// Update updates the CPU usage
func (w *Widget) Update() error {
	if w.perCore {
		// Per-core usage
		percentages, err := w.cpuProvider.Percent(100*time.Millisecond, true)
		if err != nil {
			return err
		}

		// Clamp to 0-100
		for i := range percentages {
			if percentages[i] < 0 {
				percentages[i] = 0
			}
			if percentages[i] > 100 {
				percentages[i] = 100
			}
		}

		w.mu.Lock()
		w.currentUsagePerCore = percentages
		w.hasData = true

		// Add to history (ring buffer handles capacity automatically)
		if w.displayMode == render.DisplayModeGraph {
			w.historyPerCore.Push(percentages)
		}
		w.mu.Unlock()
	} else {
		// Aggregate usage
		percentages, err := w.cpuProvider.Percent(100*time.Millisecond, false)
		if err != nil {
			return err
		}

		usage := 0.0
		if len(percentages) > 0 {
			usage = percentages[0]
		}

		// Clamp to 0-100
		if usage < 0 {
			usage = 0
		}
		if usage > 100 {
			usage = 100
		}

		w.mu.Lock()
		w.currentUsageSingle = usage
		w.hasData = true

		// Add to history (ring buffer handles capacity automatically)
		if w.displayMode == render.DisplayModeGraph {
			w.historySingle.Push(usage)
		}
		w.mu.Unlock()
	}

	return nil
}

// Render creates an image of the CPU widget
func (w *Widget) Render() (image.Image, error) {
	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Get content area and position
	content := w.GetContentArea()
	pos := w.GetPosition()

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.hasData {
		return img, nil
	}

	// For perCore mode, use grid strategy
	if w.perCore {
		// Determine border color
		borderColor := uint8(255)
		if w.fillColor >= 0 && w.fillColor <= 255 {
			borderColor = uint8(w.fillColor)
		}

		// Prepare grid data
		gridData := render.GridMetricData{
			Values:      w.currentUsagePerCore,
			ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
			Position:    pos,
			CoreBorder:  w.coreBorder,
			CoreMargin:  w.coreMargin,
			BorderColor: borderColor,
			FontFace:    w.fontFace,
			FontName:    w.fontName,
		}

		// For graph mode, transpose history from [time][core] to [core][time]
		if w.displayMode == render.DisplayModeGraph && w.historyPerCore.Len() >= 2 {
			historySlice := w.historyPerCore.ToSlice()
			numCores := len(historySlice[0])
			coreHistories := make([][]float64, numCores)
			for i := 0; i < numCores; i++ {
				coreHistories[i] = make([]float64, len(historySlice))
				for t, cores := range historySlice {
					if i < len(cores) {
						coreHistories[i][t] = cores[i]
					}
				}
			}
			gridData.History = coreHistories
		}

		w.gridStrategy.Render(img, gridData, w.Renderer)
		return img, nil
	}

	// For single-value mode, use strategy pattern
	w.strategy.Render(img, render.MetricData{
		Value:       w.currentUsageSingle,
		History:     w.historySingle.ToSlice(),
		TextFormat:  "%.0f",
		ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
		GaugeArea:   image.Rect(0, 0, pos.W, pos.H),
	}, w.Renderer)

	return img, nil
}
