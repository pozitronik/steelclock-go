// Package download fetches remote resources (fonts, game data) into local
// cache files without ever leaving a partial file under the final name.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Options controls a download.
type Options struct {
	// Timeout bounds the whole request, including reading the body.
	// Zero means no timeout.
	Timeout time.Duration
	// MaxBytes rejects bodies larger than this. Zero means no limit.
	MaxBytes int64
	// Wrap optionally wraps the response body, e.g. to report progress.
	// size is the Content-Length, or -1 if unknown.
	Wrap func(body io.Reader, size int64) io.Reader
}

// ErrTooLarge is returned when the body exceeds Options.MaxBytes.
var ErrTooLarge = errors.New("download exceeds size limit")

// ToFile downloads url into path and returns the number of bytes written.
// The body is written to a temporary file in the same directory, which is
// renamed to path only after the complete body was received and the file was
// closed. On any error the temporary file is removed and path is untouched.
// Cancelling ctx aborts the download at any point; Options.Timeout is an
// independent upper bound.
func ToFile(ctx context.Context, url, path string, opts Options) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: opts.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if opts.MaxBytes > 0 && resp.ContentLength > opts.MaxBytes {
		return 0, fmt.Errorf("%w: %d bytes, limit %d", ErrTooLarge, resp.ContentLength, opts.MaxBytes)
	}

	var body io.Reader = resp.Body
	if opts.MaxBytes > 0 {
		// Read one byte past the limit to detect an oversized body.
		body = io.LimitReader(body, opts.MaxBytes+1)
	}
	if opts.Wrap != nil {
		body = opts.Wrap(body, resp.ContentLength)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.part")
	if err != nil {
		return 0, err
	}
	tmpPath := tmp.Name()
	published := false
	defer func() {
		if !published {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	written, err := io.Copy(tmp, body)
	if err != nil {
		return written, err
	}
	if opts.MaxBytes > 0 && written > opts.MaxBytes {
		return written, fmt.Errorf("%w: limit %d bytes", ErrTooLarge, opts.MaxBytes)
	}
	if resp.ContentLength >= 0 && written != resp.ContentLength {
		return written, fmt.Errorf("incomplete download: got %d of %d bytes", written, resp.ContentLength)
	}
	if err := tmp.Close(); err != nil {
		return written, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return written, err
	}
	published = true
	return written, nil
}
