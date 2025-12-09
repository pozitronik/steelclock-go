package widget

import (
	"image"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// Widget is the interface that all widgets must implement.
// All widgets should embed *BaseWidget to get common functionality.
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

// Stoppable is an optional interface for widgets that need cleanup on shutdown.
// Widgets with goroutines, file watchers, or subscriptions should implement this.
type Stoppable interface {
	Stop()
}

// StopWidget calls Stop() on the widget if it implements Stoppable.
// Safe to call on any widget - does nothing if widget doesn't implement Stoppable.
func StopWidget(w Widget) {
	if s, ok := w.(Stoppable); ok {
		s.Stop()
	}
}

// StopWidgets calls Stop() on all widgets that implement Stoppable.
func StopWidgets(widgets []Widget) {
	for _, w := range widgets {
		StopWidget(w)
	}
}
