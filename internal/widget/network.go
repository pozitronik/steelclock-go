package widget

import (
	"fmt"
	"image"
	"time"

	"github.com/pozitronik/steelclock/internal/bitmap"
	"github.com/pozitronik/steelclock/internal/config"
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
	barBorder     bool
	rxColor       uint8
	txColor       uint8
	historyLen    int
	lastRx        uint64
	lastTx        uint64
	lastTime      time.Time
	currentRxMbps float64
	currentTxMbps float64
	rxHistory     []float64
	txHistory     []float64
	fontFace      font.Face
}

// NewNetworkWidget creates a new network widget
func NewNetworkWidget(cfg config.WidgetConfig) (*NetworkWidget, error) {
	base := NewBaseWidget(cfg)

	displayMode := cfg.Properties.DisplayMode
	if displayMode == "" {
		displayMode = "text"
	}

	fontSize := cfg.Properties.FontSize
	if fontSize == 0 {
		fontSize = 10
	}

	horizAlign := cfg.Properties.HorizontalAlign
	if horizAlign == "" {
		horizAlign = "center"
	}

	vertAlign := cfg.Properties.VerticalAlign
	if vertAlign == "" {
		vertAlign = "center"
	}

	rxColor := cfg.Properties.RxColor
	if rxColor == 0 {
		rxColor = 255
	}

	txColor := cfg.Properties.TxColor
	if txColor == 0 {
		txColor = 255
	}

	maxSpeed := cfg.Properties.MaxSpeedMbps
	if maxSpeed == 0 {
		maxSpeed = -1 // Auto-scale
	}

	historyLen := cfg.Properties.HistoryLength
	if historyLen == 0 {
		historyLen = 30
	}

	// Load font for text mode
	var fontFace font.Face
	var err error
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(cfg.Properties.Font, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &NetworkWidget{
		BaseWidget:    base,
		displayMode:   displayMode,
		interfaceName: cfg.Properties.Interface,
		maxSpeedMbps:  maxSpeed,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       cfg.Properties.Padding,
		barBorder:     cfg.Properties.BarBorder,
		rxColor:       uint8(rxColor),
		txColor:       uint8(txColor),
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

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, uint8(style.BackgroundColor))

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
	case "bar_horizontal":
		w.renderBarHorizontal(img, contentX, contentY, contentW, contentH)
	case "bar_vertical":
		w.renderBarVertical(img, contentX, contentY, contentW, contentH)
	case "graph":
		w.renderGraph(img, contentX, contentY, contentW, contentH)
	}

	return img, nil
}

func (w *NetworkWidget) renderText(img *image.Gray) {
	text := fmt.Sprintf("↓%.1f ↑%.1f", w.currentRxMbps, w.currentTxMbps)
	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
}

func (w *NetworkWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
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
