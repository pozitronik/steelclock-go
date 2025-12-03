//go:build linux

package driver

// buildPacket constructs the HID packet for sending pixel data on Linux.
// Linux hidraw with no report ID in descriptor expects data without report ID byte
// The HID descriptor shows Feature Report = 642 bytes (8 bits * 642)
// Format: [61 CMD] + [pixelData (640)] + [1 padding] = 642 bytes total
func buildPacket(pixelData []byte, width, height int) []byte {
	dataSize := width * height / 8 // 640 for 128x40
	packetSize := 642              // Fixed size matching HID descriptor

	packet := make([]byte, packetSize)
	packet[0] = 0x61 // Command byte (screen update)

	// Copy pixel data starting at byte 1
	if len(pixelData) > dataSize {
		copy(packet[1:], pixelData[:dataSize])
	} else {
		copy(packet[1:], pixelData)
	}

	// Remaining bytes stay zero (padding to reach 642)
	return packet
}
