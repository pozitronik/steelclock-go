package widget

import (
	"image"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// Widget is the interface that all widgets must implement
type Widget interface {
	// Name returns the widget's unique identifier
	Name() string

	// Update fetches new data for the widget
	Update() error

	// Render creates an image representation of the widget
	Render() (image.Image, error)

	// GetUpdateInterval returns how often the widget should update (in seconds)
	GetUpdateInterval() time.Duration

	// GetPosition returns the widget's position and size
	GetPosition() config.PositionConfig

	// GetStyle returns the widget's style configuration
	GetStyle() config.StyleConfig
}

// BaseWidget provides common functionality for all widgets
type BaseWidget struct {
	id             string
	position       config.PositionConfig
	style          config.StyleConfig
	updateInterval time.Duration

	// Auto-hide support
	autoHide        bool
	autoHideTimeout time.Duration
	lastTriggerTime time.Time
	autoHideMu      sync.RWMutex
}

// NewBaseWidget creates a new base widget
func NewBaseWidget(cfg config.WidgetConfig) *BaseWidget {
	interval := cfg.Properties.UpdateInterval
	if interval == 0 {
		interval = 1.0
	}

	autoHideTimeout := cfg.Properties.AutoHideTimeout
	if autoHideTimeout == 0 {
		autoHideTimeout = 2.0 // Default 2 seconds
	}

	return &BaseWidget{
		id:              cfg.ID,
		position:        cfg.Position,
		style:           cfg.Style,
		updateInterval:  time.Duration(interval * float64(time.Second)),
		autoHide:        cfg.Properties.AutoHide,
		autoHideTimeout: time.Duration(autoHideTimeout * float64(time.Second)),
		lastTriggerTime: time.Time{}, // Zero time = widget starts hidden if auto-hide enabled
	}
}

// Name returns the widget's ID
func (b *BaseWidget) Name() string {
	return b.id
}

// GetUpdateInterval returns the update interval
func (b *BaseWidget) GetUpdateInterval() time.Duration {
	return b.updateInterval
}

// GetPosition returns the widget's position
func (b *BaseWidget) GetPosition() config.PositionConfig {
	return b.position
}

// GetStyle returns the widget's style
func (b *BaseWidget) GetStyle() config.StyleConfig {
	return b.style
}

// TriggerAutoHide marks the widget as visible and resets the auto-hide timer
// Widgets should call this when their content changes (e.g., volume change, notification received)
func (b *BaseWidget) TriggerAutoHide() {
	if !b.autoHide {
		return
	}

	b.autoHideMu.Lock()
	b.lastTriggerTime = time.Now()
	b.autoHideMu.Unlock()
}

// ShouldHide returns true if the widget should be hidden based on auto-hide settings
// Widgets should call this in their Render() method and return nil, nil if true
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

// IsAutoHideEnabled returns whether auto-hide is enabled for this widget
func (b *BaseWidget) IsAutoHideEnabled() bool {
	return b.autoHide
}

// GetAutoHideTimeout returns the auto-hide timeout duration
func (b *BaseWidget) GetAutoHideTimeout() time.Duration {
	return b.autoHideTimeout
}
