package widget

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	// DefaultBundledWadURL is the default URL for downloading the DOOM shareware WAD
	// doom1.wad is the official shareware release, freely available
	DefaultBundledWadURL = "https://distro.ibiblio.org/slitaz/sources/packages/d/doom1.wad"
)

var (
	bundledWadURL = DefaultBundledWadURL // Can be overridden via SetBundledWadURL
)

// progressReader wraps an io.Reader and logs download progress
type progressReader struct {
	reader      io.Reader
	total       int64
	downloaded  int64
	lastLog     int64
	lastLogTime time.Time
	lastUpdate  time.Time
	startTime   time.Time
	callback    func(float64)
}

// newProgressReader creates a new progress tracking reader
func newProgressReader(reader io.Reader, total int64, callback func(float64)) *progressReader {
	return &progressReader{
		reader:      reader,
		total:       total,
		lastLogTime: time.Now(),
		lastUpdate:  time.Now(),
		startTime:   time.Now(),
		callback:    callback,
	}
}

// Read implements io.Reader and logs progress
func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)

	// Update callback frequently for smooth progress bar (every 100ms)
	if pr.callback != nil && time.Since(pr.lastUpdate) >= 100*time.Millisecond {
		if pr.total > 0 {
			progress := float64(pr.downloaded) / float64(pr.total)
			pr.callback(progress)
		}
		pr.lastUpdate = time.Now()
	}

	// Log progress every 512KB or every second
	logInterval := int64(512 * 1024)
	if pr.downloaded-pr.lastLog >= logInterval || time.Since(pr.lastLogTime) >= time.Second {
		pr.logProgress()
		pr.lastLog = pr.downloaded
		pr.lastLogTime = time.Now()
	}

	return n, err
}

// logProgress logs current download progress
func (pr *progressReader) logProgress() {
	if pr.total > 0 {
		percent := float64(pr.downloaded) / float64(pr.total) * 100
		elapsed := time.Since(pr.startTime).Seconds()
		speed := float64(pr.downloaded) / elapsed / 1024 // KB/s
		log.Printf("[DOOM] Download progress: %.1f%% (%d/%d bytes, %.1f KB/s)",
			percent, pr.downloaded, pr.total, speed)
	} else {
		log.Printf("[DOOM] Downloaded: %d bytes", pr.downloaded)
	}
}

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
	return GetWadFileWithProgress(wadName, nil, nil, nil)
}

// GetWadFileWithProgress gets WAD file with progress callback
func GetWadFileWithProgress(wadName string, progressCallback func(float64), isDownloading *bool, mu *sync.RWMutex) (string, error) {
	// Check if file exists in working directory
	if _, err := os.Stat(wadName); err == nil {
		log.Printf("[DOOM] Using existing WAD: %s", wadName)
		return wadName, nil
	}

	log.Printf("[DOOM] WAD not found, starting download...")

	// Set downloading flag
	if isDownloading != nil && mu != nil {
		mu.Lock()
		*isDownloading = true
		mu.Unlock()
	}

	// Download to working directory with progress
	downloadedFile, err := downloadWadFileWithProgress(wadName, progressCallback)
	if err != nil {
		return "", fmt.Errorf("WAD file not found and download failed: %w", err)
	}

	return downloadedFile, nil
}

// downloadWadFileWithProgress downloads WAD file with progress callback
func downloadWadFileWithProgress(wadName string, progressCallback func(float64)) (string, error) {
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

	// Get file size for progress tracking
	totalSize := resp.ContentLength
	if totalSize > 0 {
		log.Printf("[DOOM] Starting download: %.2f MB", float64(totalSize)/(1024*1024))
	} else {
		log.Printf("[DOOM] Starting download (size unknown)")
	}

	// Save to working directory
	out, err := os.Create(wadName)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = out.Close()
	}()

	// Wrap response body with progress reader
	progressR := newProgressReader(resp.Body, totalSize, progressCallback)

	// Copy with progress tracking
	written, err := io.Copy(out, progressR)
	if err != nil {
		return "", err
	}

	// Call final progress update
	if progressCallback != nil && totalSize > 0 {
		progressCallback(1.0)
	}

	log.Printf("[DOOM] Download complete: %s (%.2f MB)", wadName, float64(written)/(1024*1024))

	return wadName, nil
}
