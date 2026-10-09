//go:build windows

package driver

// buildApexPacket constructs the HID packet for sending pixel data on Windows.
// HidD_SetFeature expects the report ID as the first byte (0x00, stripped by the
// HID driver), followed by the frame command and the pixel data:
//   - legacy:      [00 ReportID] + [61] + [pixelData] + [1 padding] = 643 bytes for 128x40;
//     the device receives 642 bytes, matching its HID descriptor.
//   - framebuffer: [00 ReportID] + [1F 81] + [pixelData] = 643 bytes for 128x40;
//     no trailing byte here, because its device declares a 645-byte feature
//     report and the driver zero-pads every packet up to that length.
func buildApexPacket(p *ApexProtocol, pixelData []byte, width, height int) []byte {
	cmd := p.command()
	dataSize := width * height / 8
	trailing := 1
	if p.framebuffer {
		trailing = 0
	}

	packet := make([]byte, 1+len(cmd)+dataSize+trailing)
	packet[0] = 0x00 // Report ID (stripped by Windows HID driver)
	copy(packet[1:], cmd)
	copyApexPixels(packet[1+len(cmd):], pixelData, dataSize)

	return packet
}
