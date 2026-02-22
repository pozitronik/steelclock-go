package gpu

import "testing"

// TestNormalizeEngineType verifies that both AMD and NVIDIA engine type naming
// conventions are normalized to the same lookup keys.
func TestNormalizeEngineType(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		// NVIDIA style (single concatenated words)
		{"nvidia 3d", "3D", "3d"},
		{"nvidia videodecode", "VideoDecode", "videodecode"},
		{"nvidia videoencode", "VideoEncode", "videoencode"},
		{"nvidia copy", "Copy", "copy"},
		{"nvidia legacyoverlay", "LegacyOverlay", "legacyoverlay"},
		{"nvidia security", "Security", "security"},
		{"nvidia vr", "VR", "vr"},
		{"nvidia ofa_0", "ofa_0", "ofa"},

		// AMD style (spaces and trailing instance numbers)
		{"amd 3d", "3D", "3d"},
		{"amd video decode 1", "video decode 1", "videodecode"},
		{"amd video encode 0", "video encode 0", "videoencode"},
		{"amd video codec 0", "video codec 0", "videocodec"},
		{"amd copy", "Copy", "copy"},
		{"amd high priority compute", "high priority compute", "highprioritycompute"},
		{"amd compute 0", "compute 0", "compute"},
		{"amd compute 1", "compute 1", "compute"},
		{"amd timer 0", "timer 0", "timer"},
		{"amd security 1", "security 1", "security"},

		// Edge cases
		{"empty string", "", ""},
		{"only digits", "123", ""},
		{"trailing spaces", "3d  ", "3d"},
		{"mixed case with digits", "Video Decode 12", "videodecode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeEngineType(tt.raw)
			if got != tt.want {
				t.Errorf("normalizeEngineType(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestEngineTypeMetricsMapping verifies that normalized engine types map to
// the correct metric constants for both AMD and NVIDIA naming conventions.
func TestEngineTypeMetricsMapping(t *testing.T) {
	tests := []struct {
		name       string
		rawEngType string
		wantMetric string
		wantFound  bool
	}{
		// 3D (both vendors)
		{"3D", "3D", MetricUtilization3D, true},
		{"3d lowercase", "3d", MetricUtilization3D, true},

		// Copy (both vendors)
		{"Copy", "Copy", MetricUtilizationCopy, true},
		{"copy lowercase", "copy", MetricUtilizationCopy, true},

		// Video decode
		{"nvidia videodecode", "VideoDecode", MetricUtilizationDecode, true},
		{"amd video decode 1", "video decode 1", MetricUtilizationDecode, true},
		{"amd video decode 0", "video decode 0", MetricUtilizationDecode, true},

		// Video encode
		{"nvidia videoencode", "VideoEncode", MetricUtilizationEncode, true},
		{"amd video encode 0", "video encode 0", MetricUtilizationEncode, true},

		// AMD video codec (combined encode/decode, mapped to decode)
		{"amd video codec 0", "video codec 0", MetricUtilizationDecode, true},

		// Unknown engine types should not match
		{"nvidia legacyoverlay", "LegacyOverlay", "", false},
		{"nvidia security", "Security", "", false},
		{"nvidia vr", "VR", "", false},
		{"amd high priority compute", "high priority compute", "", false},
		{"amd compute 0", "compute 0", "", false},
		{"amd timer 0", "timer 0", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeEngineType(tt.rawEngType)
			metric, found := engineTypeMetrics[normalized]
			if found != tt.wantFound {
				t.Errorf("engineTypeMetrics[%q] (normalized from %q): found=%v, want found=%v",
					normalized, tt.rawEngType, found, tt.wantFound)
				return
			}
			if found && metric != tt.wantMetric {
				t.Errorf("engineTypeMetrics[%q] (normalized from %q) = %q, want %q",
					normalized, tt.rawEngType, metric, tt.wantMetric)
			}
		})
	}
}

// TestEngineTypeMetricsCompleteness verifies that all expected normalized keys
// are present in the engineTypeMetrics map.
func TestEngineTypeMetricsCompleteness(t *testing.T) {
	expectedKeys := []string{
		"3d",
		"copy",
		"videoencode",
		"videodecode",
		"videocodec",
	}

	for _, key := range expectedKeys {
		if _, ok := engineTypeMetrics[key]; !ok {
			t.Errorf("engineTypeMetrics missing key %q", key)
		}
	}
}
