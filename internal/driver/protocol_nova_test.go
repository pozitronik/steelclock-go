package driver

import (
	"testing"
)

func TestNovaProProtocol_Interface(t *testing.T) {
	p := &NovaProProtocol{}
	if p.Interface() != "mi_04" {
		t.Errorf("Interface() = %q, want %q", p.Interface(), "mi_04")
	}
}

func TestNovaProProtocol_DeviceFamily(t *testing.T) {
	p := &NovaProProtocol{}
	if p.DeviceFamily() != "Nova Pro" {
		t.Errorf("DeviceFamily() = %q, want %q", p.DeviceFamily(), "Nova Pro")
	}
}

func TestNovaProProtocol_ImplementsProtocol(t *testing.T) {
	var _ Protocol = (*NovaProProtocol)(nil)
}

func TestNovaProProtocol_BuildFramePackets_128x64(t *testing.T) {
	p := &NovaProProtocol{}
	pixelData := make([]byte, 128*64/8) // 1024 bytes

	packets := p.BuildFramePackets(pixelData, 128, 64)

	// 128px / 64px max strip = 2 packets
	if len(packets) != 2 {
		t.Fatalf("BuildFramePackets() returned %d packets, want 2", len(packets))
	}

	// Each packet is 1024 bytes
	for i, pkt := range packets {
		if len(pkt) != novaReportSize {
			t.Errorf("packet[%d] size = %d, want %d", i, len(pkt), novaReportSize)
		}
	}

	// First strip: dstX=0, width=64
	if packets[0][0] != novaReportID {
		t.Errorf("packet[0] report ID = 0x%02X, want 0x%02X", packets[0][0], novaReportID)
	}
	if packets[0][1] != novaScreenCommand {
		t.Errorf("packet[0] command = 0x%02X, want 0x%02X", packets[0][1], novaScreenCommand)
	}
	if packets[0][2] != 0 {
		t.Errorf("packet[0] dstX = %d, want 0", packets[0][2])
	}
	if packets[0][3] != 0 {
		t.Errorf("packet[0] dstY = %d, want 0", packets[0][3])
	}
	if packets[0][4] != 64 {
		t.Errorf("packet[0] width = %d, want 64", packets[0][4])
	}
	if packets[0][5] != 64 {
		t.Errorf("packet[0] paddedHeight = %d, want 64", packets[0][5])
	}

	// Second strip: dstX=64, width=64
	if packets[1][2] != 64 {
		t.Errorf("packet[1] dstX = %d, want 64", packets[1][2])
	}
	if packets[1][4] != 64 {
		t.Errorf("packet[1] width = %d, want 64", packets[1][4])
	}
}

func TestNovaProProtocol_BuildFramePackets_128x52(t *testing.T) {
	p := &NovaProProtocol{}
	pixelData := make([]byte, 128*52/8) // 832 bytes

	packets := p.BuildFramePackets(pixelData, 128, 52)

	if len(packets) != 2 {
		t.Fatalf("BuildFramePackets() returned %d packets, want 2", len(packets))
	}

	// Padded height should be 56 (52 rounded up to multiple of 8)
	if packets[0][5] != 56 {
		t.Errorf("paddedHeight = %d, want 56", packets[0][5])
	}
}

func TestNovaProProtocol_BuildFramePackets_64x64(t *testing.T) {
	p := &NovaProProtocol{}
	pixelData := make([]byte, 64*64/8) // 512 bytes

	packets := p.BuildFramePackets(pixelData, 64, 64)

	// 64px fits in a single strip
	if len(packets) != 1 {
		t.Fatalf("BuildFramePackets() returned %d packets, want 1", len(packets))
	}

	if packets[0][2] != 0 {
		t.Errorf("dstX = %d, want 0", packets[0][2])
	}
	if packets[0][4] != 64 {
		t.Errorf("width = %d, want 64", packets[0][4])
	}
}

func TestNovaProProtocol_BuildFramePackets_100x64(t *testing.T) {
	p := &NovaProProtocol{}
	pixelData := make([]byte, 100*64/8)

	packets := p.BuildFramePackets(pixelData, 100, 64)

	// 100px = strip of 64 + strip of 36
	if len(packets) != 2 {
		t.Fatalf("BuildFramePackets() returned %d packets, want 2", len(packets))
	}

	if packets[0][4] != 64 {
		t.Errorf("strip 0 width = %d, want 64", packets[0][4])
	}
	if packets[1][2] != 64 {
		t.Errorf("strip 1 dstX = %d, want 64", packets[1][2])
	}
	if packets[1][4] != 36 {
		t.Errorf("strip 1 width = %d, want 36", packets[1][4])
	}
}

func TestBuildNovaPacket_Size(t *testing.T) {
	packet := buildNovaPacket(0, 0, 64, 64, make([]byte, 512))

	if len(packet) != novaReportSize {
		t.Errorf("packet size = %d, want %d", len(packet), novaReportSize)
	}
}

func TestBuildNovaPacket_Header(t *testing.T) {
	tests := []struct {
		name                             string
		dstX, dstY, width, paddedHeight  byte
		wantReportID, wantCmd            byte
		wantDstX, wantDstY, wantW, wantH byte
	}{
		{
			name: "first strip 128x64",
			dstX: 0, dstY: 0, width: 64, paddedHeight: 64,
			wantReportID: 0x06, wantCmd: 0x93,
			wantDstX: 0, wantDstY: 0, wantW: 64, wantH: 64,
		},
		{
			name: "second strip 128x64",
			dstX: 64, dstY: 0, width: 64, paddedHeight: 64,
			wantReportID: 0x06, wantCmd: 0x93,
			wantDstX: 64, wantDstY: 0, wantW: 64, wantH: 64,
		},
		{
			name: "128x52 padded",
			dstX: 0, dstY: 0, width: 64, paddedHeight: 56,
			wantReportID: 0x06, wantCmd: 0x93,
			wantDstX: 0, wantDstY: 0, wantW: 64, wantH: 56,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := buildNovaPacket(tt.dstX, tt.dstY, tt.width, tt.paddedHeight, nil)

			if packet[0] != tt.wantReportID {
				t.Errorf("report ID = 0x%02X, want 0x%02X", packet[0], tt.wantReportID)
			}
			if packet[1] != tt.wantCmd {
				t.Errorf("command = 0x%02X, want 0x%02X", packet[1], tt.wantCmd)
			}
			if packet[2] != tt.wantDstX {
				t.Errorf("dstX = %d, want %d", packet[2], tt.wantDstX)
			}
			if packet[3] != tt.wantDstY {
				t.Errorf("dstY = %d, want %d", packet[3], tt.wantDstY)
			}
			if packet[4] != tt.wantW {
				t.Errorf("width = %d, want %d", packet[4], tt.wantW)
			}
			if packet[5] != tt.wantH {
				t.Errorf("paddedHeight = %d, want %d", packet[5], tt.wantH)
			}
		})
	}
}

func TestBuildNovaPacket_BitmapCopy(t *testing.T) {
	bitmap := make([]byte, 512)
	for i := range bitmap {
		bitmap[i] = byte(i % 256)
	}

	packet := buildNovaPacket(0, 0, 64, 64, bitmap)

	// Bitmap data starts at offset 6
	for i := 0; i < len(bitmap); i++ {
		if packet[novaHeaderSize+i] != bitmap[i] {
			t.Errorf("packet[%d] = 0x%02X, want 0x%02X", novaHeaderSize+i, packet[novaHeaderSize+i], bitmap[i])
			break
		}
	}

	// Rest of packet should be zero-padded
	for i := novaHeaderSize + len(bitmap); i < novaReportSize; i++ {
		if packet[i] != 0x00 {
			t.Errorf("packet[%d] = 0x%02X, want 0x00 (padding)", i, packet[i])
			break
		}
	}
}

func TestPadHeight(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{64, 64},
		{52, 56},
		{40, 40},
		{48, 48},
		{1, 8},
		{8, 8},
		{9, 16},
		{0, 0},
		{33, 40},
	}

	for _, tt := range tests {
		got := padHeight(tt.height)
		if got != tt.want {
			t.Errorf("padHeight(%d) = %d, want %d", tt.height, got, tt.want)
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_SinglePixel tests conversion of a single set pixel.
// Input: 8x8 display, pixel at (2, 3) is set.
// Row-major MSB: row 3, byte 0, bit 5 → pixelData[3] = 0x20
// Column-major LSB: column 2, page 0, bit 3 → output[2] = 0x08
func TestRowMajorMSBToColumnMajorLSB_SinglePixel(t *testing.T) {
	pixelData := make([]byte, 8) // 8x8 / 8 bits = 8 bytes (1 byte per row)
	pixelData[3] = 0x20          // Pixel at (2, 3): row 3, bit 5 (MSB first: bit 7-x)

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 8, 8, 8, 8)

	// Column 2, page 0: bit 3 should be set
	expected := byte(0x08)
	if output[2] != expected {
		t.Errorf("output[2] = 0x%02X, want 0x%02X", output[2], expected)
	}

	// All other bytes should be 0
	for i, b := range output {
		if i == 2 {
			continue
		}
		if b != 0 {
			t.Errorf("output[%d] = 0x%02X, want 0x00", i, b)
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_TopLeftPixel verifies the top-left pixel encoding.
// Pixel (0,0): row-major MSB → byte 0, bit 7 → pixelData[0] = 0x80
// Column-major LSB → column 0, page 0, bit 0 → output[0] = 0x01
func TestRowMajorMSBToColumnMajorLSB_TopLeftPixel(t *testing.T) {
	pixelData := make([]byte, 8) // 8x8
	pixelData[0] = 0x80          // Pixel (0, 0)

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 8, 8, 8, 8)

	if output[0] != 0x01 {
		t.Errorf("output[0] = 0x%02X, want 0x01", output[0])
	}
}

// TestRowMajorMSBToColumnMajorLSB_FullRow tests a full row of set pixels.
// All pixels in row 0 are set: pixelData[0] = 0xFF (for 8px wide)
// Each column's page 0 should have bit 0 set: 0x01
func TestRowMajorMSBToColumnMajorLSB_FullRow(t *testing.T) {
	pixelData := make([]byte, 8) // 8x8
	pixelData[0] = 0xFF          // All pixels in row 0

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 8, 8, 8, 8)

	for col := 0; col < 8; col++ {
		if output[col] != 0x01 {
			t.Errorf("output[%d] = 0x%02X, want 0x01", col, output[col])
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_FullColumn tests a full column of set pixels.
// All pixels in column 0 are set: bit 7 of each row's first byte.
// Column 0, page 0 should have all bits set: 0xFF
func TestRowMajorMSBToColumnMajorLSB_FullColumn(t *testing.T) {
	pixelData := make([]byte, 8) // 8x8
	for row := 0; row < 8; row++ {
		pixelData[row] = 0x80 // Column 0 pixel in each row (bit 7, MSB first)
	}

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 8, 8, 8, 8)

	// Column 0, page 0: all 8 vertical pixels set
	if output[0] != 0xFF {
		t.Errorf("output[0] = 0x%02X, want 0xFF", output[0])
	}

	// Other columns should be 0
	for col := 1; col < 8; col++ {
		if output[col] != 0x00 {
			t.Errorf("output[%d] = 0x%02X, want 0x00", col, output[col])
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_16Wide tests strip extraction from a wider display.
// 16x8 display, extract strip at srcX=8, width=8.
func TestRowMajorMSBToColumnMajorLSB_16Wide(t *testing.T) {
	pixelData := make([]byte, 16) // 16x8: 2 bytes per row, 8 rows
	// Set pixel (8, 0): second byte of row 0, bit 7
	pixelData[1] = 0x80

	output := rowMajorMSBToColumnMajorLSB(pixelData, 8, 8, 16, 8, 8)

	// Column 0 of the strip (pixel x=8), page 0, bit 0
	if output[0] != 0x01 {
		t.Errorf("output[0] = 0x%02X, want 0x01", output[0])
	}
}

// TestRowMajorMSBToColumnMajorLSB_PaddedHeight tests height padding.
// 8x5 display, paddedHeight=8. Rows 5-7 should be zero in output.
func TestRowMajorMSBToColumnMajorLSB_PaddedHeight(t *testing.T) {
	pixelData := make([]byte, 5) // 8x5: 1 byte per row, 5 rows
	// Set all pixels in all 5 rows
	for i := range pixelData {
		pixelData[i] = 0xFF
	}

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 8, 8, 5, 8)

	// Each column should have bits 0-4 set, bits 5-7 clear
	expected := byte(0x1F) // 0b00011111
	for col := 0; col < 8; col++ {
		if output[col] != expected {
			t.Errorf("output[%d] = 0x%02X, want 0x%02X", col, output[col], expected)
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_MultiplePages tests encoding with multiple pages.
// 8x16 display (2 pages): pixel at (0, 8) should be in page 1.
func TestRowMajorMSBToColumnMajorLSB_MultiplePages(t *testing.T) {
	pixelData := make([]byte, 16) // 8x16: 1 byte per row, 16 rows
	pixelData[8] = 0x80           // Pixel (0, 8): row 8, bit 7

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 8, 8, 16, 16)

	// Column 0, page 1 (index: col*pagesPerColumn + page = 0*2 + 1 = 1): bit 0 set
	if output[1] != 0x01 {
		t.Errorf("output[1] = 0x%02X, want 0x01", output[1])
	}

	// Column 0, page 0 should be 0
	if output[0] != 0x00 {
		t.Errorf("output[0] = 0x%02X, want 0x00", output[0])
	}
}

// TestRowMajorMSBToColumnMajorLSB_OutputSize verifies output buffer size.
func TestRowMajorMSBToColumnMajorLSB_OutputSize(t *testing.T) {
	tests := []struct {
		name         string
		stripWidth   int
		paddedHeight int
		wantSize     int
	}{
		{"64x64", 64, 64, 64 * 8},
		{"64x56", 64, 56, 64 * 7},
		{"64x40", 64, 40, 64 * 5},
		{"36x64", 36, 64, 36 * 8},
		{"8x8", 8, 8, 8 * 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalWidth := tt.stripWidth
			if totalWidth < 128 {
				totalWidth = 128
			}
			pixelData := make([]byte, totalWidth*tt.paddedHeight/8)

			output := rowMajorMSBToColumnMajorLSB(pixelData, 0, tt.stripWidth, totalWidth, tt.paddedHeight, tt.paddedHeight)

			if len(output) != tt.wantSize {
				t.Errorf("output size = %d, want %d", len(output), tt.wantSize)
			}
		})
	}
}

// TestRowMajorMSBToColumnMajorLSB_EmptyInput tests with all-zero input.
func TestRowMajorMSBToColumnMajorLSB_EmptyInput(t *testing.T) {
	pixelData := make([]byte, 128*64/8)

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 64, 128, 64, 64)

	for i, b := range output {
		if b != 0 {
			t.Errorf("output[%d] = 0x%02X, want 0x00 for empty input", i, b)
			break
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_AllSet tests with all pixels set.
func TestRowMajorMSBToColumnMajorLSB_AllSet(t *testing.T) {
	pixelData := make([]byte, 128*64/8)
	for i := range pixelData {
		pixelData[i] = 0xFF
	}

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 64, 128, 64, 64)

	for i, b := range output {
		if b != 0xFF {
			t.Errorf("output[%d] = 0x%02X, want 0xFF for all-set input", i, b)
			break
		}
	}
}

// TestRowMajorMSBToColumnMajorLSB_ShortInput tests graceful handling of short input data.
func TestRowMajorMSBToColumnMajorLSB_ShortInput(t *testing.T) {
	// Pixel data is shorter than expected — should not panic
	pixelData := make([]byte, 10) // Much shorter than 128x64/8
	for i := range pixelData {
		pixelData[i] = 0xFF
	}

	output := rowMajorMSBToColumnMajorLSB(pixelData, 0, 64, 128, 64, 64)

	// Should produce output without panicking
	if len(output) != 64*8 {
		t.Errorf("output size = %d, want %d", len(output), 64*8)
	}
}

func TestResolveProtocol_NovaProDevice(t *testing.T) {
	// Arctis Nova Pro Wired
	p := resolveProtocol(SteelSeriesVID, 0x12cb)
	if _, ok := p.(*NovaProProtocol); !ok {
		t.Errorf("resolveProtocol for Nova Pro Wired should return *NovaProProtocol, got %T", p)
	}
}

func TestResolveProtocol_AllNovaProDevices(t *testing.T) {
	novaProPIDs := []uint16{0x12cb, 0x12cd, 0x12e0, 0x12e5, 0x225d}

	for _, pid := range novaProPIDs {
		p := resolveProtocol(SteelSeriesVID, pid)
		if _, ok := p.(*NovaProProtocol); !ok {
			t.Errorf("resolveProtocol(0x%04X, 0x%04X) should return *NovaProProtocol, got %T",
				SteelSeriesVID, pid, p)
		}
	}
}

func BenchmarkBuildNovaPacket(b *testing.B) {
	bitmap := make([]byte, 512) // 64 * 64/8

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildNovaPacket(0, 0, 64, 64, bitmap)
	}
}

func BenchmarkRowMajorMSBToColumnMajorLSB_128x64(b *testing.B) {
	pixelData := make([]byte, 128*64/8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rowMajorMSBToColumnMajorLSB(pixelData, 0, 64, 128, 64, 64)
	}
}

func BenchmarkNovaProProtocol_BuildFramePackets_128x64(b *testing.B) {
	p := &NovaProProtocol{}
	pixelData := make([]byte, 128*64/8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.BuildFramePackets(pixelData, 128, 64)
	}
}
