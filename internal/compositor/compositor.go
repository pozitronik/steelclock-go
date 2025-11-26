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

const (
	// HeartbeatInterval is how often to send heartbeat to GameSense API
	HeartbeatInterval = 10 * time.Second

	// MaxHeartbeatFailures is how many consecutive failures before triggering backend failure callback
	MaxHeartbeatFailures = 2

	// DefaultEventName is the GameSense event name for display updates
	DefaultEventName = "STEELCLOCK_DISPLAY"
)

// Resolution represents a display resolution
type Resolution struct {
	Width  int
	Height int
}

// Compositor manages the rendering loop and API updates
type Compositor struct {
	client          gamesense.API
	layoutManager   *layout.Manager
	refreshRate     time.Duration
	eventName       string
	widgets         []widget.Widget
	stopChan        chan struct{}
	wg              sync.WaitGroup
	batchingEnabled bool
	batchSize       int
	batchSupported  bool
	frameBuffer     [][]int
	bufferMu        sync.Mutex
	resolutions     []Resolution // All resolutions to render (main + supported)

	// Pre-allocated buffers for ImageToBytes to reduce allocations in render loop
	bitmapBuffers map[string][]int

	// Backend failure handling
	OnBackendFailure     func()     // Callback when backend fails (called once per failure)
	heartbeatFailures    int        // Consecutive heartbeat failure count
	backendFailureCalled bool       // Prevents multiple callback invocations
	failureMu            sync.Mutex // Protects failure state
}

// NewCompositor creates a new compositor
func NewCompositor(client gamesense.API, layoutMgr *layout.Manager, widgets []widget.Widget, cfg *config.Config) *Compositor {
	refreshRate := time.Duration(cfg.RefreshRateMs) * time.Millisecond

	// Build list of resolutions (main display + supported resolutions)
	resolutions := []Resolution{
		{Width: cfg.Display.Width, Height: cfg.Display.Height}, // Main display resolution
	}
	for _, res := range cfg.SupportedResolutions {
		resolutions = append(resolutions, Resolution{Width: res.Width, Height: res.Height})
	}

	// Pre-allocate bitmap buffers for each resolution
	bitmapBuffers := make(map[string][]int)
	for _, res := range resolutions {
		bufferSize := (res.Width*res.Height + 7) / 8
		key := fmt.Sprintf("image-data-%dx%d", res.Width, res.Height)
		bitmapBuffers[key] = make([]int, bufferSize)
	}

	comp := &Compositor{
		client:          client,
		layoutManager:   layoutMgr,
		refreshRate:     refreshRate,
		eventName:       DefaultEventName,
		widgets:         widgets,
		stopChan:        make(chan struct{}),
		batchingEnabled: cfg.EventBatchingEnabled,
		batchSize:       cfg.EventBatchSize,
		frameBuffer:     make([][]int, 0, cfg.EventBatchSize),
		resolutions:     resolutions,
		bitmapBuffers:   bitmapBuffers,
	}

	log.Printf("Rendering for %d resolution(s):", len(resolutions))
	for _, res := range resolutions {
		log.Printf("  - %dx%d", res.Width, res.Height)
	}

	// Check if batching is supported by API (only if enabled in config)
	// FIXME: Type assertion to concrete type breaks interface abstraction.
	// Consider extending gamesense.API interface to include SupportsMultipleEvents()
	// or creating a separate BatchAPI interface for batch operations.
	if comp.batchingEnabled {
		if gsClient, ok := client.(*gamesense.Client); ok {
			comp.batchSupported = gsClient.SupportsMultipleEvents()
			if !comp.batchSupported {
				log.Println("Event batching disabled: not supported by GameSense API")
				comp.batchingEnabled = false
			} else {
				log.Printf("Event batching enabled with batch size: %d", comp.batchSize)
			}
		} else {
			log.Println("Event batching disabled: client does not support feature detection")
			comp.batchingEnabled = false
		}
	}

	return comp
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

	// Flush any remaining buffered frames
	if c.batchingEnabled {
		if err := c.flushBatch(); err != nil {
			log.Printf("Error flushing batch on stop: %v", err)
		}
	}

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

	// Render at all resolutions using pre-allocated buffers
	resolutionData := make(map[string][]int)
	for _, res := range c.resolutions {
		key := fmt.Sprintf("image-data-%dx%d", res.Width, res.Height)
		buffer := c.bitmapBuffers[key]
		bitmapData, err := bitmap.ImageToBytes(canvas, res.Width, res.Height, buffer)
		if err != nil {
			return fmt.Errorf("image conversion failed for %dx%d: %w", res.Width, res.Height, err)
		}
		resolutionData[key] = bitmapData
	}

	// If batching is enabled, buffer the frame (only main resolution for now)
	if c.batchingEnabled {
		mainKey := fmt.Sprintf("image-data-%dx%d", c.resolutions[0].Width, c.resolutions[0].Height)
		c.bufferMu.Lock()
		c.frameBuffer = append(c.frameBuffer, resolutionData[mainKey])
		shouldFlush := len(c.frameBuffer) >= c.batchSize
		c.bufferMu.Unlock()

		// Flush if buffer is full
		if shouldFlush {
			return c.flushBatch()
		}
		return nil
	}

	// Send immediately if batching disabled
	// FIXME: Type assertion to access SendScreenDataMultiRes - see FIXME in NewCompositor
	if gsClient, ok := c.client.(*gamesense.Client); ok {
		if err := gsClient.SendScreenDataMultiRes(c.eventName, resolutionData); err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
	} else {
		// Fallback to single resolution for non-Client implementations
		mainKey := fmt.Sprintf("image-data-%dx%d", c.resolutions[0].Width, c.resolutions[0].Height)
		if err := c.client.SendScreenData(c.eventName, resolutionData[mainKey]); err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
	}

	return nil
}

// flushBatch sends all buffered frames in a single request
func (c *Compositor) flushBatch() error {
	c.bufferMu.Lock()
	if len(c.frameBuffer) == 0 {
		c.bufferMu.Unlock()
		return nil
	}

	// Copy buffer to send
	framesToSend := make([][]int, len(c.frameBuffer))
	copy(framesToSend, c.frameBuffer)
	c.frameBuffer = c.frameBuffer[:0] // Clear buffer
	c.bufferMu.Unlock()

	// Send batch
	// FIXME: Type assertion to access SendMultipleScreenData - see FIXME in NewCompositor
	if gsClient, ok := c.client.(*gamesense.Client); ok {
		if err := gsClient.SendMultipleScreenData(c.eventName, framesToSend); err != nil {
			return fmt.Errorf("batch send failed: %w", err)
		}
	}

	return nil
}

// heartbeatLoop sends periodic heartbeats
func (c *Compositor) heartbeatLoop() {
	defer c.wg.Done()
	defer logPanic("heartbeatLoop")

	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := c.client.SendHeartbeat(); err != nil {
				log.Printf("Heartbeat error: %v", err)

				c.failureMu.Lock()
				c.heartbeatFailures++
				shouldNotify := c.heartbeatFailures >= MaxHeartbeatFailures && !c.backendFailureCalled && c.OnBackendFailure != nil
				if shouldNotify {
					c.backendFailureCalled = true
				}
				c.failureMu.Unlock()

				if shouldNotify {
					log.Printf("Backend failure detected after %d consecutive heartbeat failures", c.heartbeatFailures)
					// Call callback in goroutine to avoid blocking heartbeat loop
					go c.OnBackendFailure()
				}
			} else {
				// Reset failure counter on success
				c.failureMu.Lock()
				c.heartbeatFailures = 0
				c.backendFailureCalled = false
				c.failureMu.Unlock()
			}
		}
	}
}
