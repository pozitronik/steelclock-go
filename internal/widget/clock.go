package widget

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

func init() {
	Register("clock", func(cfg config.WidgetConfig) (Widget, error) {
		return NewClockWidget(cfg)
	})
}

// ClockWidget displays current time using various display modes
type ClockWidget struct {
	*BaseWidget
	displayMode ClockDisplayMode
	renderer    ClockRenderer
	currentTime time.Time
	mu          sync.RWMutex // Protects currentTime field
}

// NewClockWidget creates a new clock widget
func NewClockWidget(cfg config.WidgetConfig) (*ClockWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Extract display mode
	modeStr := helper.GetDisplayMode("text")
	displayMode := ClockDisplayMode(modeStr)
	// Map "clock_face" alias to "analog"
	if displayMode == "clock_face" {
		displayMode = ClockModeAnalog
	}

	// Create the appropriate renderer based on display mode
	renderer, err := createClockRenderer(cfg, displayMode, helper)
	if err != nil {
		return nil, err
	}

	return &ClockWidget{
		BaseWidget:  base,
		displayMode: displayMode,
		renderer:    renderer,
	}, nil
}

// createClockRenderer creates the appropriate renderer based on display mode
func createClockRenderer(cfg config.WidgetConfig, mode ClockDisplayMode, helper *ConfigHelper) (ClockRenderer, error) {
	switch mode {
	case ClockModeText:
		return createTextRenderer(cfg, helper)
	case ClockModeAnalog:
		return createAnalogRenderer(cfg, helper)
	case ClockModeBinary:
		return createBinaryRenderer(cfg), nil
	case ClockModeSegment:
		return createSegmentRenderer(cfg), nil
	default:
		return createTextRenderer(cfg, helper)
	}
}

// createTextRenderer creates a text mode clock renderer
func createTextRenderer(cfg config.WidgetConfig, helper *ConfigHelper) (*ClockTextRenderer, error) {
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Clock-specific: time format
	format := "15:04:05" // Default Go time format (HH:MM:SS)
	fontSize := 12
	fontName := textSettings.FontName
	if cfg.Text != nil {
		if cfg.Text.Format != "" {
			format = convertStrftimeToGo(cfg.Text.Format)
		}
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
	}

	// Load font for text mode
	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	return NewClockTextRenderer(ClockTextConfig{
		FontFace:   fontFace,
		FontName:   fontName,
		HorizAlign: textSettings.HorizAlign,
		VertAlign:  textSettings.VertAlign,
		Padding:    padding,
		Format:     format,
	}), nil
}

// createAnalogRenderer creates an analog mode clock renderer
func createAnalogRenderer(cfg config.WidgetConfig, helper *ConfigHelper) (*ClockAnalogRenderer, error) {
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Analog mode settings
	showSeconds := true
	showTicks := true
	if cfg.Analog != nil {
		showSeconds = cfg.Analog.ShowSeconds
		showTicks = cfg.Analog.ShowTicks
	}

	// Colors for analog mode (defaults to white)
	faceColor := 255
	hourColor := 255
	minuteColor := 255
	secondColor := 255
	if cfg.Analog != nil && cfg.Analog.Colors != nil {
		if cfg.Analog.Colors.Face != nil {
			faceColor = *cfg.Analog.Colors.Face
		}
		if cfg.Analog.Colors.Hour != nil {
			hourColor = *cfg.Analog.Colors.Hour
		}
		if cfg.Analog.Colors.Minute != nil {
			minuteColor = *cfg.Analog.Colors.Minute
		}
		if cfg.Analog.Colors.Second != nil {
			secondColor = *cfg.Analog.Colors.Second
		}
	}

	return NewClockAnalogRenderer(ClockAnalogConfig{
		HorizAlign:  textSettings.HorizAlign,
		VertAlign:   textSettings.VertAlign,
		Padding:     padding,
		ShowSeconds: showSeconds,
		ShowTicks:   showTicks,
		FaceColor:   faceColor,
		HourColor:   hourColor,
		MinuteColor: minuteColor,
		SecondColor: secondColor,
	}), nil
}

// createBinaryRenderer creates a binary mode clock renderer
func createBinaryRenderer(cfg config.WidgetConfig) *ClockBinaryRenderer {
	// Binary mode settings (defaults)
	binaryConfig := NewClockBinaryConfig()

	if cfg.Binary != nil {
		if cfg.Binary.Format != "" {
			binaryConfig.Format = cfg.Binary.Format
		}
		if cfg.Binary.Style != "" {
			binaryConfig.Style = cfg.Binary.Style
		}
		if cfg.Binary.Layout != "" {
			binaryConfig.Layout = cfg.Binary.Layout
		}
		binaryConfig.ShowLabels = cfg.Binary.ShowLabels
		binaryConfig.ShowHint = cfg.Binary.ShowHint
		if cfg.Binary.DotSize > 0 {
			binaryConfig.DotSize = cfg.Binary.DotSize
		}
		if cfg.Binary.DotSpacing >= 0 {
			binaryConfig.DotSpacing = cfg.Binary.DotSpacing
		}
		if cfg.Binary.DotStyle != "" {
			binaryConfig.DotStyle = cfg.Binary.DotStyle
		}
		if cfg.Binary.OnColor != nil {
			binaryConfig.OnColor = *cfg.Binary.OnColor
		}
		if cfg.Binary.OffColor != nil {
			binaryConfig.OffColor = *cfg.Binary.OffColor
		}
	}

	return NewClockBinaryRenderer(binaryConfig)
}

// createSegmentRenderer creates a segment mode clock renderer
func createSegmentRenderer(cfg config.WidgetConfig) *ClockSegmentRenderer {
	// Segment mode settings (defaults)
	segmentConfig := NewClockSegmentConfig()

	if cfg.Segment != nil {
		if cfg.Segment.Format != "" {
			segmentConfig.Format = cfg.Segment.Format
		}
		if cfg.Segment.DigitHeight > 0 {
			segmentConfig.DigitHeight = cfg.Segment.DigitHeight
		}
		if cfg.Segment.SegmentThickness > 0 {
			segmentConfig.SegmentThickness = cfg.Segment.SegmentThickness
		}
		if cfg.Segment.SegmentStyle != "" {
			segmentConfig.SegmentStyle = cfg.Segment.SegmentStyle
		}
		if cfg.Segment.DigitSpacing >= 0 {
			segmentConfig.DigitSpacing = cfg.Segment.DigitSpacing
		}
		if cfg.Segment.ColonStyle != "" {
			segmentConfig.ColonStyle = cfg.Segment.ColonStyle
		}
		if cfg.Segment.ColonBlink != nil {
			segmentConfig.ColonBlink = *cfg.Segment.ColonBlink
		}
		if cfg.Segment.OnColor != nil {
			segmentConfig.OnColor = *cfg.Segment.OnColor
		}
		if cfg.Segment.OffColor != nil {
			segmentConfig.OffColor = *cfg.Segment.OffColor
		}
		if cfg.Segment.Flip != nil {
			if cfg.Segment.Flip.Style != "" {
				segmentConfig.FlipStyle = cfg.Segment.Flip.Style
			}
			if cfg.Segment.Flip.Speed > 0 {
				segmentConfig.FlipSpeed = cfg.Segment.Flip.Speed
			}
		}
	}

	return NewClockSegmentRenderer(segmentConfig)
}

// Update updates the current time
func (w *ClockWidget) Update() error {
	w.mu.Lock()
	w.currentTime = time.Now()
	w.mu.Unlock()
	return nil
}

// Render creates an image of the clock
func (w *ClockWidget) Render() (image.Image, error) {
	// Check if time needs to be updated
	w.mu.RLock()
	isEmpty := w.currentTime.IsZero()
	currentTime := w.currentTime
	w.mu.RUnlock()

	// Update time if not set
	if isEmpty {
		if err := w.Update(); err != nil {
			return nil, fmt.Errorf("failed to update clock: %w", err)
		}
		w.mu.RLock()
		currentTime = w.currentTime
		w.mu.RUnlock()
	}

	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	// Delegate rendering to the appropriate renderer
	width, height := w.Dimensions()
	if err := w.renderer.Render(img, currentTime, 0, 0, width, height); err != nil {
		return nil, fmt.Errorf("failed to render clock: %w", err)
	}

	return img, nil
}

// NeedsUpdate returns true if the renderer needs faster refresh (e.g., during animations)
func (w *ClockWidget) NeedsUpdate() bool {
	return w.renderer.NeedsUpdate()
}
