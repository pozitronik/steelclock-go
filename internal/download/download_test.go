package download

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// assertFiles checks that the directory holds exactly the expected names,
// i.e. no temporary file was left behind.
func assertFiles(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("directory contains %v, want %v", got, want)
	}
}

func TestToFile(t *testing.T) {
	const content = "complete resource body"

	tests := []struct {
		name    string
		handler http.HandlerFunc
		opts    Options
		wantErr error // nil: success; errAny: any error
	}{
		{
			name: "success with content length",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(len(content)))
				_, _ = io.WriteString(w, content)
			},
		},
		{
			name: "success without content length",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.(http.Flusher).Flush() // forces chunked encoding
				_, _ = io.WriteString(w, content)
			},
		},
		{
			name: "truncated body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = io.WriteString(w, "partial")
			},
			wantErr: errAny,
		},
		{
			name: "HTTP error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "missing", http.StatusNotFound)
			},
			wantErr: errAny,
		},
		{
			name: "declared size over limit",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(len(content)))
				_, _ = io.WriteString(w, content)
			},
			opts:    Options{MaxBytes: 5},
			wantErr: ErrTooLarge,
		},
		{
			name: "undeclared size over limit",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.(http.Flusher).Flush()
				_, _ = io.WriteString(w, content)
			},
			opts:    Options{MaxBytes: 5},
			wantErr: ErrTooLarge,
		},
		{
			name: "size exactly at limit",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, content)
			},
			opts: Options{MaxBytes: int64(len(content))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			dir := t.TempDir()
			path := filepath.Join(dir, "resource.bin")
			written, err := ToFile(context.Background(), server.URL, path, tt.opts)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ToFile() error = %v", err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("downloaded file missing: %v", err)
				}
				if string(data) != content || written != int64(len(content)) {
					t.Errorf("file = %q (%d bytes written), want %q", data, written, content)
				}
				assertFiles(t, dir, "resource.bin")
				return
			}

			if err == nil {
				t.Fatal("ToFile() error = nil, want error")
			}
			if tt.wantErr != errAny && !errors.Is(err, tt.wantErr) {
				t.Errorf("ToFile() error = %v, want %v", err, tt.wantErr)
			}
			assertFiles(t, dir)
		})
	}
}

// errAny marks test cases that expect some error without a specific type.
var errAny = errors.New("any error")

func TestToFile_KeepsExistingFileOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "partial")
	}))
	defer server.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "resource.bin")
	if err := os.WriteFile(path, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := ToFile(context.Background(), server.URL, path, Options{}); err == nil {
		t.Fatal("ToFile() error = nil, want error")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "previous" {
		t.Errorf("existing file = %q (%v), want it unchanged", data, err)
	}
	assertFiles(t, dir, "resource.bin")
}

func TestToFile_Timeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "start")
		w.(http.Flusher).Flush()
		<-release // stall the body
	}))
	defer server.Close()
	defer close(release)

	dir := t.TempDir()
	start := time.Now()
	_, err := ToFile(context.Background(), server.URL, filepath.Join(dir, "resource.bin"), Options{Timeout: 200 * time.Millisecond})
	if err == nil {
		t.Fatal("ToFile() error = nil, want timeout")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("ToFile() took %v, want it bounded by the timeout", elapsed)
	}
	assertFiles(t, dir)
}

func TestToFile_Wrap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "4")
		_, _ = io.WriteString(w, "data")
	}))
	defer server.Close()

	var gotSize int64
	wrapped := false
	opts := Options{Wrap: func(body io.Reader, size int64) io.Reader {
		wrapped, gotSize = true, size
		return body
	}}
	if _, err := ToFile(context.Background(), server.URL, filepath.Join(t.TempDir(), "resource.bin"), opts); err != nil {
		t.Fatalf("ToFile() error = %v", err)
	}
	if !wrapped || gotSize != 4 {
		t.Errorf("Wrap called = %v with size %d, want true with 4", wrapped, gotSize)
	}
}

func TestToFile_Cancel(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "start")
		w.(http.Flusher).Flush()
		<-release // stall the body
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	dir := t.TempDir()
	start := time.Now()
	_, err := ToFile(ctx, server.URL, filepath.Join(dir, "resource.bin"), Options{Timeout: time.Hour})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ToFile() error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("ToFile() took %v after cancel, want it to return promptly", elapsed)
	}
	assertFiles(t, dir)
}
