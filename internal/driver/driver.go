// Package driver provides direct USB HID communication with SteelSeries OLED displays,
// bypassing the GameSense backend.
package driver

import (
	"fmt"
	"sync"
)

// DeviceInfo contains information about a connected device
type DeviceInfo struct {
	VID          uint16 // Vendor ID
	PID          uint16 // Product ID
	Path         string // Device path
	ProductName  string // Product name (if available)
	Manufacturer string // Manufacturer name (if available)
	Interface    string // Interface identifier (e.g., "mi_01")
}

// Driver interface for USB HID communication with OLED displays
type Driver interface {
	// Open finds and opens a device connection
	Open() error

	// Close closes the device connection
	Close() error

	// SendFrame sends raw pixel data to the display
	// pixelData should be packed bits (1 bit per pixel), size = width * height / 8
	SendFrame(pixelData []byte) error

	// IsConnected returns true if device is currently connected
	IsConnected() bool

	// DeviceInfo returns information about the connected device
	DeviceInfo() DeviceInfo

	// Reconnect attempts to reconnect to the device
	Reconnect() error
}

// Config holds driver configuration
type Config struct {
	VID       uint16 // Vendor ID (0 = auto-detect)
	PID       uint16 // Product ID (0 = auto-detect)
	Interface string // USB interface (default "mi_01")
	Width     int    // Display width in pixels
	Height    int    // Display height in pixels
}

// HIDDriver implements Driver interface using USB HID
type HIDDriver struct {
	config     Config
	handle     DeviceHandle
	deviceInfo DeviceInfo
	connected  bool
	mu         sync.RWMutex
}

// DeviceHandle is a platform-specific device handle type
// Defined in platform-specific files (hid_windows.go, hid_unix.go)

// NewDriver creates a new HID driver with the given configuration
func NewDriver(cfg Config) *HIDDriver {
	if cfg.Interface == "" {
		cfg.Interface = "mi_01"
	}
	if cfg.Width == 0 {
		cfg.Width = 128
	}
	if cfg.Height == 0 {
		cfg.Height = 40
	}

	return &HIDDriver{
		config: cfg,
	}
}

// Open finds and opens a device connection
func (d *HIDDriver) Open() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return nil // Already connected
	}

	// Find device
	var devicePath string
	var err error

	if d.config.VID != 0 && d.config.PID != 0 {
		// Use specified VID/PID
		devicePath, err = findDevicePath(d.config.VID, d.config.PID, d.config.Interface)
	} else {
		// Auto-detect from known devices
		devicePath, err = autoDetectDevice(d.config.Interface)
	}

	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// Open device
	handle, err := openDevice(devicePath)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}

	d.handle = handle
	d.connected = true
	d.deviceInfo = DeviceInfo{
		VID:       d.config.VID,
		PID:       d.config.PID,
		Path:      devicePath,
		Interface: d.config.Interface,
	}

	return nil
}

// Close closes the device connection
func (d *HIDDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return nil
	}

	err := closeDevice(d.handle)
	d.connected = false
	d.handle = InvalidHandle

	return err
}

// SendFrame sends raw pixel data to the display
func (d *HIDDriver) SendFrame(pixelData []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return fmt.Errorf("device not connected")
	}

	// Build packet: [00 ReportID] + [61 CMD] + [32 Padding] + [Data]
	packet := buildPacket(pixelData, d.config.Width, d.config.Height)

	// Send via HID SetFeature
	if err := sendFeatureReport(d.handle, packet); err != nil {
		// Mark as disconnected on send failure
		d.connected = false
		_ = closeDevice(d.handle)
		d.handle = InvalidHandle
		return fmt.Errorf("send failed: %w", err)
	}

	return nil
}

// IsConnected returns true if device is currently connected
func (d *HIDDriver) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected
}

// DeviceInfo returns information about the connected device
func (d *HIDDriver) DeviceInfo() DeviceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.deviceInfo
}

// Reconnect attempts to reconnect to the device
func (d *HIDDriver) Reconnect() error {
	// Close existing connection if any
	_ = d.Close()
	// Try to open again
	return d.Open()
}

// buildPacket is defined in platform-specific files:
// - packet_windows.go: Format [00 ReportID] + [61 CMD] + [16 Padding] + [pixelData] = 658 bytes
// - packet_linux.go: Format [61 CMD] + [1 Padding] + [pixelData] = 642 bytes
