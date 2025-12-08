package app

import (
	"fmt"

	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// WidgetManager handles widget creation and compositor setup.
// It consolidates the logic for creating widgets, layout managers, and compositors.
type WidgetManager struct{}

// NewWidgetManager creates a new widget manager.
func NewWidgetManager() *WidgetManager {
	return &WidgetManager{}
}

// CompositorSetup contains all components needed to run a compositor.
type CompositorSetup struct {
	Compositor *compositor.Compositor
	Widgets    []widget.Widget
	Layout     *layout.Manager
}

// CreateFromConfig creates widgets and compositor from configuration.
// Returns NoWidgetsError if no widgets are enabled in the config.
func (m *WidgetManager) CreateFromConfig(client gamesense.API, cfg *config.Config) (*CompositorSetup, error) {
	widgets, err := widget.CreateWidgets(cfg.Widgets)
	if err != nil {
		return nil, fmt.Errorf("failed to create widgets: %w", err)
	}

	if len(widgets) == 0 {
		return nil, &NoWidgetsError{}
	}

	return m.createSetup(client, widgets, cfg.Display, cfg), nil
}

// CreateErrorDisplay creates a compositor setup for displaying an error message.
func (m *WidgetManager) CreateErrorDisplay(client gamesense.API, message string, width, height int) *CompositorSetup {
	errorWidget := widget.NewErrorWidget(width, height, message)
	widgets := []widget.Widget{errorWidget}

	displayCfg := config.DisplayConfig{
		Width:      width,
		Height:     height,
		Background: 0,
	}

	errorCfg := &config.Config{
		GameName:        config.DefaultGameName,
		GameDisplayName: config.DefaultGameDisplay,
		RefreshRateMs:   ErrorDisplayRefreshRateMs,
		Display:         displayCfg,
		Widgets:         []config.WidgetConfig{},
	}

	return m.createSetup(client, widgets, displayCfg, errorCfg)
}

// createSetup creates the compositor setup with the given components.
func (m *WidgetManager) createSetup(client gamesense.API, widgets []widget.Widget, displayCfg config.DisplayConfig, cfg *config.Config) *CompositorSetup {
	layoutMgr := layout.NewManager(displayCfg, widgets)
	comp := compositor.NewCompositor(client, layoutMgr, widgets, cfg)

	return &CompositorSetup{
		Compositor: comp,
		Widgets:    widgets,
		Layout:     layoutMgr,
	}
}
