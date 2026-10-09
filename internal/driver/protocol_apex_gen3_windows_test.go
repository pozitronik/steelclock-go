//go:build windows

package driver

import (
	"testing"
)

// Windows packet format: ReportID(1) + CMD(2: 1F 81) + Data(640) = 643 bytes.
// The device's true feature-report length (645, discovered via HID capabilities
// at Open time) is padded on by the generic driver-level logic, not here.
const (
	gen3TestPacketSize = 643
	gen3TestDataOffset = 3
)

func TestBuildApexGen3Packet_Size(t *testing.T) {
	pixelData := make([]byte, 640)

	packet := buildApexGen3Packet(pixelData, 128, 40)

	if len(packet) != gen3TestPacketSize {
		t.Errorf("buildApexGen3Packet() size = %d, want %d", len(packet), gen3TestPacketSize)
	}
}

func TestBuildApexGen3Packet_ReportID(t *testing.T) {
	packet := buildApexGen3Packet(make([]byte, 640), 128, 40)

	if packet[0] != 0x00 {
		t.Errorf("packet[0] (ReportID) = 0x%02X, want 0x00", packet[0])
	}
}

func TestBuildApexGen3Packet_Command(t *testing.T) {
	packet := buildApexGen3Packet(make([]byte, 640), 128, 40)

	if packet[1] != 0x1F || packet[2] != 0x81 {
		t.Errorf("packet[1:3] (CMD) = 0x%02X 0x%02X, want 0x1F 0x81", packet[1], packet[2])
	}
}

func TestBuildApexGen3Packet_DataCopy(t *testing.T) {
	pixelData := make([]byte, 640)
	for i := range pixelData {
		pixelData[i] = byte(i % 256)
	}

	packet := buildApexGen3Packet(pixelData, 128, 40)

	for i := 0; i < len(pixelData); i++ {
		if packet[gen3TestDataOffset+i] != pixelData[i] {
			t.Errorf("packet[%d] = 0x%02X, want 0x%02X", gen3TestDataOffset+i, packet[gen3TestDataOffset+i], pixelData[i])
			break
		}
	}
}

func TestBuildApexGen3Packet_ShortData(t *testing.T) {
	pixelData := make([]byte, 100)
	for i := range pixelData {
		pixelData[i] = 0xFF
	}

	packet := buildApexGen3Packet(pixelData, 128, 40)

	for i := 0; i < 100; i++ {
		if packet[gen3TestDataOffset+i] != 0xFF {
			t.Errorf("packet[%d] = 0x%02X, want 0xFF", gen3TestDataOffset+i, packet[gen3TestDataOffset+i])
			break
		}
	}

	for i := 100; i < 640; i++ {
		if packet[gen3TestDataOffset+i] != 0x00 {
			t.Errorf("packet[%d] = 0x%02X, want 0x00 (padding)", gen3TestDataOffset+i, packet[gen3TestDataOffset+i])
			break
		}
	}
}

func TestBuildApexGen3Packet_LongData(t *testing.T) {
	pixelData := make([]byte, 1000)
	for i := range pixelData {
		pixelData[i] = 0xAA
	}

	packet := buildApexGen3Packet(pixelData, 128, 40)

	if len(packet) != gen3TestPacketSize {
		t.Errorf("buildApexGen3Packet() size with long data = %d, want %d", len(packet), gen3TestPacketSize)
	}

	for i := 0; i < 640; i++ {
		if packet[gen3TestDataOffset+i] != 0xAA {
			t.Errorf("packet[%d] = 0x%02X, want 0xAA", gen3TestDataOffset+i, packet[gen3TestDataOffset+i])
			break
		}
	}
}

func TestApexGen3Protocol_Interface(t *testing.T) {
	p := &ApexGen3Protocol{}
	if p.Interface() != "mi_01" {
		t.Errorf("Interface() = %q, want %q", p.Interface(), "mi_01")
	}
}

func TestApexGen3Protocol_ImplementsProtocol(t *testing.T) {
	var _ Protocol = (*ApexGen3Protocol)(nil)
}

func TestApexGen3Protocol_PadToReportLength(t *testing.T) {
	p := &ApexGen3Protocol{}
	if !protocolWantsReportLengthPadding(p) {
		t.Error("ApexGen3Protocol should opt into report-length padding")
	}
}

func TestProtocolWantsReportLengthPadding_DefaultFalse(t *testing.T) {
	// A protocol that does not implement ReportLengthPadding (e.g. the
	// classic Apex family) must not be padded.
	if protocolWantsReportLengthPadding(&ApexProtocol{}) {
		t.Error("ApexProtocol should not opt into report-length padding")
	}
}

func TestResolveProtocol_ApexProTklGen3(t *testing.T) {
	p := resolveProtocol(SteelSeriesVID, 0x1628)
	if _, ok := p.(*ApexGen3Protocol); !ok {
		t.Errorf("resolveProtocol for Apex Pro TKL Gen 3 should return *ApexGen3Protocol, got %T", p)
	}
}
