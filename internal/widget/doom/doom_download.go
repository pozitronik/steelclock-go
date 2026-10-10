package doom

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/download"
)

const (
	// DefaultBundledWadURL is the default URL for downloading the DOOM shareware WAD
	// doom1.wad is the official shareware release, freely available
	DefaultBundledWadURL = "https://distro.ibiblio.org/slitaz/sources/packages/d/doom1.wad"

	// wadDownloadTimeout bounds the whole WAD download, so a stalled server
	// cannot block the DOOM worker (and shutdown) indefinitely
	wadDownloadTimeout = 10 * time.Minute
	// wadDownloadMaxBytes rejects implausibly large downloads (doom1.wad is
	// ~4 MB; full IWADs and large PWADs stay well below this)
	wadDownloadMaxBytes = 256 << 20
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

// GetWadFile gets WAD file from working directory and downloads if necessary
// Only accepts filename, not path (e.g., "doom1.wad", not "path/to/doom1.wad")
// bundledWadURL: custom URL for download (empty = use default)
func GetWadFile(wadName string, bundledWadURL string) (string, error) {
	return GetWadFileWithProgress(context.Background(), wadName, bundledWadURL, nil, nil, nil)
}

// GetWadFileWithProgress gets WAD file with progress callback
// bundledWadURL: custom URL for download (empty = use default)
// Cancelling ctx aborts a running download.
func GetWadFileWithProgress(ctx context.Context, wadName string, bundledWadURL string, progressCallback func(float64), isDownloading *bool, mu *sync.RWMutex) (string, error) {
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
	downloadedFile, err := downloadWadFileWithProgress(ctx, wadName, bundledWadURL, progressCallback)
	if err != nil {
		return "", fmt.Errorf("WAD file not found and download failed: %w", err)
	}

	return downloadedFile, nil
}

// downloadWadFileWithProgress downloads WAD file with progress callback
// bundledWadURL: custom URL for download (empty = use default)
func downloadWadFileWithProgress(ctx context.Context, wadName string, bundledWadURL string, progressCallback func(float64)) (string, error) {
	// Use default URL if not specified
	downloadURL := bundledWadURL
	if downloadURL == "" {
		downloadURL = DefaultBundledWadURL
	}

	log.Printf("[DOOM] Downloading %s from: %s", wadName, downloadURL)

	// Download into the working directory. The file appears under its final
	// name only when complete, so an interrupted download is retried later.
	var totalSize int64
	opts := download.Options{
		Timeout:  wadDownloadTimeout,
		MaxBytes: wadDownloadMaxBytes,
		Wrap: func(body io.Reader, size int64) io.Reader {
			totalSize = size
			if size > 0 {
				log.Printf("[DOOM] Starting download: %.2f MB", float64(size)/(1024*1024))
			} else {
				log.Printf("[DOOM] Starting download (size unknown)")
			}
			return newProgressReader(body, size, progressCallback)
		},
	}
	written, err := download.ToFile(ctx, downloadURL, wadName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to download WAD: %w", err)
	}

	// Call final progress update
	if progressCallback != nil && totalSize > 0 {
		progressCallback(1.0)
	}

	log.Printf("[DOOM] Download complete: %s (%.2f MB)", wadName, float64(written)/(1024*1024))

	return wadName, nil
}
