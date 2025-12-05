//go:build windows

package driver

// buildPacket constructs the HID packet for sending pixel data on Windows.
// Windows HidD_SetFeature expects report ID as first byte (stripped by HID driver).
// After Report ID is stripped, device receives: [61 CMD] + [data] + [1 padding]
// This matches the Linux implementation format.
// Format: [00 ReportID] + [61 CMD] + [pixelData] + [1 padding]
func buildPacket(pixelData []byte, width, height int) []byte {
	dataSize := width * height / 8
	// ReportID(1) + CMD(1) + Data + Padding(1)
	// After Windows strips ReportID, device gets: CMD(1) + Data + Padding(1)
	packetSize := 1 + 1 + dataSize + 1

	packet := make([]byte, packetSize)
	packet[0] = 0x00 // Report ID (stripped by Windows HID driver)
	packet[1] = 0x61 // Command byte

	// Copy pixel data starting at byte 2
	if len(pixelData) > dataSize {
		copy(packet[2:], pixelData[:dataSize])
	} else {
		copy(packet[2:], pixelData)
	}

	// Last byte stays zero (trailing padding)
	return packet
}
