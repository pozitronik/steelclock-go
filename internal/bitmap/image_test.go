package bitmap

import (
	"image"
	"image/color"
	"testing"
)

func TestNewGrayscaleImage(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		background uint8
	}{
		{"Small black", 10, 10, 0},
		{"Small white", 10, 10, 255},
		{"Display size", 128, 40, 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := NewGrayscaleImage(tt.width, tt.height, tt.background)

			if img.Bounds().Dx() != tt.width {
				t.Errorf("width = %d, want %d", img.Bounds().Dx(), tt.width)
			}
			if img.Bounds().Dy() != tt.height {
				t.Errorf("height = %d, want %d", img.Bounds().Dy(), tt.height)
			}

			// Check background color
			pixel := img.GrayAt(0, 0).Y
			if pixel != tt.background {
				t.Errorf("background = %d, want %d", pixel, tt.background)
			}
		})
	}
}

func TestImageToBytes(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		expected int // expected byte count
	}{
		{"Standard display", 128, 40, 640},
		{"Single row", 8, 1, 1},
		{"Single column", 1, 8, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewGray(image.Rect(0, 0, tt.width, tt.height))
			bytes, err := ImageToBytes(img, tt.width, tt.height)

			if err != nil {
				t.Fatalf("ImageToBytes() error = %v", err)
			}

			if len(bytes) != tt.expected {
				t.Errorf("len(bytes) = %d, want %d", len(bytes), tt.expected)
			}
		})
	}
}

func TestImageToBytesPattern(t *testing.T) {
	// Create a simple pattern: white-black-white-black...
	img := image.NewGray(image.Rect(0, 0, 8, 1))
	for x := 0; x < 8; x++ {
		if x%2 == 0 {
			img.SetGray(x, 0, color.Gray{Y: 255})
		} else {
			img.SetGray(x, 0, color.Gray{Y: 0})
		}
	}

	bytes, err := ImageToBytes(img, 8, 1)
	if err != nil {
		t.Fatalf("ImageToBytes() error = %v", err)
	}

	// Should produce 0b10101010 = 170 (MSB first)
	if len(bytes) != 1 {
		t.Fatalf("len(bytes) = %d, want 1", len(bytes))
	}
	if bytes[0] != 170 {
		t.Errorf("bytes[0] = %d, want 170", bytes[0])
	}
}

func TestFloydSteinbergDither(t *testing.T) {
	// Create gradient image
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			// Create gradient from 0 to 255
			val := uint8(x * 25)
			img.SetGray(x, y, color.Gray{Y: val})
		}
	}

	result := FloydSteinbergDither(img)

	// Check that result is monochrome (only 0 or 255)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			val := result.GrayAt(x, y).Y
			if val != 0 && val != 255 {
				t.Errorf("pixel at (%d,%d) = %d, want 0 or 255", x, y, val)
			}
		}
	}
}

func TestDrawBorder(t *testing.T) {
	img := NewGrayscaleImage(10, 10, 0)
	DrawBorder(img, 255)

	// Check corners
	corners := [][2]int{{0, 0}, {9, 0}, {0, 9}, {9, 9}}
	for _, corner := range corners {
		x, y := corner[0], corner[1]
		pixel := img.GrayAt(x, y).Y
		if pixel != 255 {
			t.Errorf("corner (%d,%d) = %d, want 255", x, y, pixel)
		}
	}

	// Check center (should be background)
	centerPixel := img.GrayAt(5, 5).Y
	if centerPixel != 0 {
		t.Errorf("center = %d, want 0", centerPixel)
	}
}

func TestResizeImage_Upscale(t *testing.T) {
	// Create small 4x4 image
	srcImg := image.NewGray(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			srcImg.SetGray(x, y, color.Gray{Y: 100})
		}
	}

	// Resize to 8x8 (upscale)
	result := ResizeImage(srcImg, 8, 8)

	if result.Bounds().Dx() != 8 {
		t.Errorf("Resized width = %d, want 8", result.Bounds().Dx())
	}

	if result.Bounds().Dy() != 8 {
		t.Errorf("Resized height = %d, want 8", result.Bounds().Dy())
	}

	// Verify pixels were sampled
	grayResult, ok := result.(*image.Gray)
	if !ok {
		t.Fatal("ResizeImage did not return *image.Gray")
	}

	// At least some pixels should have value 100
	hasCorrectColor := false
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if grayResult.GrayAt(x, y).Y == 100 {
				hasCorrectColor = true
				break
			}
		}
	}

	if !hasCorrectColor {
		t.Error("Resized image does not contain expected pixel values")
	}
}

func TestResizeImage_Downscale(t *testing.T) {
	// Create 16x16 image
	srcImg := image.NewGray(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			srcImg.SetGray(x, y, color.Gray{Y: 150})
		}
	}

	// Resize to 8x8 (downscale)
	result := ResizeImage(srcImg, 8, 8)

	if result.Bounds().Dx() != 8 {
		t.Errorf("Resized width = %d, want 8", result.Bounds().Dx())
	}

	if result.Bounds().Dy() != 8 {
		t.Errorf("Resized height = %d, want 8", result.Bounds().Dy())
	}
}

func TestResizeImage_SameSize(t *testing.T) {
	// Create 10x10 image
	srcImg := image.NewGray(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			srcImg.SetGray(x, y, color.Gray{Y: uint8(x * 25)})
		}
	}

	// Resize to same size (10x10)
	result := ResizeImage(srcImg, 10, 10)

	if result.Bounds().Dx() != 10 || result.Bounds().Dy() != 10 {
		t.Errorf("Resized size = %dx%d, want 10x10", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestResizeImage_ToSmall(t *testing.T) {
	// Create 100x100 image
	srcImg := image.NewGray(image.Rect(0, 0, 100, 100))

	// Resize to very small 1x1
	result := ResizeImage(srcImg, 1, 1)

	if result.Bounds().Dx() != 1 || result.Bounds().Dy() != 1 {
		t.Errorf("Resized size = %dx%d, want 1x1", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestResizeImage_ToLarge(t *testing.T) {
	// Create small 2x2 image
	srcImg := image.NewGray(image.Rect(0, 0, 2, 2))
	srcImg.SetGray(0, 0, color.Gray{Y: 255})
	srcImg.SetGray(1, 0, color.Gray{Y: 200})
	srcImg.SetGray(0, 1, color.Gray{Y: 150})
	srcImg.SetGray(1, 1, color.Gray{Y: 100})

	// Resize to large 100x100
	result := ResizeImage(srcImg, 100, 100)

	if result.Bounds().Dx() != 100 || result.Bounds().Dy() != 100 {
		t.Errorf("Resized size = %dx%d, want 100x100", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestResizeImage_AspectRatioChange(t *testing.T) {
	// Create 16x9 image
	srcImg := image.NewGray(image.Rect(0, 0, 16, 9))

	// Resize to 4x3 (different aspect ratio)
	result := ResizeImage(srcImg, 4, 3)

	if result.Bounds().Dx() != 4 || result.Bounds().Dy() != 3 {
		t.Errorf("Resized size = %dx%d, want 4x3", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestResizeImage_NonGrayInput(t *testing.T) {
	// Create RGBA image
	srcImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			srcImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// Resize should handle RGBA input and convert to Gray
	result := ResizeImage(srcImg, 5, 5)

	if result.Bounds().Dx() != 5 || result.Bounds().Dy() != 5 {
		t.Errorf("Resized size = %dx%d, want 5x5", result.Bounds().Dx(), result.Bounds().Dy())
	}

	_, ok := result.(*image.Gray)
	if !ok {
		t.Error("ResizeImage should always return *image.Gray")
	}
}

func TestResizeImage_OffsetBounds(t *testing.T) {
	// Create image with non-zero Min
	srcImg := image.NewGray(image.Rect(10, 10, 20, 20))
	for y := 10; y < 20; y++ {
		for x := 10; x < 20; x++ {
			srcImg.SetGray(x, y, color.Gray{Y: 128})
		}
	}

	// Resize should handle offset bounds correctly
	result := ResizeImage(srcImg, 5, 5)

	if result.Bounds().Dx() != 5 || result.Bounds().Dy() != 5 {
		t.Errorf("Resized size = %dx%d, want 5x5", result.Bounds().Dx(), result.Bounds().Dy())
	}
}

func TestImageToBytes_WrongSize(t *testing.T) {
	// Create 8x8 image but request 16x16 conversion
	img := image.NewGray(image.Rect(0, 0, 8, 8))

	// ImageToBytes will resize the image to match requested dimensions
	bytes, err := ImageToBytes(img, 16, 16)
	if err != nil {
		t.Errorf("ImageToBytes() should resize mismatched image, got error: %v", err)
	}

	// Should produce bytes for 16x16 image
	expectedBytes := (16*16 + 7) / 8
	if len(bytes) != expectedBytes {
		t.Errorf("len(bytes) = %d, want %d for 16x16 image", len(bytes), expectedBytes)
	}
}

func TestImageToBytes_EmptyImage(t *testing.T) {
	// Create 0x0 image
	img := image.NewGray(image.Rect(0, 0, 0, 0))

	bytes, err := ImageToBytes(img, 0, 0)
	if err != nil {
		t.Errorf("ImageToBytes() with 0x0 image should not error, got: %v", err)
	}

	if len(bytes) != 0 {
		t.Errorf("ImageToBytes() for 0x0 image should return empty slice, got %d bytes", len(bytes))
	}
}

func TestFloydSteinbergDither_AllWhite(t *testing.T) {
	// Create all-white image
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	result := FloydSteinbergDither(img)

	// All pixels should remain white
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if result.GrayAt(x, y).Y != 255 {
				t.Errorf("Pixel at (%d,%d) = %d, want 255", x, y, result.GrayAt(x, y).Y)
			}
		}
	}
}

func TestFloydSteinbergDither_AllBlack(t *testing.T) {
	// Create all-black image
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	// Default is black, no need to set

	result := FloydSteinbergDither(img)

	// All pixels should remain black
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if result.GrayAt(x, y).Y != 0 {
				t.Errorf("Pixel at (%d,%d) = %d, want 0", x, y, result.GrayAt(x, y).Y)
			}
		}
	}
}

func TestFloydSteinbergDither_MidGray(t *testing.T) {
	// Create mid-gray image (128)
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetGray(x, y, color.Gray{Y: 128})
		}
	}

	result := FloydSteinbergDither(img)

	// Count black and white pixels
	blackPixels := 0
	whitePixels := 0

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			val := result.GrayAt(x, y).Y
			if val == 0 {
				blackPixels++
			} else if val == 255 {
				whitePixels++
			}
		}
	}

	// For mid-gray, should have approximately equal black and white
	total := blackPixels + whitePixels
	if total != 100 {
		t.Errorf("Total pixels = %d, want 100", total)
	}

	// Should have some of both colors
	if blackPixels == 0 || whitePixels == 0 {
		t.Errorf("Mid-gray dither should produce both black (%d) and white (%d)", blackPixels, whitePixels)
	}
}

func TestNewGrayscaleImage_ZeroSize(t *testing.T) {
	img := NewGrayscaleImage(0, 0, 128)

	if img.Bounds().Dx() != 0 || img.Bounds().Dy() != 0 {
		t.Errorf("Size = %dx%d, want 0x0", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestNewGrayscaleImage_LargeSize(t *testing.T) {
	// Create large image
	img := NewGrayscaleImage(1000, 1000, 64)

	if img.Bounds().Dx() != 1000 || img.Bounds().Dy() != 1000 {
		t.Errorf("Size = %dx%d, want 1000x1000", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Verify background in corner
	if img.GrayAt(0, 0).Y != 64 {
		t.Errorf("Background = %d, want 64", img.GrayAt(0, 0).Y)
	}
}
