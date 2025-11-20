package widget

import (
	"image"
	"testing"
	"time"
)

func TestNewErrorWidget(t *testing.T) {
	widget := NewErrorWidget(128, 40, "TEST")

	if widget == nil {
		t.Fatal("NewErrorWidget() returned nil")
	}

	if widget.Name() != "error_display" {
		t.Errorf("Name() = %s, want error_display", widget.Name())
	}

	if widget.message != "TEST" {
		t.Errorf("message = %s, want TEST", widget.message)
	}

	if !widget.flashState {
		t.Error("flashState should start as true")
	}
}

func TestErrorWidget_Update(t *testing.T) {
	widget := NewErrorWidget(128, 40, "CONFIG")

	// Initial state
	initialState := widget.flashState

	// Update immediately - should not change yet
	_ = widget.Update()
	if widget.flashState != initialState {
		t.Error("flashState should not change immediately")
	}

	// Wait for flash period
	time.Sleep(550 * time.Millisecond)
	_ = widget.Update()

	if widget.flashState == initialState {
		t.Error("flashState should toggle after flash period")
	}
}

func TestErrorWidget_Render_FlashOn(t *testing.T) {
	widget := NewErrorWidget(128, 40, "CONFIG")
	widget.flashState = true

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	grayImg := img.(*image.Gray)

	// When flash is on, should have visible pixels
	hasPixels := false
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
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
		t.Error("Render() with flash on should have visible pixels")
	}
}

func TestErrorWidget_Render_FlashOff(t *testing.T) {
	widget := NewErrorWidget(128, 40, "CONFIG")
	widget.flashState = false

	img, err := widget.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	grayImg := img.(*image.Gray)

	// When flash is off, should be all black (background)
	hasNonZeroPixels := false
	for y := 0; y < 40; y++ {
		for x := 0; x < 128; x++ {
			if grayImg.GrayAt(x, y).Y > 0 {
				hasNonZeroPixels = true
				break
			}
		}
		if hasNonZeroPixels {
			break
		}
	}

	if hasNonZeroPixels {
		t.Error("Render() with flash off should have no visible pixels")
	}
}

func TestErrorWidget_Messages(t *testing.T) {
	messages := []string{
		"CONFIG",
		"NO WIDGETS",
		"ERROR",
	}

	for _, msg := range messages {
		t.Run(msg, func(t *testing.T) {
			widget := NewErrorWidget(128, 40, msg)
			widget.flashState = true

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v for message %s", err, msg)
			}

			if img == nil {
				t.Fatalf("Render() returned nil image for message %s", msg)
			}

			// Verify image has content
			grayImg := img.(*image.Gray)
			hasContent := false
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					if grayImg.GrayAt(x, y).Y > 0 {
						hasContent = true
						break
					}
				}
				if hasContent {
					break
				}
			}

			if !hasContent {
				t.Errorf("Render() for message %s has no visible content", msg)
			}
		})
	}
}

func TestErrorWidget_DifferentSizes(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{128, 40},
		{64, 32},
		{256, 64},
	}

	for _, size := range sizes {
		t.Run("size", func(t *testing.T) {
			widget := NewErrorWidget(size.width, size.height, "TEST")

			img, err := widget.Render()
			if err != nil {
				t.Fatalf("Render() error = %v for size %dx%d", err, size.width, size.height)
			}

			if img.Bounds().Dx() != size.width {
				t.Errorf("image width = %d, want %d", img.Bounds().Dx(), size.width)
			}

			if img.Bounds().Dy() != size.height {
				t.Errorf("image height = %d, want %d", img.Bounds().Dy(), size.height)
			}
		})
	}
}
