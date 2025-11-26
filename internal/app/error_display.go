package app

import (
	"fmt"
	"log"

	"github.com/pozitronik/steelclock-go/internal/compositor"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// ErrorDisplayRefreshRateMs is the refresh rate for error display (flash interval)
const ErrorDisplayRefreshRateMs = 500

// startWithErrorDisplay creates and runs an error display widget
func (a *App) startWithErrorDisplay(message string, width, height int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Printf("Starting error display: %s", message)

	errorClient := a.client
	if errorClient == nil {
		log.Println("No existing client, creating one for error display...")

		var err error
		if a.lastGoodConfig != nil {
			errorClient, err = CreateBackendClient(a.lastGoodConfig)
		} else {
			errorClient, err = gamesense.NewClient(config.DefaultGameName, config.DefaultGameDisplay)
			if err == nil {
				err = errorClient.RegisterGame(DeveloperName, 0)
			}
		}
		if err != nil {
			log.Printf("ERROR: Failed to create client for error display: %v", err)
			return fmt.Errorf("failed to create client: %w", err)
		}
	}

	if err := errorClient.RegisterGame(DeveloperName, 0); err != nil {
		log.Printf("ERROR: Failed to register game for error display: %v", err)
		return fmt.Errorf("failed to register game: %w", err)
	}

	if err := errorClient.BindScreenEvent(EventName, DeviceType); err != nil {
		log.Printf("ERROR: Failed to bind screen event for error display: %v", err)
		return fmt.Errorf("failed to bind screen event: %w", err)
	}

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

	layoutMgr := layout.NewManager(displayCfg, widgets)
	a.comp = compositor.NewCompositor(errorClient, layoutMgr, widgets, errorCfg)

	if err := a.comp.Start(); err != nil {
		log.Printf("ERROR: Failed to start compositor for error display: %v", err)
		return fmt.Errorf("failed to start compositor: %w", err)
	}

	log.Println("Error display started - screen will flash error message")
	log.Println("Fix the configuration file and reload to continue")

	return nil
}
