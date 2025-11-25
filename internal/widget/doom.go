package widget

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/AndreRenaud/gore"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// DoomWidget displays DOOM running on the device
type DoomWidget struct {
	*BaseWidget

	// DOOM engine state
	wadFile       string
	bundledWadURL string // Custom URL for WAD download (empty = use default)
	currentImg    *image.Gray
	mu            sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
	started       bool

	// Download state
	isDownloading    bool
	downloadProgress float64 // 0.0 to 1.0
	downloadError    error

	// Rendering
	scale float64 // Downscale factor from DOOM resolution to display
}

// NewDoomWidget creates a new DOOM widget
func NewDoomWidget(cfg config.WidgetConfig) (*DoomWidget, error) {
	base := NewBaseWidget(cfg)

	// Get WAD file name from config
	wadName := cfg.Wad
	if wadName == "" {
		wadName = "doom1.wad"
	}

	// Get bundled WAD URL from config (empty = use default)
	bundledWadURL := ""
	if cfg.BundledWadURL != nil {
		bundledWadURL = *cfg.BundledWadURL
	}

	// Calculate scale factor (DOOM renders at 320x200, we display at 128x40)
	scaleX := float64(cfg.Position.W) / 320.0
	scaleY := float64(cfg.Position.H) / 200.0
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	w := &DoomWidget{
		BaseWidget:    base,
		wadFile:       wadName,
		bundledWadURL: bundledWadURL,
		scale:         scale,
		stopChan:      make(chan struct{}),
	}

	// Initialize DOOM in background (handles WAD download if needed)
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
			w.mu.Lock()
			w.downloadError = fmt.Errorf("panic: %v", r)
			w.mu.Unlock()
		}
	}()

	// Progress callback for download
	progressCallback := func(progress float64) {
		w.mu.Lock()
		w.downloadProgress = progress
		w.mu.Unlock()
	}

	// Get WAD file (may download with progress updates)
	wadFile, err := GetWadFileWithProgress(w.wadFile, w.bundledWadURL, progressCallback, &w.isDownloading, &w.mu)
	if err != nil {
		log.Printf("[DOOM] Failed to get WAD file: %v", err)
		w.mu.Lock()
		w.downloadError = err
		w.isDownloading = false
		w.mu.Unlock()
		return
	}

	w.mu.Lock()
	w.wadFile = wadFile
	w.isDownloading = false
	w.started = true
	w.mu.Unlock()

	log.Printf("[DOOM] Starting engine with WAD: %s", wadFile)

	// Run DOOM main loop with demo playback
	args := []string{"-iwad", wadFile}
	done := make(chan struct{})
	go func() {
		gore.Run(w, args)
		close(done)
	}()

	// Wait for either completion or stop signal
	select {
	case <-done:
		log.Printf("[DOOM] Engine stopped normally")
	case <-w.stopChan:
		log.Printf("[DOOM] Stop requested, terminating engine...")
		// Signal DOOM engine to quit
		gore.Stop()

		// Wait for goroutine to exit with timeout
		select {
		case <-done:
			log.Printf("[DOOM] Engine stopped cleanly")
		case <-time.After(2 * time.Second):
			log.Printf("[DOOM] WARNING: Engine did not stop within 2 seconds")
			// Goroutine may still be running, but we'll exit anyway
			// The GetEvent() callback will prevent further processing
		}
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
func (w *DoomWidget) SetTitle(_ string) {
	// No-op for embedded display
}

// GetEvent implements DoomFrontend interface
func (w *DoomWidget) GetEvent(event *gore.DoomEvent) bool {
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
func (w *DoomWidget) CacheSound(_ string, _ []byte) {
	// No-op - no audio on OLED display
}

// PlaySound implements DoomFrontend interface
func (w *DoomWidget) PlaySound(_ string, _, _, _ int) {
	// No-op - no audio on OLED display
}

// Render renders the current DOOM frame or download progress
func (w *DoomWidget) Render() (image.Image, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	pos := w.GetPosition()
	img := image.NewGray(image.Rect(0, 0, pos.W, pos.H))

	// Show download error if any
	if w.downloadError != nil {
		text := "Download failed!"
		textWidth := glyphs.MeasureText(text, glyphs.Font3x5)
		textX := (pos.W - textWidth) / 2
		glyphs.DrawText(img, text, textX, pos.H/2-4, glyphs.Font3x5, color.Gray{Y: 255})
		return img, nil
	}

	// Show download progress bar
	if w.isDownloading {
		w.drawProgressBar(img, w.downloadProgress, pos.W, pos.H)
		return img, nil
	}

	// Show DOOM frame if available
	if w.currentImg != nil {
		return w.currentImg, nil
	}

	// Return empty image while DOOM is initializing
	return img, nil
}

// drawProgressBar renders a download progress bar
func (w *DoomWidget) drawProgressBar(img *image.Gray, progress float64, width, height int) {
	// Draw title using 3×5 pixel font
	title := "Downloading DOOM"
	titleWidth := glyphs.MeasureText(title, glyphs.Font3x5)
	titleX := (width - titleWidth) / 2
	glyphs.DrawText(img, title, titleX, 4, glyphs.Font3x5, color.Gray{Y: 255})

	// Progress bar dimensions
	barWidth := width - 20
	barHeight := 8
	barX := 10
	barY := height/2 - barHeight/2

	// Draw border
	for x := barX; x < barX+barWidth; x++ {
		img.SetGray(x, barY, color.Gray{Y: 255})
		img.SetGray(x, barY+barHeight-1, color.Gray{Y: 255})
	}
	for y := barY; y < barY+barHeight; y++ {
		img.SetGray(barX, y, color.Gray{Y: 255})
		img.SetGray(barX+barWidth-1, y, color.Gray{Y: 255})
	}

	// Draw filled portion
	fillWidth := int(float64(barWidth-2) * progress)
	for y := barY + 1; y < barY+barHeight-1; y++ {
		for x := barX + 1; x < barX+1+fillWidth; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	// Draw percentage text using 3×5 pixel font
	percentText := fmt.Sprintf("%.0f%%", progress*100)
	textWidth := glyphs.MeasureText(percentText, glyphs.Font3x5)
	textX := (width - textWidth) / 2
	glyphs.DrawText(img, percentText, textX, barY+barHeight+6, glyphs.Font3x5, color.Gray{Y: 255})
}

// Update is called periodically
func (w *DoomWidget) Update() error {
	// DOOM updates itself in background goroutine
	return nil
}

// Stop stops the DOOM engine
func (w *DoomWidget) Stop() {
	// Close stop channel to signal runDoom to exit
	// runDoom() will call gore.Stop() and wait for cleanup
	close(w.stopChan)

	// Wait for runDoom goroutine to complete
	// This ensures gore.Run() goroutine has exited
	w.wg.Wait()
}
