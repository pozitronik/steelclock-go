package compositor

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// Compositor manages the rendering loop and API updates
type Compositor struct {
	client        gamesense.API
	layoutManager *layout.Manager
	refreshRate   time.Duration
	eventName     string
	widgets       []widget.Widget
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewCompositor creates a new compositor
func NewCompositor(client gamesense.API, layoutMgr *layout.Manager, widgets []widget.Widget, cfg *config.Config) *Compositor {
	refreshRate := time.Duration(cfg.RefreshRateMs) * time.Millisecond

	return &Compositor{
		client:        client,
		layoutManager: layoutMgr,
		refreshRate:   refreshRate,
		eventName:     "STEELCLOCK_DISPLAY",
		widgets:       widgets,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the rendering loop
func (c *Compositor) Start() error {
	log.Println("Compositor starting...")

	// Start widget update threads
	for _, w := range c.widgets {
		c.wg.Add(1)
		go c.widgetUpdateLoop(w)
	}

	// Start rendering loop
	c.wg.Add(1)
	go c.renderLoop()

	// Start heartbeat
	c.wg.Add(1)
	go c.heartbeatLoop()

	log.Println("Compositor started")
	return nil
}

// Stop stops the compositor
func (c *Compositor) Stop() {
	log.Println("Compositor stopping...")
	close(c.stopChan)
	c.wg.Wait()
	log.Println("Compositor stopped")
}

// logPanic writes panic information to panic.log
func logPanic(context string) {
	if r := recover(); r != nil {
		logFile, err := os.OpenFile("panic.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open panic.log: %v", err)
			return
		}
		defer func() {
			if closeErr := logFile.Close(); closeErr != nil {
				log.Printf("Failed to close panic.log: %v", closeErr)
			}
		}()

		panicMsg := fmt.Sprintf("\n=== PANIC at %s ===\nContext: %s\nError: %v\n\nStack trace:\n%s\n",
			time.Now().Format("2006-01-02 15:04:05"), context, r, debug.Stack())

		if _, err := logFile.WriteString(panicMsg); err != nil {
			log.Printf("Failed to write to panic.log: %v", err)
		}
		log.Print(panicMsg)
	}
}

// widgetUpdateLoop periodically updates a widget
func (c *Compositor) widgetUpdateLoop(w widget.Widget) {
	defer c.wg.Done()
	defer logPanic(fmt.Sprintf("widgetUpdateLoop for %s", w.Name()))

	ticker := time.NewTicker(w.GetUpdateInterval())
	defer ticker.Stop()

	// Initial update
	if err := w.Update(); err != nil {
		log.Printf("Widget %s update error: %v", w.Name(), err)
	}

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := w.Update(); err != nil {
				log.Printf("Widget %s update error: %v", w.Name(), err)
			}
		}
	}
}

// renderLoop periodically renders and sends frames
func (c *Compositor) renderLoop() {
	defer c.wg.Done()
	defer logPanic("renderLoop")

	ticker := time.NewTicker(c.refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := c.renderFrame(); err != nil {
				log.Printf("Render error: %v", err)
			}
		}
	}
}

// renderFrame renders and sends a single frame
func (c *Compositor) renderFrame() error {
	// Composite all widgets
	canvas, err := c.layoutManager.Composite()
	if err != nil {
		return fmt.Errorf("composite failed: %w", err)
	}

	// Convert to bytes
	bitmapData, err := bitmap.ImageToBytes(canvas, 128, 40)
	if err != nil {
		return fmt.Errorf("image conversion failed: %w", err)
	}

	// Send to display
	if err := c.client.SendScreenData(c.eventName, bitmapData); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}

	return nil
}

// heartbeatLoop sends periodic heartbeats
func (c *Compositor) heartbeatLoop() {
	defer c.wg.Done()
	defer logPanic("heartbeatLoop")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := c.client.SendHeartbeat(); err != nil {
				log.Printf("Heartbeat error: %v", err)
			}
		}
	}
}
