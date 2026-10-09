package memory

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestWidget_UsageText verifies the values the widget hands to the token
// formatter: used and total GB in their own places, and the percentage.
func TestWidget_UsageText(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		usedGB  float64
		totalGB float64
		percent float64
		want    string
	}{
		{"used and percent", "R {used}GB {percent}%", 4, 16, 25, "R 4.0GB 25%"},
		{"used and total not swapped", "{used}/{total}", 6.5, 32, 20.3, "6.5/32.0"},
		{"percent only", "{percent}%", 1, 2, 50, "50%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(config.WidgetConfig{
				Type:     "memory",
				ID:       "test_memory_tokens",
				Enabled:  config.BoolPtr(true),
				Position: config.PositionConfig{W: 128, H: 20},
				Mode:     "text",
				Text:     &config.TextConfig{Format: tt.format},
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			w.usedGB, w.totalGB, w.currentValue = tt.usedGB, tt.totalGB, tt.percent

			if got := w.usageText(); got != tt.want {
				t.Errorf("usageText() = %q, want %q", got, tt.want)
			}
		})
	}
}
