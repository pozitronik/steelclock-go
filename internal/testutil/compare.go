package testutil

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
)

// FrameDiff represents the difference between two frames
type FrameDiff struct {
	Identical       bool    // True if frames are exactly equal
	DifferentPixels int     // Number of differing pixels
	TotalPixels     int     // Total pixels compared
	DifferenceRatio float64 // Ratio of different pixels (0.0 - 1.0)
	MaxDifference   int     // Maximum per-pixel difference
	AverageDiff     float64 // Average difference across all pixels
	DiffPositions   []int   // Indices of differing pixels (limited to first 100)
	ExpectedSize    int
	ActualSize      int
	SizeMismatch    bool
}

// CompareFrames compares two frames and returns detailed diff information
func CompareFrames(expected, actual []int) *FrameDiff {
	diff := &FrameDiff{
		ExpectedSize: len(expected),
		ActualSize:   len(actual),
		TotalPixels:  len(expected) * 8, // Each byte contains 8 pixels (1-bit)
	}

	if len(expected) != len(actual) {
		diff.SizeMismatch = true
		return diff
	}

	if len(expected) == 0 {
		diff.Identical = true
		return diff
	}

	totalDiff := 0
	for i := 0; i < len(expected); i++ {
		if expected[i] != actual[i] {
			byteDiff := countBitDifferences(expected[i], actual[i])
			diff.DifferentPixels += byteDiff
			totalDiff += absDiff(expected[i], actual[i])

			if absDiff(expected[i], actual[i]) > diff.MaxDifference {
				diff.MaxDifference = absDiff(expected[i], actual[i])
			}

			if len(diff.DiffPositions) < 100 {
				diff.DiffPositions = append(diff.DiffPositions, i)
			}
		}
	}

	diff.Identical = diff.DifferentPixels == 0
	diff.DifferenceRatio = float64(diff.DifferentPixels) / float64(diff.TotalPixels)
	diff.AverageDiff = float64(totalDiff) / float64(len(expected))

	return diff
}

// CompareFramesWithTolerance compares frames allowing some pixel differences
func CompareFramesWithTolerance(expected, actual []int, tolerancePixels int) bool {
	diff := CompareFrames(expected, actual)
	return diff.DifferentPixels <= tolerancePixels
}

// CompareFramesWithRatio compares frames allowing a ratio of differences
func CompareFramesWithRatio(expected, actual []int, maxRatio float64) bool {
	diff := CompareFrames(expected, actual)
	return diff.DifferenceRatio <= maxRatio
}

// CompareRegion compares a rectangular region of two frames.
// Frames are 128x40 packed as 640 bytes (128/8 * 40 = 16 * 40)
func CompareRegion(expected, actual []int, x, y, width, height int) *FrameDiff {
	const frameWidth = 128
	const frameHeight = 40
	const bytesPerRow = frameWidth / 8 // 16 bytes per row

	diff := &FrameDiff{
		TotalPixels: width * height,
	}

	if len(expected) != 640 || len(actual) != 640 {
		diff.SizeMismatch = true
		return diff
	}

	// Validate bounds
	if x < 0 || y < 0 || x+width > frameWidth || y+height > frameHeight {
		diff.SizeMismatch = true
		return diff
	}

	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			byteIdx := row*bytesPerRow + col/8
			bitIdx := 7 - (col % 8) // MSB first

			expectedBit := (expected[byteIdx] >> bitIdx) & 1
			actualBit := (actual[byteIdx] >> bitIdx) & 1

			if expectedBit != actualBit {
				diff.DifferentPixels++
				if len(diff.DiffPositions) < 100 {
					diff.DiffPositions = append(diff.DiffPositions, row*frameWidth+col)
				}
			}
		}
	}

	diff.Identical = diff.DifferentPixels == 0
	diff.DifferenceRatio = float64(diff.DifferentPixels) / float64(diff.TotalPixels)

	return diff
}

// CountSetPixels counts the number of white (set) pixels in a frame
func CountSetPixels(frame []int) int {
	count := 0
	for _, b := range frame {
		count += countBits(b)
	}
	return count
}

// IsBlankFrame returns true if the frame has no set pixels
func IsBlankFrame(frame []int) bool {
	for _, b := range frame {
		if b != 0 {
			return false
		}
	}
	return true
}

// IsFullFrame returns true if all pixels are set
func IsFullFrame(frame []int) bool {
	for _, b := range frame {
		if b != 255 {
			return false
		}
	}
	return true
}

// GetPixel returns the value of a single pixel (0 or 1)
func GetPixel(frame []int, x, y int) int {
	const bytesPerRow = 16 // 128 / 8

	if len(frame) != 640 || x < 0 || x >= 128 || y < 0 || y >= 40 {
		return -1 // Invalid
	}

	byteIdx := y*bytesPerRow + x/8
	bitIdx := 7 - (x % 8)
	return (frame[byteIdx] >> bitIdx) & 1
}

// GetRow extracts a single row of pixels as 128 individual bits
func GetRow(frame []int, row int) []int {
	const bytesPerRow = 16

	if len(frame) != 640 || row < 0 || row >= 40 {
		return nil
	}

	pixels := make([]int, 128)
	startByte := row * bytesPerRow

	for i := 0; i < bytesPerRow; i++ {
		b := frame[startByte+i]
		for bit := 0; bit < 8; bit++ {
			pixels[i*8+bit] = (b >> (7 - bit)) & 1
		}
	}

	return pixels
}

// FrameToASCII converts a frame to ASCII art for visualization
func FrameToASCII(frame []int) string {
	if len(frame) != 640 {
		return fmt.Sprintf("<invalid frame size: %d>", len(frame))
	}

	var sb strings.Builder
	sb.WriteString("+" + strings.Repeat("-", 128) + "+\n")

	for row := 0; row < 40; row++ {
		sb.WriteString("|")
		pixels := GetRow(frame, row)
		for _, p := range pixels {
			if p == 1 {
				sb.WriteString("#")
			} else {
				sb.WriteString(" ")
			}
		}
		sb.WriteString("|\n")
	}

	sb.WriteString("+" + strings.Repeat("-", 128) + "+")
	return sb.String()
}

// FrameDiffToASCII creates ASCII visualization showing differences
// Expected pixels shown as '.', actual as '+', matching set as '#', matching unset as ' '
func FrameDiffToASCII(expected, actual []int) string {
	if len(expected) != 640 || len(actual) != 640 {
		return "<size mismatch>"
	}

	var sb strings.Builder
	sb.WriteString("Legend: ' '=both unset, '#'=both set, 'E'=expected only, 'A'=actual only\n")
	sb.WriteString("+" + strings.Repeat("-", 128) + "+\n")

	for row := 0; row < 40; row++ {
		sb.WriteString("|")
		expectedPixels := GetRow(expected, row)
		actualPixels := GetRow(actual, row)

		for i := 0; i < 128; i++ {
			e, a := expectedPixels[i], actualPixels[i]
			switch {
			case e == 1 && a == 1:
				sb.WriteString("#") // Both set
			case e == 0 && a == 0:
				sb.WriteString(" ") // Both unset
			case e == 1 && a == 0:
				sb.WriteString("E") // Expected only
			case e == 0 && a == 1:
				sb.WriteString("A") // Actual only
			}
		}
		sb.WriteString("|\n")
	}

	sb.WriteString("+" + strings.Repeat("-", 128) + "+")
	return sb.String()
}

// SaveFrameAsPNG saves a frame as a PNG file for debugging
func SaveFrameAsPNG(frame []int, filename string) error {
	if len(frame) != 640 {
		return fmt.Errorf("invalid frame size: %d", len(frame))
	}

	img := image.NewGray(image.Rect(0, 0, 128, 40))

	for row := 0; row < 40; row++ {
		pixels := GetRow(frame, row)
		for col, p := range pixels {
			if p == 1 {
				img.Set(col, row, color.White)
			} else {
				img.Set(col, row, color.Black)
			}
		}
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	return png.Encode(f, img)
}

// SaveDiffAsPNG saves a diff visualization as PNG
// Red = expected only, Green = actual only, White = both set, Black = both unset
func SaveDiffAsPNG(expected, actual []int, filename string) error {
	if len(expected) != 640 || len(actual) != 640 {
		return fmt.Errorf("invalid frame sizes")
	}

	img := image.NewRGBA(image.Rect(0, 0, 128, 40))

	for row := 0; row < 40; row++ {
		expectedPixels := GetRow(expected, row)
		actualPixels := GetRow(actual, row)

		for col := 0; col < 128; col++ {
			e, a := expectedPixels[col], actualPixels[col]
			var c color.RGBA
			switch {
			case e == 1 && a == 1:
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255} // White - both set
			case e == 0 && a == 0:
				c = color.RGBA{R: 0, G: 0, B: 0, A: 255} // Black - both unset
			case e == 1 && a == 0:
				c = color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red - expected only
			case e == 0 && a == 1:
				c = color.RGBA{R: 0, G: 255, B: 0, A: 255} // Green - actual only
			}
			img.Set(col, row, c)
		}
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	return png.Encode(f, img)
}

// Helper functions

func countBits(n int) int {
	count := 0
	for n != 0 {
		count += n & 1
		n >>= 1
	}
	return count
}

func countBitDifferences(a, b int) int {
	return countBits(a ^ b)
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
