//go:build linux
// +build linux

package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseUevent_ValidFormat(t *testing.T) {
	// Create temp file with valid uevent content
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	content := `HID_ID=0003:00001038:00001612
HID_NAME=SteelSeries Apex 7
HID_PHYS=usb-0000:00:14.0-2/input1
HID_UNIQ=
MODALIAS=hid:b0003g0001v00001038p00001612
`
	if err := os.WriteFile(ueventPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	vid, pid, hidName, err := parseUevent(ueventPath)
	if err != nil {
		t.Fatalf("parseUevent() error = %v", err)
	}

	if vid != 0x1038 {
		t.Errorf("vid = 0x%04X, want 0x1038", vid)
	}
	if pid != 0x1612 {
		t.Errorf("pid = 0x%04X, want 0x1612", pid)
	}
	if hidName != "SteelSeries Apex 7" {
		t.Errorf("hidName = %q, want %q", hidName, "SteelSeries Apex 7")
	}
}

func TestParseUevent_ApexPro(t *testing.T) {
	// Test with Apex Pro TKL format
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	content := `HID_ID=0003:00001038:00001610
HID_NAME=SteelSeries Apex Pro TKL
HID_PHYS=usb-0000:00:14.0-4/input1
HID_UNIQ=
MODALIAS=hid:b0003g0001v00001038p00001610
`
	if err := os.WriteFile(ueventPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	vid, pid, hidName, err := parseUevent(ueventPath)
	if err != nil {
		t.Fatalf("parseUevent() error = %v", err)
	}

	if vid != 0x1038 {
		t.Errorf("vid = 0x%04X, want 0x1038", vid)
	}
	if pid != 0x1610 {
		t.Errorf("pid = 0x%04X, want 0x1610", pid)
	}
	if hidName != "SteelSeries Apex Pro TKL" {
		t.Errorf("hidName = %q, want %q", hidName, "SteelSeries Apex Pro TKL")
	}
}

func TestParseUevent_NoVIDPID(t *testing.T) {
	// Create temp file with missing HID_ID
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	content := `HID_NAME=SteelSeries Apex 7
HID_PHYS=usb-0000:00:14.0-2/input1
`
	if err := os.WriteFile(ueventPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	_, _, _, err := parseUevent(ueventPath)
	if err == nil {
		t.Error("parseUevent() should return error when HID_ID is missing")
	}
}

func TestParseUevent_MalformedHIDID(t *testing.T) {
	// Create temp file with malformed HID_ID
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	content := `HID_ID=malformed
HID_NAME=Unknown Device
`
	if err := os.WriteFile(ueventPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	_, _, _, err := parseUevent(ueventPath)
	if err == nil {
		t.Error("parseUevent() should return error for malformed HID_ID")
	}
}

func TestParseUevent_InvalidHexValues(t *testing.T) {
	// Create temp file with invalid hex values
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	content := `HID_ID=0003:ZZZZZZZZ:YYYYYYYY
HID_NAME=Invalid Device
`
	if err := os.WriteFile(ueventPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	_, _, _, err := parseUevent(ueventPath)
	if err == nil {
		t.Error("parseUevent() should return error for invalid hex values")
	}
}

func TestParseUevent_FileNotFound(t *testing.T) {
	_, _, _, err := parseUevent("/nonexistent/path/uevent")
	if err == nil {
		t.Error("parseUevent() should return error for non-existent file")
	}
}

func TestParseUevent_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	if err := os.WriteFile(ueventPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	_, _, _, err := parseUevent(ueventPath)
	if err == nil {
		t.Error("parseUevent() should return error for empty file")
	}
}

func TestParseUevent_OnlyHIDID(t *testing.T) {
	// Create temp file with only HID_ID (no HID_NAME)
	tmpDir := t.TempDir()
	ueventPath := filepath.Join(tmpDir, "uevent")

	content := `HID_ID=0003:00001038:00001612
`
	if err := os.WriteFile(ueventPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write uevent file: %v", err)
	}

	vid, pid, hidName, err := parseUevent(ueventPath)
	if err != nil {
		t.Fatalf("parseUevent() error = %v", err)
	}

	if vid != 0x1038 {
		t.Errorf("vid = 0x%04X, want 0x1038", vid)
	}
	if pid != 0x1612 {
		t.Errorf("pid = 0x%04X, want 0x1612", pid)
	}
	if hidName != "" {
		t.Errorf("hidName = %q, want empty", hidName)
	}
}

func TestGetInterfaceFromPath_Invalid(t *testing.T) {
	// Test with non-existent path
	result := getInterfaceFromPath("/nonexistent/path")
	if result != "" {
		t.Errorf("getInterfaceFromPath() = %q, want empty for non-existent path", result)
	}
}

func TestUSBInterfaceRegex(t *testing.T) {
	// Test the USB interface regex pattern directly
	// Pattern: \d+-\d+:\d+\.(\d+) matches bus-port:config.interface
	testCases := []struct {
		path    string
		wantNum string
		wantOK  bool
	}{
		{"3-2:1.0", "0", true},
		{"3-2:1.1", "1", true},
		{"4-1:2.3", "3", true},
		{"/sys/devices/3-2:1.1/hidraw/hidraw0", "1", true},
		{"invalid", "", false},
		{"", "", false},
		{"no-match-here", "", false},
	}

	for _, tc := range testCases {
		matches := usbInterfaceRegex.FindStringSubmatch(tc.path)
		if tc.wantOK {
			if len(matches) < 2 {
				t.Errorf("regex failed to match %q", tc.path)
				continue
			}
			if matches[1] != tc.wantNum {
				t.Errorf("regex on %q: got %q, want %q", tc.path, matches[1], tc.wantNum)
			}
		} else {
			if len(matches) >= 2 {
				t.Errorf("regex should not match %q, but got %v", tc.path, matches)
			}
		}
	}
}
