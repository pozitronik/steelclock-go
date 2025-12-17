// Package screenmirror provides a widget that captures and displays screen content.
package screenmirror

import (
	"image"
)

// DisplayInfo contains information about a display.
type DisplayInfo struct {
	// Index is the display index (0-based).
	Index int
	// Name is the display name (if available).
	Name string
	// Width is the display width in pixels.
	Width int
	// Height is the display height in pixels.
	Height int
	// X is the display X position in virtual screen coordinates.
	X int
	// Y is the display Y position in virtual screen coordinates.
	Y int
	// IsPrimary indicates if this is the primary display.
	IsPrimary bool
}

// CaptureRegion defines a rectangular region to capture.
type CaptureRegion struct {
	X      int
	Y      int
	Width  int
	Height int
}

// WindowTarget defines which window to capture.
type WindowTarget struct {
	// Title is a substring to match in the window title.
	Title string
	// Class is the window class name to match (Windows-specific).
	Class string
	// Active indicates to capture the currently active window.
	Active bool
}

// DisplaySelector specifies which display to capture.
// Either Index or Name should be set, not both.
// If both are nil/empty, the primary display is used.
type DisplaySelector struct {
	// Index is the display index (nil = not specified).
	// Use -1 for all monitors combined.
	Index *int
	// Name is the display name to match (partial, case-insensitive).
	Name string
}

// CaptureConfig contains the configuration for screen capture.
type CaptureConfig struct {
	// Display specifies which display to capture.
	Display DisplaySelector
	// Region is the capture region (nil = full display).
	Region *CaptureRegion
	// Window is the window to capture (nil = capture display/region).
	Window *WindowTarget
}

// ScreenCapture abstracts platform-specific screen capture.
type ScreenCapture interface {
	// Capture captures a frame from the screen.
	// Returns an RGBA image of the captured content.
	Capture() (*image.RGBA, error)

	// Close releases any resources held by the capture.
	Close()

	// IsAvailable returns true if the capture method is available.
	IsAvailable() bool

	// GetDisplayInfo returns information about the captured display.
	GetDisplayInfo() DisplayInfo
}
