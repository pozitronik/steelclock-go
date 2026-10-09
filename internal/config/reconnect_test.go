package config

import (
	"strings"
	"testing"
)

func TestApplyDefaults_ReconnectInterval(t *testing.T) {
	tests := []struct {
		name string
		set  int
		want int
	}{
		{"unset uses the default", 0, DefaultReconnectIntervalMs},
		{"explicit value is kept", 5000, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ReconnectIntervalMs: tt.set}
			applyDefaults(cfg)
			if cfg.ReconnectIntervalMs != tt.want {
				t.Errorf("ReconnectIntervalMs = %d, want %d", cfg.ReconnectIntervalMs, tt.want)
			}
		})
	}
}

func TestValidate_ReconnectInterval(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"unset", 0, false},
		{"minimum", MinReconnectIntervalMs, false},
		{"default", DefaultReconnectIntervalMs, false},
		{"maximum", MaxReconnectIntervalMs, false},
		{"below minimum", MinReconnectIntervalMs - 1, true},
		{"above maximum", MaxReconnectIntervalMs + 1, true},
		{"negative", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGlobalConfig(&Config{ReconnectIntervalMs: tt.value})
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateGlobalConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "reconnect_interval_ms") {
				t.Errorf("error %q should name reconnect_interval_ms", err)
			}
		})
	}
}
