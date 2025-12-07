package widget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"github.com/shirou/gopsutil/v4/net"
	"golang.org/x/image/font"
)

// NetworkWidget displays network I/O
type NetworkWidget struct {
	*BaseWidget
	displayMode   string
	interfaceName *string
	maxSpeedBps   float64 // Max speed in bytes per second (-1 for auto-scale)
	fontSize      int
	fontName      string
	horizAlign    string
	vertAlign     string
	padding       int
	barDirection  string
	barBorder     bool
	rxColor       int // -1 means transparent/no fill (skip drawing)
	txColor       int // -1 means transparent/no fill (skip drawing)
	rxNeedleColor int // -1 means transparent (skip drawing)
	txNeedleColor int // -1 means transparent (skip drawing)
	historyLen    int
	unit          string // "auto", "bps", "Kbps", "Mbps", "Gbps", "B/s", "KB/s", "MB/s", "GB/s", "KiB/s", "MiB/s", "GiB/s"
	showUnit      bool   // Show unit suffix in text mode
	converter     *shared.ByteRateConverter
	lastRx        uint64
	lastTx        uint64
	lastTime      time.Time
	currentRxBps  float64 // Current RX speed in bytes per second
	currentTxBps  float64 // Current TX speed in bytes per second
	rxHistory     *shared.RingBuffer[float64]
	txHistory     *shared.RingBuffer[float64]
	fontFace      font.Face
	mu            sync.RWMutex // Protects currentRxBps, currentTxBps, rxHistory, txHistory
}

// NewNetworkWidget creates a new network widget
func NewNetworkWidget(cfg config.WidgetConfig) (*NetworkWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := helper.GetDisplayMode("text")
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
	case "bar":
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Rx != nil {
				rxColor = *cfg.Bar.Colors.Rx
			}
			if cfg.Bar.Colors.Tx != nil {
				txColor = *cfg.Bar.Colors.Tx
			}
		}
	case "graph":
		if cfg.Graph != nil && cfg.Graph.Colors != nil {
			if cfg.Graph.Colors.Rx != nil {
				rxColor = *cfg.Graph.Colors.Rx
			}
			if cfg.Graph.Colors.Tx != nil {
				txColor = *cfg.Graph.Colors.Tx
			}
		}
	case "gauge":
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
	fontFace, err := helper.LoadFontForTextMode(displayMode)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return &NetworkWidget{
		BaseWidget:    base,
		displayMode:   displayMode,
		interfaceName: cfg.Interface,
		maxSpeedBps:   maxSpeedBps,
		fontSize:      textSettings.FontSize,
		fontName:      textSettings.FontName,
		horizAlign:    textSettings.HorizAlign,
		vertAlign:     textSettings.VertAlign,
		padding:       padding,
		barDirection:  barSettings.Direction,
		barBorder:     barSettings.Border,
		rxColor:       rxColor,
		txColor:       txColor,
		rxNeedleColor: rxNeedleColor,
		txNeedleColor: txNeedleColor,
		historyLen:    graphSettings.HistoryLen,
		unit:          unit,
		showUnit:      showUnit,
		converter:     converter,
		rxHistory:     shared.NewRingBuffer[float64](graphSettings.HistoryLen),
		txHistory:     shared.NewRingBuffer[float64](graphSettings.HistoryLen),
		fontFace:      fontFace,
	}, nil
}

// convertToUnit converts bytes per second to the specified unit
func (w *NetworkWidget) convertToUnit(bps float64, unitName string) (float64, string) {
	return w.converter.Convert(bps, unitName)
}

// formatNetValue formats a value with appropriate precision
func formatNetValue(value float64) string {
	if value >= 100 {
		return fmt.Sprintf("%.0f", value)
	} else if value >= 10 {
		return fmt.Sprintf("%.1f", value)
	}
	return fmt.Sprintf("%.2f", value)
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
			rxDelta := float64(rx-w.lastRx) / elapsed
			txDelta := float64(tx-w.lastTx) / elapsed

			w.mu.Lock()
			w.currentRxBps = rxDelta
			w.currentTxBps = txDelta

			// Add to history (store raw bytes per second)
			if w.displayMode == "graph" {
				w.rxHistory.Push(rxDelta)
				w.txHistory.Push(txDelta)
			}
			w.mu.Unlock()
		}
	}

	w.lastRx = rx
	w.lastTx = tx
	w.lastTime = now

	return nil
}

// Render creates an image of the network widget
func (w *NetworkWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	contentX := w.padding
	contentY := w.padding
	contentW := pos.W - w.padding*2
	contentH := pos.H - w.padding*2

	switch w.displayMode {
	case "text":
		w.renderText(img)
	case "bar":
		if w.barDirection == "vertical" {
			w.renderBarVertical(img, contentX, contentY, contentW, contentH)
		} else {
			w.renderBarHorizontal(img, contentX, contentY, contentW, contentH)
		}
	case "graph":
		w.renderGraph(img, contentX, contentY, contentW, contentH)
	case "gauge":
		w.renderGauge(img, pos)
	}

	return img, nil
}

func (w *NetworkWidget) renderText(img *image.Gray) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var text string
	if w.unit == "auto" {
		// Auto-scale each value independently
		rxVal, rxUnit := w.converter.AutoScale(w.currentRxBps)
		txVal, txUnit := w.converter.AutoScale(w.currentTxBps)

		// Always show unit for auto mode to distinguish scales
		text = fmt.Sprintf("↓%s%s ↑%s%s", formatNetValue(rxVal), rxUnit, formatNetValue(txVal), txUnit)
	} else {
		// Fixed unit for both values
		rxVal, unitName := w.convertToUnit(w.currentRxBps, w.unit)
		txVal, _ := w.convertToUnit(w.currentTxBps, w.unit)

		if w.showUnit {
			text = fmt.Sprintf("↓%s ↑%s %s", formatNetValue(rxVal), formatNetValue(txVal), unitName)
		} else {
			text = fmt.Sprintf("↓%s ↑%s", formatNetValue(rxVal), formatNetValue(txVal))
		}
	}

	bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
}

func (w *NetworkWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.currentRxBps, w.currentTxBps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	rxPercent := (w.currentRxBps / maxSpeed) * 100
	txPercent := (w.currentTxBps / maxSpeed) * 100

	bitmap.DrawDualHorizontalBar(img, x, y, width, height, rxPercent, txPercent, w.rxColor, w.txColor, w.barBorder)
}

func (w *NetworkWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		maxSpeed = max(w.currentRxBps, w.currentTxBps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	rxPercent := (w.currentRxBps / maxSpeed) * 100
	txPercent := (w.currentTxBps / maxSpeed) * 100

	bitmap.DrawDualVerticalBar(img, x, y, width, height, rxPercent, txPercent, w.rxColor, w.txColor, w.barBorder)
}

func (w *NetworkWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.rxHistory.Len() < 2 {
		return
	}

	// Get history slices
	rxData := w.rxHistory.ToSlice()
	txData := w.txHistory.ToSlice()

	// Normalize to 0-100 scale using bytes per second
	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		// Find max in history
		maxSpeed = 1.0
		for _, v := range rxData {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
		for _, v := range txData {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
	}

	rxPercent := make([]float64, len(rxData))
	txPercent := make([]float64, len(txData))

	for i := range rxData {
		rxPercent[i] = (rxData[i] / maxSpeed) * 100
		txPercent[i] = (txData[i] / maxSpeed) * 100
	}

	// Draw both graphs overlaid (RX and TX)
	bitmap.DrawDualGraph(img, x, y, width, height, rxPercent, txPercent, w.historyLen, w.rxColor, w.rxColor, w.txColor, w.txColor)
}

func (w *NetworkWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Calculate max speed for percentage
	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.currentRxBps, w.currentTxBps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	// Calculate percentages
	rxPercent := (w.currentRxBps / maxSpeed) * 100
	txPercent := (w.currentTxBps / maxSpeed) * 100

	// Clamp to 0-100
	if rxPercent < 0 {
		rxPercent = 0
	}
	if rxPercent > 100 {
		rxPercent = 100
	}
	if txPercent < 0 {
		txPercent = 0
	}
	if txPercent > 100 {
		txPercent = 100
	}

	// Draw dual gauge: outer (RX) and inner (TX)
	// Convert colors, treating -1 as 0 (invisible on black)
	rxCol := uint8(0)
	rxNeedleCol := uint8(0)
	txCol := uint8(0)
	txNeedleCol := uint8(0)
	if w.rxColor >= 0 {
		rxCol = uint8(w.rxColor)
	}
	if w.rxNeedleColor >= 0 {
		rxNeedleCol = uint8(w.rxNeedleColor)
	}
	if w.txColor >= 0 {
		txCol = uint8(w.txColor)
	}
	if w.txNeedleColor >= 0 {
		txNeedleCol = uint8(w.txNeedleColor)
	}
	bitmap.DrawDualGauge(img, pos, rxPercent, txPercent, rxCol, rxNeedleCol, txCol, txNeedleCol)
}
