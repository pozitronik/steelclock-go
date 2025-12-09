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

// --- Additional Edge Case Tests for Coverage ---

func TestCompareFrames_EmptyFrames(t *testing.T) {
	// Both frames empty
	diff := CompareFrames([]byte{}, []byte{})
	if !diff.Identical {
		t.Error("Empty frames should be identical")
	}
	if diff.TotalPixels != 0 {
		t.Errorf("Expected 0 total pixels for empty frames, got %d", diff.TotalPixels)
	}
}

func TestCompareFrames_ManyDifferences(t *testing.T) {
	// Create frames with many differences to test DiffPositions limit
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// Set every byte different (640 differences, each contributes to DiffPositions)
	for i := range frame1 {
		frame1[i] = 0xFF
		frame2[i] = 0x00
	}

	diff := CompareFrames(frame1, frame2)

	// Should cap DiffPositions at 100
	if len(diff.DiffPositions) != 100 {
		t.Errorf("DiffPositions should be capped at 100, got %d", len(diff.DiffPositions))
	}

	// But should still count all different pixels
	if diff.DifferentPixels != 640*8 {
		t.Errorf("Expected %d different pixels, got %d", 640*8, diff.DifferentPixels)
	}
}

func TestCompareRegion_InvalidFrameSizes(t *testing.T) {
	tests := []struct {
		name     string
		expected []byte
		actual   []byte
	}{
		{"expected too small", make([]byte, 320), make([]byte, 640)},
		{"actual too small", make([]byte, 640), make([]byte, 320)},
		{"both too small", make([]byte, 100), make([]byte, 100)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diff := CompareRegion(tc.expected, tc.actual, 0, 0, 8, 8)
			if !diff.SizeMismatch {
				t.Error("Should report size mismatch")
			}
		})
	}
}

func TestCompareRegion_InvalidBounds(t *testing.T) {
	frame := make([]byte, 640)

	tests := []struct {
		name string
		x, y int
		w, h int
	}{
		{"negative x", -1, 0, 8, 8},
		{"negative y", 0, -1, 8, 8},
		{"x + width > 128", 125, 0, 8, 8},
		{"y + height > 40", 0, 38, 8, 8},
		{"far out of bounds", 200, 200, 10, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diff := CompareRegion(frame, frame, tc.x, tc.y, tc.w, tc.h)
			if !diff.SizeMismatch {
				t.Error("Should report size mismatch for invalid bounds")
			}
		})
	}
}

func TestCompareRegion_ManyDifferentPixels(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// Make entire frames different
	for i := range frame1 {
		frame1[i] = 0xFF
	}

	// Compare large region to test DiffPositions cap
	diff := CompareRegion(frame1, frame2, 0, 0, 128, 40)

	// Should cap at 100
	if len(diff.DiffPositions) != 100 {
		t.Errorf("DiffPositions should be capped at 100, got %d", len(diff.DiffPositions))
	}
}

func TestGetRow_InvalidInputs(t *testing.T) {
	frame := make([]byte, 640)

	tests := []struct {
		name  string
		frame []byte
		row   int
	}{
		{"invalid frame size", make([]byte, 100), 0},
		{"negative row", frame, -1},
		{"row too large", frame, 40},
		{"row way too large", frame, 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetRow(tc.frame, tc.row)
			if result != nil {
				t.Error("Expected nil for invalid input")
			}
		})
	}
}

func TestFrameToASCII_InvalidSize(t *testing.T) {
	invalidFrame := make([]byte, 100)
	ascii := FrameToASCII(invalidFrame)
	if ascii == "" {
		t.Error("Should return non-empty string for invalid frame")
	}
	if len(ascii) < 20 {
		t.Error("Should contain error message")
	}
}

func TestFrameDiffToASCII_SizeMismatch(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 100)

	ascii := FrameDiffToASCII(frame1, frame2)
	if ascii != "<size mismatch>" {
		t.Errorf("Expected '<size mismatch>', got '%s'", ascii)
	}

	// Also test when expected is wrong size
	ascii2 := FrameDiffToASCII(frame2, frame1)
	if ascii2 != "<size mismatch>" {
		t.Errorf("Expected '<size mismatch>', got '%s'", ascii2)
	}
}

func TestSaveFrameAsPNG_InvalidFrameSize(t *testing.T) {
	invalidFrame := make([]byte, 100)
	err := SaveFrameAsPNG(invalidFrame, "/tmp/test.png")
	if err == nil {
		t.Error("Expected error for invalid frame size")
	}
}

func TestSaveFrameAsPNG_InvalidPath(t *testing.T) {
	frame := make([]byte, 640)
	// Use an invalid path
	err := SaveFrameAsPNG(frame, "/nonexistent/directory/file.png")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestSaveDiffAsPNG_InvalidFrameSizes(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 100)

	err := SaveDiffAsPNG(frame1, frame2, "/tmp/test.png")
	if err == nil {
		t.Error("Expected error for invalid frame sizes")
	}

	err = SaveDiffAsPNG(frame2, frame1, "/tmp/test.png")
	if err == nil {
		t.Error("Expected error for invalid frame sizes")
	}
}

func TestSaveDiffAsPNG_InvalidPath(t *testing.T) {
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	err := SaveDiffAsPNG(frame1, frame2, "/nonexistent/directory/file.png")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestAbsDiff_BothDirections(t *testing.T) {
	// Test through CompareFrames with specific values
	// to ensure both branches of absDiff are covered
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// a > b case
	frame1[0] = 200
	frame2[0] = 100

	// a <= b case (a < b)
	frame1[1] = 50
	frame2[1] = 150

	diff := CompareFrames(frame1, frame2)
	// Just verify it doesn't panic and produces valid results
	if diff.MaxDifference < 100 {
		t.Errorf("MaxDifference should be at least 100, got %d", diff.MaxDifference)
	}
}

func TestGetPixel_InvalidFrame(t *testing.T) {
	// Test invalid frame size
	invalidFrame := make([]byte, 100)
	result := GetPixel(invalidFrame, 0, 0)
	if result != -1 {
		t.Errorf("Expected -1 for invalid frame, got %d", result)
	}

	// Test invalid y coordinate
	validFrame := make([]byte, 640)
	result = GetPixel(validFrame, 0, -1)
	if result != -1 {
		t.Errorf("Expected -1 for negative y, got %d", result)
	}

	result = GetPixel(validFrame, 0, 40)
	if result != -1 {
		t.Errorf("Expected -1 for y=40, got %d", result)
	}
}

func TestFrameDiffToASCII_AllCases(t *testing.T) {
	// Create frames that exercise all 4 cases in the switch statement
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// Set up pixels to cover all 4 cases:
	// Position 0,0 - both set (case 1)
	// Position 1,0 - expected only (case 3)
	// Position 2,0 - actual only (case 4)
	// Position 3,0 - both unset (case 2) - default

	// Byte 0: bits 7,6,5,4,3,2,1,0 correspond to x positions 0,1,2,3,4,5,6,7
	// Set position 0 in both frames (bit 7)
	frame1[0] |= 0x80 // Position 0,0 set in expected
	frame2[0] |= 0x80 // Position 0,0 set in actual -> both set

	// Set position 1 in expected only (bit 6)
	frame1[0] |= 0x40 // Position 1,0 set in expected only -> expected only

	// Set position 2 in actual only (bit 5)
	frame2[0] |= 0x20 // Position 2,0 set in actual only -> actual only

	// Position 3 (bit 4) is unset in both -> both unset

	ascii := FrameDiffToASCII(frame1, frame2)

	// Verify the output contains all expected characters
	if len(ascii) == 0 {
		t.Error("FrameDiffToASCII should return non-empty string")
	}

	// The first line contains the legend, second line the border
	// Content starts from line 3
	// Should contain '#' (both set), 'E' (expected only), 'A' (actual only), ' ' (both unset)
	if !containsChar(ascii, '#') {
		t.Error("ASCII should contain '#' for both-set pixels")
	}
	if !containsChar(ascii, 'E') {
		t.Error("ASCII should contain 'E' for expected-only pixels")
	}
	if !containsChar(ascii, 'A') {
		t.Error("ASCII should contain 'A' for actual-only pixels")
	}
}

func TestSaveDiffAsPNG_AllCases(t *testing.T) {
	// Create frames that exercise all 4 color cases
	frame1 := make([]byte, 640)
	frame2 := make([]byte, 640)

	// Same setup as TestFrameDiffToASCII_AllCases
	frame1[0] |= 0x80 // Position 0 - both set (white)
	frame2[0] |= 0x80

	frame1[0] |= 0x40 // Position 1 - expected only (red)

	frame2[0] |= 0x20 // Position 2 - actual only (green)

	// Position 3 is both unset (black)

	tmpFile := os.TempDir() + "/test_diff_all_cases.png"
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile)

	err := SaveDiffAsPNG(frame1, frame2, tmpFile)
	if err != nil {
		t.Errorf("SaveDiffAsPNG failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Diff PNG file was not created")
	}
}

// Helper function to check if string contains a character
func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}
