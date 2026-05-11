//go:build darwin

package driver

// buildApexPacket constructs the HID packet for sending pixel data on macOS.
// macOS IOHIDDeviceSetReport takes the report ID separately from the data buffer.
// sendHIDFeatureReport strips data[0] as the report ID before calling IOKit.
// Format: [00 ReportID] + [61 CMD] + [pixelData] + [1 padding]
// After stripping: device receives [61 CMD] + [pixelData] + [1 padding] = 642 bytes
func buildApexPacket(pixelData []byte, width, height int) []byte {
	dataSize := width * height / 8
	packetSize := 1 + 1 + dataSize + 1 // ReportID(1) + CMD(1) + Data + Padding(1)

	packet := make([]byte, packetSize)
	packet[0] = 0x00 // Report ID (stripped by sendHIDFeatureReport before IOKit call)
	packet[1] = 0x61 // Command byte

	if len(pixelData) > dataSize {
		copy(packet[2:], pixelData[:dataSize])
	} else {
		copy(packet[2:], pixelData)
	}

	return packet
}
