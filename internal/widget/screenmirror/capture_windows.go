//go:build windows

package screenmirror

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Windows API constants
const (
	SM_CXSCREEN        = 0
	SM_CYSCREEN        = 1
	SM_XVIRTUALSCREEN  = 76
	SM_YVIRTUALSCREEN  = 77
	SM_CXVIRTUALSCREEN = 78
	SM_CYVIRTUALSCREEN = 79

	SRCCOPY        = 0x00CC0020
	DIB_RGB_COLORS = 0
	BI_RGB         = 0

	BITSPIXEL = 12
)

// Windows API functions
var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	gdi32                      = syscall.NewLazyDLL("gdi32.dll")
	procGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	procGetDC                  = user32.NewProc("GetDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
	procGetDesktopWindow       = user32.NewProc("GetDesktopWindow")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procGetDIBits              = gdi32.NewProc("GetDIBits")

	// Window capture
	procFindWindowW          = user32.NewProc("FindWindowW")
	procGetForegroundWindow  = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procEnumWindows          = user32.NewProc("EnumWindows")
	procIsWindowVisible      = user32.NewProc("IsWindowVisible")
	procPrintWindow          = user32.NewProc("PrintWindow")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procClientToScreen       = user32.NewProc("ClientToScreen")
)

// BITMAPINFOHEADER structure for GetDIBits
type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// RECT structure
type rect struct {
	Left, Top, Right, Bottom int32
}

// POINT structure
type point struct {
	X, Y int32
}

// windowsCapture implements ScreenCapture for Windows using GDI.
type windowsCapture struct {
	cfg         CaptureConfig
	displayInfo DisplayInfo
	mu          sync.Mutex

	// Capture region
	captureX      int
	captureY      int
	captureWidth  int
	captureHeight int

	// Window capture
	targetHwnd uintptr
}

// newScreenCapture creates a Windows-specific screen capture.
func newScreenCapture(cfg CaptureConfig) (ScreenCapture, error) {
	c := &windowsCapture{
		cfg: cfg,
	}

	// Initialize display info
	if err := c.initializeDisplay(); err != nil {
		return nil, err
	}

	return c, nil
}

// initializeDisplay sets up display information and capture region.
func (c *windowsCapture) initializeDisplay() error {
	// Get virtual screen dimensions (all monitors)
	virtualX, _, _ := procGetSystemMetrics.Call(SM_XVIRTUALSCREEN)
	virtualY, _, _ := procGetSystemMetrics.Call(SM_YVIRTUALSCREEN)
	virtualWidth, _, _ := procGetSystemMetrics.Call(SM_CXVIRTUALSCREEN)
	virtualHeight, _, _ := procGetSystemMetrics.Call(SM_CYVIRTUALSCREEN)

	// Get primary screen dimensions
	primaryWidth, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	primaryHeight, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)

	// Set up display info (primary monitor for now)
	c.displayInfo = DisplayInfo{
		Index:     0,
		Name:      "Primary",
		Width:     int(primaryWidth),
		Height:    int(primaryHeight),
		X:         0,
		Y:         0,
		IsPrimary: true,
	}

	// Determine capture region
	if c.cfg.Window != nil {
		// Window capture mode - find the target window
		hwnd, err := c.findTargetWindow()
		if err != nil {
			return err
		}
		c.targetHwnd = hwnd

		// Get window bounds
		var r rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		c.captureX = int(r.Left)
		c.captureY = int(r.Top)
		c.captureWidth = int(r.Right - r.Left)
		c.captureHeight = int(r.Bottom - r.Top)
	} else if c.cfg.Region != nil {
		// Region capture mode
		c.captureX = c.cfg.Region.X
		c.captureY = c.cfg.Region.Y
		c.captureWidth = c.cfg.Region.Width
		c.captureHeight = c.cfg.Region.Height
	} else {
		// Full screen capture (primary monitor)
		c.captureX = 0
		c.captureY = 0
		c.captureWidth = int(primaryWidth)
		c.captureHeight = int(primaryHeight)

		// If display index is specified and > 0, use virtual screen
		if c.cfg.DisplayIndex != nil && *c.cfg.DisplayIndex > 0 {
			c.captureX = int(virtualX)
			c.captureY = int(virtualY)
			c.captureWidth = int(virtualWidth)
			c.captureHeight = int(virtualHeight)
			c.displayInfo.Name = "Virtual"
		}
	}

	return nil
}

// findTargetWindow finds the window to capture based on configuration.
func (c *windowsCapture) findTargetWindow() (uintptr, error) {
	if c.cfg.Window == nil {
		return 0, fmt.Errorf("no window configuration")
	}

	// Active window
	if c.cfg.Window.Active {
		hwnd, _, _ := procGetForegroundWindow.Call()
		if hwnd == 0 {
			return 0, fmt.Errorf("no active window found")
		}
		return hwnd, nil
	}

	// Find by window class
	if c.cfg.Window.Class != "" {
		classPtr, _ := syscall.UTF16PtrFromString(c.cfg.Window.Class)
		hwnd, _, _ := procFindWindowW.Call(
			uintptr(unsafe.Pointer(classPtr)),
			0,
		)
		if hwnd != 0 {
			return hwnd, nil
		}
	}

	// Find by title substring
	if c.cfg.Window.Title != "" {
		hwnd := c.findWindowByTitle(c.cfg.Window.Title)
		if hwnd != 0 {
			return hwnd, nil
		}
		return 0, fmt.Errorf("window with title containing '%s' not found", c.cfg.Window.Title)
	}

	return 0, fmt.Errorf("no valid window target specified")
}

// findWindowByTitle finds a window by title substring.
func (c *windowsCapture) findWindowByTitle(titleSubstr string) uintptr {
	var foundHwnd uintptr
	titleLower := strings.ToLower(titleSubstr)

	// Callback for EnumWindows
	callback := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		// Check if window is visible
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1 // Continue enumeration
		}

		// Get window title
		length, _, _ := procGetWindowTextLengthW.Call(hwnd)
		if length == 0 {
			return 1
		}

		buf := make([]uint16, length+1)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), length+1)
		title := syscall.UTF16ToString(buf)

		if strings.Contains(strings.ToLower(title), titleLower) {
			foundHwnd = hwnd
			return 0 // Stop enumeration
		}
		return 1
	})

	procEnumWindows.Call(callback, 0)
	return foundHwnd
}

// Capture captures a frame from the screen.
func (c *windowsCapture) Capture() (*image.RGBA, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update window position if capturing a window
	if c.cfg.Window != nil {
		if c.cfg.Window.Active {
			// Re-find active window each frame
			hwnd, _, _ := procGetForegroundWindow.Call()
			if hwnd != 0 {
				c.targetHwnd = hwnd
			}
		}

		if c.targetHwnd != 0 {
			var r rect
			procGetWindowRect.Call(c.targetHwnd, uintptr(unsafe.Pointer(&r)))
			c.captureX = int(r.Left)
			c.captureY = int(r.Top)
			c.captureWidth = int(r.Right - r.Left)
			c.captureHeight = int(r.Bottom - r.Top)
		}
	}

	if c.captureWidth <= 0 || c.captureHeight <= 0 {
		return nil, fmt.Errorf("invalid capture dimensions")
	}

	// Get screen DC
	desktopHwnd, _, _ := procGetDesktopWindow.Call()
	srcDC, _, _ := procGetDC.Call(desktopHwnd)
	if srcDC == 0 {
		return nil, fmt.Errorf("failed to get screen DC")
	}
	defer procReleaseDC.Call(desktopHwnd, srcDC)

	// Create compatible DC and bitmap
	memDC, _, _ := procCreateCompatibleDC.Call(srcDC)
	if memDC == 0 {
		return nil, fmt.Errorf("failed to create compatible DC")
	}
	defer procDeleteDC.Call(memDC)

	bitmap, _, _ := procCreateCompatibleBitmap.Call(srcDC, uintptr(c.captureWidth), uintptr(c.captureHeight))
	if bitmap == 0 {
		return nil, fmt.Errorf("failed to create bitmap")
	}
	defer procDeleteObject.Call(bitmap)

	// Select bitmap into memory DC
	oldBitmap, _, _ := procSelectObject.Call(memDC, bitmap)
	defer procSelectObject.Call(memDC, oldBitmap)

	// Copy screen to memory DC
	ret, _, _ := procBitBlt.Call(
		memDC,
		0, 0,
		uintptr(c.captureWidth), uintptr(c.captureHeight),
		srcDC,
		uintptr(c.captureX), uintptr(c.captureY),
		SRCCOPY,
	)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt failed")
	}

	// Get bitmap bits
	img := image.NewRGBA(image.Rect(0, 0, c.captureWidth, c.captureHeight))

	// Set up BITMAPINFOHEADER
	bi := bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       int32(c.captureWidth),
		Height:      -int32(c.captureHeight), // Negative for top-down DIB
		Planes:      1,
		BitCount:    32,
		Compression: BI_RGB,
	}

	// Get DIB bits
	ret, _, _ = procGetDIBits.Call(
		memDC,
		bitmap,
		0,
		uintptr(c.captureHeight),
		uintptr(unsafe.Pointer(&img.Pix[0])),
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	// Windows uses BGRA, Go uses RGBA - swap channels
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+2] = img.Pix[i+2], img.Pix[i] // Swap B and R
	}

	return img, nil
}

// Close releases resources.
func (c *windowsCapture) Close() {
	// No persistent resources to release with GDI approach
}

// IsAvailable returns true if screen capture is available.
func (c *windowsCapture) IsAvailable() bool {
	return c.captureWidth > 0 && c.captureHeight > 0
}

// GetDisplayInfo returns information about the captured display.
func (c *windowsCapture) GetDisplayInfo() DisplayInfo {
	return c.displayInfo
}
