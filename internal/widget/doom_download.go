package widget

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	// DefaultBundledWadURL is the default URL for downloading the DOOM shareware WAD
	// doom1.wad is the official shareware release, freely available
	DefaultBundledWadURL = "https://distro.ibiblio.org/slitaz/sources/packages/d/doom1.wad"
)

var (
	bundledWadURL = DefaultBundledWadURL // Can be overridden via SetBundledWadURL
)

// SetBundledWadURL sets the URL for downloading the bundled WAD file
// This should be called at application startup if a custom URL is configured
func SetBundledWadURL(url string) {
	if url != "" {
		bundledWadURL = url
	}
}

// GetWadFile gets WAD file from working directory and downloads if necessary
// Only accepts filename, not path (e.g., "doom1.wad", not "path/to/doom1.wad")
func GetWadFile(wadName string) (string, error) {
	log.Printf("[DOOM] Looking for WAD file: %s", wadName)

	// Check if file exists in working directory
	if _, err := os.Stat(wadName); err == nil {
		log.Printf("[DOOM] WAD file found: %s", wadName)
		return wadName, nil
	}

	log.Printf("[DOOM] WAD file not found, attempting download...")

	// Download to working directory
	downloadedFile, err := downloadWadFile(wadName)
	if err != nil {
		return "", fmt.Errorf("WAD file not found and download failed: %w", err)
	}

	log.Printf("[DOOM] WAD file ready: %s", downloadedFile)
	return downloadedFile, nil
}

// downloadWadFile downloads WAD file to working directory
func downloadWadFile(wadName string) (string, error) {
	log.Printf("[DOOM] Downloading %s from: %s", wadName, bundledWadURL)

	// Download WAD from configured URL
	resp, err := http.Get(bundledWadURL)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download WAD: HTTP %d", resp.StatusCode)
	}

	log.Printf("[DOOM] Downloading WAD (%d bytes)...", resp.ContentLength)

	// Save to working directory
	out, err := os.Create(wadName)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = out.Close()
	}()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[DOOM] Downloaded %d bytes to: %s", written, wadName)

	return wadName, nil
}
