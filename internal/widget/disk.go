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

// DiskUnit represents a disk speed unit
type DiskUnit struct {
	Name     string  // Display name (e.g., "MB/s")
	Divisor  float64 // Divisor to convert from bytes
	IsBinary bool    // True for binary units (KiB, MiB, GiB)
}

// Predefined disk units
var diskUnits = map[string]DiskUnit{
	"B/s":   {Name: "B/s", Divisor: 1, IsBinary: false},
	"KB/s":  {Name: "KB/s", Divisor: 1000, IsBinary: false},
	"MB/s":  {Name: "MB/s", Divisor: 1000000, IsBinary: false},
	"GB/s":  {Name: "GB/s", Divisor: 1000000000, IsBinary: false},
	"KiB/s": {Name: "KiB/s", Divisor: 1024, IsBinary: true},
	"MiB/s": {Name: "MiB/s", Divisor: 1048576, IsBinary: true},
	"GiB/s": {Name: "GiB/s", Divisor: 1073741824, IsBinary: true},
}

// autoScaleUnits defines the order for auto-scaling (decimal units)
var autoScaleUnitsDecimal = []string{"B/s", "KB/s", "MB/s", "GB/s"}

// autoScaleUnitsBinary defines the order for auto-scaling (binary units)
var autoScaleUnitsBinary = []string{"B/s", "KiB/s", "MiB/s", "GiB/s"}

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
	lastRead        uint64
	lastWrite       uint64
	lastTime        time.Time
	currentReadBps  float64 // Current read speed in bytes per second
	currentWriteBps float64 // Current write speed in bytes per second
	readHistory     []float64
	writeHistory    []float64
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
	if unit != "auto" {
		if _, ok := diskUnits[unit]; !ok {
			unit = "MB/s" // Fallback to default
		}
	}

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
		readHistory:  make([]float64, 0, graphSettings.HistoryLen),
		writeHistory: make([]float64, 0, graphSettings.HistoryLen),
		fontFace:     fontFace,
	}, nil
}

// convertToUnit converts bytes per second to the specified unit
func (w *DiskWidget) convertToUnit(bps float64, unitName string) (float64, string) {
	if unitName == "auto" {
		return w.autoScale(bps)
	}

	unit, ok := diskUnits[unitName]
	if !ok {
		// Fallback to MB/s
		unit = diskUnits["MB/s"]
	}

	return bps / unit.Divisor, unit.Name
}

// autoScale automatically selects the best unit based on the value
func (w *DiskWidget) autoScale(bps float64) (float64, string) {
	// Determine if we should use binary or decimal units
	// Use binary if the configured unit (when not auto) is binary
	useBinary := false
	if w.unit != "auto" {
		if u, ok := diskUnits[w.unit]; ok {
			useBinary = u.IsBinary
		}
	}

	var units []string
	if useBinary {
		units = autoScaleUnitsBinary
	} else {
		units = autoScaleUnitsDecimal
	}

	// Find the best unit (largest unit where value >= 1)
	selectedUnit := units[0]
	for _, unitName := range units {
		unit := diskUnits[unitName]
		if bps/unit.Divisor >= 1 {
			selectedUnit = unitName
		} else {
			break
		}
	}

	unit := diskUnits[selectedUnit]
	return bps / unit.Divisor, unit.Name
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
		readVal, readUnit := w.autoScale(w.currentReadBps)
		writeVal, writeUnit := w.autoScale(w.currentWriteBps)

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

	// Split into two halves: Read top, Write bottom
	halfH := height / 2

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

	// Only draw if color is not transparent (-1)
	if w.readColor >= 0 {
		bitmap.DrawHorizontalBar(img, x, y, width, halfH, readPercent, uint8(w.readColor), w.barBorder)
	}
	if w.writeColor >= 0 {
		bitmap.DrawHorizontalBar(img, x, y+halfH, width, height-halfH, writePercent, uint8(w.writeColor), w.barBorder)
	}
}

func (w *DiskWidget) renderBarVertical(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Split into two halves: Read left, Write right
	halfW := width / 2

	maxSpeed := w.maxSpeedBps
	if maxSpeed < 0 {
		maxSpeed = max(w.currentReadBps, w.currentWriteBps)
		if maxSpeed < 1 {
			maxSpeed = 1
		}
	}

	readPercent := (w.currentReadBps / maxSpeed) * 100
	writePercent := (w.currentWriteBps / maxSpeed) * 100

	// Only draw if color is not transparent (-1)
	if w.readColor >= 0 {
		bitmap.DrawVerticalBar(img, x, y, halfW, height, readPercent, uint8(w.readColor), w.barBorder)
	}
	if w.writeColor >= 0 {
		bitmap.DrawVerticalBar(img, x+halfW, y, width-halfW, height, writePercent, uint8(w.writeColor), w.barBorder)
	}
}

func (w *DiskWidget) renderGraph(img *image.Gray, x, y, width, height int) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if len(w.readHistory) < 2 {
		return
	}

	// Normalize to 0-100 scale using bytes per second
	maxSpeed := w.maxSpeedBps
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

	// Draw both graphs (Read and Write overlaid) if color is not -1 (transparent)
	// Each channel uses the same color for both fill and line
	if w.readColor >= 0 {
		bitmap.DrawGraph(img, x, y, width, height, readPercent, w.historyLen, w.readColor, w.readColor)
	}
	if w.writeColor >= 0 {
		bitmap.DrawGraph(img, x, y, width, height, writePercent, w.historyLen, w.writeColor, w.writeColor)
	}
}
