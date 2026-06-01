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
	protocol   Protocol
	handle     DeviceHandle
	deviceInfo DeviceInfo
	connected  bool
	mu         sync.RWMutex
}

// DeviceHandle is a platform-specific device handle type
// Defined in platform-specific files (hid_windows.go, hid_unix.go)

// NewDriver creates a new HID driver with the given configuration
func NewDriver(cfg Config) *HIDDriver {
	protocol := resolveProtocol(cfg.VID, cfg.PID)

	if cfg.Interface == "" {
		cfg.Interface = protocol.Interface()
	}
	if cfg.Width == 0 {
		cfg.Width = 128
	}
	if cfg.Height == 0 {
		cfg.Height = 40
	}

	return &HIDDriver{
		config:   cfg,
		protocol: protocol,
	}
}

// deviceInterface returns the USB interface a known device exposes its display
// on, derived from the device's protocol (Apex keyboards on mi_01, Nova Pro
// family on mi_04). The protocol is the single source of truth for this.
func deviceInterface(dev *KnownDevice) string {
	if dev.NewProtocol != nil {
		return dev.NewProtocol().Interface()
	}
	return (&ApexProtocol{}).Interface()
}

// autoDetectKnownDevice scans KnownDevices and returns the first one whose
// VID/PID is present on its expected USB interface. Searching each device on
// its own interface (rather than a single fixed one) is what lets Nova Pro
// devices — whose OLED lives on mi_04, not the keyboards' mi_01 — be found.
func autoDetectKnownDevice() (*KnownDevice, string, error) {
	for i := range KnownDevices {
		dev := &KnownDevices[i]
		path, err := findDevicePath(dev.VID, dev.PID, deviceInterface(dev))
		if err == nil {
			return dev, path, nil
		}
	}
	return nil, "", fmt.Errorf("no known SteelSeries device found")
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
		// Auto-detect from known devices. Each device family is searched on its
		// own USB interface and, once matched, its protocol/interface/IDs are
		// adopted so the correct frame format is used for whatever was found.
		var matched *KnownDevice
		matched, devicePath, err = autoDetectKnownDevice()
		if err == nil {
			d.protocol = resolveProtocol(matched.VID, matched.PID)
			d.config.VID = matched.VID
			d.config.PID = matched.PID
			d.config.Interface = deviceInterface(matched)
		}
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

	// Build packets using the device-specific protocol
	packets := d.protocol.BuildFramePackets(pixelData, d.config.Width, d.config.Height)

	// Send each packet via HID SetFeature
	for _, packet := range packets {
		if err := sendFeatureReport(d.handle, packet); err != nil {
			// Mark as disconnected on send failure
			d.connected = false
			_ = closeDevice(d.handle)
			d.handle = InvalidHandle
			return fmt.Errorf("send failed: %w", err)
		}
	}

	return nil
}

// SendRawPacket sends a pre-built packet directly to the device via HID SetFeature.
// Used for control packets (brightness, return-to-UI) that bypass the protocol's frame building.
func (d *HIDDriver) SendRawPacket(packet []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return fmt.Errorf("device not connected")
	}

	if err := sendFeatureReport(d.handle, packet); err != nil {
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

// buildApexPacket is defined in platform-specific files:
// - protocol_apex_windows.go: Format [00 ReportID] + [61 CMD] + [pixelData] + [1 Padding] = 643 bytes (Report ID stripped by OS)
// - protocol_apex_linux.go: Format [61 CMD] + [pixelData] + [1 Padding] = 642 bytes
// Both send the same data to device: [61 CMD] + [pixelData] + [1 Padding] = 642 bytes (for 128x40)
