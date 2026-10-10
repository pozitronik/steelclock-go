//go:build linux

package driver

import (
	"errors"
	"os"
	"testing"
)

// returnToUIPacket is the framebuffer variant's return-to-UI output report,
// zero-padded or trimmed to n bytes.
func returnToUIPacket(n int) []byte {
	packet := make([]byte, n)
	copy(packet, []byte{0x00, 0x1F, 0x82})
	return packet
}

func TestClient_ReturnToUI_Transport(t *testing.T) {
	tests := []struct {
		name        string
		protocol    Protocol
		outputLen   int
		lenKnown    bool
		wantErr     bool
		wantWritten []byte // nil: nothing may be written
	}{
		{"legacy Apex sends nothing", &ApexProtocol{}, 0, false, false, nil},
		{"unknown length sends the packet as built", NewApexFramebufferProtocol(), 0, false, false, returnToUIPacket(65)},
		{"declared length matches", NewApexFramebufferProtocol(), 65, true, false, returnToUIPacket(65)},
		{"longer declared length is zero-padded", NewApexFramebufferProtocol(), 70, true, false, returnToUIPacket(70)},
		{"shorter declared length drops zero padding", NewApexFramebufferProtocol(), 40, true, false, returnToUIPacket(40)},
		{"no declared output report sends nothing", NewApexFramebufferProtocol(), 0, true, true, nil},
		{"declared length cutting the command sends nothing", NewApexFramebufferProtocol(), 2, true, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, path := openRegularFileHandle(t)
			d := &HIDDriver{
				protocol:             tt.protocol,
				handle:               handle,
				connected:            true,
				outputReportLen:      tt.outputLen,
				outputReportLenKnown: tt.lenKnown,
			}
			c := &Client{driver: d}

			err := c.ReturnToUI()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReturnToUI() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !d.IsConnected() {
				t.Error("a packet that was never sent must not mark the device disconnected")
			}

			written, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(written) != string(tt.wantWritten) {
				t.Errorf("written = % X, want % X", written, tt.wantWritten)
			}
		})
	}
}

func TestClient_ReturnToUI_NoOutputReportError(t *testing.T) {
	handle, _ := openRegularFileHandle(t)
	c := &Client{driver: &HIDDriver{
		protocol:             NewApexFramebufferProtocol(),
		handle:               handle,
		connected:            true,
		outputReportLenKnown: true,
	}}
	if err := c.ReturnToUI(); !errors.Is(err, errNoOutputReport) {
		t.Errorf("ReturnToUI() error = %v, want errNoOutputReport", err)
	}
}

func TestClient_ReturnToUI_Disconnected(t *testing.T) {
	c := &Client{driver: &HIDDriver{protocol: NewApexFramebufferProtocol(), handle: InvalidHandle}}
	if err := c.ReturnToUI(); err == nil {
		t.Error("ReturnToUI() on a disconnected device error = nil, want error")
	}
}

func TestSendOutputReport_InvalidInput(t *testing.T) {
	if err := sendOutputReport(InvalidHandle, []byte{0}); err == nil {
		t.Error("sendOutputReport(InvalidHandle) error = nil, want error")
	}
	handle, _ := openRegularFileHandle(t)
	if err := sendOutputReport(handle, nil); err == nil {
		t.Error("sendOutputReport(empty) error = nil, want error")
	}
}
