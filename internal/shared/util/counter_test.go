package util

import "testing"

func TestCounterRate(t *testing.T) {
	tests := []struct {
		name     string
		current  uint64
		previous uint64
		elapsed  float64
		want     float64
	}{
		{"growth over one second", 1500, 500, 1, 1000},
		{"growth over half a second", 1500, 1000, 0.5, 1000},
		{"unchanged", 500, 500, 1, 0},
		{"counter reset", 50, 100, 1, 0},
		{"reset to zero", 0, 1 << 40, 1, 0},
		{"zero elapsed", 1500, 500, 0, 0},
		{"negative elapsed", 1500, 500, -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CounterRate(tt.current, tt.previous, tt.elapsed); got != tt.want {
				t.Errorf("CounterRate(%d, %d, %g) = %g, want %g", tt.current, tt.previous, tt.elapsed, got, tt.want)
			}
		})
	}
}
