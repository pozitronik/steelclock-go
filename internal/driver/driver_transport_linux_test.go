//go:build linux

package driver

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// openRegularFileHandle opens a regular file as a stand-in device handle:
// write() succeeds on it, while the HIDIOCSFEATURE ioctl fails (ENOTTY). That
// tells the two transports apart without HID hardware.
func openRegularFileHandle(t *testing.T) (DeviceHandle, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hidraw")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Close(fd) })
	return DeviceHandle(fd), path
}

func TestSendPacket_Transport(t *testing.T) {
	tests := []struct {
		name        string
		protocol    Protocol
		wantErr     bool // feature-only transport: the ioctl fails on a regular file
		wantWritten bool // default transport: write() lands in the file
	}{
		{"legacy Apex uses write() first", &ApexProtocol{}, false, true},
		{"framebuffer Apex skips write()", NewApexFramebufferProtocol(), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, path := openRegularFileHandle(t)
			d := &HIDDriver{protocol: tt.protocol, handle: handle}

			packet := tt.protocol.BuildFramePackets(make([]byte, 640), 128, 40)[0]
			err := d.sendPacket(packet)
			if (err != nil) != tt.wantErr {
				t.Fatalf("sendPacket() error = %v, wantErr %v", err, tt.wantErr)
			}

			info, statErr := os.Stat(path)
			if statErr != nil {
				t.Fatal(statErr)
			}
			if written := info.Size() > 0; written != tt.wantWritten {
				t.Errorf("packet written via write() = %v, want %v", written, tt.wantWritten)
			}
		})
	}
}

func TestSetFeatureReport_InvalidInput(t *testing.T) {
	if err := setFeatureReport(InvalidHandle, []byte{0}); err == nil {
		t.Error("setFeatureReport(InvalidHandle) error = nil, want error")
	}
	handle, _ := openRegularFileHandle(t)
	if err := setFeatureReport(handle, nil); err == nil {
		t.Error("setFeatureReport(empty) error = nil, want error")
	}
}
