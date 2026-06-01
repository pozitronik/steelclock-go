package driver

import (
	"strings"
	"testing"
)

func TestFormatDeviceDiagnostic_KnownNovaPro(t *testing.T) {
	devices := []DeviceInfo{
		{VID: SteelSeriesVID, PID: 0x12cd, Interface: "mi_04", Path: `\\?\hid#vid_1038&pid_12cd&mi_04#x`},
		{VID: SteelSeriesVID, PID: 0x12cd, Interface: "mi_00", Path: `\\?\hid#vid_1038&pid_12cd&mi_00#x`},
		{VID: 0x046d, PID: 0xc52b, Interface: "mi_01", Path: `\\?\hid#vid_046d&pid_c52b&mi_01#x`}, // unrelated
	}

	report := FormatDeviceDiagnostic(devices)

	for _, want := range []string{
		"VID 0x1038): 2",      // two SteelSeries interfaces
		"mi_04",               // OLED interface listed
		"PID 0x12cd is KNOWN", // cross-referenced
		"Arctis Nova Pro Wireless (Base Station)", // name
		"128x64",                                // size
		"Nova Pro",                              // protocol family
		"1 other non-SteelSeries HID interface", // the Logitech device counted
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q\n---\n%s", want, report)
		}
	}
}

func TestFormatDeviceDiagnostic_UnknownPID(t *testing.T) {
	devices := []DeviceInfo{
		{VID: SteelSeriesVID, PID: 0x9999, Interface: "mi_04", Path: `\\?\hid#vid_1038&pid_9999&mi_04#x`},
	}

	report := FormatDeviceDiagnostic(devices)

	if !strings.Contains(report, "PID 0x9999 is NOT in SteelClock's known-device table") {
		t.Errorf("expected unknown-PID note, got:\n%s", report)
	}
}

func TestFormatDeviceDiagnostic_NoSteelSeries(t *testing.T) {
	devices := []DeviceInfo{
		{VID: 0x046d, PID: 0xc52b, Interface: "mi_01"},
	}

	report := FormatDeviceDiagnostic(devices)

	if !strings.Contains(report, "No SteelSeries (VID 0x1038) HID interfaces found") {
		t.Errorf("expected no-SteelSeries message, got:\n%s", report)
	}
	if !strings.Contains(report, "1 other non-SteelSeries HID interface") {
		t.Errorf("expected other-device count, got:\n%s", report)
	}
}
