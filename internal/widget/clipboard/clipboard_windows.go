//go:build windows

package clipboard

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32                      = syscall.NewLazyDLL("user32.dll")
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	shell32                     = syscall.NewLazyDLL("shell32.dll")
	procOpenClipboard           = user32.NewProc("OpenClipboard")
	procCloseClipboard          = user32.NewProc("CloseClipboard")
	procGetClipboardData        = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvail  = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardSequenceNum = user32.NewProc("GetClipboardSequenceNumber")
	procGlobalLock              = kernel32.NewProc("GlobalLock")
	procGlobalUnlock            = kernel32.NewProc("GlobalUnlock")
	procDragQueryFileW          = shell32.NewProc("DragQueryFileW")
)

// Clipboard formats
const (
	cfUnicodeText = 13 // CF_UNICODETEXT
	cfBitmap      = 2  // CF_BITMAP
	cfDIB         = 8  // CF_DIB
	cfHDrop       = 15 // CF_HDROP
	cfHTML        = 49 // CF_HTML (registered format, but commonly this ID)
)

// windowsClipboardReader implements ClipboardReader for Windows.
type windowsClipboardReader struct {
	lastSeqNum uint32
	mu         sync.Mutex
}

// newClipboardReader creates a Windows-specific clipboard reader.
func newClipboardReader() (ClipboardReader, error) {
	r := &windowsClipboardReader{}
	// Get initial sequence number
	seqNum, _, _ := procGetClipboardSequenceNum.Call()
	r.lastSeqNum = uint32(seqNum)
	return r, nil
}

// HasChanged returns true if the clipboard content has changed since last check.
func (r *windowsClipboardReader) HasChanged() bool {
	seqNum, _, _ := procGetClipboardSequenceNum.Call()
	currentSeq := uint32(seqNum)

	r.mu.Lock()
	defer r.mu.Unlock()

	if currentSeq != r.lastSeqNum {
		r.lastSeqNum = currentSeq
		return true
	}
	return false
}

// Read returns the current clipboard content and type.
func (r *windowsClipboardReader) Read() (string, ContentType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Open clipboard
	ret, _, err := procOpenClipboard.Call(0)
	if ret == 0 {
		return "", TypeUnknown, fmt.Errorf("failed to open clipboard: %w", err)
	}
	defer procCloseClipboard.Call()

	// Check available formats in order of preference
	// Text
	if isFormatAvailable(cfUnicodeText) {
		text, err := getUnicodeText()
		if err != nil {
			return "", TypeText, err
		}
		if text == "" {
			return "", TypeEmpty, nil
		}
		return text, TypeText, nil
	}

	// Files (HDROP)
	if isFormatAvailable(cfHDrop) {
		files, err := getFiles()
		if err != nil {
			return "", TypeFiles, err
		}
		if len(files) == 0 {
			return "", TypeEmpty, nil
		}
		return formatFileList(files), TypeFiles, nil
	}

	// Image (Bitmap or DIB)
	if isFormatAvailable(cfBitmap) || isFormatAvailable(cfDIB) {
		// We only report image metadata, not actual content
		return "[Image]", TypeImage, nil
	}

	// HTML
	if isFormatAvailable(cfHTML) {
		return "[HTML]", TypeHTML, nil
	}

	// Check if clipboard is empty by trying to get any data
	return "", TypeEmpty, nil
}

// Close releases resources.
func (r *windowsClipboardReader) Close() error {
	return nil
}

// isFormatAvailable checks if a clipboard format is available.
func isFormatAvailable(format uint32) bool {
	ret, _, _ := procIsClipboardFormatAvail.Call(uintptr(format))
	return ret != 0
}

// getUnicodeText retrieves Unicode text from clipboard.
func getUnicodeText() (string, error) {
	hMem, _, err := procGetClipboardData.Call(uintptr(cfUnicodeText))
	if hMem == 0 {
		return "", fmt.Errorf("failed to get clipboard data: %w", err)
	}

	ptr, _, err := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", fmt.Errorf("failed to lock memory: %w", err)
	}
	defer procGlobalUnlock.Call(hMem)

	// Convert UTF-16 to string
	// nolint:govet // ptr is a valid uintptr from GlobalLock syscall, conversion to unsafe.Pointer is correct
	text := utf16PtrToString((*uint16)(unsafe.Pointer(ptr)))
	return text, nil
}

// getFiles retrieves file list from HDROP clipboard data.
func getFiles() ([]string, error) {
	hMem, _, err := procGetClipboardData.Call(uintptr(cfHDrop))
	if hMem == 0 {
		return nil, fmt.Errorf("failed to get clipboard data: %w", err)
	}

	// Get file count
	count, _, _ := procDragQueryFileW.Call(hMem, 0xFFFFFFFF, 0, 0)
	if count == 0 {
		return nil, nil
	}

	files := make([]string, 0, count)
	buf := make([]uint16, 260) // MAX_PATH

	for i := uintptr(0); i < count; i++ {
		procDragQueryFileW.Call(hMem, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		files = append(files, syscall.UTF16ToString(buf))
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
	if idx := strings.LastIndexAny(firstFile, "/\\"); idx >= 0 {
		firstFile = firstFile[idx+1:]
	}

	if len(files) == 1 {
		return firstFile
	}

	return fmt.Sprintf("%s (+%d more)", firstFile, len(files)-1)
}

// utf16PtrToString converts a null-terminated UTF-16 string to Go string.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}

	// Find length
	end := unsafe.Pointer(p)
	n := 0
	for *(*uint16)(end) != 0 {
		end = unsafe.Pointer(uintptr(end) + 2)
		n++
	}

	if n == 0 {
		return ""
	}

	// Convert to slice and then to string
	s := unsafe.Slice(p, n)
	return syscall.UTF16ToString(s)
}
