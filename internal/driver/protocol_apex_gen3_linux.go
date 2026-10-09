//go:build linux

package driver

// buildApexGen3Packet constructs the HID feature report for sending pixel data
// to the Apex Pro TKL 2023 (PID 0x1628) OLED on Linux.
//
// UNVERIFIED ON REAL LINUX HARDWARE. This layout mirrors the one documented by
// the community reference implementation for the same OLED command
// (https://github.com/SilasDaSilva/apex-pro-tkl-gen3-linux, PROTOCOL.md):
// report ID 0x00 + CMD(2: 1F 81) + 640 bytes of pixel data + a trailing zero
// byte, 644 bytes total. Like HidD_SetFeature on Windows, the kernel strips
// report ID 0 for unnumbered reports, so the device receives 643 bytes. The
// only difference from protocol_apex_gen3_windows.go is the trailing byte,
// which Windows gets from driver-level padding to the declared report length
// instead. Needs confirmation on real Linux hardware before being considered
// reliable.
func buildApexGen3Packet(pixelData []byte, width, height int) []byte {
	dataSize := width * height / 8
	packetSize := 1 + 2 + dataSize + 1 // ReportID(1) + CMD(2: 1F 81) + Data + trailing 0x00

	packet := make([]byte, packetSize)
	packet[0] = 0x00 // Report ID
	packet[1] = 0x1F
	packet[2] = 0x81

	if len(pixelData) > dataSize {
		copy(packet[3:], pixelData[:dataSize])
	} else {
		copy(packet[3:], pixelData)
	}
	// packet[len(packet)-1] stays 0x00 (trailing byte per reference layout).

	return packet
}
