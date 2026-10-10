package doom

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestGetWadFile_FileExists(t *testing.T) {
	// Create a temporary WAD file
	tmpFile := "test_existing.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	content := []byte("test wad content")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test getting existing file (empty URL = use default)
	result, err := GetWadFile(tmpFile, "")
	if err != nil {
		t.Errorf("GetWadFile() error = %v, want nil", err)
	}

	if result != tmpFile {
		t.Errorf("GetWadFile() = %s, want %s", result, tmpFile)
	}
}

func TestGetWadFile_FileNotExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	// Test with non-existent file - will attempt download
	// This is a network test and will actually download the file
	nonexistent := "test_download_12345.wad"
	defer func() { _ = os.Remove(nonexistent) }() // Clean up

	result, err := GetWadFile(nonexistent, "")
	if err != nil {
		// Network error or download failed - this is acceptable in tests
		t.Logf("GetWadFile() error = %v (network test may fail)", err)
		return
	}

	// If download succeeded, verify file was created
	if _, err := os.Stat(result); err != nil {
		t.Errorf("Downloaded file does not exist: %v", err)
	}
}

func TestGetWadFileWithProgress_FileExists(t *testing.T) {
	// Create a temporary WAD file
	tmpFile := "test_progress.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	content := []byte("test wad content")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test getting existing file with progress callback
	progressCalled := false
	progressCallback := func(progress float64) {
		progressCalled = true
	}

	var isDownloading bool
	var mu sync.RWMutex

	result, err := GetWadFileWithProgress(context.Background(), tmpFile, "", progressCallback, &isDownloading, &mu)
	if err != nil {
		t.Errorf("GetWadFileWithProgress() error = %v, want nil", err)
	}

	if result != tmpFile {
		t.Errorf("GetWadFileWithProgress() = %s, want %s", result, tmpFile)
	}

	// Progress callback should NOT be called if file exists
	if progressCalled {
		t.Error("Progress callback should not be called when file already exists")
	}

	// isDownloading should remain false
	if isDownloading {
		t.Error("isDownloading should be false when file already exists")
	}
}

func TestGetWadFileWithProgress_ProgressCallback(t *testing.T) {
	// Create a temporary WAD file to test progress tracking
	tmpFile := "test_callback.wad"
	defer func() { _ = os.Remove(tmpFile) }()

	content := []byte("test wad content")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var progressValues []float64
	progressCallback := func(progress float64) {
		progressValues = append(progressValues, progress)
	}

	var isDownloading bool
	var mu sync.RWMutex

	_, err := GetWadFileWithProgress(context.Background(), tmpFile, "", progressCallback, &isDownloading, &mu)
	if err != nil {
		t.Errorf("GetWadFileWithProgress() error = %v", err)
	}

	// When file exists, no progress updates should occur
	if len(progressValues) > 0 {
		t.Errorf("Progress callback called %d times, want 0 when file exists", len(progressValues))
	}
}

func TestProgressReader_Read(t *testing.T) {
	// Create a simple reader
	testData := []byte("Hello, World! This is test data for progress tracking.")
	testReader := &testReaderWrapper{data: testData, pos: 0}

	progressCalled := 0
	var lastProgress float64

	callback := func(progress float64) {
		progressCalled++
		lastProgress = progress
	}

	pr := newProgressReader(testReader, int64(len(testData)), callback)

	// Read all data in chunks
	buffer := make([]byte, 10)
	totalRead := 0

	for {
		n, err := pr.Read(buffer)
		totalRead += n

		if err != nil {
			break
		}
	}

	// Should have read all data
	if totalRead != len(testData) {
		t.Errorf("Read %d bytes, want %d", totalRead, len(testData))
	}

	// Progress callback may not be called if read completes too fast
	// (due to 100ms threshold), but lastProgress should be calculated
	if progressCalled > 0 {
		// If called, last progress should be reasonable
		if lastProgress < 0 || lastProgress > 1.0 {
			t.Errorf("lastProgress = %f, want between 0 and 1", lastProgress)
		}
	}
}

func TestProgressReader_NoCallback(t *testing.T) {
	testData := []byte("Test data without callback")
	testReader := &testReaderWrapper{data: testData, pos: 0}

	// Create progress reader without callback
	pr := newProgressReader(testReader, int64(len(testData)), nil)

	buffer := make([]byte, 10)
	totalRead := 0

	for {
		n, err := pr.Read(buffer)
		totalRead += n

		if err != nil {
			break
		}
	}

	// Should still read all data
	if totalRead != len(testData) {
		t.Errorf("Read %d bytes, want %d", totalRead, len(testData))
	}
}

// testReaderWrapper is a simple io.Reader for testing
type testReaderWrapper struct {
	data []byte
	pos  int
}

func (r *testReaderWrapper) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, os.ErrClosed // Use as EOF equivalent
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TestGetWadFile_InterruptedDownload checks that an interrupted download
// leaves no WAD behind, so the next attempt downloads again instead of using
// the partial file.
func TestGetWadFile_InterruptedDownload(t *testing.T) {
	var truncate atomic.Bool
	truncate.Store(true)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		if truncate.Load() {
			w.Header().Set("Content-Length", "100")
			_, _ = fmt.Fprint(w, "PWADpartial")
			return
		}
		_, _ = fmt.Fprint(w, "PWADcomplete")
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "test.wad")
	if _, err := GetWadFile(path, server.URL); err == nil {
		t.Fatal("GetWadFile() error = nil for a truncated body")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("truncated download left the WAD file behind")
	}

	truncate.Store(false)
	result, err := GetWadFile(path, server.URL)
	if err != nil {
		t.Fatalf("GetWadFile() retry error = %v", err)
	}
	data, err := os.ReadFile(result)
	if err != nil || string(data) != "PWADcomplete" {
		t.Errorf("WAD file = %q (%v), want the complete body", data, err)
	}
	if n := requests.Load(); n != 2 {
		t.Errorf("server got %d requests, want 2", n)
	}
}

// TestStop_AbortsWadDownload checks that stopping the widget (app exit or
// profile reload) does not wait for a stalled WAD download.
func TestStop_AbortsWadDownload(t *testing.T) {
	// Only the first DOOM widget per process may start the engine; allow this
	// one to reach the download and restore the flag afterwards.
	doomHasRunMu.Lock()
	previous := doomHasRun
	doomHasRun = false
	doomHasRunMu.Unlock()
	t.Cleanup(func() {
		doomHasRunMu.Lock()
		doomHasRun = previous
		doomHasRunMu.Unlock()
	})

	requested := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested <- struct{}{}
		select { // withhold the response until released or the client gives up
		case <-release:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	defer close(release)

	t.Chdir(t.TempDir())
	url := server.URL
	w, err := New(config.WidgetConfig{
		Type:          "doom",
		Position:      config.PositionConfig{W: 128, H: 40},
		Wad:           "stalled.wad",
		BundledWadURL: &url,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	select {
	case <-requested:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}

	stopped := make(chan struct{})
	go func() {
		w.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() waited for the stalled download")
	}
}
