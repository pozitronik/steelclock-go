package bitmap

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

const (
	// DefaultBundledFontURL is the default URL for downloading the bundled font
	DefaultBundledFontURL = "https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf"
)

var (
	fontCache      = make(map[string]*opentype.Font)
	fontCacheMutex sync.RWMutex            // Protects fontCache map access
	bundledFontURL = DefaultBundledFontURL // Can be overridden via SetBundledFontURL
	// fontMutex protects concurrent access to font.Face operations
	// font.Face from golang.org/x/image/font is not thread-safe
	fontMutex sync.Mutex
)

// SetBundledFontURL sets the URL for downloading the bundled font
// This should be called at application startup if a custom URL is configured
func SetBundledFontURL(url string) {
	if url != "" {
		bundledFontURL = url
	}
}

// LoadFont loads a TrueType font
func LoadFont(fontName string, size int) (font.Face, error) {
	var fontPath string

	// Try to resolve font path if font name is specified
	if fontName != "" {
		fontPath = resolveFontPath(fontName)
	}

	// If no font path found (either fontName was empty or not resolved),
	// try to download bundled font
	if fontPath == "" {
		bundledPath, err := downloadBundledFont()
		if err == nil {
			fontPath = bundledPath
		} else {
			// Use basic font as last resort
			return basicfont.Face7x13, nil
		}
	}

	// Load the font
	ttf, err := loadTTF(fontPath)
	if err != nil {
		return basicfont.Face7x13, nil
	}

	// Create font face using opentype package
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})

	if err != nil {
		return basicfont.Face7x13, nil
	}

	return face, nil
}

// loadTTF loads a TrueType font file
func loadTTF(path string) (*opentype.Font, error) {
	// Check cache with read lock
	fontCacheMutex.RLock()
	if cached, ok := fontCache[path]; ok {
		fontCacheMutex.RUnlock()
		return cached, nil
	}
	fontCacheMutex.RUnlock()

	// Font not in cache, load it
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ttf, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}

	// Store in cache with write lock
	fontCacheMutex.Lock()
	fontCache[path] = ttf
	fontCacheMutex.Unlock()

	return ttf, nil
}

// resolveFontPath resolves font name to file path
func resolveFontPath(fontName string) string {
	// Check if it's already a path
	if _, err := os.Stat(fontName); err == nil {
		return fontName
	}

	// Windows fonts directory
	windowsFonts := filepath.Join("C:", "Windows", "Fonts")

	// Common font mappings
	fontMappings := map[string]string{
		"arial":       "arial.ttf",
		"consolas":    "consola.ttf",
		"courier new": "cour.ttf",
		"verdana":     "verdana.ttf",
		"tahoma":      "tahoma.ttf",
	}

	// Try exact match
	if fileName, ok := fontMappings[fontName]; ok {
		path := filepath.Join(windowsFonts, fileName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try direct file name
	path := filepath.Join(windowsFonts, fontName+".ttf")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

// downloadBundledFont downloads and returns path to bundled font
func downloadBundledFont() (string, error) {
	fontsDir := filepath.Join(".", "fonts")
	fontPath := filepath.Join(fontsDir, "FSEX302.ttf")

	// Check if already downloaded
	if _, err := os.Stat(fontPath); err == nil {
		return fontPath, nil
	}

	// Create fonts directory
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		return "", err
	}

	// Download font from configured URL
	resp, err := http.Get(bundledFontURL)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download font: HTTP %d", resp.StatusCode)
	}

	// Save to file
	out, err := os.Create(fontPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	return fontPath, nil
}

// measureTextUnsafe measures the width and height of text without locking
// Internal use only - caller must hold fontMutex
func measureTextUnsafe(text string, face font.Face) (int, int) {
	drawer := &font.Drawer{
		Face: face,
	}

	advance := drawer.MeasureString(text)
	width := advance.Ceil()

	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()

	return width, height
}

// MeasureText measures the width and height of text
func MeasureText(text string, face font.Face) (int, int) {
	// Protect font face access - font.Face is not thread-safe
	fontMutex.Lock()
	defer fontMutex.Unlock()

	return measureTextUnsafe(text, face)
}
