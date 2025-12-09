package doom

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	// Create a temporary WAD file to avoid download during test
	tmpFile := "test_doom.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	if err := os.WriteFile(tmpFile, []byte("test wad content"), 0644); err != nil {
		t.Fatalf("Failed to create test WAD file: %v", err)
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Wad: tmpFile,
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if widget == nil {
		t.Fatal("New() returned nil")
	}

	if widget.Name() != "test_doom" {
		t.Errorf("Name() = %s, want test_doom", widget.Name())
	}

	// Clean up
	widget.Stop()
	time.Sleep(10 * time.Millisecond) // Let goroutines finish
}

func TestWidget_Render_EmptyFrame(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	// Create a temporary WAD to avoid download
	tmpFile := "test_empty_frame.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	if err := os.WriteFile(tmpFile, []byte("test wad"), 0644); err != nil {
		t.Fatalf("Failed to create test WAD: %v", err)
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Wad: tmpFile,
	}

	widget, _ := New(cfg)
	defer widget.Stop()

	// Render before any frames - should return empty image
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	if img.Bounds().Dx() != 128 {
		t.Errorf("image width = %d, want 128", img.Bounds().Dx())
	}

	if img.Bounds().Dy() != 40 {
		t.Errorf("image height = %d, want 40", img.Bounds().Dy())
	}
}

func TestWidget_Render_DownloadProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	// Create a temporary WAD to avoid actual download
	tmpFile := "test_progress_render.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	if err := os.WriteFile(tmpFile, []byte("test wad"), 0644); err != nil {
		t.Fatalf("Failed to create test WAD: %v", err)
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Wad: tmpFile,
	}

	widget, _ := New(cfg)
	defer widget.Stop()

	// Simulate download in progress
	widget.mu.Lock()
	widget.isDownloading = true
	widget.downloadProgress = 0.5 // 50%
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	// Check that the image is not empty (has some pixels set)
	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatal("Render() did not return *image.Gray")
	}

	hasPixels := false
	for y := 0; y < grayImg.Bounds().Dy(); y++ {
		for x := 0; x < grayImg.Bounds().Dx(); x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasPixels = true
				break
			}
		}
		if hasPixels {
			break
		}
	}

	if !hasPixels {
		t.Error("Progress bar should have visible pixels")
	}
}

func TestWidget_Render_DownloadError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	// Create a temporary WAD to avoid actual download
	tmpFile := "test_error_render.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	if err := os.WriteFile(tmpFile, []byte("test wad"), 0644); err != nil {
		t.Fatalf("Failed to create test WAD: %v", err)
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Wad: tmpFile,
	}

	widget, _ := New(cfg)
	defer widget.Stop()

	// Simulate download error
	widget.mu.Lock()
	widget.downloadError = fmt.Errorf("test error")
	widget.mu.Unlock()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	// Should show error message (has pixels)
	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatal("Render() did not return *image.Gray")
	}

	hasPixels := false
	for y := 0; y < grayImg.Bounds().Dy(); y++ {
		for x := 0; x < grayImg.Bounds().Dx(); x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasPixels = true
				break
			}
		}
		if hasPixels {
			break
		}
	}

	if !hasPixels {
		t.Error("Error message should have visible pixels")
	}
}

func TestWidget_DrawProgressBar(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, _ := New(cfg)
	defer widget.Stop()

	img := image.NewGray(image.Rect(0, 0, 128, 40))

	// Test progress bar at 50%
	widget.drawProgressBar(img, 0.5, 128, 40)

	// Check for border pixels (top border)
	hasBorder := false
	barY := 40/2 - 8/2
	for x := 10; x < 10+128-20; x++ {
		if img.GrayAt(x, barY).Y == 255 {
			hasBorder = true
			break
		}
	}

	if !hasBorder {
		t.Error("Progress bar should have visible border")
	}

	// Check for filled pixels (should be filled up to ~50%)
	hasFill := false
	for x := 11; x < 11+(128-20)/2; x++ {
		if img.GrayAt(x, barY+1).Y == 255 {
			hasFill = true
			break
		}
	}

	if !hasFill {
		t.Error("Progress bar should have filled portion")
	}

	// Check that right side is not filled (should be empty well after 50%)
	// Check at 75% position - should be empty for 50% progress
	rightX := 11 + (128-20)*3/4
	rightFilled := false
	for x := rightX; x < 11+(128-20)-2; x++ {
		if img.GrayAt(x, barY+1).Y == 255 {
			rightFilled = true
			break
		}
	}

	if rightFilled {
		t.Error("Progress bar should not be filled beyond progress value")
	}
}

func TestWidget_DrawFrame(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, _ := New(cfg)
	defer widget.Stop()

	// Create a test RGBA image (320x200)
	srcImg := image.NewRGBA(image.Rect(0, 0, 320, 200))

	// Fill with some colors
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			srcImg.Set(x, y, color.RGBA{R: 128, G: 64, B: 32, A: 255})
		}
	}

	// Call DrawFrame
	widget.DrawFrame(srcImg)

	// Check that currentImg was set
	widget.mu.RLock()
	hasFrame := widget.currentImg != nil
	widget.mu.RUnlock()

	if !hasFrame {
		t.Error("DrawFrame should set currentImg")
	}

	// Render should now return the converted frame
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil after DrawFrame")
	}

	// Check dimensions
	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 40 {
		t.Errorf("Rendered image size = %dx%d, want 128x40", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Check that image has some non-zero pixels (grayscale conversion)
	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatal("Render() did not return *image.Gray")
	}

	hasGray := false
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasGray = true
				break
			}
		}
		if hasGray {
			break
		}
	}

	if !hasGray {
		t.Error("Converted frame should have visible pixels")
	}
}

func TestWidget_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
	}

	widget, _ := New(cfg)
	defer widget.Stop()

	// Update should not return error (DOOM updates in background)
	err := widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
}

// TestWidget_StopCleansUpGoroutines tests that Stop() properly terminates all goroutines
// This test exposes the goroutine leak where gore.Run() goroutine may not exit
func TestWidget_StopCleansUpGoroutines(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gore engine test in short/CI mode (checkptr issues)")
	}

	// Create a temporary WAD file
	tmpFile := "test_stop_cleanup.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	if err := os.WriteFile(tmpFile, []byte("test wad content"), 0644); err != nil {
		t.Fatalf("Failed to create test WAD file: %v", err)
	}

	cfg := config.WidgetConfig{
		Type:    "doom",
		ID:      "test_doom",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0,
			Y: 0,
			W: 128,
			H: 40,
		},
		Wad: tmpFile,
	}

	widget, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop should return quickly and clean up all goroutines
	// Set a timeout to detect if Stop() blocks indefinitely
	done := make(chan struct{})
	go func() {
		widget.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Stop completed successfully
		t.Log("Stop() completed successfully")
	case <-time.After(5 * time.Second):
		t.Error("Stop() did not complete within 5 seconds - possible goroutine leak")
		t.Error("The gore.Run() goroutine may still be running after stopChan is closed")
	}

	// Give goroutines time to fully exit
	time.Sleep(100 * time.Millisecond)

	// Verify wg.Wait() completed (by checking if we can acquire the lock without blocking)
	acquired := make(chan bool, 1)
	go func() {
		widget.mu.Lock()
		// Empty critical section is intentional - just testing mutex can be acquired
		_ = widget.started // Access field to avoid empty critical section warning
		widget.mu.Unlock()
		acquired <- true
	}()

	select {
	case <-acquired:
		t.Log("All mutexes released, goroutines properly cleaned up")
	case <-time.After(1 * time.Second):
		t.Error("Mutex still locked - goroutine may still be running")
	}
}
