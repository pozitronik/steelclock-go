//go:build linux

package screenmirror

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// DisplayIndexAll is the special index value for capturing all monitors combined.
// Note: This must match the Windows constant value.
const DisplayIndexAll = -1

// linuxMonitorInfo holds information about a single monitor from xrandr.
type linuxMonitorInfo struct {
	Name      string
	Width     int
	Height    int
	X         int
	Y         int
	IsPrimary bool
	Index     int
}

// linuxCapture implements ScreenCapture for Linux using ffmpeg.
type linuxCapture struct {
	cfg         CaptureConfig
	displayInfo DisplayInfo
	mu          sync.Mutex

	// Capture parameters
	display       string // X11 display (e.g., ":0.0")
	captureX      int
	captureY      int
	captureWidth  int
	captureHeight int

	// Window capture
	targetWindowID string
}

// newScreenCapture creates a Linux-specific screen capture.
func newScreenCapture(cfg CaptureConfig) (ScreenCapture, error) {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found: screen_mirror requires ffmpeg for screen capture")
	}

	c := &linuxCapture{
		cfg:     cfg,
		display: os.Getenv("DISPLAY"),
	}

	if c.display == "" {
		c.display = ":0"
	}

	// Initialize display info
	if err := c.initializeDisplay(); err != nil {
		return nil, err
	}

	return c, nil
}

// initializeDisplay sets up display information and capture region.
func (c *linuxCapture) initializeDisplay() error {
	// Determine capture region
	if c.cfg.Window != nil {
		// Window capture mode
		windowID, err := c.findTargetWindow()
		if err != nil {
			return err
		}
		c.targetWindowID = windowID

		// Get window geometry
		x, y, w, h := c.getWindowGeometry(windowID)
		c.captureX = x
		c.captureY = y
		c.captureWidth = w
		c.captureHeight = h

		c.displayInfo = DisplayInfo{
			Index: -1,
			Name:  "Window",
		}
		return nil
	}

	if c.cfg.Region != nil {
		// Region capture mode
		c.captureX = c.cfg.Region.X
		c.captureY = c.cfg.Region.Y
		c.captureWidth = c.cfg.Region.W
		c.captureHeight = c.cfg.Region.H

		c.displayInfo = DisplayInfo{
			Index:  -1,
			Name:   "Region",
			Width:  c.captureWidth,
			Height: c.captureHeight,
			X:      c.captureX,
			Y:      c.captureY,
		}
		return nil
	}

	// Display capture mode - enumerate monitors
	monitors := c.enumerateMonitors()

	// Check for "all monitors" mode (display index = -1)
	if c.cfg.Display.Index != nil && *c.cfg.Display.Index == DisplayIndexAll {
		return c.initializeVirtualScreen(monitors)
	}

	// Try to find monitor by name first (if specified)
	if c.cfg.Display.Name != "" {
		if monitor := c.findMonitorByName(monitors, c.cfg.Display.Name); monitor != nil {
			c.setMonitorCapture(monitor)
			return nil
		}
		// Name specified but not found - return error
		return fmt.Errorf("display '%s' not found", c.cfg.Display.Name)
	}

	// Try to find monitor by index
	if c.cfg.Display.Index != nil {
		if monitor := c.findMonitorByIndex(monitors, *c.cfg.Display.Index); monitor != nil {
			c.setMonitorCapture(monitor)
			return nil
		}
		// Index specified but out of range - return error
		return fmt.Errorf("display index %d not found (available: 0-%d)", *c.cfg.Display.Index, len(monitors)-1)
	}

	// Default: use primary monitor
	if monitor := c.findPrimaryMonitor(monitors); monitor != nil {
		c.setMonitorCapture(monitor)
		return nil
	}

	// Fallback to virtual screen if no monitors found
	return c.initializeVirtualScreen(monitors)
}

// enumerateMonitors returns a list of all connected monitors using xrandr.
func (c *linuxCapture) enumerateMonitors() []linuxMonitorInfo {
	var monitors []linuxMonitorInfo

	cmd := exec.Command("xrandr", "--query")
	output, err := cmd.Output()
	if err != nil {
		return monitors
	}

	// Parse xrandr output for connected monitors
	// Example lines:
	// HDMI-1 connected primary 1920x1080+0+0 (normal left inverted right x axis y axis) 527mm x 296mm
	// DP-1 connected 1920x1080+1920+0 (normal left inverted right x axis y axis) 527mm x 296mm
	// eDP-1 disconnected (normal left inverted right x axis y axis)
	connectedRegex := regexp.MustCompile(`^(\S+)\s+connected\s+(primary\s+)?(\d+)x(\d+)\+(\d+)\+(\d+)`)

	lines := strings.Split(string(output), "\n")
	index := 0
	for _, line := range lines {
		matches := connectedRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		isPrimary := matches[2] != ""
		width, _ := strconv.Atoi(matches[3])
		height, _ := strconv.Atoi(matches[4])
		x, _ := strconv.Atoi(matches[5])
		y, _ := strconv.Atoi(matches[6])

		monitors = append(monitors, linuxMonitorInfo{
			Name:      name,
			Width:     width,
			Height:    height,
			X:         x,
			Y:         y,
			IsPrimary: isPrimary,
			Index:     index,
		})
		index++
	}

	return monitors
}

// findMonitorByIndex finds a monitor by its index.
func (c *linuxCapture) findMonitorByIndex(monitors []linuxMonitorInfo, index int) *linuxMonitorInfo {
	for i := range monitors {
		if monitors[i].Index == index {
			return &monitors[i]
		}
	}
	return nil
}

// findMonitorByName finds a monitor by partial name match (case-insensitive).
func (c *linuxCapture) findMonitorByName(monitors []linuxMonitorInfo, name string) *linuxMonitorInfo {
	nameLower := strings.ToLower(name)
	for i := range monitors {
		if strings.Contains(strings.ToLower(monitors[i].Name), nameLower) {
			return &monitors[i]
		}
	}
	return nil
}

// findPrimaryMonitor finds the primary monitor.
func (c *linuxCapture) findPrimaryMonitor(monitors []linuxMonitorInfo) *linuxMonitorInfo {
	for i := range monitors {
		if monitors[i].IsPrimary {
			return &monitors[i]
		}
	}
	// Fallback to first monitor if no primary found
	if len(monitors) > 0 {
		return &monitors[0]
	}
	return nil
}

// initializeVirtualScreen sets up capture for all monitors combined.
func (c *linuxCapture) initializeVirtualScreen(monitors []linuxMonitorInfo) error {
	// Calculate bounding box of all monitors
	minX, minY := 0, 0
	maxX, maxY := 0, 0

	if len(monitors) > 0 {
		minX = monitors[0].X
		minY = monitors[0].Y
		maxX = monitors[0].X + monitors[0].Width
		maxY = monitors[0].Y + monitors[0].Height

		for _, m := range monitors[1:] {
			if m.X < minX {
				minX = m.X
			}
			if m.Y < minY {
				minY = m.Y
			}
			if m.X+m.Width > maxX {
				maxX = m.X + m.Width
			}
			if m.Y+m.Height > maxY {
				maxY = m.Y + m.Height
			}
		}
	} else {
		// Fallback: get screen resolution the old way
		width, height := c.getScreenResolution()
		if width == 0 || height == 0 {
			width = 1920
			height = 1080
		}
		maxX = width
		maxY = height
	}

	c.captureX = minX
	c.captureY = minY
	c.captureWidth = maxX - minX
	c.captureHeight = maxY - minY

	c.displayInfo = DisplayInfo{
		Index:     DisplayIndexAll,
		Name:      "All Monitors",
		Width:     c.captureWidth,
		Height:    c.captureHeight,
		X:         c.captureX,
		Y:         c.captureY,
		IsPrimary: false,
	}

	return nil
}

// setMonitorCapture configures capture for a specific monitor.
func (c *linuxCapture) setMonitorCapture(monitor *linuxMonitorInfo) {
	c.captureX = monitor.X
	c.captureY = monitor.Y
	c.captureWidth = monitor.Width
	c.captureHeight = monitor.Height

	c.displayInfo = DisplayInfo{
		Index:     monitor.Index,
		Name:      monitor.Name,
		Width:     monitor.Width,
		Height:    monitor.Height,
		X:         monitor.X,
		Y:         monitor.Y,
		IsPrimary: monitor.IsPrimary,
	}
}

// getScreenResolution gets screen resolution using xrandr.
func (c *linuxCapture) getScreenResolution() (int, int) {
	cmd := exec.Command("xrandr", "--query")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	// Parse output for current resolution (line with *)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "*") {
			// Extract resolution from line like "   1920x1080     60.00*+"
			fields := strings.Fields(line)
			if len(fields) > 0 {
				parts := strings.Split(fields[0], "x")
				if len(parts) == 2 {
					w, _ := strconv.Atoi(parts[0])
					h, _ := strconv.Atoi(parts[1])
					if w > 0 && h > 0 {
						return w, h
					}
				}
			}
		}
	}

	return 0, 0
}

// findTargetWindow finds the window to capture based on configuration.
func (c *linuxCapture) findTargetWindow() (string, error) {
	if c.cfg.Window == nil {
		return "", fmt.Errorf("no window configuration")
	}

	// Active window
	if c.cfg.Window.Active {
		return c.getActiveWindowID()
	}

	// Find by title
	if c.cfg.Window.Title != "" {
		return c.findWindowByTitle(c.cfg.Window.Title)
	}

	// Find by class
	if c.cfg.Window.Class != "" {
		return c.findWindowByClass(c.cfg.Window.Class)
	}

	return "", fmt.Errorf("no valid window target specified")
}

// getActiveWindowID returns the ID of the currently active window.
func (c *linuxCapture) getActiveWindowID() (string, error) {
	// Try xdotool first
	cmd := exec.Command("xdotool", "getactivewindow")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Fallback to xprop
	cmd = exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get active window: %w", err)
	}

	// Parse output like "_NET_ACTIVE_WINDOW(WINDOW): window id # 0x1234567"
	parts := strings.Split(string(output), "#")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1]), nil
	}

	return "", fmt.Errorf("could not parse active window ID")
}

// findWindowByTitle finds a window by title substring using xdotool.
func (c *linuxCapture) findWindowByTitle(titleSubstr string) (string, error) {
	cmd := exec.Command("xdotool", "search", "--name", titleSubstr)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("window with title '%s' not found", titleSubstr)
	}

	// Take first result
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0], nil
	}

	return "", fmt.Errorf("window with title '%s' not found", titleSubstr)
}

// findWindowByClass finds a window by class name using xdotool.
func (c *linuxCapture) findWindowByClass(className string) (string, error) {
	cmd := exec.Command("xdotool", "search", "--class", className)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("window with class '%s' not found", className)
	}

	// Take first result
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0], nil
	}

	return "", fmt.Errorf("window with class '%s' not found", className)
}

// getWindowGeometry gets window position and size.
func (c *linuxCapture) getWindowGeometry(windowID string) (x, y, w, h int) {
	// Use xdotool getwindowgeometry
	cmd := exec.Command("xdotool", "getwindowgeometry", "--shell", windowID)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 640, 480 // Default fallback
	}

	// Parse output like:
	// WINDOW=12345
	// X=100
	// Y=200
	// WIDTH=800
	// HEIGHT=600
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		switch key {
		case "X":
			x = val
		case "Y":
			y = val
		case "WIDTH":
			w = val
		case "HEIGHT":
			h = val
		}
	}

	if w == 0 {
		w = 640
	}
	if h == 0 {
		h = 480
	}

	return x, y, w, h
}

// Capture captures a frame from the screen using ffmpeg.
func (c *linuxCapture) Capture() (*image.RGBA, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update window position if capturing a window
	if c.cfg.Window != nil && c.cfg.Window.Active {
		// Re-find active window each frame
		windowID, err := c.getActiveWindowID()
		if err == nil {
			c.targetWindowID = windowID
			x, y, w, h := c.getWindowGeometry(windowID)
			c.captureX = x
			c.captureY = y
			c.captureWidth = w
			c.captureHeight = h
		}
	} else if c.targetWindowID != "" {
		// Update geometry for tracked window
		x, y, w, h := c.getWindowGeometry(c.targetWindowID)
		c.captureX = x
		c.captureY = y
		c.captureWidth = w
		c.captureHeight = h
	}

	if c.captureWidth <= 0 || c.captureHeight <= 0 {
		return nil, fmt.Errorf("invalid capture dimensions")
	}

	// Build ffmpeg command for single frame capture
	// Input: x11grab from display
	// Output: raw RGB to stdout
	grabInput := fmt.Sprintf("%s+%d,%d", c.display, c.captureX, c.captureY)
	videoSize := fmt.Sprintf("%dx%d", c.captureWidth, c.captureHeight)

	cmd := exec.Command("ffmpeg",
		"-f", "x11grab",
		"-video_size", videoSize,
		"-i", grabInput,
		"-frames:v", "1",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-loglevel", "error",
		"-",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg capture failed: %w (%s)", err, stderr.String())
	}

	// Convert raw RGB24 data to RGBA image
	data := stdout.Bytes()
	expectedSize := c.captureWidth * c.captureHeight * 3
	if len(data) != expectedSize {
		return nil, fmt.Errorf("unexpected data size: got %d, expected %d", len(data), expectedSize)
	}

	img := image.NewRGBA(image.Rect(0, 0, c.captureWidth, c.captureHeight))

	for y := 0; y < c.captureHeight; y++ {
		for x := 0; x < c.captureWidth; x++ {
			srcIdx := (y*c.captureWidth + x) * 3
			dstIdx := (y*c.captureWidth + x) * 4
			img.Pix[dstIdx] = data[srcIdx]     // R
			img.Pix[dstIdx+1] = data[srcIdx+1] // G
			img.Pix[dstIdx+2] = data[srcIdx+2] // B
			img.Pix[dstIdx+3] = 255            // A
		}
	}

	return img, nil
}

// Close releases resources.
func (c *linuxCapture) Close() {
	// No persistent resources to release
}

// IsAvailable returns true if screen capture is available.
func (c *linuxCapture) IsAvailable() bool {
	return c.captureWidth > 0 && c.captureHeight > 0
}

// GetDisplayInfo returns information about the captured display.
func (c *linuxCapture) GetDisplayInfo() DisplayInfo {
	return c.displayInfo
}
