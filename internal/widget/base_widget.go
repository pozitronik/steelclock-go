package widget

import (
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// ContentArea represents the drawable area within a widget after accounting for padding.
type ContentArea struct {
	X      int // Left offset
	Y      int // Top offset
	Width  int // Available width
	Height int // Available height
}

// BaseWidget provides common functionality for all widgets.
// Widgets should embed *BaseWidget and call NewBaseWidget() in their constructor.
//
// BaseWidget handles:
//   - Widget identification (ID/Name)
//   - Position and size (GetPosition)
//   - Style configuration (GetStyle)
//   - Update interval timing
//   - Auto-hide functionality for transient widgets
//   - Padding and content area calculation
//   - Canvas creation with background color
//   - Border drawing
//
// Usage example:
//
//	type MyWidget struct {
//	    *BaseWidget
//	    // widget-specific fields
//	}
//
//	func NewMyWidget(cfg config.WidgetConfig) (*MyWidget, error) {
//	    base := NewBaseWidget(cfg)
//	    return &MyWidget{BaseWidget: base}, nil
//	}
//
//	func (w *MyWidget) Render() (image.Image, error) {
//	    img := w.CreateCanvas()           // Creates image with background
//	    w.ApplyBorder(img)                // Draws border if configured
//	    content := w.GetContentArea()     // Gets padded content area
//	    // ... draw content using content.X, content.Y, content.Width, content.Height
//	    return img, nil
//	}
type BaseWidget struct {
	id             string
	position       config.PositionConfig
	style          config.StyleConfig
	updateInterval time.Duration
	padding        int

	// Auto-hide support
	autoHide        bool
	autoHideTimeout time.Duration
	lastTriggerTime time.Time
	autoHideMu      sync.RWMutex
}

// NewBaseWidget creates a new base widget from configuration.
// It extracts common widget settings including:
//   - ID (from cfg.ID)
//   - Position (from cfg.Position)
//   - Style (from cfg.Style, with defaults if nil)
//   - Update interval (from cfg.UpdateInterval, defaults to 1 second)
//   - Auto-hide settings (from cfg.AutoHide)
//   - Padding (from cfg.Style.Padding, defaults to 0)
func NewBaseWidget(cfg config.WidgetConfig) *BaseWidget {
	interval := cfg.UpdateInterval
	if interval == 0 {
		interval = 1.0
	}

	// Extract auto-hide settings
	autoHide := false
	autoHideTimeout := 2.0 // Default 2 seconds
	if cfg.AutoHide != nil {
		autoHide = cfg.AutoHide.Enabled
		if cfg.AutoHide.Timeout > 0 {
			autoHideTimeout = cfg.AutoHide.Timeout
		}
	}

	// Extract style (handle nil pointer)
	style := config.StyleConfig{}
	if cfg.Style != nil {
		style = *cfg.Style
	}

	// Extract padding from style
	padding := 0
	if cfg.Style != nil && cfg.Style.Padding > 0 {
		padding = cfg.Style.Padding
	}

	return &BaseWidget{
		id:              cfg.ID,
		position:        cfg.Position,
		style:           style,
		updateInterval:  time.Duration(interval * float64(time.Second)),
		padding:         padding,
		autoHide:        autoHide,
		autoHideTimeout: time.Duration(autoHideTimeout * float64(time.Second)),
		lastTriggerTime: time.Time{}, // Zero time = widget starts hidden if auto-hide enabled
	}
}

// --- Core Interface Methods ---

// Name returns the widget's unique identifier.
func (b *BaseWidget) Name() string {
	return b.id
}

// GetUpdateInterval returns how often the widget should update its data.
func (b *BaseWidget) GetUpdateInterval() time.Duration {
	return b.updateInterval
}

// GetPosition returns the widget's position and dimensions.
func (b *BaseWidget) GetPosition() config.PositionConfig {
	return b.position
}

// GetStyle returns the widget's style configuration.
func (b *BaseWidget) GetStyle() config.StyleConfig {
	return b.style
}

// --- Rendering Helpers ---

// CreateCanvas creates a new grayscale image with the widget's dimensions and background color.
// This is typically the first step in a widget's Render() method.
//
// The background color is determined by the style configuration:
//   - If Background is -1 (transparent), uses black (0) as the actual color
//   - Otherwise uses the configured Background value (0-255)
func (b *BaseWidget) CreateCanvas() *image.Gray {
	pos := b.position
	return bitmap.NewGrayscaleImage(pos.W, pos.H, b.GetRenderBackgroundColor())
}

// ApplyBorder draws a border around the canvas if border is enabled in the style.
// Border is enabled when style.Border >= 0 (the value is the border color).
// Border is disabled when style.Border < 0 (typically -1).
func (b *BaseWidget) ApplyBorder(img *image.Gray) {
	if b.style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(b.style.Border))
	}
}

// GetContentArea returns the drawable area after accounting for padding.
// Use this to determine where to draw widget content.
//
// Returns ContentArea with X, Y offsets and available Width, Height.
// If padding is 0, returns the full widget dimensions.
func (b *BaseWidget) GetContentArea() ContentArea {
	pos := b.position
	return ContentArea{
		X:      b.padding,
		Y:      b.padding,
		Width:  pos.W - b.padding*2,
		Height: pos.H - b.padding*2,
	}
}

// GetPadding returns the widget's padding value.
func (b *BaseWidget) GetPadding() int {
	return b.padding
}

// GetRenderBackgroundColor returns the background color to use when rendering.
// Handles the special case of -1 (transparent) by returning 0 (black).
// The compositor will skip black pixels for transparent widgets.
func (b *BaseWidget) GetRenderBackgroundColor() uint8 {
	if b.style.Background == -1 {
		return 0 // Use black as background for transparent widgets
	}
	return uint8(b.style.Background)
}

// --- Auto-Hide Support ---

// TriggerAutoHide marks the widget as visible and resets the auto-hide timer.
// Widgets should call this when their content changes (e.g., volume change, notification received).
// Has no effect if auto-hide is not enabled for this widget.
func (b *BaseWidget) TriggerAutoHide() {
	if !b.autoHide {
		return
	}

	b.autoHideMu.Lock()
	b.lastTriggerTime = time.Now()
	b.autoHideMu.Unlock()
}

// ShouldHide returns true if the widget should be hidden based on auto-hide settings.
// Widgets should call this in their Render() method and return nil, nil if true.
//
// Returns false if:
//   - Auto-hide is not enabled
//   - The widget was recently triggered and timeout hasn't expired
//
// Returns true if:
//   - Auto-hide is enabled AND (never triggered OR timeout expired)
func (b *BaseWidget) ShouldHide() bool {
	if !b.autoHide {
		return false
	}

	b.autoHideMu.RLock()
	defer b.autoHideMu.RUnlock()

	// If never triggered, widget stays hidden
	if b.lastTriggerTime.IsZero() {
		return true
	}

	// Check if timeout expired
	return time.Since(b.lastTriggerTime) > b.autoHideTimeout
}

// IsAutoHideEnabled returns whether auto-hide is enabled for this widget.
func (b *BaseWidget) IsAutoHideEnabled() bool {
	return b.autoHide
}

// GetAutoHideTimeout returns the auto-hide timeout duration.
func (b *BaseWidget) GetAutoHideTimeout() time.Duration {
	return b.autoHideTimeout
}

// --- Dimension Helpers ---

// Width returns the widget's width in pixels.
func (b *BaseWidget) Width() int {
	return b.position.W
}

// Height returns the widget's height in pixels.
func (b *BaseWidget) Height() int {
	return b.position.H
}

// Dimensions returns the widget's width and height.
func (b *BaseWidget) Dimensions() (width, height int) {
	return b.position.W, b.position.H
}
