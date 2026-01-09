package clipboard

import (
	"strings"
	"testing"
)

// BenchmarkFormatContent_Current benchmarks the current implementation
// to measure allocations from chained ReplaceAll calls.
//
// Expected: Multiple allocations per call due to:
// - 4 ReplaceAll calls for whitespace handling
// - 4 ReplaceAll calls for template substitution
// - Each ReplaceAll creates a new string
func BenchmarkFormatContent_Current(b *testing.B) {
	w := &Widget{
		cfg: Config{
			TextFormat:    "{type}: {content} ({length} chars) - {preview}",
			MaxLength:     100,
			ShowInvisible: false,
		},
	}

	// Realistic clipboard content with whitespace
	content := "Line1\r\nLine2\nLine3\tColumn"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = w.formatContent(content, TypeText)
	}
}

// BenchmarkFormatContent_ShowInvisible benchmarks with ShowInvisible enabled
func BenchmarkFormatContent_ShowInvisible(b *testing.B) {
	w := &Widget{
		cfg: Config{
			TextFormat:    "{type}: {content} ({length} chars)",
			MaxLength:     100,
			ShowInvisible: true,
		},
	}

	content := "Line1\r\nLine2\nLine3\tColumn"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = w.formatContent(content, TypeText)
	}
}

// BenchmarkFormatContent_LongContent benchmarks with longer content
func BenchmarkFormatContent_LongContent(b *testing.B) {
	w := &Widget{
		cfg: Config{
			TextFormat:    "{content}",
			MaxLength:     1000,
			ShowInvisible: false,
		},
	}

	// Simulate a longer clipboard content (500 chars with mixed whitespace)
	content := strings.Repeat("Word\tMore\nText\r\n", 25)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = w.formatContent(content, TypeText)
	}
}

// BenchmarkReplaceAllChain demonstrates the allocation cost of chained ReplaceAll
func BenchmarkReplaceAllChain(b *testing.B) {
	content := "Line1\r\nLine2\nLine3\tColumn"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := content
		result = strings.ReplaceAll(result, "\r\n", " ")
		result = strings.ReplaceAll(result, "\n", " ")
		result = strings.ReplaceAll(result, "\r", "")
		result = strings.ReplaceAll(result, "\t", " ")
		_ = result
	}
}

// BenchmarkStringBuilder_SinglePass demonstrates the optimized approach
func BenchmarkStringBuilder_SinglePass(b *testing.B) {
	content := "Line1\r\nLine2\nLine3\tColumn"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.Grow(len(content))

		for j := 0; j < len(content); j++ {
			c := content[j]
			switch c {
			case '\r':
				if j+1 < len(content) && content[j+1] == '\n' {
					sb.WriteByte(' ')
					j++ // Skip the \n
				}
				// Skip standalone \r
			case '\n':
				sb.WriteByte(' ')
			case '\t':
				sb.WriteByte(' ')
			default:
				sb.WriteByte(c)
			}
		}
		_ = sb.String()
	}
}
