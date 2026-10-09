package render

import (
	"fmt"
	"strings"
)

// IsUsageTokenFormat reports whether a usage metric's text format uses the
// {used}/{total}/{percent} tokens rather than a plain printf format. Any "{"
// selects token mode: a printf format for a single percentage never has one.
func IsUsageTokenFormat(format string) bool {
	return strings.Contains(format, "{")
}

// FormatUsageTokens renders a usage text format, replacing {used} and {total}
// with gigabytes (one decimal) and {percent} with the usage percentage (no
// decimals). Unknown tokens are left unchanged.
func FormatUsageTokens(format string, usedGB, totalGB, percent float64) string {
	return NewTokenFormatter().
		Set("used", fmt.Sprintf("%.1f", usedGB)).
		Set("total", fmt.Sprintf("%.1f", totalGB)).
		Set("percent", fmt.Sprintf("%.0f", percent)).
		Format(format)
}
