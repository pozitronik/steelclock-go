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
