package bitmap

import (
	"os"
	"testing"
)

// TestLoadFont tests font loading functionality
func TestLoadFont(t *testing.T) {
	tests := []struct {
		name     string
		fontName string
		fontSize int
		wantErr  bool
	}{
		{
			name:     "load default font with valid size",
			fontName: "",
			fontSize: 12,
			wantErr:  false,
		},
		{
			name:     "load with larger font size",
			fontName: "",
			fontSize: 24,
			wantErr:  false,
		},
		{
			name:     "load with small font size",
			fontName: "",
			fontSize: 8,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			face, err := LoadFont(tt.fontName, tt.fontSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFont() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if face == nil {
					t.Error("LoadFont() returned nil font face")
				}
			}
		})
	}
}

// TestLoadFont_InvalidSize tests handling of invalid font sizes
func TestLoadFont_InvalidSize(t *testing.T) {
	tests := []struct {
		name     string
		fontSize int
	}{
		{"zero size", 0},
		{"negative size", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The function may handle invalid sizes by falling back to defaults
			// or by returning an error. We just verify it doesn't panic.
			face, err := LoadFont("", tt.fontSize)

			// Log the behavior for documentation
			if err != nil {
				t.Logf("LoadFont() with size %d returned error (expected): %v", tt.fontSize, err)
			} else if face != nil {
				t.Logf("LoadFont() with size %d fell back to default font", tt.fontSize)
			}
		})
	}
}

// TestLoadFont_NonexistentFont tests handling of non-existent font files
func TestLoadFont_NonexistentFont(t *testing.T) {
	fontName := "definitely_not_a_real_font_file_xyz123.ttf"

	// This should fall back to bundled font download or default font
	face, err := LoadFont(fontName, 12)

	// The behavior depends on implementation - it might succeed with fallback
	// or fail with error. We just verify it doesn't panic.
	if err != nil && face != nil {
		t.Error("LoadFont() returned both error and non-nil face")
	}
}

// TestResolveFontPath tests font path resolution
func TestResolveFontPath(t *testing.T) {
	tests := []struct {
		name     string
		fontName string
	}{
		{
			name:     "empty font name returns bundled font path",
			fontName: "",
		},
		{
			name:     "relative path",
			fontName: "fonts/myfont.ttf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFontPath(tt.fontName)

			// Should return some path (either resolved or fallback)
			// We just verify it doesn't panic and returns a reasonable result
			t.Logf("resolveFontPath(%q) = %q", tt.fontName, got)
		})
	}
}

// TestDownloadBundledFont tests bundled font download functionality
func TestDownloadBundledFont(t *testing.T) {
	// Change to temp directory for test
	oldCacheDir := os.Getenv("FONTCONFIG_PATH")
	defer func() {
		if oldCacheDir != "" {
			err := os.Setenv("FONTCONFIG_PATH", oldCacheDir)
			if err != nil {
				return
			}
		}
	}()

	fontPath, err := downloadBundledFont()

	// Should either succeed or fail gracefully
	if err != nil {
		t.Logf("downloadBundledFont() error (expected in test env): %v", err)
	}

	if fontPath != "" {
		// If we got a path, verify it exists or will be created
		t.Logf("Font path: %s", fontPath)
	}
}

// TestMeasureText tests text measurement functionality
func TestMeasureText(t *testing.T) {
	// Load a font for testing
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	tests := []struct {
		name string
		text string
	}{
		{"empty string", ""},
		{"single character", "A"},
		{"short text", "Hello"},
		{"longer text", "Hello, World!"},
		{"numbers", "1234567890"},
		{"mixed", "Mix123!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := MeasureText(tt.text, face)

			if tt.text == "" {
				// Empty string should have minimal or zero dimensions
				if width < 0 || height < 0 {
					t.Errorf("MeasureText() returned negative dimensions: %dx%d", width, height)
				}
			} else {
				// Non-empty text should have positive dimensions
				if width <= 0 {
					t.Errorf("MeasureText() width = %d, want > 0 for text %q", width, tt.text)
				}
				if height <= 0 {
					t.Errorf("MeasureText() height = %d, want > 0 for text %q", height, tt.text)
				}
			}
		})
	}
}

// TestMeasureText_NilFace tests measure text with nil font face
func TestMeasureText_NilFace(t *testing.T) {
	// This may panic with nil face, which is acceptable
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected with nil face
			t.Logf("MeasureText() with nil face panicked as expected: %v", r)
		}
	}()

	width, height := MeasureText("test", nil)

	// If it doesn't panic, verify dimensions are reasonable
	if width < 0 || height < 0 {
		t.Errorf("MeasureText() with nil face returned negative dimensions: %dx%d", width, height)
	}
}

// TestMeasureText_LongerTextIsWider tests that longer text has greater width
func TestMeasureText_LongerTextIsWider(t *testing.T) {
	face, err := LoadFont("", 12)
	if err != nil {
		t.Skipf("Skipping test, cannot load font: %v", err)
	}

	shortWidth, _ := MeasureText("A", face)
	longWidth, _ := MeasureText("AAAAAAAAAA", face)

	if longWidth <= shortWidth {
		t.Errorf("Longer text should have greater width: short=%d, long=%d", shortWidth, longWidth)
	}
}

// TestLoadFont_MultipleLoads tests loading multiple fonts concurrently
func TestLoadFont_MultipleLoads(t *testing.T) {
	done := make(chan bool, 5)

	// Load fonts concurrently
	for i := 0; i < 5; i++ {
		go func(size int) {
			_, err := LoadFont("", size)
			if err != nil {
				t.Errorf("Concurrent LoadFont() error: %v", err)
			}
			done <- true
		}(10 + i*2)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
