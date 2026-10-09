package driver

// Apex OLED frame commands.
var (
	// apexLegacyCommand is the frame command of the original Apex 7/Pro/5 firmware.
	apexLegacyCommand = []byte{0x61}
	// apexFramebufferCommand is the live-framebuffer write of newer firmware
	// (device command 0x1F, sub-command 0x81), e.g. the PID 0x1628 Apex Pro TKL
	// 2023. The legacy command is silently accepted there but never shown.
	apexFramebufferCommand = []byte{0x1F, 0x81}
	// apexReturnToUICommand makes the framebuffer variant's OLED show the
	// default SteelSeries screen again (device command 0x1F, sub-command 0x82).
	apexReturnToUICommand = []byte{0x1F, 0x82}
)

// apexOutputReportLen is the framebuffer variant's HID output report length:
// report ID plus 64 bytes.
const apexOutputReportLen = 65

// ApexProtocol implements the Protocol interface for SteelSeries Apex keyboards:
// row-major MSB encoding, single packet per frame, mi_01 interface. The zero
// value speaks the original 0x61 frame command; NewApexFramebufferProtocol
// returns the variant for newer firmware.
type ApexProtocol struct {
	// framebuffer selects the newer firmware's 0x1F 0x81 live-framebuffer
	// command. Its frames are feature reports that the device declares longer
	// than the frame, so this variant also opts into report-length padding and
	// the strict feature-report transport.
	framebuffer bool
}

// NewApexFramebufferProtocol returns the Apex protocol variant for newer
// firmware that takes frames as 0x1F 0x81 live-framebuffer writes.
func NewApexFramebufferProtocol() *ApexProtocol {
	return &ApexProtocol{framebuffer: true}
}

// command returns the frame command prefix for this protocol variant.
func (p *ApexProtocol) command() []byte {
	if p.framebuffer {
		return apexFramebufferCommand
	}
	return apexLegacyCommand
}

// BuildFramePackets builds a single HID packet for the Apex keyboard display.
func (p *ApexProtocol) BuildFramePackets(pixelData []byte, width, height int) [][]byte {
	return [][]byte{buildApexPacket(p, pixelData, width, height)}
}

// Interface returns the default USB interface for Apex keyboards.
func (p *ApexProtocol) Interface() string {
	return "mi_01"
}

// DeviceFamily returns the device family name.
func (p *ApexProtocol) DeviceFamily() string {
	if p.framebuffer {
		return "Apex Keyboard (framebuffer command)"
	}
	return "Apex Keyboard"
}

// PadToReportLength opts the framebuffer variant into report-length padding:
// its device declares a feature report longer than the frame packet (645 vs.
// 643 bytes on Windows), so the driver zero-pads up to the discovered length.
// The legacy variant keeps its exact packet size.
func (p *ApexProtocol) PadToReportLength() bool {
	return p.framebuffer
}

// RequiresFeatureReport makes the framebuffer variant skip the Linux write()
// attempt: its frames are feature reports, while the interface's output report
// is only 65 bytes. The legacy variant keeps the write()-first transport it has
// always worked with.
func (p *ApexProtocol) RequiresFeatureReport() bool {
	return p.framebuffer
}

// BuildReturnToUIPacket builds the framebuffer variant's return-to-UI command, a
// HID output report: [00 ReportID] + [1F 82] + zero padding = 65 bytes. The
// legacy command set has no such command, so it returns nil.
func (p *ApexProtocol) BuildReturnToUIPacket() []byte {
	if !p.framebuffer {
		return nil
	}
	packet := make([]byte, apexOutputReportLen)
	copy(packet[1:], apexReturnToUICommand)
	return packet
}

// ReturnToUIIsOutputReport reports that the framebuffer variant's return-to-UI
// command is an output report, unlike its frames, which are feature reports.
func (p *ApexProtocol) ReturnToUIIsOutputReport() bool {
	return p.framebuffer
}

// copyApexPixels copies at most dataSize bytes of pixel data into dst.
func copyApexPixels(dst, pixelData []byte, dataSize int) {
	if len(pixelData) > dataSize {
		pixelData = pixelData[:dataSize]
	}
	copy(dst, pixelData)
}

// resolveProtocol determines the appropriate protocol for a device based on VID/PID.
// Returns ApexProtocol as the default when the device is not found or has no specific protocol.
func resolveProtocol(vid, pid uint16) Protocol {
	if vid != 0 && pid != 0 {
		for _, dev := range KnownDevices {
			if dev.VID == vid && dev.PID == pid {
				if dev.NewProtocol != nil {
					return dev.NewProtocol()
				}
				break
			}
		}
	}
	return &ApexProtocol{}
}
