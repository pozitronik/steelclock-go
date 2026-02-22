package gpu

import "strings"

// engineTypeMetrics maps normalized PDH engine type names (lowercased, spaces/digits stripped)
// to metric constants. Both AMD and NVIDIA naming conventions are handled:
//   - AMD uses spaces: "video decode 1", "video encode 0", "video codec 0"
//   - NVIDIA uses concatenated words: "videodecode", "videoencode"
//
// Normalization strips spaces, trailing digits, and underscores, so both styles
// reduce to the same key (e.g., "videodecode").
var engineTypeMetrics = map[string]string{
	"3d":          MetricUtilization3D,
	"copy":        MetricUtilizationCopy,
	"videoencode": MetricUtilizationEncode,
	"videodecode": MetricUtilizationDecode,
	"videocodec":  MetricUtilizationDecode, // AMD "video codec" is a combined encode/decode engine
}

// normalizeEngineType converts a raw PDH engine type string into a normalized
// form suitable for lookup in engineTypeMetrics.
//
// Normalization:
//  1. Lowercase
//  2. Strip trailing digits (e.g., "video decode 1" -> "video decode ")
//  3. Remove spaces (e.g., "video decode " -> "videodecode")
//
// This handles both AMD style ("video decode 1", "video encode 0") and
// NVIDIA style ("videodecode", "videoencode") uniformly.
func normalizeEngineType(raw string) string {
	s := strings.ToLower(raw)
	// Strip trailing digits: "video decode 1" -> "video decode "
	s = strings.TrimRight(s, "0123456789")
	// Remove all spaces: "video decode " -> "videodecode"
	s = strings.ReplaceAll(s, " ", "")
	// Trim trailing underscores that may remain
	s = strings.TrimRight(s, "_")
	return s
}
