package widget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/shirou/gopsutil/v4/net"
	"golang.org/x/image/font"
)

// NetworkWidget displays network I/O
type NetworkWidget struct {
	*BaseWidget
	displayMode   string
	interfaceName *string
	maxSpeedMbps  float64
	fontSize      int
	horizAlign    string
	vertAlign     string
	padding       int
	barDirection  string
	barBorder     bool
	rxColor       uint8
	txColor       uint8
	rxNeedleColor uint8
	txNeedleColor uint8
	historyLen    int
	lastRx        uint64
	lastTx        uint64
	lastTime      time.Time
	currentRxMbps float64
	currentTxMbps float64
	rxHistory     []float64
	txHistory     []float64
	fontFace      font.Face
	mu            sync.RWMutex // Protects currentRxMbps, currentTxMbps, rxHistory, txHistory
}

// NewNetworkWidget creates a new network widget
func NewNetworkWidget(cfg config.WidgetConfig) (*NetworkWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Mode
	if displayMode == "" {
		displayMode = "text"
	}

	// Extract text settings
	fontSize := 10
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
		fontName = cfg.Text.Font
		if cfg.Text.Align != nil {
			if cfg.Text.Align.H != "" {
				horizAlign = cfg.Text.Align.H
			}
			if cfg.Text.Align.V != "" {
				vertAlign = cfg.Text.Align.V
			}
		}
	}

	// Extract padding from style
	if cfg.Style != nil {
		padding = cfg.Style.Padding
	}

	// Extract colors
	rxColor := 255
	txColor := 255
	rxNeedleColor := 255
	txNeedleColor := 200
	if cfg.Colors != nil {
		if cfg.Colors.Rx != nil {
			rxColor = *cfg.Colors.Rx
		}
		if cfg.Colors.Tx != nil {
			txColor = *cfg.Colors.Tx
		}
		if cfg.Colors.RxNeedle != nil {
			rxNeedleColor = *cfg.Colors.RxNeedle
		}
		if cfg.Colors.TxNeedle != nil {
			txNeedleColor = *cfg.Colors.TxNeedle
		}
	}

	// Max speed
	maxSpeed := cfg.MaxSpeedMbps
	if maxSpeed == 0 {
		maxSpeed = -1 // Auto-scale
	}

	// Extract graph settings
	historyLen := 30
	if cfg.Graph != nil && cfg.Graph.History > 0 {
		historyLen = cfg.Graph.History
	}

	// Extract bar settings
	barDirection := "horizontal"
	barBorder := false
	if cfg.Bar != nil {
		if cfg.Bar.Direction != "" {
			barDirection = cfg.Bar.Direction
		}
		barBorder = cfg.Bar.Border
	}

	// Load font for text mode
	var fontFace font.Face
	var err error
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(fontName, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &NetworkWidget{
		BaseWidget:    base,
		displayMode:   displayMode,
		interfaceName: cfg.Interface,
		maxSpeedMbps:  maxSpeed,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       padding,
		barDirection:  barDirection,
		barBorder:     barBorder,
		rxColor:       uint8(rxColor),
		txColor:       uint8(txColor),
		rxNeedleColor: uint8(rxNeedleColor),
		txNeedleColor: uint8(txNeedleColor),
		historyLen:    historyLen,
		rxHistory:     make([]float64, 0, historyLen),
		txHistory:     make([]float64, 0, historyLen),
		fontFace:      fontFace,
	}, nil
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
			// Calculate Mbps
			rxDelta := float64(rx-w.lastRx) * 8 / 1000000 / elapsed // bits to Mbps
			txDelta := float64(tx-w.lastTx) * 8 / 1000000 / elapsed

			w.mu.Lock()
			w.currentRxMbps = rxDelta
			w.currentTxMbps = txDelta

			// Add to history
			if w.displayMode == "graph" {
				w.rxHistory = append(w.rxHistory, rxDelta)
				if len(w.rxHistory) > w.historyLen {
					w.rxHistory = w.rxHistory[1:]
				}

				w.txHistory = append(w.txHistory, txDelta)
				if len(w.txHistory) > w.historyLen {
					w.txHistory = w.txHistory[1:]
				}
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

	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
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

	text := fmt.Sprintf("↓%.1f ↑%.1f", w.currentRxMbps, w.currentTxMbps)
	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
}

func (w *NetworkWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Split into two halves: RX top, TX bottom
	halfH := height / 2

	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.currentRxMbps, w.currentTxMbps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	rxPercent := (w.currentRxMbps / maxSpeed) * 100
	txPercent := (w.currentTxMbps / maxSpeed) * 100

	bitmap.DrawHorizontalBar(img, x, y, width, halfH, rxPercent, w.rxColor, w.barBorder)
	bitmap.DrawHorizontalBar(img, x, y+halfH, width, height-halfH, txPercent, w.txColor, w.barBorder)
}

func (w *NetworkWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Split into two halves: RX left, TX right
	halfW := width / 2

	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		maxSpeed = max(w.currentRxMbps, w.currentTxMbps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	rxPercent := (w.currentRxMbps / maxSpeed) * 100
	txPercent := (w.currentTxMbps / maxSpeed) * 100

	bitmap.DrawVerticalBar(img, x, y, halfW, height, rxPercent, w.rxColor, w.barBorder)
	bitmap.DrawVerticalBar(img, x+halfW, y, width-halfW, height, txPercent, w.txColor, w.barBorder)
}

func (w *NetworkWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.rxHistory) < 2 {
		return
	}

	// Normalize to 0-100 scale
	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		// Find max in history
		maxSpeed = 1.0
		for _, v := range w.rxHistory {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
		for _, v := range w.txHistory {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
	}

	rxPercent := make([]float64, len(w.rxHistory))
	txPercent := make([]float64, len(w.txHistory))

	for i := range w.rxHistory {
		rxPercent[i] = (w.rxHistory[i] / maxSpeed) * 100
		txPercent[i] = (w.txHistory[i] / maxSpeed) * 100
	}

	// Draw both graphs (RX and TX overlaid)
	bitmap.DrawGraph(img, x, y, width, height, rxPercent, w.historyLen, w.rxColor)
	bitmap.DrawGraph(img, x, y, width, height, txPercent, w.historyLen, w.txColor)
}

func (w *NetworkWidget) renderGauge(img *image.Gray, pos config.PositionConfig) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Calculate max speed for percentage
	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.currentRxMbps, w.currentTxMbps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	// Calculate percentages
	rxPercent := (w.currentRxMbps / maxSpeed) * 100
	txPercent := (w.currentTxMbps / maxSpeed) * 100

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
	bitmap.DrawDualGauge(img, pos, rxPercent, txPercent, w.rxColor, w.rxNeedleColor, w.txColor, w.txNeedleColor)
}
