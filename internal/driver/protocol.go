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

// ReportLengthPadding is an optional interface for protocols whose fixed-size
// packets may be shorter than the device's declared HID feature-report length
// and should be zero-padded up to it. Opt-in, because padding every protocol's
// packets risks changing behavior on devices that already work correctly today
// (e.g. the classic Apex family, whose packet size already matches).
type ReportLengthPadding interface {
	PadToReportLength() bool
}

// FeatureReportTransport marks protocols whose packets must be sent strictly as
// HID feature reports. The default Linux transport tries hidraw write() (an
// output report) first, which a device can accept without ever acting on it.
type FeatureReportTransport interface {
	RequiresFeatureReport() bool
}

// protocolRequiresFeatureReport reports whether a protocol's packets must be
// sent strictly as HID feature reports.
func protocolRequiresFeatureReport(p Protocol) bool {
	if f, ok := p.(FeatureReportTransport); ok {
		return f.RequiresFeatureReport()
	}
	return false
}

// protocolWantsReportLengthPadding reports whether a protocol's packets should
// be padded to the device's discovered feature-report length.
func protocolWantsReportLengthPadding(p Protocol) bool {
	if r, ok := p.(ReportLengthPadding); ok {
		return r.PadToReportLength()
	}
	return false
}
