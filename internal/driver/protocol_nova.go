package driver

// Nova Pro protocol constants
const (
	novaReportID      = 0x06 // HID Report ID for Nova Pro feature reports
	novaScreenCommand = 0x93 // Command byte for screen update
	novaHeaderSize    = 6    // [ReportID, Command, DstX, DstY, Width, PaddedHeight]
	novaReportSize    = 1024 // Fixed HID feature report size
	novaMaxStripWidth = 64   // Maximum pixels per strip (hardware limit)
)

// NovaProProtocol implements the Protocol interface for SteelSeries Nova Pro headsets
// and GameDAC Gen 2 devices. Uses Report ID 0x06, command 0x93, column-major LSB encoding,
// and splits frames into 64-pixel-wide strips.
type NovaProProtocol struct{}

// BuildFramePackets converts row-major MSB pixel data into Nova Pro HID packets.
// For displays wider than 64 pixels, the frame is split into multiple 64px-wide strips.
func (p *NovaProProtocol) BuildFramePackets(pixelData []byte, width, height int) [][]byte {
	paddedHeight := padHeight(height)

	var packets [][]byte
	for srcX := 0; srcX < width; srcX += novaMaxStripWidth {
		stripWidth := width - srcX
		if stripWidth > novaMaxStripWidth {
			stripWidth = novaMaxStripWidth
		}

		bitmapData := rowMajorMSBToColumnMajorLSB(pixelData, srcX, stripWidth, width, height, paddedHeight)
		packet := buildNovaPacket(byte(srcX), 0, byte(stripWidth), byte(paddedHeight), bitmapData)
		packets = append(packets, packet)
	}

	return packets
}

// Interface returns the default USB interface for Nova Pro devices.
func (p *NovaProProtocol) Interface() string {
	return "mi_04"
}

// DeviceFamily returns the device family name.
func (p *NovaProProtocol) DeviceFamily() string {
	return "Nova Pro"
}

// buildNovaPacket constructs a 1024-byte HID feature report for Nova Pro devices.
// Both Linux and Windows use the same format: Report ID 0x06 as first byte.
// This is because Nova Pro uses numbered reports (ID=0x06), and both platforms
// require the report ID as the first byte of the buffer for numbered reports.
func buildNovaPacket(dstX, dstY, width, paddedHeight byte, bitmapData []byte) []byte {
	packet := make([]byte, novaReportSize)
	packet[0] = novaReportID
	packet[1] = novaScreenCommand
	packet[2] = dstX
	packet[3] = dstY
	packet[4] = width
	packet[5] = paddedHeight
	copy(packet[novaHeaderSize:], bitmapData)
	return packet
}

// padHeight rounds height up to the nearest multiple of 8.
func padHeight(height int) int {
	return (height + 7) &^ 7
}

// rowMajorMSBToColumnMajorLSB converts pixel data from row-major MSB format
// (as produced by bitmap.ImageToBytes) to column-major LSB format (as needed by Nova Pro).
//
// Input format (row-major MSB):
//   - Pixels stored row by row, left to right, top to bottom
//   - Each byte holds 8 horizontal pixels, MSB = leftmost pixel
//
// Output format (column-major LSB):
//   - Pixels stored column by column
//   - Each byte holds 8 vertical pixels (a "page"), LSB = topmost pixel
//   - Columns are stored sequentially: all pages of column 0, then column 1, etc.
func rowMajorMSBToColumnMajorLSB(pixelData []byte, srcX, stripWidth, totalWidth, height, paddedHeight int) []byte {
	bytesPerInputRow := totalWidth / 8
	pagesPerColumn := paddedHeight / 8
	output := make([]byte, stripWidth*pagesPerColumn)

	for col := 0; col < stripWidth; col++ {
		pixelX := srcX + col
		inputByteCol := pixelX / 8
		inputBitMask := byte(1 << uint(7-pixelX%8)) // MSB first

		for page := 0; page < pagesPerColumn; page++ {
			var outByte byte
			for bit := 0; bit < 8; bit++ {
				pixelY := page*8 + bit
				if pixelY >= height {
					break // Padding rows stay 0
				}
				inputByteIndex := pixelY*bytesPerInputRow + inputByteCol
				if inputByteIndex < len(pixelData) && pixelData[inputByteIndex]&inputBitMask != 0 {
					outByte |= 1 << uint(bit) // LSB first
				}
			}
			output[col*pagesPerColumn+page] = outByte
		}
	}

	return output
}
