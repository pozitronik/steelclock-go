package testutil

import (
	"os"
	"testing"
)

func TestCompareFrames_Identical(t *testing.T) {
	frame := make([]byte, 640)
	frame[0] = 0xFF
	frame[100] = 0xAA

	diff := CompareFrames(frame, frame)

	if !diff.Identical {
		t.Error("Identical frames should be marked as identical")
	}
	if diff.DifferentPixels != 0 {
		t.Errorf("Expected 0 different pixels, got %d", diff.DifferentPixels)
	}
}

func TestCompareFrames_Different(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	frame1[0] = 0xFF // 8 bits set
	frame2[0] = 0x00 // 0 bits set

	diff := CompareFrames(frame1, frame2)

	if diff.Identical {
		t.Error("Different frames should not be marked as identical")
	}
	if diff.DifferentPixels != 8 {
		t.Errorf("Expected 8 different pixels, got %d", diff.DifferentPixels)
	}
}

func TestCompareFrames_SizeMismatch(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 320)

	diff := CompareFrames(frame1, frame2)

	if !diff.SizeMismatch {
		t.Error("Should detect size mismatch")
	}
}

func TestCompareFramesWithTolerance(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	frame1[0] = 0xFF // 8 pixels different
	frame2[0] = 0x00

	// Should pass with tolerance of 8
	if !CompareFramesWithTolerance(frame1, frame2, 8) {
		t.Error("Should pass with tolerance of 8")
	}

	// Should fail with tolerance of 7
	if CompareFramesWithTolerance(frame1, frame2, 7) {
		t.Error("Should fail with tolerance of 7")
	}
}

func TestCompareFramesWithRatio(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// Set all pixels in first frame
	for i := range frame1 {
		frame1[i] = 0xFF
	}
	// Leave second frame blank
	// Total pixels = 640 * 8 = 5120, all different

	// 100% different - should fail at 50% tolerance
	if CompareFramesWithRatio(frame1, frame2, 0.5) {
		t.Error("Should fail when 100% of pixels differ with 50% tolerance")
	}

	// Should pass at 100% tolerance
	if !CompareFramesWithRatio(frame1, frame2, 1.0) {
		t.Error("Should pass with 100% tolerance")
	}
}

func TestCompareRegion(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// Set a pixel at position (0,0) in frame1
	// Byte 0, bit 7 (MSB)
	frame1[0] = 0x80

	// Compare the first 8x8 region
	diff := CompareRegion(frame1, frame2, 0, 0, 8, 8)

	if diff.Identical {
		t.Error("Regions should not be identical")
	}
	if diff.DifferentPixels != 1 {
		t.Errorf("Expected 1 different pixel in region, got %d", diff.DifferentPixels)
	}

	// Compare a region that doesn't include the difference
	diff2 := CompareRegion(frame1, frame2, 8, 0, 8, 8)
	if !diff2.Identical {
		t.Error("Region without difference should be identical")
	}
}

func TestCountSetPixels(t *testing.T) {
	frame := make([]byte, 640)
	frame[0] = 0xFF // 8 pixels
	frame[1] = 0x0F // 4 pixels

	count := CountSetPixels(frame)
	if count != 12 {
		t.Errorf("Expected 12 set pixels, got %d", count)
	}
}

func TestIsBlankFrame(t *testing.T) {
	blankFrame := make([]byte, 640)
	nonBlankFrame := make([]byte, 640)
	nonBlankFrame[100] = 1

	if !IsBlankFrame(blankFrame) {
		t.Error("Blank frame should be detected as blank")
	}
	if IsBlankFrame(nonBlankFrame) {
		t.Error("Non-blank frame should not be detected as blank")
	}
}

func TestIsFullFrame(t *testing.T) {
	fullFrame := make([]byte, 640)
	for i := range fullFrame {
		fullFrame[i] = 255
	}
	nonFullFrame := make([]byte, 640)
	for i := range nonFullFrame {
		nonFullFrame[i] = 255
	}
	nonFullFrame[0] = 0

	if !IsFullFrame(fullFrame) {
		t.Error("Full frame should be detected as full")
	}
	if IsFullFrame(nonFullFrame) {
		t.Error("Non-full frame should not be detected as full")
	}
}

func TestGetPixel(t *testing.T) {
	frame := make([]byte, 640)
	// Set pixel at (0,0) - byte 0, bit 7
	frame[0] = 0x80
	// Set pixel at (7,0) - byte 0, bit 0
	frame[0] |= 0x01
	// Set pixel at (8,0) - byte 1, bit 7
	frame[1] = 0x80

	tests := []struct {
		x, y     int
		expected int
	}{
		{0, 0, 1},    // First pixel
		{1, 0, 0},    // Second pixel
		{7, 0, 1},    // 8th pixel (still byte 0)
		{8, 0, 1},    // 9th pixel (byte 1)
		{9, 0, 0},    // 10th pixel
		{0, 1, 0},    // First pixel of second row
		{-1, 0, -1},  // Invalid
		{128, 0, -1}, // Invalid
	}

	for _, tc := range tests {
		got := GetPixel(frame, tc.x, tc.y)
		if got != tc.expected {
			t.Errorf("GetPixel(%d, %d): expected %d, got %d", tc.x, tc.y, tc.expected, got)
		}
	}
}

func TestGetRow(t *testing.T) {
	frame := make([]byte, 640)
	// Set first byte of row 0 to all 1s
	frame[0] = 0xFF
	// Set first byte of row 1 to 0xAA (10101010)
	frame[16] = 0xAA

	row0 := GetRow(frame, 0)
	if len(row0) != 128 {
		t.Errorf("Expected row length 128, got %d", len(row0))
	}
	// First 8 pixels should all be 1
	for i := 0; i < 8; i++ {
		if row0[i] != 1 {
			t.Errorf("Row 0, pixel %d: expected 1, got %d", i, row0[i])
		}
	}
	// Pixel 8 should be 0
	if row0[8] != 0 {
		t.Errorf("Row 0, pixel 8: expected 0, got %d", row0[8])
	}

	row1 := GetRow(frame, 1)
	// 0xAA = 10101010, so pixels 0,2,4,6 are 1, pixels 1,3,5,7 are 0
	expectedRow1 := []int{1, 0, 1, 0, 1, 0, 1, 0}
	for i, expected := range expectedRow1 {
		if row1[i] != expected {
			t.Errorf("Row 1, pixel %d: expected %d, got %d", i, expected, row1[i])
		}
	}
}

func TestFrameToASCII(t *testing.T) {
	frame := make([]byte, 640)
	// Set first 4 pixels of first row
	frame[0] = 0xF0 // 11110000

	ascii := FrameToASCII(frame)

	if len(ascii) == 0 {
		t.Error("FrameToASCII returned empty string")
	}
	// Should contain the border and at least one line with content
	if ascii[0] != '+' {
		t.Error("Should start with border character")
	}
}

func TestFrameDiffToASCII(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)
	frame1[0] = 0xF0 // Expected only
	frame2[1] = 0x0F // Actual only

	ascii := FrameDiffToASCII(frame1, frame2)

	if len(ascii) == 0 {
		t.Error("FrameDiffToASCII returned empty string")
	}
}

func TestSaveFrameAsPNG(t *testing.T) {
	frame := make([]byte, 640)
	frame[0] = 0xFF // Some visible pixels

	tmpFile := os.TempDir() + "/test_frame.png"
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile)

	err := SaveFrameAsPNG(frame, tmpFile)
	if err != nil {
		t.Errorf("SaveFrameAsPNG failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("PNG file was not created")
	}
}

func TestSaveDiffAsPNG(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)
	frame1[0] = 0xF0
	frame2[1] = 0x0F

	tmpFile := os.TempDir() + "/test_diff.png"
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile)

	err := SaveDiffAsPNG(frame1, frame2, tmpFile)
	if err != nil {
		t.Errorf("SaveDiffAsPNG failed: %v", err)
	}

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Diff PNG file was not created")
	}
}

func TestFrameDiff_DifferenceRatio(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// All pixels in byte 0 different (8 pixels out of 5120 total)
	frame1[0] = 0xFF

	diff := CompareFrames(frame1, frame2)

	expectedRatio := 8.0 / 5120.0
	if diff.DifferenceRatio != expectedRatio {
		t.Errorf("Expected ratio %f, got %f", expectedRatio, diff.DifferenceRatio)
	}
}
