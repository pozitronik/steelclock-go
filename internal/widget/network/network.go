package network

import (
	"fmt"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
	"github.com/pozitronik/steelclock-go/internal/shared"
	widgetbase "github.com/pozitronik/steelclock-go/internal/shared/base"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("network", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Widget displays network I/O (RX/TX)
type Widget struct {
	*widgetbase.DualIOWidget
	interfaceName   *string
	networkProvider metrics.NetworkProvider

	// State for delta calculation
	lastRx   uint64
	lastTx   uint64
	lastTime time.Time
}

// New creates a new network widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := render.DisplayMode(helper.GetDisplayMode(config.ModeText))
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()

	// Extract network-specific colors (rx/tx)
	rxColor := 255
	txColor := 255
	rxNeedleColor := 255
	txNeedleColor := 200

	switch displayMode {
	case render.DisplayModeBar:
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Rx != nil {
				rxColor = *cfg.Bar.Colors.Rx
			}
			if cfg.Bar.Colors.Tx != nil {
				txColor = *cfg.Bar.Colors.Tx
			}
		}
	case render.DisplayModeGraph:
		if cfg.Graph != nil && cfg.Graph.Colors != nil {
			if cfg.Graph.Colors.Rx != nil {
				rxColor = *cfg.Graph.Colors.Rx
			}
			if cfg.Graph.Colors.Tx != nil {
				txColor = *cfg.Graph.Colors.Tx
			}
		}
	case render.DisplayModeGauge:
		if cfg.Gauge != nil && cfg.Gauge.Colors != nil {
			if cfg.Gauge.Colors.Rx != nil {
				rxColor = *cfg.Gauge.Colors.Rx
			}
			if cfg.Gauge.Colors.Tx != nil {
				txColor = *cfg.Gauge.Colors.Tx
			}
			if cfg.Gauge.Colors.RxNeedle != nil {
				rxNeedleColor = *cfg.Gauge.Colors.RxNeedle
			}
			if cfg.Gauge.Colors.TxNeedle != nil {
				txNeedleColor = *cfg.Gauge.Colors.TxNeedle
			}
		}
	}

	// Max speed - convert from Mbps (config) to bytes per second (internal)
	// Config uses Mbps for backward compatibility
	maxSpeedBps := float64(-1) // Auto-scale by default
	if cfg.MaxSpeedMbps > 0 {
		// Convert Mbps to bytes per second: Mbps * 1000000 / 8
		maxSpeedBps = cfg.MaxSpeedMbps * 1000000 / 8
	}

	// Unit selection - default to "Mbps" for backward compatibility
	unit := cfg.Unit
	if unit == "" {
		unit = "Mbps"
	}
	// Validate unit
	if unit != "auto" && !util.IsValidUnit(unit) {
		unit = "Mbps" // Fallback to default
	}

	// Create byte rate converter
	converter := util.NewByteRateConverter(unit)

	// Show unit suffix in text mode
	showUnit := false
	if cfg.Text != nil && cfg.Text.ShowUnit != nil {
		showUnit = *cfg.Text.ShowUnit
	}

	// Load font for text mode
	fontFace, err := bitmap.LoadFontForTextMode(string(displayMode), textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Create dual metric renderer
	renderer := render.NewDualMetricRenderer(
		render.DualBarConfig{
			Direction:      barSettings.Direction,
			Border:         barSettings.Border,
			PrimaryColor:   rxColor,
			SecondaryColor: txColor,
		},
		render.DualGraphConfig{
			HistoryLen:    graphSettings.HistoryLen,
			PrimaryFill:   rxColor,
			PrimaryLine:   rxColor,
			SecondaryFill: txColor,
			SecondaryLine: txColor,
		},
		render.DualGaugeConfig{
			PrimaryArcColor:      uint8(max(0, rxColor)),
			PrimaryNeedleColor:   uint8(max(0, rxNeedleColor)),
			SecondaryArcColor:    uint8(max(0, txColor)),
			SecondaryNeedleColor: uint8(max(0, txNeedleColor)),
		},
		render.TextConfig{
			FontFace:   fontFace,
			FontName:   textSettings.FontName,
			HorizAlign: textSettings.HorizAlign,
			VertAlign:  textSettings.VertAlign,
			Padding:    padding,
		},
	)

	// Create base dual I/O widget
	baseDualIO := widgetbase.NewDualIOWidget(widgetbase.DualIOConfig{
		Base:          base,
		DisplayMode:   displayMode,
		Padding:       padding,
		MaxSpeedBps:   maxSpeedBps,
		Unit:          unit,
		ShowUnit:      showUnit,
		SupportsGauge: true,
		TextConfig: widgetbase.DualIOTextConfig{
			PrimaryPrefix:   "↓",
			SecondaryPrefix: "↑",
		},
		Converter:  converter,
		Renderer:   renderer,
		HistoryLen: graphSettings.HistoryLen,
	})

	return &Widget{
		DualIOWidget:    baseDualIO,
		interfaceName:   cfg.Interface,
		networkProvider: metrics.DefaultNetwork,
	}, nil
}

// Update updates the network stats
func (w *Widget) Update() error {
	stats, err := w.networkProvider.IOCounters()
	if err != nil {
		return err
	}

	// Find the interface
	var rx, tx uint64
	if w.interfaceName != nil && *w.interfaceName != "" {
		// Use specified interface
		for _, stat := range stats {
			if stat.Name == *w.interfaceName {
				rx = stat.BytesRecv
				tx = stat.BytesSent
				break
			}
		}
	} else {
		// Sum all interfaces
		for _, stat := range stats {
			rx += stat.BytesRecv
			tx += stat.BytesSent
		}
	}

	now := time.Now()

	if !w.lastTime.IsZero() {
		elapsed := now.Sub(w.lastTime).Seconds()
		if elapsed > 0 {
			// Calculate bytes per second
			rxBps := float64(rx-w.lastRx) / elapsed
			txBps := float64(tx-w.lastTx) / elapsed

			// Update base widget values
			w.SetValuesAndHistory(rxBps, txBps, w.IsGraphMode())
		}
	}

	w.lastRx = rx
	w.lastTx = tx
	w.lastTime = now

	return nil
}
