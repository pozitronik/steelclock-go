//go:build linux

package clipboard

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// linuxClipboardReader implements ClipboardReader for Linux.
// It uses external tools (wl-paste, xclip, xsel) for clipboard access.
type linuxClipboardReader struct {
	tool        string // "wl-paste", "xclip", or "xsel"
	lastHash    [16]byte
	hasLastHash bool
	mu          sync.Mutex
}

// newClipboardReader creates a Linux-specific clipboard reader.
func newClipboardReader() (ClipboardReader, error) {
	r := &linuxClipboardReader{}

	// Detect available tool (prefer Wayland, then X11)
	tools := []string{"wl-paste", "xclip", "xsel"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			r.tool = tool
			return r, nil
		}
	}

	return nil, fmt.Errorf("no clipboard tool available (tried: wl-paste, xclip, xsel)")
}

// HasChanged returns true if the clipboard content has changed since last check.
func (r *linuxClipboardReader) HasChanged() bool {
	content, err := r.readRaw()
	if err != nil {
		return false
	}

	hash := md5.Sum(content)

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.hasLastHash || hash != r.lastHash {
		r.lastHash = hash
		r.hasLastHash = true
		return true
	}
	return false
}

// Read returns the current clipboard content and type.
func (r *linuxClipboardReader) Read() (string, ContentType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// First check what type of content is available
	contentType := r.detectContentType()

	switch contentType {
	case TypeText:
		content, err := r.readRaw()
		if err != nil {
			return "", TypeText, err
		}
		text := string(content)
		if text == "" {
			return "", TypeEmpty, nil
		}
		return text, TypeText, nil

	case TypeImage:
		// Get image info if possible
		info := r.getImageInfo()
		return info, TypeImage, nil

	case TypeFiles:
		files, err := r.readFiles()
		if err != nil {
			return "", TypeFiles, err
		}
		return formatFileList(files), TypeFiles, nil

	case TypeHTML:
		return "[HTML]", TypeHTML, nil

	case TypeEmpty:
		return "", TypeEmpty, nil

	default:
		return "[Unknown]", TypeUnknown, nil
	}
}

// Close releases resources.
func (r *linuxClipboardReader) Close() error {
	return nil
}

// readRaw reads raw clipboard content as bytes.
func (r *linuxClipboardReader) readRaw() ([]byte, error) {
	var cmd *exec.Cmd

	switch r.tool {
	case "wl-paste":
		cmd = exec.Command("wl-paste", "--no-newline")
	case "xclip":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	case "xsel":
		cmd = exec.Command("xsel", "--clipboard", "--output")
	default:
		return nil, fmt.Errorf("no clipboard tool configured")
	}

	output, err := cmd.Output()
	if err != nil {
		// Empty clipboard often returns error
		return nil, nil
	}

	return output, nil
}

// detectContentType determines what type of content is in the clipboard.
func (r *linuxClipboardReader) detectContentType() ContentType {
	targets := r.getTargets()
	if len(targets) == 0 {
		return TypeEmpty
	}

	// Check for common MIME types
	hasText := false
	hasImage := false
	hasFiles := false
	hasHTML := false

	for _, target := range targets {
		target = strings.ToLower(target)
		switch {
		case strings.Contains(target, "text/plain") ||
			strings.Contains(target, "utf8_string") ||
			strings.Contains(target, "string"):
			hasText = true
		case strings.Contains(target, "image/"):
			hasImage = true
		case strings.Contains(target, "text/uri-list") ||
			strings.Contains(target, "x-special/gnome-copied-files"):
			hasFiles = true
		case strings.Contains(target, "text/html"):
			hasHTML = true
		}
	}

	// Priority: files > image > HTML > text
	if hasFiles {
		return TypeFiles
	}
	if hasImage {
		return TypeImage
	}
	if hasHTML {
		return TypeHTML
	}
	if hasText {
		return TypeText
	}

	return TypeUnknown
}

// getTargets returns the list of available clipboard formats/targets.
func (r *linuxClipboardReader) getTargets() []string {
	var cmd *exec.Cmd

	switch r.tool {
	case "wl-paste":
		cmd = exec.Command("wl-paste", "--list-types")
	case "xclip":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	case "xsel":
		// xsel doesn't support listing targets, fall back to text
		return []string{"text/plain"}
	default:
		return nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	targets := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			targets = append(targets, line)
		}
	}

	return targets
}

// getImageInfo tries to get image dimensions if available.
func (r *linuxClipboardReader) getImageInfo() string {
	// Try to get image dimensions using imagemagick if available
	// This is optional - if not available, just return generic info
	if _, err := exec.LookPath("identify"); err != nil {
		return "[Image]"
	}

	var cmd *exec.Cmd
	switch r.tool {
	case "wl-paste":
		// Pipe wl-paste to identify
		cmd = exec.Command("bash", "-c", "wl-paste | identify -format '%wx%h' -")
	case "xclip":
		cmd = exec.Command("bash", "-c", "xclip -selection clipboard -t image/png -o | identify -format '%wx%h' -")
	default:
		return "[Image]"
	}

	output, err := cmd.Output()
	if err != nil {
		return "[Image]"
	}

	dims := strings.TrimSpace(string(output))
	if dims != "" {
		return fmt.Sprintf("[Image: %s]", dims)
	}

	return "[Image]"
}

// readFiles reads file paths from clipboard (for copied files).
func (r *linuxClipboardReader) readFiles() ([]string, error) {
	var cmd *exec.Cmd

	switch r.tool {
	case "wl-paste":
		cmd = exec.Command("wl-paste", "-t", "text/uri-list", "--no-newline")
	case "xclip":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-t", "text/uri-list", "-o")
	default:
		return nil, nil
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	// Parse URI list (file:// URIs)
	lines := bytes.Split(output, []byte("\n"))
	files := make([]string, 0, len(lines))

	for _, line := range lines {
		path := string(bytes.TrimSpace(line))
		if path == "" {
			continue
		}

		// Remove file:// prefix
		if strings.HasPrefix(path, "file://") {
			path = strings.TrimPrefix(path, "file://")
			// URL decode if needed (basic)
			path = strings.ReplaceAll(path, "%20", " ")
		}

		files = append(files, path)
	}

	return files, nil
}

// formatFileList formats the file list for display.
func formatFileList(files []string) string {
	if len(files) == 0 {
		return ""
	}

	// Get just the filename from the first path
	firstFile := files[0]
	if idx := strings.LastIndex(firstFile, "/"); idx >= 0 {
		firstFile = firstFile[idx+1:]
	}

	if len(files) == 1 {
		return firstFile
	}

	return fmt.Sprintf("%s (+%d more)", firstFile, len(files)-1)
}
