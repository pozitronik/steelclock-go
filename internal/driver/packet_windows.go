//go:build windows

package driver

// buildPacket constructs the HID packet for sending pixel data on Windows.
// Windows HidD_SetFeature expects report ID as first byte
// Format: [00 ReportID] + [61 CMD] + [16 Padding] + [pixelData]
func buildPacket(pixelData []byte, width, height int) []byte {
	dataSize := width * height / 8
	packetSize := 18 + dataSize // ReportID(1) + CMD(1) + Padding(16) + Data

	packet := make([]byte, packetSize)
	packet[0] = 0x00 // Report ID
	packet[1] = 0x61 // Command byte
	// Bytes 2-17 stay zero (padding)

	// Copy pixel data, truncate or pad as needed
	if len(pixelData) > dataSize {
		copy(packet[18:], pixelData[:dataSize])
	} else {
		copy(packet[18:], pixelData)
	}

	return packet
}
