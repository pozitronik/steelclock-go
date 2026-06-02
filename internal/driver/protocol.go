package driver

// Protocol defines device-specific HID packet building.
// Different SteelSeries device families (Apex keyboards, Nova Pro headsets, etc.)
// use different HID report formats and encoding schemes.
type Protocol interface {
	// BuildFramePackets converts row-major MSB pixel data into HID packets.
	// Returns multiple packets when data must be split (e.g., Nova Pro strips).
	BuildFramePackets(pixelData []byte, width, height int) [][]byte

	// Interface returns the default USB interface for this protocol (e.g., "mi_01").
	Interface() string

	// DeviceFamily returns a human-readable name for logging.
	DeviceFamily() string
}

// BrightnessSupport is an optional interface for protocols that support display brightness.
type BrightnessSupport interface {
	BuildBrightnessPacket(level int) []byte
}

// UIReturnSupport is an optional interface for protocols that support returning to device UI.
type UIReturnSupport interface {
	BuildReturnToUIPacket() []byte
}

// ScreenByCapability marks protocols whose OLED interface must be located at
// open time by scanning HID feature-report capabilities, because the interface
// varies across device generations (e.g. the Nova Pro family exposes its screen
// on mi_04 on older units but on mi_03/collection 01 on the Omni).
type ScreenByCapability interface {
	DetectScreenInterface() bool
}

// protocolDetectsScreenInterface reports whether a protocol's screen interface
// should be discovered by HID capability rather than a fixed interface string.
func protocolDetectsScreenInterface(p Protocol) bool {
	if s, ok := p.(ScreenByCapability); ok {
		return s.DetectScreenInterface()
	}
	return false
}
