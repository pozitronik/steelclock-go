package bitmap

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

// NewGrayscaleImage creates a new grayscale image
func NewGrayscaleImage(width, height int, bgColor uint8) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))

	// Fill with background color
	if bgColor != 0 {
		fillColor := color.Gray{Y: bgColor}
		draw.Draw(img, img.Bounds(), &image.Uniform{C: fillColor}, image.Point{}, draw.Src)
	}

	return img
}

// ImageToBytes converts an image to byte array for GameSense API
// Format: Monochrome, MSB first, row-major order
func ImageToBytes(img image.Image, width, height int) ([]int, error) {
	// Resize if needed
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		img = ResizeImage(img, width, height)
	}

	// Convert to grayscale first
	grayImg := toGrayscale(img)

	// Apply Floyd-Steinberg dithering to convert to 1-bit
	monoImg := FloydSteinbergDither(grayImg)

	// Convert to byte array
	bytes := make([]int, (width*height+7)/8)
	byteIndex := 0
	bitIndex := 0
	var currentByte byte

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get pixel value (0 or 1)
			pixel := monoImg.GrayAt(x, y).Y

			// Set bit (MSB first)
			if pixel > 0 {
				currentByte |= 1 << (7 - bitIndex)
			}

			bitIndex++
			if bitIndex == 8 {
				bytes[byteIndex] = int(currentByte)
				byteIndex++
				currentByte = 0
				bitIndex = 0
			}
		}
	}

	// Handle remaining bits
	if bitIndex > 0 {
		bytes[byteIndex] = int(currentByte)
	}

	expectedSize := (width*height + 7) / 8
	if len(bytes) != expectedSize {
		return nil, fmt.Errorf("unexpected bitmap size: got %d, expected %d", len(bytes), expectedSize)
	}

	return bytes, nil
}

// toGrayscale converts any image to grayscale
func toGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)
	return gray
}

// ResizeImage resizes an image using nearest neighbor
func ResizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewGray(image.Rect(0, 0, width, height))
	srcBounds := img.Bounds()

	scaleX := float64(srcBounds.Dx()) / float64(width)
	scaleY := float64(srcBounds.Dy()) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)
			dst.Set(x, y, img.At(srcX+srcBounds.Min.X, srcY+srcBounds.Min.Y))
		}
	}

	return dst
}

// FloydSteinbergDither applies Floyd-Steinberg dithering to convert grayscale to 1-bit
func FloydSteinbergDither(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Create a copy to work with
	result := image.NewGray(bounds)
	draw.Draw(result, bounds, img, bounds.Min, draw.Src)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldPixel := result.GrayAt(x, y).Y
			newPixel := uint8(0)
			if oldPixel > 127 {
				newPixel = 255
			}
			result.SetGray(x, y, color.Gray{Y: newPixel})

			quantError := int(oldPixel) - int(newPixel)

			// Distribute error to neighboring pixels
			if x+1 < width {
				distributeError(result, x+1, y, quantError, 7.0/16.0)
			}
			if y+1 < height {
				if x > 0 {
					distributeError(result, x-1, y+1, quantError, 3.0/16.0)
				}
				distributeError(result, x, y+1, quantError, 5.0/16.0)
				if x+1 < width {
					distributeError(result, x+1, y+1, quantError, 1.0/16.0)
				}
			}
		}
	}

	return result
}

func distributeError(img *image.Gray, x, y int, quantError int, factor float64) {
	current := img.GrayAt(x, y).Y
	newVal := int(current) + int(float64(quantError)*factor)
	if newVal < 0 {
		newVal = 0
	}
	if newVal > 255 {
		newVal = 255
	}
	img.SetGray(x, y, color.Gray{Y: uint8(newVal)})
}
