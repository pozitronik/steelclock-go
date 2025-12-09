package widget

import (
	"fmt"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"github.com/shirou/gopsutil/v4/net"
)

func init() {
	Register("network", func(cfg config.WidgetConfig) (Widget, error) {
		return NewNetworkWidget(cfg)
	})
}

// NetworkWidget displays network I/O (RX/TX)
type NetworkWidget struct {
	*shared.BaseDualIOWidget
	base          *BaseWidget
	interfaceName *string

	// State for delta calculation
	lastRx   uint64
	lastTx   uint64
	lastTime time.Time
}

// NewNetworkWidget creates a new network widget
func NewNetworkWidget(cfg config.WidgetConfig) (*NetworkWidget, error) {
	base := NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := shared.DisplayMode(helper.GetDisplayMode(config.ModeText))
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
	case shared.DisplayModeBar:
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Rx != nil {
				rxColor = *cfg.Bar.Colors.Rx
			}
			if cfg.Bar.Colors.Tx != nil {
				txColor = *cfg.Bar.Colors.Tx
			}
		}
	case shared.DisplayModeGraph:
		if cfg.Graph != nil && cfg.Graph.Colors != nil {
			if cfg.Graph.Colors.Rx != nil {
				rxColor = *cfg.Graph.Colors.Rx
			}
			if cfg.Graph.Colors.Tx != nil {
				txColor = *cfg.Graph.Colors.Tx
			}
		}
	case shared.DisplayModeGauge:
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
	if unit != "auto" && !shared.IsValidUnit(unit) {
		unit = "Mbps" // Fallback to default
	}

	// Create byte rate converter
	converter := shared.NewByteRateConverter(unit)

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
	renderer := shared.NewDualMetricRenderer(
		shared.DualBarConfig{
			Direction:      barSettings.Direction,
			Border:         barSettings.Border,
			PrimaryColor:   rxColor,
			SecondaryColor: txColor,
		},
		shared.DualGraphConfig{
			HistoryLen:    graphSettings.HistoryLen,
			PrimaryFill:   rxColor,
			PrimaryLine:   rxColor,
			SecondaryFill: txColor,
			SecondaryLine: txColor,
		},
		shared.DualGaugeConfig{
			PrimaryArcColor:      uint8(max(0, rxColor)),
			PrimaryNeedleColor:   uint8(max(0, rxNeedleColor)),
			SecondaryArcColor:    uint8(max(0, txColor)),
			SecondaryNeedleColor: uint8(max(0, txNeedleColor)),
		},
		shared.TextConfig{
			FontFace:   fontFace,
			FontName:   textSettings.FontName,
			HorizAlign: textSettings.HorizAlign,
			VertAlign:  textSettings.VertAlign,
			Padding:    padding,
		},
	)

	// Create base dual I/O widget
	baseDualIO := shared.NewBaseDualIOWidget(shared.BaseDualIOConfig{
		Base:          base,
		DisplayMode:   displayMode,
		Padding:       padding,
		MaxSpeedBps:   maxSpeedBps,
		Unit:          unit,
		ShowUnit:      showUnit,
		SupportsGauge: true,
		TextConfig: shared.DualIOTextConfig{
			PrimaryPrefix:   "↓",
			SecondaryPrefix: "↑",
		},
		Converter:  converter,
		Renderer:   renderer,
		HistoryLen: graphSettings.HistoryLen,
	})

	return &NetworkWidget{
		BaseDualIOWidget: baseDualIO,
		base:             base,
		interfaceName:    cfg.Interface,
	}, nil
}

// Name returns the widget's ID
func (w *NetworkWidget) Name() string {
	return w.base.Name()
}

// GetUpdateInterval returns the update interval
func (w *NetworkWidget) GetUpdateInterval() time.Duration {
	return w.base.GetUpdateInterval()
}

// GetPosition returns the widget's position
func (w *NetworkWidget) GetPosition() config.PositionConfig {
	return w.base.GetPosition()
}

// GetStyle returns the widget's style
func (w *NetworkWidget) GetStyle() config.StyleConfig {
	return w.base.GetStyle()
}

// Update updates the network stats
func (w *NetworkWidget) Update() error {
	stats, err := net.IOCounters(true)
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
