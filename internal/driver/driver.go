// Package driver provides direct USB HID communication with SteelSeries OLED displays,
// bypassing the GameSense backend.
package driver

import (
	"fmt"
	"log"
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

	// HID capabilities (populated for SteelSeries interfaces during diagnostics;
	// HasCaps is false when they could not be read). Report lengths include the
	// report ID byte, matching the Windows HIDP_CAPS convention.
	HasCaps          bool
	UsagePage        uint16
	Usage            uint16
	InputReportLen   int
	OutputReportLen  int
	FeatureReportLen int
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
	// reportLen is the device's expected HID feature-report length (including the
	// report ID), discovered from capabilities at Open. Packets shorter than this
	// are zero-padded up to it. 0 means no padding (use the packet as built).
	reportLen int
	// reportID is the HID feature report ID the device actually accepts, discovered
	// at Open for capability-detected screens. When known, it overrides the report
	// ID byte the protocol builds (older Nova units use 0x06, the Omni uses 0x01).
	reportID      byte
	reportIDKnown bool
	mu            sync.RWMutex
}

// screenReportIDCandidates are tried first when discovering a screen's feature
// report ID: 0x06 (older Nova/GameDAC), 0x01 (Nova Pro Omni), 0x00 (unnumbered).
var screenReportIDCandidates = []byte{0x06, 0x01, 0x00}

// discoverFeatureReportID returns the first HID feature report ID the device
// accepts a full-size (featureLen) report under — a valid ID is accepted, others
// error. Sends benign all-zero frames. Tries the known candidates first, then
// scans the rest.
func discoverFeatureReportID(handle DeviceHandle, featureLen int) (byte, bool) {
	if featureLen < 1 {
		return 0, false
	}
	accepts := func(rid byte) bool {
		buf := make([]byte, featureLen)
		buf[0] = rid
		return sendFeatureReport(handle, buf) == nil
	}
	tried := make(map[byte]bool, 256)
	for _, rid := range screenReportIDCandidates {
		tried[rid] = true
		if accepts(rid) {
			return rid, true
		}
	}
	for r := 0; r <= 0xFF; r++ {
		if tried[byte(r)] {
			continue
		}
		if accepts(byte(r)) {
			return byte(r), true
		}
	}
	return 0, false
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

// findScreenInterfaceByCaps locates the OLED interface of a VID/PID by HID
// capability rather than a fixed interface string. Among that device's
// SteelSeries collections it picks the one with the largest FEATURE report that
// is big enough to carry a frame, returning its device path and report length.
// This is what lets the Nova Pro family work across generations whose OLED lives
// on different interfaces (older units expose it on mi_04, the Omni on mi_03
// collection 01, while mi_04 there is an unrelated consumer-control collection).
// Returns ok=false when capabilities are unavailable (e.g. Linux) or no suitable
// collection is found, so the caller can fall back to the interface-based path.
func findScreenInterfaceByCaps(vid, pid uint16, minFeatureLen int) (path string, featureLen int, ok bool) {
	devices, err := EnumerateDevices()
	if err != nil {
		return "", 0, false
	}
	for _, d := range devices {
		if d.VID != vid || d.PID != pid || !d.HasCaps {
			continue
		}
		if d.FeatureReportLen >= minFeatureLen && d.FeatureReportLen > featureLen {
			featureLen = d.FeatureReportLen
			path = d.Path
		}
	}
	return path, featureLen, path != ""
}

// Open finds and opens a device connection
func (d *HIDDriver) Open() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return nil // Already connected
	}

	// Resolve VID/PID (auto-detecting from the known-device table when not
	// configured) and the protocol that goes with them.
	var devicePath string
	var err error

	if d.config.VID != 0 && d.config.PID != 0 {
		devicePath, err = findDevicePath(d.config.VID, d.config.PID, d.config.Interface)
		if err != nil {
			return fmt.Errorf("device not found: %w", err)
		}
	} else {
		var matched *KnownDevice
		matched, devicePath, err = autoDetectKnownDevice()
		if err != nil {
			return fmt.Errorf("device not found: %w", err)
		}
		d.protocol = resolveProtocol(matched.VID, matched.PID)
		d.config.VID = matched.VID
		d.config.PID = matched.PID
		d.config.Interface = deviceInterface(matched)
	}

	// Protocols whose OLED interface varies by device generation (Nova Pro)
	// locate the screen collection by HID feature-report capability, overriding
	// the interface-derived path. This finds mi_03/col01 on the Omni and mi_04 on
	// older units, and yields the exact report length to pad frames to.
	if protocolDetectsScreenInterface(d.protocol) {
		minLen := d.config.Width * d.config.Height / 8
		if p, flen, found := findScreenInterfaceByCaps(d.config.VID, d.config.PID, minLen); found {
			devicePath = p
			d.reportLen = flen
			log.Printf("Direct driver: selected screen interface by capability (feature report %d bytes): %s", flen, p)
		} else {
			log.Printf("Direct driver: no capability-matched screen interface for %04X:%04X, using %s",
				d.config.VID, d.config.PID, d.config.Interface)
		}
	}

	// Open device
	handle, err := openDevice(devicePath)
	if err != nil {
		return fmt.Errorf("failed to open device: %w", err)
	}

	d.handle = handle
	d.connected = true

	// For capability-detected screens, discover the feature report ID the device
	// actually accepts (older Nova units use 0x06, the Omni uses 0x01) and override
	// whatever the protocol builds. Without this the device rejects frames with
	// "the parameter is incorrect".
	if protocolDetectsScreenInterface(d.protocol) && d.reportLen > 0 {
		if rid, ok := discoverFeatureReportID(handle, d.reportLen); ok {
			d.reportID = rid
			d.reportIDKnown = true
			log.Printf("Direct driver: using screen feature report ID 0x%02X", rid)
		} else {
			log.Printf("Direct driver: could not determine an accepted feature report ID")
		}
	}

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

	// Send each packet using the transport the device protocol requires.
	for _, packet := range packets {
		if err := d.sendPacket(packet); err != nil {
			// Mark as disconnected on send failure
			d.connected = false
			_ = closeDevice(d.handle)
			d.handle = InvalidHandle
			return fmt.Errorf("send failed: %w", err)
		}
	}

	return nil
}

// sendPacket delivers a single HID feature report, zero-padding it up to the
// device's expected report length when that is known (some devices, e.g. the
// Nova Pro Omni, require a fixed 1036-byte feature report rather than 1024).
func (d *HIDDriver) sendPacket(packet []byte) error {
	if d.reportLen > len(packet) {
		buf := make([]byte, d.reportLen)
		copy(buf, packet)
		packet = buf
	}
	// Override the report ID byte with the one the device actually accepts.
	if d.reportIDKnown && len(packet) > 0 {
		packet[0] = d.reportID
	}
	return sendFeatureReport(d.handle, packet)
}

// SendRawPacket sends a pre-built packet directly to the device as a HID feature
// report (padded to the device's report length when known). Used for control
// packets (brightness, return-to-UI) that bypass the protocol's frame building.
func (d *HIDDriver) SendRawPacket(packet []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return fmt.Errorf("device not connected")
	}

	if err := d.sendPacket(packet); err != nil {
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
