package driver

// ApexProtocol implements the Protocol interface for SteelSeries Apex keyboards.
// Apex keyboards use cmd 0x61, row-major MSB encoding, single packet per frame.
type ApexProtocol struct{}

// BuildFramePackets builds a single HID packet for the Apex keyboard display.
func (p *ApexProtocol) BuildFramePackets(pixelData []byte, width, height int) [][]byte {
	return [][]byte{buildApexPacket(pixelData, width, height)}
}

// Interface returns the default USB interface for Apex keyboards.
func (p *ApexProtocol) Interface() string {
	return "mi_01"
}

// DeviceFamily returns the device family name.
func (p *ApexProtocol) DeviceFamily() string {
	return "Apex Keyboard"
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
