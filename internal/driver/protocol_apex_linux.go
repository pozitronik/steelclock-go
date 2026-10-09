//go:build linux

package driver

// apexFrameBytes is the pixel payload of an Apex OLED frame (128x40 / 8).
// Linux packets have a fixed size matching the HID descriptor.
const apexFrameBytes = 640

// buildApexPacket constructs the HID packet for sending pixel data on Linux.
// Packets have a fixed size matching the HID descriptor (oversized pixel data
// is truncated to fit):
//   - legacy: [61] + [pixelData] + [1 padding] = 642 bytes. The device has no
//     numbered reports, so the command byte goes first and hidraw sends the
//     buffer as-is.
//   - framebuffer: [00 ReportID] + [1F 81] + [pixelData] + [1 padding] = 644
//     bytes, mirroring the community reference implementation
//     (https://github.com/SilasDaSilva/apex-pro-tkl-gen3-linux, PROTOCOL.md).
//     The kernel strips report ID 0 for unnumbered reports, so the device
//     receives 643 bytes. UNVERIFIED ON REAL LINUX HARDWARE.
func buildApexPacket(p *ApexProtocol, pixelData []byte, width, height int) []byte {
	prefix := p.command()
	if p.framebuffer {
		prefix = append([]byte{0x00}, prefix...) // Report ID (stripped by the kernel)
	}
	dataSize := width * height / 8

	packet := make([]byte, len(prefix)+apexFrameBytes+1)
	copy(packet, prefix)
	copyApexPixels(packet[len(prefix):], pixelData, dataSize)

	return packet
}
