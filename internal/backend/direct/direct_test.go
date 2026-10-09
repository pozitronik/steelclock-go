package direct

import (
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewBackend_InvalidVID(t *testing.T) {
	cfg := &config.Config{
		DirectDriver: &config.DirectDriverConfig{
			VID: "ZZZZ", // not valid hex
		},
		Display: config.DisplayConfig{Width: 128, Height: 40},
	}

	_, err := newBackend(cfg)
	if err == nil {
		t.Fatal("expected error for invalid VID hex")
	}
	if !strings.Contains(err.Error(), "invalid VID") {
		t.Errorf("error = %q, should mention 'invalid VID'", err.Error())
	}
}

func TestNewBackend_InvalidPID(t *testing.T) {
	cfg := &config.Config{
		DirectDriver: &config.DirectDriverConfig{
			VID: "1038", // valid hex
			PID: "GHIJ", // not valid hex
		},
		Display: config.DisplayConfig{Width: 128, Height: 40},
	}

	_, err := newBackend(cfg)
	if err == nil {
		t.Fatal("expected error for invalid PID hex")
	}
	if !strings.Contains(err.Error(), "invalid PID") {
		t.Errorf("error = %q, should mention 'invalid PID'", err.Error())
	}
}

func TestNewBackend_VIDOverflow(t *testing.T) {
	cfg := &config.Config{
		DirectDriver: &config.DirectDriverConfig{
			VID: "FFFFF", // exceeds uint16
		},
		Display: config.DisplayConfig{Width: 128, Height: 40},
	}

	_, err := newBackend(cfg)
	if err == nil {
		t.Fatal("expected error for VID overflow")
	}
}

func TestNewBackend_NoDirectDriverConfig(t *testing.T) {
	// When DirectDriver is nil, VID/PID default to 0 and we proceed to driver.NewClient
	// which will fail without hardware. This just verifies no panic on nil config.
	cfg := &config.Config{
		Display: config.DisplayConfig{Width: 128, Height: 40},
	}

	_, err := newBackend(cfg)
	// Expected to fail at driver.NewClient level (no hardware), not at config parsing
	if err == nil {
		t.Skip("driver.NewClient succeeded unexpectedly (hardware present?)")
	}
	// Should NOT be a VID/PID parsing error
	if strings.Contains(err.Error(), "invalid VID") || strings.Contains(err.Error(), "invalid PID") {
		t.Errorf("unexpected parsing error: %v", err)
	}
}

func TestDriverConfig(t *testing.T) {
	cfg := &config.Config{
		DirectDriver: &config.DirectDriverConfig{VID: "1038", PID: "1612", Interface: "mi_01"},
		Display:      config.DisplayConfig{Width: 128, Height: 40},
	}

	got, err := driverConfig(cfg)
	if err != nil {
		t.Fatalf("driverConfig() error = %v", err)
	}
	if got.VID != 0x1038 || got.PID != 0x1612 || got.Interface != "mi_01" || got.Width != 128 || got.Height != 40 {
		t.Errorf("driverConfig() = %+v", got)
	}
}

func TestProbe(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{"invalid VID is never available", &config.Config{
			DirectDriver: &config.DirectDriverConfig{VID: "ZZZZ"},
		}},
		{"device that is not connected", &config.Config{
			DirectDriver: &config.DirectDriverConfig{VID: "1038", PID: "FFFF", Interface: "mi_01"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if probe(tt.cfg) {
				t.Error("probe() = true, want false")
			}
		})
	}
}
