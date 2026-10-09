package driver

// ApexGen3Protocol implements the Protocol interface for the SteelSeries Apex
// Pro TKL 2023 hardware revision identified by PID 0x1628 (USB product string
// "SteelSeries Apex Pro TKL 2023", confirmed via Windows device enumeration),
// whose OLED firmware expects a different command prefix than the original
// Apex Pro/7 family and the existing PID 0x1632 "Apex Pro TKL (2023)" entry.
// Confirmed by reverse-engineering (community protocol docs + USB capture) and
// verified on real hardware (Windows): live framebuffer writes use cmd bytes
// 0x1F 0x81 (not the legacy 0x61), same row-major MSB pixel encoding, single
// packet per frame, same mi_01 interface.
type ApexGen3Protocol struct{}

// BuildFramePackets builds a single HID packet for the Gen 3 Apex keyboard display.
func (p *ApexGen3Protocol) BuildFramePackets(pixelData []byte, width, height int) [][]byte {
	return [][]byte{buildApexGen3Packet(pixelData, width, height)}
}

// Interface returns the default USB interface for Gen 3 Apex keyboards.
func (p *ApexGen3Protocol) Interface() string {
	return "mi_01"
}

// DeviceFamily returns the device family name.
func (p *ApexGen3Protocol) DeviceFamily() string {
	return "Apex Keyboard (Gen 3)"
}

// PadToReportLength opts this protocol into generic report-length padding:
// PID 0x1628 declares a feature report a couple of bytes longer than this
// protocol's fixed packet size (645 vs. 643 on Windows), so the driver pads
// with zero bytes up to the discovered length. Other Apex protocols do not
// implement this interface and are unaffected, so this cannot change
// behavior for already-working devices like the classic Apex Pro (2023).
func (p *ApexGen3Protocol) PadToReportLength() bool {
	return true
}
