package widget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
	"github.com/shirou/gopsutil/v4/disk"
	"golang.org/x/image/font"
)

func init() {
	Register("disk", func(cfg config.WidgetConfig) (Widget, error) {
		return NewDiskWidget(cfg)
	})
}

// DiskWidget displays disk I/O
type DiskWidget struct {
	*BaseWidget
	displayMode     string
	diskName        *string
	maxSpeedBps     float64 // Max speed in bytes per second (-1 for auto-scale)
	fontSize        int
	fontName        string
	horizAlign      string
	vertAlign       string
	padding         int
	barDirection    string
	barBorder       bool
	readColor       int // -1 means transparent/no fill (skip drawing)
	writeColor      int // -1 means transparent/no fill (skip drawing)
	historyLen      int
	unit            string // "auto", "B/s", "KB/s", "MB/s", "GB/s", "KiB/s", "MiB/s", "GiB/s"
	showUnit        bool   // Show unit suffix in text mode
	converter       *shared.ByteRateConverter
	lastRead        uint64
	lastWrite       uint64
	lastTime        time.Time
	currentReadBps  float64 // Current read speed in bytes per second
	currentWriteBps float64 // Current write speed in bytes per second
	readHistory     *shared.RingBuffer[float64]
	writeHistory    *shared.RingBuffer[float64]
	fontFace        font.Face
	mu              sync.RWMutex // Protects currentReadBps, currentWriteBps, readHistory, writeHistory
}

// NewDiskWidget creates a new disk widget
func NewDiskWidget(cfg config.WidgetConfig) (*DiskWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := helper.GetDisplayMode("text")
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()

	// Extract disk-specific colors (read/write)
	readColor := 255
	writeColor := 255
	switch displayMode {
	case "bar":
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Read != nil {
				readColor = *cfg.Bar.Colors.Read
			}
			if cfg.Bar.Colors.Write != nil {
				writeColor = *cfg.Bar.Colors.Write
			}
		}
	case "graph":
		if cfg.Graph != nil && cfg.Graph.Colors != nil {
			if cfg.Graph.Colors.Read != nil {
				readColor = *cfg.Graph.Colors.Read
			}
			if cfg.Graph.Colors.Write != nil {
				writeColor = *cfg.Graph.Colors.Write
			}
		}
	}

	// Max speed - convert from MB/s (config) to B/s (internal)
	maxSpeedBps := float64(-1) // Auto-scale by default
	if cfg.MaxSpeedMbps > 0 {
		maxSpeedBps = cfg.MaxSpeedMbps * 1000000 // Convert MB/s to B/s
	}

	// Unit selection - default to "MB/s" for backward compatibility
	unit := cfg.Unit
	if unit == "" {
		unit = "MB/s"
	}
	// Validate unit
	if unit != "auto" && !shared.IsValidUnit(unit) {
		unit = "MB/s" // Fallback to default
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

	return &DiskWidget{
		BaseWidget:   base,
		displayMode:  displayMode,
		diskName:     cfg.Disk,
		maxSpeedBps:  maxSpeedBps,
		fontSize:     textSettings.FontSize,
		fontName:     textSettings.FontName,
		horizAlign:   textSettings.HorizAlign,
		vertAlign:    textSettings.VertAlign,
		padding:      padding,
		barDirection: barSettings.Direction,
		barBorder:    barSettings.Border,
		readColor:    readColor,
		writeColor:   writeColor,
		historyLen:   graphSettings.HistoryLen,
		unit:         unit,
		showUnit:     showUnit,
		converter:    converter,
		readHistory:  shared.NewRingBuffer[float64](graphSettings.HistoryLen),
		writeHistory: shared.NewRingBuffer[float64](graphSettings.HistoryLen),
		fontFace:     fontFace,
	}, nil
}

// convertToUnit converts bytes per second to the specified unit
func (w *DiskWidget) convertToUnit(bps float64, unitName string) (float64, string) {
	return w.converter.Convert(bps, unitName)
}

// formatValue formats a value with appropriate precision
func formatValue(value float64) string {
	if value >= 100 {
		return fmt.Sprintf("%.0f", value)
	} else if value >= 10 {
		return fmt.Sprintf("%.1f", value)
	}
	return fmt.Sprintf("%.2f", value)
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
			// Calculate bytes per second
			readDelta := float64(readBytes-w.lastRead) / elapsed
			writeDelta := float64(writeBytes-w.lastWrite) / elapsed

			w.mu.Lock()
			w.currentReadBps = readDelta
			w.currentWriteBps = writeDelta

			// Add to history (store raw bytes per second)
			if w.displayMode == "graph" {
				w.readHistory.Push(readDelta)
				w.writeHistory.Push(writeDelta)
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
	}

	return img, nil
}

func (w *DiskWidget) renderText(img *image.Gray) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var text string
	if w.unit == "auto" {
		// Auto-scale each value independently
		readVal, readUnit := w.converter.AutoScale(w.currentReadBps)
		writeVal, writeUnit := w.converter.AutoScale(w.currentWriteBps)

		if w.showUnit {
			text = fmt.Sprintf("R%s%s W%s%s", formatValue(readVal), readUnit, formatValue(writeVal), writeUnit)
		} else {
			// Even without show_unit, include abbreviated unit for auto mode to distinguish scales
			text = fmt.Sprintf("R%s%s W%s%s", formatValue(readVal), readUnit, formatValue(writeVal), writeUnit)
		}
	} else {
		// Fixed unit for both values
		readVal, unitName := w.convertToUnit(w.currentReadBps, w.unit)
		writeVal, _ := w.convertToUnit(w.currentWriteBps, w.unit)

		if w.showUnit {
			text = fmt.Sprintf("R%s W%s %s", formatValue(readVal), formatValue(writeVal), unitName)
		} else {
			text = fmt.Sprintf("R%s W%s", formatValue(readVal), formatValue(writeVal))
		}
	}

	bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
}

func (w *DiskWidget) renderBarHorizontal(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		// Auto-scale
		maxSpeed = max(w.currentReadBps, w.currentWriteBps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	readPercent := (w.currentReadBps / maxSpeed) * 100
	writePercent := (w.currentWriteBps / maxSpeed) * 100

	bitmap.DrawDualHorizontalBar(img, x, y, width, height, readPercent, writePercent, w.readColor, w.writeColor, w.barBorder)
}

func (w *DiskWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		maxSpeed = max(w.currentReadBps, w.currentWriteBps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	readPercent := (w.currentReadBps / maxSpeed) * 100
	writePercent := (w.currentWriteBps / maxSpeed) * 100

	bitmap.DrawDualVerticalBar(img, x, y, width, height, readPercent, writePercent, w.readColor, w.writeColor, w.barBorder)
}

func (w *DiskWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.readHistory.Len() < 2 {
		return
	}

	// Get history slices
	readData := w.readHistory.ToSlice()
	writeData := w.writeHistory.ToSlice()

	// Normalize to 0-100 scale using bytes per second
	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		// Find max in history
		maxSpeed = 1.0
		for _, v := range readData {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
		for _, v := range writeData {
			if v > maxSpeed {
				maxSpeed = v
			}
		}
	}

	readPercent := make([]float64, len(readData))
	writePercent := make([]float64, len(writeData))

	for i := range readData {
		readPercent[i] = (readData[i] / maxSpeed) * 100
		writePercent[i] = (writeData[i] / maxSpeed) * 100
	}

	// Draw both graphs overlaid (read and write)
	bitmap.DrawDualGraph(img, x, y, width, height, readPercent, writePercent, w.historyLen, w.readColor, w.readColor, w.writeColor, w.writeColor)
}
