package driver

import (
	"bytes"
	"errors"
	"testing"
)

func TestFitOutputReport(t *testing.T) {
	packet := []byte{0x00, 0x1F, 0x82, 0x00, 0x00}

	tests := []struct {
		name      string
		length    int
		known     bool
		want      []byte
		wantErr   bool
		wantErrIs error // checked with errors.Is when set
	}{
		{"unknown length keeps the packet", 0, false, packet, false, nil},
		{"exact length", 5, true, packet, false, nil},
		{"longer length pads with zeros", 7, true, []byte{0x00, 0x1F, 0x82, 0, 0, 0, 0}, false, nil},
		{"shorter length drops zero padding", 3, true, []byte{0x00, 0x1F, 0x82}, false, nil},
		{"shorter length cutting data is refused", 2, true, nil, true, nil},
		{"no output report", 0, true, nil, true, errNoOutputReport},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fitOutputReport(packet, tt.length, tt.known)
			if (err != nil) != tt.wantErr {
				t.Fatalf("fitOutputReport() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("fitOutputReport() error = %v, want %v", err, tt.wantErrIs)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("fitOutputReport() = % X, want % X", got, tt.want)
			}
		})
	}
}
