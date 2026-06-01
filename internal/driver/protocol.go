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

// OutputReportSupport marks protocols whose HID reports must be delivered as
// output reports (via HidD_SetOutputReport on Windows / write() on Linux) rather
// than feature reports. The Nova Pro OLED interface (mi_04) rejects feature
// reports with ERROR_INVALID_FUNCTION, so its frames must go out as output reports.
type OutputReportSupport interface {
	UsesOutputReport() bool
}

// protocolUsesOutputReport reports whether a protocol opts into HID output
// reports instead of feature reports.
func protocolUsesOutputReport(p Protocol) bool {
	if or, ok := p.(OutputReportSupport); ok {
		return or.UsesOutputReport()
	}
	return false
}
