package widget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/shirou/gopsutil/v4/disk"
	"golang.org/x/image/font"
)

// DiskWidget displays disk I/O
type DiskWidget struct {
	*BaseWidget
	displayMode      string
	diskName         *string
	maxSpeedMbps     float64
	fontSize         int
	horizAlign       string
	vertAlign        string
	padding          int
	barBorder        bool
	readColor        uint8
	writeColor       uint8
	historyLen       int
	lastRead         uint64
	lastWrite        uint64
	lastTime         time.Time
	currentReadMbps  float64
	currentWriteMbps float64
	readHistory      []float64
	writeHistory     []float64
	fontFace         font.Face
	mu               sync.RWMutex // Protects currentReadMbps, currentWriteMbps, readHistory, writeHistory
}

// NewDiskWidget creates a new disk widget
func NewDiskWidget(cfg config.WidgetConfig) (*DiskWidget, error) {
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

	readColor := cfg.Properties.ReadColor
	if readColor == 0 {
		readColor = 255
	}

	writeColor := cfg.Properties.WriteColor
	if writeColor == 0 {
		writeColor = 255
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

	return &DiskWidget{
		BaseWidget:   base,
		displayMode:  displayMode,
		diskName:     cfg.Properties.DiskName,
		maxSpeedMbps: maxSpeed,
		fontSize:     fontSize,
		horizAlign:   horizAlign,
		vertAlign:    vertAlign,
		padding:      cfg.Properties.Padding,
		barBorder:    cfg.Properties.BarBorder,
		readColor:    uint8(readColor),
		writeColor:   uint8(writeColor),
		historyLen:   historyLen,
		readHistory:  make([]float64, 0, historyLen),
		writeHistory: make([]float64, 0, historyLen),
		fontFace:     fontFace,
	}, nil
}

// Update updates the disk stats
func (w *DiskWidget) Update() error {
	stats, err := disk.IOCounters()
	if err != nil {
		return err
	}

	// Find the disk
	var readBytes, writeBytes uint64
	if w.diskName != nil && *w.diskName != "" {
		// Use specified disk
		if stat, ok := stats[*w.diskName]; ok {
			readBytes = stat.ReadBytes
			writeBytes = stat.WriteBytes
		}
	} else {
		// Sum all disks
		for _, stat := range stats {
			readBytes += stat.ReadBytes
			writeBytes += stat.WriteBytes
		}
	}

	now := time.Now()

	if !w.lastTime.IsZero() {
		elapsed := now.Sub(w.lastTime).Seconds()
		if elapsed > 0 {
			// Calculate MBps
			readDelta := float64(readBytes-w.lastRead) / 1000000 / elapsed
			writeDelta := float64(writeBytes-w.lastWrite) / 1000000 / elapsed

			w.mu.Lock()
			w.currentReadMbps = readDelta
			w.currentWriteMbps = writeDelta

			// Add to history
			if w.displayMode == "graph" {
				w.readHistory = append(w.readHistory, readDelta)
				if len(w.readHistory) > w.historyLen {
					w.readHistory = w.readHistory[1:]
				}

				w.writeHistory = append(w.writeHistory, writeDelta)
				if len(w.writeHistory) > w.historyLen {
					w.writeHistory = w.writeHistory[1:]
				}
			}
			w.mu.Unlock()
		}
	}

	w.lastRead = readBytes
	w.lastWrite = writeBytes
	w.lastTime = now

	return nil
}

// Render creates an image of the disk widget
func (w *DiskWidget) Render() (image.Image, error) {
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
	case "bar_horizontal":
		w.renderBarHorizontal(img, contentX, contentY, contentW, contentH)
	case "bar_vertical":
		w.renderBarVertical(img, contentX, contentY, contentW, contentH)
	case "graph":
		w.renderGraph(img, contentX, contentY, contentW, contentH)
	}

	return img, nil
}

func (w *DiskWidget) renderText(img *image.Gray) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	text := fmt.Sprintf("R%.1f W%.1f", w.currentReadMbps, w.currentWriteMbps)
	bitmap.DrawAlignedText(img, text, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
}

func (w *DiskWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Split into two halves: Read top, Write bottom
	halfH := height / 2

	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.currentReadMbps, w.currentWriteMbps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	readPercent := (w.currentReadMbps / maxSpeed) * 100
	writePercent := (w.currentWriteMbps / maxSpeed) * 100

	bitmap.DrawHorizontalBar(img, x, y, width, halfH, readPercent, w.readColor, w.barBorder)
	bitmap.DrawHorizontalBar(img, x, y+halfH, width, height-halfH, writePercent, w.writeColor, w.barBorder)
}

func (w *DiskWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Split into two halves: Read left, Write right
	halfW := width / 2

	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		maxSpeed = max(w.currentReadMbps, w.currentWriteMbps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	readPercent := (w.currentReadMbps / maxSpeed) * 100
	writePercent := (w.currentWriteMbps / maxSpeed) * 100

	bitmap.DrawVerticalBar(img, x, y, halfW, height, readPercent, w.readColor, w.barBorder)
	bitmap.DrawVerticalBar(img, x+halfW, y, width-halfW, height, writePercent, w.writeColor, w.barBorder)
}

func (w *DiskWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.readHistory) < 2 {
		return
	}

	// Normalize to 0-100 scale
	maxSpeed := w.maxSpeedMbps
	if maxSpeed < 0 {
		// Find max in history
		maxSpeed = 1.0
		for _, v := range w.readHistory {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
		for _, v := range w.writeHistory {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
	}

	readPercent := make([]float64, len(w.readHistory))
	writePercent := make([]float64, len(w.writeHistory))

	for i := range w.readHistory {
		readPercent[i] = (w.readHistory[i] / maxSpeed) * 100
		writePercent[i] = (w.writeHistory[i] / maxSpeed) * 100
	}

	// Draw both graphs (Read and Write overlaid)
	bitmap.DrawGraph(img, x, y, width, height, readPercent, w.historyLen, w.readColor)
	bitmap.DrawGraph(img, x, y, width, height, writePercent, w.historyLen, w.writeColor)
}
