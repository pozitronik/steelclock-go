package widget

import (
	"image"
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
}

// NewBaseWidget creates a new base widget
func NewBaseWidget(cfg config.WidgetConfig) *BaseWidget {
	interval := cfg.Properties.UpdateInterval
	if interval == 0 {
		interval = 1.0
	}

	return &BaseWidget{
		id:             cfg.ID,
		position:       cfg.Position,
		style:          cfg.Style,
		updateInterval: time.Duration(interval * float64(time.Second)),
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
