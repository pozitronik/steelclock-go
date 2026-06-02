package driver

import "testing"

func TestHexList(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"empty", nil, "(none)"},
		{"single", []byte{0x06}, "0x06"},
		{"multiple", []byte{0x00, 0x06, 0xFF}, "0x00 0x06 0xFF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hexList(tt.in); got != tt.want {
				t.Errorf("hexList(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
