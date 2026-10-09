package gpu

import "testing"

// TestWidget_UsageText verifies the values the widget hands to the token
// formatter: used GB derived from the percentage and the adapter total.
func TestWidget_UsageText(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		totalGB float64
		percent float64
		want    string
	}{
		{"used and percent", "V {used}GB {percent}%", 16, 50, "V 8.0GB 50%"},
		{"used and total not swapped", "{used}/{total}", 12, 25, "3.0/12.0"},
		{"non-memory metric has no total", "{used}/{total} {percent}%", 0, 75, "0.0/0.0 75%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Widget{totalMemoryGB: tt.totalGB, currentValue: tt.percent}

			if got := w.usageText(tt.format); got != tt.want {
				t.Errorf("usageText(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}
