//go:build darwin

package clipboard

import (
	"crypto/md5"
	"os/exec"
	"strings"
)

// darwinReader reads clipboard content via pbpaste on macOS.
type darwinReader struct {
	lastHash [md5.Size]byte
}

// newReader creates a clipboard reader for macOS using pbpaste.
func newReader() (Reader, error) {
	return &darwinReader{}, nil
}

// HasChanged returns true if the clipboard content has changed since the last Read.
func (r *darwinReader) HasChanged() bool {
	content, _, _ := r.readClipboard()
	hash := md5.Sum([]byte(content))
	return hash != r.lastHash
}

// Read returns the current clipboard content and its type.
func (r *darwinReader) Read() (string, ContentType, error) {
	content, contentType, err := r.readClipboard()
	if err != nil {
		return "", TypeUnknown, err
	}
	r.lastHash = md5.Sum([]byte(content))
	return content, contentType, nil
}

// Close is a no-op on macOS (no resources to release).
func (r *darwinReader) Close() error {
	return nil
}

// readClipboard reads the clipboard using pbpaste and determines content type.
func (r *darwinReader) readClipboard() (string, ContentType, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", TypeEmpty, nil
	}

	content := string(out)
	if content == "" {
		return "", TypeEmpty, nil
	}

	// Detect content type
	contentType := detectContentType(content)
	return content, contentType, nil
}

// detectContentType attempts to classify the clipboard content.
func detectContentType(content string) ContentType {
	trimmed := strings.TrimSpace(content)

	if trimmed == "" {
		return TypeEmpty
	}

	// Check for HTML content
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
		return TypeHTML
	}

	// Check for file paths (macOS paths start with /)
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "/") {
		allPaths := true
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "/") {
				allPaths = false
				break
			}
		}
		if allPaths {
			return TypeFiles
		}
	}

	return TypeText
}
