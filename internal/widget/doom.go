package widget

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sync"
	"time"

	"github.com/AndreRenaud/gore"
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// Package-level state to track if DOOM has been run in this process.
// The gore library uses global state and cannot be safely restarted.
var (
	doomHasRun   bool
	doomHasRunMu sync.Mutex
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

	// Render mode settings
	renderMode      string  // "normal", "contrast", "posterize", "threshold", "dither", "gamma"
	posterizeLevels int     // Number of gray levels for posterize mode
	thresholdValue  int     // Cutoff for threshold mode
	gamma           float64 // Gamma value for gamma mode
	contrastBoost   float64 // Contrast multiplier for gamma mode
	ditherSize      int     // Bayer matrix size for dither mode
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

	// Render mode settings with defaults
	renderMode := "normal"
	posterizeLevels := 4
	thresholdValue := 128
	gamma := 1.5
	contrastBoost := 1.2
	ditherSize := 4

	if cfg.Doom != nil {
		if cfg.Doom.RenderMode != "" {
			renderMode = cfg.Doom.RenderMode
		}
		if cfg.Doom.PosterizeLevels > 0 {
			posterizeLevels = cfg.Doom.PosterizeLevels
			if posterizeLevels < 2 {
				posterizeLevels = 2
			} else if posterizeLevels > 16 {
				posterizeLevels = 16
			}
		}
		if cfg.Doom.ThresholdValue > 0 {
			thresholdValue = cfg.Doom.ThresholdValue
			if thresholdValue > 255 {
				thresholdValue = 255
			}
		}
		if cfg.Doom.Gamma > 0 {
			gamma = cfg.Doom.Gamma
			if gamma < 0.1 {
				gamma = 0.1
			} else if gamma > 3.0 {
				gamma = 3.0
			}
		}
		if cfg.Doom.ContrastBoost > 0 {
			contrastBoost = cfg.Doom.ContrastBoost
			if contrastBoost < 1.0 {
				contrastBoost = 1.0
			} else if contrastBoost > 3.0 {
				contrastBoost = 3.0
			}
		}
		if cfg.Doom.DitherSize > 0 {
			ditherSize = cfg.Doom.DitherSize
			// Clamp to valid Bayer matrix sizes
			if ditherSize <= 2 {
				ditherSize = 2
			} else if ditherSize <= 4 {
				ditherSize = 4
			} else {
				ditherSize = 8
			}
		}
	}

	w := &DoomWidget{
		BaseWidget:      base,
		wadFile:         wadName,
		bundledWadURL:   bundledWadURL,
		scale:           scale,
		stopChan:        make(chan struct{}),
		renderMode:      renderMode,
		posterizeLevels: posterizeLevels,
		thresholdValue:  thresholdValue,
		gamma:           gamma,
		contrastBoost:   contrastBoost,
		ditherSize:      ditherSize,
	}

	// Initialize DOOM in background (handles WAD download if needed)
	w.wg.Add(1)
	go w.runDoom()

	return w, nil
}

// runDoom runs the DOOM engine in a background goroutine
func (w *DoomWidget) runDoom() {
	defer w.wg.Done()

	// Check if DOOM has already been run in this process.
	// The gore library uses global state and cannot be safely restarted.
	doomHasRunMu.Lock()
	if doomHasRun {
		doomHasRunMu.Unlock()
		log.Printf("[DOOM] Cannot restart DOOM - engine has global state. Restart application to play DOOM again.")
		w.mu.Lock()
		w.downloadError = fmt.Errorf("restart app to play DOOM")
		w.mu.Unlock()
		return
	}
	doomHasRun = true
	doomHasRunMu.Unlock()

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
		log.Printf("[DOOM] First frame received, size: %dx%d, render mode: %s", img.Bounds().Dx(), img.Bounds().Dy(), w.renderMode)
	}
	w.mu.Unlock()

	pos := w.GetPosition()

	// Create grayscale image at display resolution
	grayImg := image.NewGray(image.Rect(0, 0, pos.W, pos.H))

	// Downsample and convert RGBA to grayscale
	bounds := img.Bounds()
	scaleX := float64(bounds.Dx()) / float64(pos.W)
	scaleY := float64(bounds.Dy()) / float64(pos.H)

	// First pass: convert to grayscale and find min/max for contrast modes
	grayValues := make([][]uint8, pos.H)
	var minGray, maxGray uint8 = 255, 0

	for y := 0; y < pos.H; y++ {
		grayValues[y] = make([]uint8, pos.W)
		for x := 0; x < pos.W; x++ {
			// Sample from source image
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			// Get RGBA color
			r, g, b, _ := img.At(srcX, srcY).RGBA()

			// Convert to grayscale using standard luminance formula
			// Y = 0.299*R + 0.587*G + 0.114*B
			gray := uint8((299*r + 587*g + 114*b) / 1000 / 256)
			grayValues[y][x] = gray

			if gray < minGray {
				minGray = gray
			}
			if gray > maxGray {
				maxGray = gray
			}
		}
	}

	// Second pass: apply render mode
	for y := 0; y < pos.H; y++ {
		for x := 0; x < pos.W; x++ {
			gray := grayValues[y][x]
			var finalGray uint8

			switch w.renderMode {
			case "contrast":
				finalGray = w.applyContrast(gray, minGray, maxGray)
			case "posterize":
				finalGray = w.applyPosterize(gray)
			case "threshold":
				finalGray = w.applyThreshold(gray)
			case "dither":
				finalGray = w.applyDither(gray, x, y)
			case "gamma":
				finalGray = w.applyGamma(gray, minGray, maxGray)
			default: // "normal"
				finalGray = gray
			}

			grayImg.SetGray(x, y, color.Gray{Y: finalGray})
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

	// Show error if any
	if w.downloadError != nil {
		text := w.downloadError.Error()
		textWidth := glyphs.MeasureText(text, glyphs.Font3x5)
		textX := (pos.W - textWidth) / 2
		if textX < 0 {
			textX = 0
		}
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
	bitmap.DrawHorizontalLine(img, barX, barX+barWidth-1, barY, 255)
	bitmap.DrawHorizontalLine(img, barX, barX+barWidth-1, barY+barHeight-1, 255)
	bitmap.DrawVerticalLine(img, barX, barY, barY+barHeight-1, 255)
	bitmap.DrawVerticalLine(img, barX+barWidth-1, barY, barY+barHeight-1, 255)

	// Draw filled portion
	fillWidth := int(float64(barWidth-2) * progress)
	if fillWidth > 0 {
		bitmap.DrawFilledRectangle(img, barX+1, barY+1, fillWidth, barHeight-2, 255)
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

// applyContrast applies auto-contrast stretching (histogram stretching)
// Maps the actual min-max range to full 0-255 range
func (w *DoomWidget) applyContrast(gray, minGray, maxGray uint8) uint8 {
	if maxGray == minGray {
		return gray
	}
	// Stretch to full range
	stretched := float64(gray-minGray) * 255.0 / float64(maxGray-minGray)
	if stretched > 255 {
		return 255
	}
	return uint8(stretched)
}

// applyPosterize reduces the image to N discrete gray levels
func (w *DoomWidget) applyPosterize(gray uint8) uint8 {
	levels := w.posterizeLevels
	if levels < 2 {
		levels = 2
	}
	// Quantize to N levels, then map back to 0-255
	step := 256 / levels
	level := int(gray) / step
	if level >= levels {
		level = levels - 1
	}
	// Map level back to 0-255 range
	return uint8(level * 255 / (levels - 1))
}

// applyThreshold converts to pure black/white based on threshold value
func (w *DoomWidget) applyThreshold(gray uint8) uint8 {
	if int(gray) >= w.thresholdValue {
		return 255
	}
	return 0
}

// applyDither applies ordered dithering using Bayer matrix
func (w *DoomWidget) applyDither(gray uint8, x, y int) uint8 {
	// Bayer matrices for different sizes
	var threshold float64

	switch w.ditherSize {
	case 2:
		// 2x2 Bayer matrix
		bayer2 := [2][2]float64{
			{0.0 / 4.0, 2.0 / 4.0},
			{3.0 / 4.0, 1.0 / 4.0},
		}
		threshold = bayer2[y%2][x%2]
	case 8:
		// 8x8 Bayer matrix
		bayer8 := [8][8]float64{
			{0.0 / 64.0, 32.0 / 64.0, 8.0 / 64.0, 40.0 / 64.0, 2.0 / 64.0, 34.0 / 64.0, 10.0 / 64.0, 42.0 / 64.0},
			{48.0 / 64.0, 16.0 / 64.0, 56.0 / 64.0, 24.0 / 64.0, 50.0 / 64.0, 18.0 / 64.0, 58.0 / 64.0, 26.0 / 64.0},
			{12.0 / 64.0, 44.0 / 64.0, 4.0 / 64.0, 36.0 / 64.0, 14.0 / 64.0, 46.0 / 64.0, 6.0 / 64.0, 38.0 / 64.0},
			{60.0 / 64.0, 28.0 / 64.0, 52.0 / 64.0, 20.0 / 64.0, 62.0 / 64.0, 30.0 / 64.0, 54.0 / 64.0, 22.0 / 64.0},
			{3.0 / 64.0, 35.0 / 64.0, 11.0 / 64.0, 43.0 / 64.0, 1.0 / 64.0, 33.0 / 64.0, 9.0 / 64.0, 41.0 / 64.0},
			{51.0 / 64.0, 19.0 / 64.0, 59.0 / 64.0, 27.0 / 64.0, 49.0 / 64.0, 17.0 / 64.0, 57.0 / 64.0, 25.0 / 64.0},
			{15.0 / 64.0, 47.0 / 64.0, 7.0 / 64.0, 39.0 / 64.0, 13.0 / 64.0, 45.0 / 64.0, 5.0 / 64.0, 37.0 / 64.0},
			{63.0 / 64.0, 31.0 / 64.0, 55.0 / 64.0, 23.0 / 64.0, 61.0 / 64.0, 29.0 / 64.0, 53.0 / 64.0, 21.0 / 64.0},
		}
		threshold = bayer8[y%8][x%8]
	default: // 4x4 (default)
		// 4x4 Bayer matrix
		bayer4 := [4][4]float64{
			{0.0 / 16.0, 8.0 / 16.0, 2.0 / 16.0, 10.0 / 16.0},
			{12.0 / 16.0, 4.0 / 16.0, 14.0 / 16.0, 6.0 / 16.0},
			{3.0 / 16.0, 11.0 / 16.0, 1.0 / 16.0, 9.0 / 16.0},
			{15.0 / 16.0, 7.0 / 16.0, 13.0 / 16.0, 5.0 / 16.0},
		}
		threshold = bayer4[y%4][x%4]
	}

	// Compare normalized gray value against threshold
	normalizedGray := float64(gray) / 255.0
	if normalizedGray > threshold {
		return 255
	}
	return 0
}

// applyGamma applies gamma correction with optional contrast boost
// First stretches contrast, then applies gamma curve
func (w *DoomWidget) applyGamma(gray, minGray, maxGray uint8) uint8 {
	// First apply contrast stretching
	var normalized float64
	if maxGray == minGray {
		normalized = float64(gray) / 255.0
	} else {
		normalized = float64(gray-minGray) / float64(maxGray-minGray)
	}

	// Apply contrast boost (expand around 0.5)
	if w.contrastBoost > 1.0 {
		normalized = (normalized-0.5)*w.contrastBoost + 0.5
		if normalized < 0 {
			normalized = 0
		} else if normalized > 1 {
			normalized = 1
		}
	}

	// Apply gamma correction: output = input^(1/gamma)
	// gamma > 1 brightens midtones, gamma < 1 darkens them
	gammaCorrected := math.Pow(normalized, 1.0/w.gamma)

	result := gammaCorrected * 255.0
	if result > 255 {
		return 255
	}
	return uint8(result)
}
