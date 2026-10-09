package render

import "testing"

func TestIsUsageTokenFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   bool
	}{
		{"token format", "R {used}GB {percent}%", true},
		{"single token", "{percent}", true},
		{"unknown token", "{foo}", true},
		{"printf format", "%.0f%%", false},
		{"printf with label", "RAM %.1f%%", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUsageTokenFormat(tt.format); got != tt.want {
				t.Errorf("IsUsageTokenFormat(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

func TestFormatUsageTokens(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		usedGB  float64
		totalGB float64
		percent float64
		want    string
	}{
		{"used and percent", "R {used}GB {percent}%", 10.44, 16, 65.2, "R 10.4GB 65%"},
		{"all tokens", "{used}/{total}GB ({percent}%)", 8, 32, 25, "8.0/32.0GB (25%)"},
		{"repeated token", "{percent}% {percent}%", 1, 2, 50, "50% 50%"},
		{"rounding", "{used} {total} {percent}", 0.05, 15.96, 99.5, "0.1 16.0 100"},
		{"zero values", "V {used}GB {percent}%", 0, 0, 0, "V 0.0GB 0%"},
		{"unknown token kept", "{used} {foo}", 1.5, 2, 3, "1.5 {foo}"},
		{"no tokens", "RAM", 1, 2, 3, "RAM"},
		{"empty format", "", 1, 2, 3, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatUsageTokens(tt.format, tt.usedGB, tt.totalGB, tt.percent)
			if got != tt.want {
				t.Errorf("FormatUsageTokens(%q, %v, %v, %v) = %q, want %q",
					tt.format, tt.usedGB, tt.totalGB, tt.percent, got, tt.want)
			}
		})
	}
}
