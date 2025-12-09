package widget

import (
	"fmt"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/metrics"
	"github.com/pozitronik/steelclock-go/internal/widget/shared"
)

func init() {
	Register("disk", func(cfg config.WidgetConfig) (Widget, error) {
		return NewDiskWidget(cfg)
	})
}

// DiskWidget displays disk I/O (Read/Write)
type DiskWidget struct {
	*shared.BaseDualIOWidget
	base         *BaseWidget
	diskName     *string
	diskProvider metrics.DiskProvider

	// State for delta calculation
	lastRead  uint64
	lastWrite uint64
	lastTime  time.Time
}

// NewDiskWidget creates a new disk widget
func NewDiskWidget(cfg config.WidgetConfig) (*DiskWidget, error) {
	base := NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract common settings using helper
	displayMode := shared.DisplayMode(helper.GetDisplayMode(config.ModeText))
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()

	// Extract disk-specific colors (read/write)
	readColor := 255
	writeColor := 255

	switch displayMode {
	case shared.DisplayModeBar:
		if cfg.Bar != nil && cfg.Bar.Colors != nil {
			if cfg.Bar.Colors.Read != nil {
				readColor = *cfg.Bar.Colors.Read
			}
			if cfg.Bar.Colors.Write != nil {
				writeColor = *cfg.Bar.Colors.Write
			}
		}
	case shared.DisplayModeGraph:
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
	fontFace, err := bitmap.LoadFontForTextMode(string(displayMode), textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Create dual metric renderer (disk doesn't use gauge mode)
	renderer := shared.NewDualMetricRenderer(
		shared.DualBarConfig{
			Direction:      barSettings.Direction,
			Border:         barSettings.Border,
			PrimaryColor:   readColor,
			SecondaryColor: writeColor,
		},
		shared.DualGraphConfig{
			HistoryLen:    graphSettings.HistoryLen,
			PrimaryFill:   readColor,
			PrimaryLine:   readColor,
			SecondaryFill: writeColor,
			SecondaryLine: writeColor,
		},
		shared.DualGaugeConfig{}, // Not used for disk
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
		SupportsGauge: false, // Disk doesn't support gauge mode
		TextConfig: shared.DualIOTextConfig{
			PrimaryPrefix:   "R",
			SecondaryPrefix: "W",
		},
		Converter:  converter,
		Renderer:   renderer,
		HistoryLen: graphSettings.HistoryLen,
	})

	return &DiskWidget{
		BaseDualIOWidget: baseDualIO,
		base:             base,
		diskName:         cfg.Disk,
		diskProvider:     metrics.DefaultDisk,
	}, nil
}

// Name returns the widget's ID
func (w *DiskWidget) Name() string {
	return w.base.Name()
}

// GetUpdateInterval returns the update interval
func (w *DiskWidget) GetUpdateInterval() time.Duration {
	return w.base.GetUpdateInterval()
}

// GetPosition returns the widget's position
func (w *DiskWidget) GetPosition() config.PositionConfig {
	return w.base.GetPosition()
}

// GetStyle returns the widget's style
func (w *DiskWidget) GetStyle() config.StyleConfig {
	return w.base.GetStyle()
}

// Update updates the disk stats
func (w *DiskWidget) Update() error {
	stats, err := w.diskProvider.IOCounters()
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
			readBps := float64(readBytes-w.lastRead) / elapsed
			writeBps := float64(writeBytes-w.lastWrite) / elapsed

			// Update base widget values
			w.SetValuesAndHistory(readBps, writeBps, w.IsGraphMode())
		}
	}

	w.lastRead = readBytes
	w.lastWrite = writeBytes
	w.lastTime = now

	return nil
}
