package widget

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// ClockWidget displays current time
type ClockWidget struct {
	*BaseWidget
	format      string
	fontSize    int
	fontName    string
	horizAlign  string
	vertAlign   string
	padding     int
	displayMode string
	currentTime string
	mu          sync.RWMutex // Protects currentTime field
	fontFace    font.Face
	// Analog mode settings
	showSeconds bool
	showTicks   bool
	// Colors for analog mode
	faceColor   int
	hourColor   int
	minuteColor int
	secondColor int
}

// NewClockWidget creates a new clock widget
func NewClockWidget(cfg config.WidgetConfig) (*ClockWidget, error) {
	base := NewBaseWidget(cfg)

	// Extract text settings (with defaults from loader)
	format := "15:04:05" // Default Go time format (HH:MM:SS)
	fontSize := 12
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Format != "" {
			// Convert Python strftime to Go format
			format = convertStrftimeToGo(cfg.Text.Format)
		}
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

	displayMode := cfg.Mode
	if displayMode == "" {
		displayMode = "text" // Default to text display
	}

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
	if cfg.Colors != nil {
		if cfg.Colors.Face != nil {
			faceColor = *cfg.Colors.Face
		}
		if cfg.Colors.Hour != nil {
			hourColor = *cfg.Colors.Hour
		}
		if cfg.Colors.Minute != nil {
			minuteColor = *cfg.Colors.Minute
		}
		if cfg.Colors.Second != nil {
			secondColor = *cfg.Colors.Second
		}
	}

	// Load font only for text mode
	var fontFace font.Face
	var err error
	if displayMode == "text" {
		fontFace, err = bitmap.LoadFont(fontName, fontSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load font: %w", err)
		}
	}

	return &ClockWidget{
		BaseWidget:  base,
		format:      format,
		fontSize:    fontSize,
		fontName:    fontName,
		horizAlign:  horizAlign,
		vertAlign:   vertAlign,
		padding:     padding,
		displayMode: displayMode,
		fontFace:    fontFace,
		showSeconds: showSeconds,
		showTicks:   showTicks,
		faceColor:   faceColor,
		hourColor:   hourColor,
		minuteColor: minuteColor,
		secondColor: secondColor,
	}, nil
}

// Update updates the current time
func (w *ClockWidget) Update() error {
	w.mu.Lock()
	w.currentTime = time.Now().Format(w.format)
	w.mu.Unlock()
	return nil
}

// Render creates an image of the clock
func (w *ClockWidget) Render() (image.Image, error) {
	// Check if time needs to be updated
	w.mu.RLock()
	isEmpty := w.currentTime == ""
	w.mu.RUnlock()

	// Update time if not set
	if isEmpty {
		if err := w.Update(); err != nil {
			return nil, fmt.Errorf("failed to update clock: %w", err)
		}
	}

	pos := w.GetPosition()
	style := w.GetStyle()

	// Create image with background
	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	// Draw border if enabled
	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	// Render based on display mode
	switch w.displayMode {
	case "analog", "clock_face":
		w.renderClockFace(img)
	default: // "text"
		// Get current time with read lock and copy to local variable
		w.mu.RLock()
		timeStr := w.currentTime
		w.mu.RUnlock()
		bitmap.DrawAlignedText(img, timeStr, w.fontFace, w.horizAlign, w.vertAlign, w.padding)
	}

	return img, nil
}

// renderClockFace draws an analog clock face with hour, minute, and second hands
//
//nolint:gocyclo // Complex geometric calculations for clock face rendering
func (w *ClockWidget) renderClockFace(img *image.Gray) {
	pos := w.GetPosition()

	// Calculate maximum radius that fits within widget bounds
	maxRadius := pos.H / 2
	if pos.W/2 < maxRadius {
		maxRadius = pos.W / 2
	}
	radius := maxRadius - w.padding - 2 // Account for padding and edge margin

	if radius < 5 {
		return // Too small to draw
	}

	// Calculate center position based on alignment
	var centerX, centerY int

	// Horizontal alignment
	switch w.horizAlign {
	case "left":
		centerX = radius + w.padding + 2
	case "right":
		centerX = pos.W - radius - w.padding - 2
	default: // "center"
		centerX = pos.W / 2
	}

	// Vertical alignment
	switch w.vertAlign {
	case "top":
		centerY = radius + w.padding + 2
	case "bottom":
		centerY = pos.H - radius - w.padding - 2
	default: // "center"
		centerY = pos.H / 2
	}

	faceC := color.Gray{Y: uint8(w.faceColor)}
	hourC := color.Gray{Y: uint8(w.hourColor)}
	minuteC := color.Gray{Y: uint8(w.minuteColor)}
	secondC := color.Gray{Y: uint8(w.secondColor)}

	// Draw clock face circle
	bitmap.DrawCircle(img, centerX, centerY, radius, faceC)

	// Draw hour markers if enabled
	if w.showTicks {
		for hour := 0; hour < 12; hour++ {
			angle := float64(hour) * 30.0 // 30 degrees per hour
			rad := (angle - 90.0) * math.Pi / 180.0

			// Outer point on circle
			x1 := centerX + int(float64(radius)*math.Cos(rad))
			y1 := centerY + int(float64(radius)*math.Sin(rad))

			// Inner point (tick mark length)
			tickLen := 2
			if hour%3 == 0 {
				tickLen = 4 // Longer ticks at 12, 3, 6, 9
			}
			x2 := centerX + int(float64(radius-tickLen)*math.Cos(rad))
			y2 := centerY + int(float64(radius-tickLen)*math.Sin(rad))

			bitmap.DrawLine(img, x1, y1, x2, y2, faceC)
		}
	}

	// Get current time
	now := time.Now()
	hour := now.Hour() % 12
	minute := now.Minute()
	second := now.Second()

	// Calculate hand angles (in degrees, 0 = 12 o'clock, clockwise)
	// Subtract 90 to convert from 0=3 o'clock to 0=12 o'clock
	hourAngle := (float64(hour)*30.0 + float64(minute)*0.5 - 90.0) * math.Pi / 180.0
	minuteAngle := (float64(minute)*6.0 + float64(second)*0.1 - 90.0) * math.Pi / 180.0
	secondAngle := (float64(second)*6.0 - 90.0) * math.Pi / 180.0

	// Draw hour hand (short and thick)
	hourLen := int(float64(radius) * 0.5)
	hourX := centerX + int(float64(hourLen)*math.Cos(hourAngle))
	hourY := centerY + int(float64(hourLen)*math.Sin(hourAngle))
	bitmap.DrawLine(img, centerX, centerY, hourX, hourY, hourC)

	// Draw minute hand (medium length)
	minuteLen := int(float64(radius) * 0.75)
	minuteX := centerX + int(float64(minuteLen)*math.Cos(minuteAngle))
	minuteY := centerY + int(float64(minuteLen)*math.Sin(minuteAngle))
	bitmap.DrawLine(img, centerX, centerY, minuteX, minuteY, minuteC)

	// Draw second hand (long and thin) if enabled
	if w.showSeconds {
		secondLen := int(float64(radius) * 0.9)
		secondX := centerX + int(float64(secondLen)*math.Cos(secondAngle))
		secondY := centerY + int(float64(secondLen)*math.Sin(secondAngle))
		bitmap.DrawLine(img, centerX, centerY, secondX, secondY, secondC)
	}

	// Draw center dot
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if centerX+dx >= 0 && centerX+dx < pos.W && centerY+dy >= 0 && centerY+dy < pos.H {
				img.Set(centerX+dx, centerY+dy, faceC)
			}
		}
	}
}

// convertStrftimeToGo converts Python strftime format to Go time format
// This is a simplified converter for common formats
func convertStrftimeToGo(strftime string) string {
	// Map common strftime patterns to Go format
	replacements := map[string]string{
		"%H:%M:%S": "15:04:05",
		"%H:%M":    "15:04",
		"%Y-%m-%d": "2006-01-02",
		"%d.%m.%Y": "02.01.2006",
		"%Y":       "2006",
		"%m":       "01",
		"%d":       "02",
		"%H":       "15",
		"%M":       "04",
		"%S":       "05",
	}

	result := strftime
	for old, goFmt := range replacements {
		if result == old {
			return goFmt
		}
	}

	// If no exact match found, try common patterns
	// For more complex formats, users should use Go format directly
	return result
}
