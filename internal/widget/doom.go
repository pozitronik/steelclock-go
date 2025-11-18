package widget

import (
	"image"
	"image/color"
	"log"
	"sync"

	"github.com/AndreRenaud/gore"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// DoomWidget displays DOOM running on the device
type DoomWidget struct {
	*BaseWidget

	// DOOM engine state
	wadFile    string
	currentImg *image.Gray
	mu         sync.RWMutex
	stopChan   chan struct{}
	wg         sync.WaitGroup
	started    bool

	// Rendering
	scale float64 // Downscale factor from DOOM resolution to display
}

// NewDoomWidget creates a new DOOM widget
func NewDoomWidget(cfg config.WidgetConfig) (*DoomWidget, error) {
	base := NewBaseWidget(cfg)

	// Get WAD file name from properties
	wadName := cfg.Properties.WadName
	if wadName == "" {
		wadName = "doom1.wad"
	}

	// Get WAD file from working directory (will auto-download if not found)
	wadFile, err := GetWadFile(wadName)
	if err != nil {
		return nil, err
	}

	log.Printf("[DOOM] Using WAD file: %s", wadFile)

	// Calculate scale factor (DOOM renders at 320x200, we display at 128x40)
	scaleX := float64(cfg.Position.W) / 320.0
	scaleY := float64(cfg.Position.H) / 200.0
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	w := &DoomWidget{
		BaseWidget: base,
		wadFile:    wadFile,
		scale:      scale,
		stopChan:   make(chan struct{}),
	}

	// Initialize DOOM in background
	w.wg.Add(1)
	go w.runDoom()

	return w, nil
}

// runDoom runs the DOOM engine in a background goroutine
func (w *DoomWidget) runDoom() {
	defer w.wg.Done()

	// Catch any panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DOOM] PANIC in Gore.Run: %v", r)
		}
	}()

	w.mu.Lock()
	w.started = true
	w.mu.Unlock()

	log.Printf("[DOOM] Starting DOOM engine with WAD: %s", w.wadFile)

	// Run DOOM main loop with demo playback
	// The engine will call our DrawFrame() method for each frame
	args := []string{"-iwad", w.wadFile}
	log.Printf("[DOOM] Gore.Run args: %v", args)

	// Set a timeout to detect if Gore hangs
	done := make(chan struct{})
	go func() {
		gore.Run(w, args)
		close(done)
		log.Printf("[DOOM] Gore.Run exited normally")
	}()

	// Wait for either completion or stop signal
	select {
	case <-done:
		log.Printf("[DOOM] Gore finished")
	case <-w.stopChan:
		log.Printf("[DOOM] Stop requested while Gore was running")
	}
}

// DrawFrame implements DoomFrontend interface - called by DOOM engine with each frame
func (w *DoomWidget) DrawFrame(img *image.RGBA) {
	select {
	case <-w.stopChan:
		return
	default:
	}

	// Log first frame only
	w.mu.Lock()
	if w.currentImg == nil {
		log.Printf("[DOOM] First frame received, size: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
	w.mu.Unlock()

	pos := w.GetPosition()

	// Create grayscale image at display resolution
	grayImg := image.NewGray(image.Rect(0, 0, pos.W, pos.H))

	// Downsample and convert RGBA to grayscale
	bounds := img.Bounds()
	scaleX := float64(bounds.Dx()) / float64(pos.W)
	scaleY := float64(bounds.Dy()) / float64(pos.H)

	for y := 0; y < pos.H; y++ {
		for x := 0; x < pos.W; x++ {
			// Sample from source image
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			// Get RGBA color
			r, g, b, _ := img.At(srcX, srcY).RGBA()

			// Convert to grayscale using standard luminance formula
			// Y = 0.299*R + 0.587*G + 0.114*B
			gray := uint8((299*r + 587*g + 114*b) / 1000 / 256)

			grayImg.SetGray(x, y, color.Gray{Y: gray})
		}
	}

	// Store the frame
	w.mu.Lock()
	w.currentImg = grayImg
	w.mu.Unlock()
}

// SetTitle implements DoomFrontend interface
func (w *DoomWidget) SetTitle(title string) {
	log.Printf("[DOOM] SetTitle called: %s", title)
	// No-op for embedded display
}

// GetEvent implements DoomFrontend interface
func (w *DoomWidget) GetEvent(event *gore.DoomEvent) bool {
	// Only log first few calls to avoid spam
	w.mu.Lock()
	if w.currentImg == nil {
		log.Printf("[DOOM] GetEvent called (waiting for first frame)")
	}
	w.mu.Unlock()

	select {
	case <-w.stopChan:
		event.Type = gore.Ev_quit
		return true
	default:
	}

	// Return false to let demo play automatically
	return false
}

// CacheSound implements DoomFrontend interface
func (w *DoomWidget) CacheSound(name string, data []byte) {
	// Only log first call
	w.mu.Lock()
	if w.currentImg == nil {
		log.Printf("[DOOM] CacheSound called: %s (%d bytes)", name, len(data))
	}
	w.mu.Unlock()
	// No-op - no audio on OLED display
}

// PlaySound implements DoomFrontend interface
func (w *DoomWidget) PlaySound(name string, channel, vol, sep int) {
	// No-op - no audio on OLED display
}

// Render renders the current DOOM frame
func (w *DoomWidget) Render() (image.Image, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.currentImg == nil {
		// Return empty image while DOOM is loading
		pos := w.GetPosition()
		return image.NewGray(image.Rect(0, 0, pos.W, pos.H)), nil
	}

	return w.currentImg, nil
}

// Update is called periodically
func (w *DoomWidget) Update() error {
	// DOOM updates itself in background goroutine
	return nil
}

// Stop stops the DOOM engine
func (w *DoomWidget) Stop() {
	w.mu.Lock()
	started := w.started
	w.mu.Unlock()

	if started {
		gore.Stop()
	}

	close(w.stopChan)
	w.wg.Wait()
}
