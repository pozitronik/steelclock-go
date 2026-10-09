//go:build windows

package driver

// buildApexGen3Packet constructs the HID packet for sending pixel data to Gen 3
// Apex keyboards on Windows. Windows HidD_SetFeature expects report ID as first
// byte (stripped by the HID driver). After the ID is stripped, the device
// receives: [1F 81 CMD] + [pixelData]. Any additional padding needed to reach
// the device's true feature-report length is applied generically by the driver
// (see HIDDriver.reportLen / sendPacket) once the real length is discovered
// from HID capabilities, so this only needs to build the meaningful prefix.
func buildApexGen3Packet(pixelData []byte, width, height int) []byte {
	dataSize := width * height / 8
	packetSize := 1 + 2 + dataSize // ReportID(1) + CMD(2: 1F 81) + Data

	packet := make([]byte, packetSize)
	packet[0] = 0x00 // Report ID (stripped by Windows HID driver)
	packet[1] = 0x1F
	packet[2] = 0x81

	if len(pixelData) > dataSize {
		copy(packet[3:], pixelData[:dataSize])
	} else {
		copy(packet[3:], pixelData)
	}

	return packet
}
