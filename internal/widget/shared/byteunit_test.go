package shared

import (
	"math"
	"testing"
)

func TestDetermineUnitFamily(t *testing.T) {
	tests := []struct {
		unitName string
		expected UnitFamily
	}{
		{"B/s", UnitFamilyBytesDecimal},
		{"KB/s", UnitFamilyBytesDecimal},
		{"MB/s", UnitFamilyBytesDecimal},
		{"GB/s", UnitFamilyBytesDecimal},
		{"KiB/s", UnitFamilyBytesBinary},
		{"MiB/s", UnitFamilyBytesBinary},
		{"GiB/s", UnitFamilyBytesBinary},
		{"bps", UnitFamilyBits},
		{"Kbps", UnitFamilyBits},
		{"Mbps", UnitFamilyBits},
		{"Gbps", UnitFamilyBits},
		{"unknown", UnitFamilyBytesDecimal}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.unitName, func(t *testing.T) {
			got := DetermineUnitFamily(tt.unitName)
			if got != tt.expected {
				t.Errorf("DetermineUnitFamily(%q) = %v, want %v", tt.unitName, got, tt.expected)
			}
		})
	}
}

func TestIsValidUnit(t *testing.T) {
	validUnits := []string{"B/s", "KB/s", "MB/s", "GB/s", "KiB/s", "MiB/s", "GiB/s", "bps", "Kbps", "Mbps", "Gbps"}
	invalidUnits := []string{"bytes", "megabytes", "auto", "", "invalid"}

	for _, unit := range validUnits {
		if !IsValidUnit(unit) {
			t.Errorf("IsValidUnit(%q) = false, want true", unit)
		}
	}

	for _, unit := range invalidUnits {
		if IsValidUnit(unit) {
			t.Errorf("IsValidUnit(%q) = true, want false", unit)
		}
	}
}

func TestNewByteRateConverter(t *testing.T) {
	tests := []struct {
		defaultUnit    string
		expectedFamily UnitFamily
	}{
		{"MB/s", UnitFamilyBytesDecimal},
		{"MiB/s", UnitFamilyBytesBinary},
		{"Mbps", UnitFamilyBits},
	}

	for _, tt := range tests {
		t.Run(tt.defaultUnit, func(t *testing.T) {
			c := NewByteRateConverter(tt.defaultUnit)
			if c.GetFamily() != tt.expectedFamily {
				t.Errorf("NewByteRateConverter(%q).GetFamily() = %v, want %v",
					tt.defaultUnit, c.GetFamily(), tt.expectedFamily)
			}
		})
	}
}

func TestByteRateConverter_Convert(t *testing.T) {
	c := NewByteRateConverter("MB/s")

	tests := []struct {
		bps      float64
		unitName string
		wantVal  float64
		wantUnit string
	}{
		{1000000, "MB/s", 1.0, "MB/s"},
		{1048576, "MiB/s", 1.0, "MiB/s"},
		{125000, "Mbps", 1.0, "Mbps"}, // 125000 B/s = 1 Mbps
		{1000, "KB/s", 1.0, "KB/s"},
		{1024, "KiB/s", 1.0, "KiB/s"},
		{1000000000, "GB/s", 1.0, "GB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.unitName, func(t *testing.T) {
			gotVal, gotUnit := c.Convert(tt.bps, tt.unitName)
			if math.Abs(gotVal-tt.wantVal) > 0.001 || gotUnit != tt.wantUnit {
				t.Errorf("Convert(%v, %q) = (%v, %q), want (%v, %q)",
					tt.bps, tt.unitName, gotVal, gotUnit, tt.wantVal, tt.wantUnit)
			}
		})
	}
}

func TestByteRateConverter_ConvertAuto(t *testing.T) {
	c := NewByteRateConverter("MB/s")

	// Test auto-scaling with bytes decimal family
	val, unit := c.Convert(1500000, "auto")
	if unit != "MB/s" {
		t.Errorf("Convert(1500000, auto) unit = %q, want MB/s", unit)
	}
	if math.Abs(val-1.5) > 0.001 {
		t.Errorf("Convert(1500000, auto) value = %v, want 1.5", val)
	}
}

func TestByteRateConverter_AutoScale(t *testing.T) {
	tests := []struct {
		name        string
		defaultUnit string
		bps         float64
		wantUnit    string
	}{
		// Bytes decimal family
		{"small bytes", "MB/s", 500, "B/s"},
		{"kilobytes", "MB/s", 50000, "KB/s"},
		{"megabytes", "MB/s", 5000000, "MB/s"},
		{"gigabytes", "MB/s", 5000000000, "GB/s"},

		// Bytes binary family
		{"kibibytes", "MiB/s", 50000, "KiB/s"},
		{"mebibytes", "MiB/s", 5000000, "MiB/s"},

		// Bits family
		{"kilobits", "Mbps", 5000, "Kbps"},
		{"megabits", "Mbps", 500000, "Mbps"},
		{"gigabits", "Mbps", 500000000, "Gbps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewByteRateConverter(tt.defaultUnit)
			_, gotUnit := c.AutoScale(tt.bps)
			if gotUnit != tt.wantUnit {
				t.Errorf("AutoScale(%v) with default %q: unit = %q, want %q",
					tt.bps, tt.defaultUnit, gotUnit, tt.wantUnit)
			}
		})
	}
}

func TestByteRateConverter_SetFamily(t *testing.T) {
	c := NewByteRateConverter("MB/s")

	// Should start with bytes decimal
	if c.GetFamily() != UnitFamilyBytesDecimal {
		t.Errorf("Initial family = %v, want %v", c.GetFamily(), UnitFamilyBytesDecimal)
	}

	// Change to bits
	c.SetFamily(UnitFamilyBits)
	if c.GetFamily() != UnitFamilyBits {
		t.Errorf("After SetFamily(Bits): family = %v, want %v", c.GetFamily(), UnitFamilyBits)
	}

	// Auto-scale should now use bits
	_, unit := c.AutoScale(500000)
	if unit != "Mbps" {
		t.Errorf("AutoScale after SetFamily(Bits): unit = %q, want Mbps", unit)
	}
}

func TestAutoScaleBytes(t *testing.T) {
	tests := []struct {
		bps       float64
		useBinary bool
		wantUnit  string
	}{
		{500, false, "B/s"},
		{5000, false, "KB/s"},
		{5000000, false, "MB/s"},
		{5000000000, false, "GB/s"},
		{5000, true, "KiB/s"},
		{5000000, true, "MiB/s"},
		{5000000000, true, "GiB/s"},
	}

	for _, tt := range tests {
		_, gotUnit := AutoScaleBytes(tt.bps, tt.useBinary)
		if gotUnit != tt.wantUnit {
			t.Errorf("AutoScaleBytes(%v, %v) unit = %q, want %q",
				tt.bps, tt.useBinary, gotUnit, tt.wantUnit)
		}
	}
}

func TestAutoScaleBits(t *testing.T) {
	tests := []struct {
		bps      float64
		wantUnit string
	}{
		{10, "bps"},
		{5000, "Kbps"},
		{500000, "Mbps"},
		{500000000, "Gbps"},
	}

	for _, tt := range tests {
		_, gotUnit := AutoScaleBits(tt.bps)
		if gotUnit != tt.wantUnit {
			t.Errorf("AutoScaleBits(%v) unit = %q, want %q", tt.bps, gotUnit, tt.wantUnit)
		}
	}
}

func TestConvertToUnit(t *testing.T) {
	tests := []struct {
		bps      float64
		unitName string
		wantVal  float64
		wantUnit string
	}{
		{1000000, "MB/s", 1.0, "MB/s"},
		{1048576, "MiB/s", 1.0, "MiB/s"},
		{125000, "Mbps", 1.0, "Mbps"},
		{1000000, "invalid", 1000000, "B/s"}, // Falls back to B/s
	}

	for _, tt := range tests {
		t.Run(tt.unitName, func(t *testing.T) {
			gotVal, gotUnit := ConvertToUnit(tt.bps, tt.unitName)
			if math.Abs(gotVal-tt.wantVal) > 0.001 || gotUnit != tt.wantUnit {
				t.Errorf("ConvertToUnit(%v, %q) = (%v, %q), want (%v, %q)",
					tt.bps, tt.unitName, gotVal, gotUnit, tt.wantVal, tt.wantUnit)
			}
		})
	}
}

func TestAllUnitsContainsExpected(t *testing.T) {
	expectedUnits := []string{
		"B/s", "KB/s", "MB/s", "GB/s",
		"KiB/s", "MiB/s", "GiB/s",
		"bps", "Kbps", "Mbps", "Gbps",
	}

	for _, unit := range expectedUnits {
		if _, ok := AllUnits[unit]; !ok {
			t.Errorf("AllUnits missing expected unit %q", unit)
		}
	}

	// Check count (should be 11 unique units, but B/s appears in multiple maps)
	if len(AllUnits) != 11 {
		t.Errorf("len(AllUnits) = %d, want 11", len(AllUnits))
	}
}
