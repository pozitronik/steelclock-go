package bitmap

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

var fontCache = make(map[string]*opentype.Font)

// LoadFont loads a TrueType font
func LoadFont(fontName string, size int) (font.Face, error) {
	if fontName == "" {
		// Use basic font as fallback
		return basicfont.Face7x13, nil
	}

	// Try to load system font or bundled font
	fontPath := resolveFontPath(fontName)
	if fontPath == "" {
		// Try to download bundled font
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
	// Check cache
	if cached, ok := fontCache[path]; ok {
		return cached, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ttf, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}

	fontCache[path] = ttf
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

	// Download font
	url := "https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf"
	resp, err := http.Get(url)
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

// MeasureText measures the width and height of text
func MeasureText(text string, face font.Face) (int, int) {
	drawer := &font.Drawer{
		Face: face,
	}

	advance := drawer.MeasureString(text)
	width := advance.Ceil()

	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()

	return width, height
}
