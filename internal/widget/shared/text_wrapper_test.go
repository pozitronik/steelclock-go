package shared

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

func TestSplitByNewlines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "no newlines",
			input:    "hello world",
			expected: []string{"hello world"},
		},
		{
			name:     "single newline",
			input:    "hello\nworld",
			expected: []string{"hello", "world"},
		},
		{
			name:     "multiple newlines",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "carriage return",
			input:    "line1\rline2",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "trailing newline",
			input:    "hello\n",
			expected: []string{"hello", ""},
		},
		{
			name:     "leading newline",
			input:    "\nhello",
			expected: []string{"", "hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitByNewlines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitByNewlines(%q) = %v (len %d), want %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("SplitByNewlines(%q)[%d] = %q, want %q",
						tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single word",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "two words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "multiple spaces",
			input:    "hello   world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "tabs",
			input:    "hello\tworld",
			expected: []string{"hello", "world"},
		},
		{
			name:     "leading space",
			input:    " hello",
			expected: []string{"hello"},
		},
		{
			name:     "trailing space",
			input:    "hello ",
			expected: []string{"hello"},
		},
		{
			name:     "mixed whitespace",
			input:    "  hello \t world  ",
			expected: []string{"hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitIntoWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitIntoWords(%q) = %v (len %d), want %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("SplitIntoWords(%q)[%d] = %q, want %q",
						tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestNewTextWrapper(t *testing.T) {
	// Load a test font
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	t.Run("default mode is normal", func(t *testing.T) {
		wrapper := NewTextWrapper(fontFace, "", 100, "")
		if wrapper.mode != WrapModeNormal {
			t.Errorf("Expected default mode %q, got %q", WrapModeNormal, wrapper.mode)
		}
	})

	t.Run("explicit mode is set", func(t *testing.T) {
		wrapper := NewTextWrapper(fontFace, "", 100, WrapModeBreakAll)
		if wrapper.mode != WrapModeBreakAll {
			t.Errorf("Expected mode %q, got %q", WrapModeBreakAll, wrapper.mode)
		}
	})
}

func TestTextWrapper_Wrap(t *testing.T) {
	// Load a test font
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	t.Run("empty text returns nil", func(t *testing.T) {
		wrapper := NewTextWrapper(fontFace, "", 100, WrapModeNormal)
		result := wrapper.Wrap("")
		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})

	t.Run("short text fits on one line", func(t *testing.T) {
		wrapper := NewTextWrapper(fontFace, "", 200, WrapModeNormal)
		result := wrapper.Wrap("Hello")
		if len(result) != 1 {
			t.Errorf("Expected 1 line, got %d", len(result))
		}
		if result[0] != "Hello" {
			t.Errorf("Expected 'Hello', got %q", result[0])
		}
	})

	t.Run("text with newline splits into lines", func(t *testing.T) {
		wrapper := NewTextWrapper(fontFace, "", 200, WrapModeNormal)
		result := wrapper.Wrap("Hello\nWorld")
		if len(result) != 2 {
			t.Errorf("Expected 2 lines, got %d: %v", len(result), result)
			return
		}
		if result[0] != "Hello" {
			t.Errorf("Expected 'Hello', got %q", result[0])
		}
		if result[1] != "World" {
			t.Errorf("Expected 'World', got %q", result[1])
		}
	})

	t.Run("long text wraps at word boundary", func(t *testing.T) {
		// Use a very narrow width to force wrapping
		wrapper := NewTextWrapper(fontFace, "", 40, WrapModeNormal)
		result := wrapper.Wrap("Hello World")
		if len(result) < 2 {
			t.Errorf("Expected text to wrap into multiple lines, got %d: %v", len(result), result)
		}
	})

	t.Run("break-all mode breaks at character", func(t *testing.T) {
		wrapper := NewTextWrapper(fontFace, "", 30, WrapModeBreakAll)
		result := wrapper.Wrap("ABCDEFGH")
		if len(result) < 2 {
			t.Errorf("Expected text to wrap into multiple lines in break-all mode, got %d: %v", len(result), result)
		}
	})
}

func TestTextWrapper_SetMaxWidth(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	wrapper := NewTextWrapper(fontFace, "", 100, WrapModeNormal)
	wrapper.SetMaxWidth(50)

	if wrapper.maxWidth != 50 {
		t.Errorf("Expected maxWidth 50, got %d", wrapper.maxWidth)
	}
}

func TestTextWrapper_TruncateWithEllipsis(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	wrapper := NewTextWrapper(fontFace, "", 100, WrapModeNormal)

	t.Run("empty lines returns empty", func(t *testing.T) {
		result := wrapper.TruncateWithEllipsis(nil, 50)
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got %v", result)
		}
	})

	t.Run("lines that fit are unchanged", func(t *testing.T) {
		lines := []string{"line1", "line2"}
		result := wrapper.TruncateWithEllipsis(lines, 100)
		if len(result) != 2 {
			t.Errorf("Expected 2 lines, got %d", len(result))
		}
	})

	t.Run("excess lines are truncated with ellipsis", func(t *testing.T) {
		lines := []string{"line1", "line2", "line3", "line4", "line5"}
		// With a small height, we should get truncation
		lineHeight := wrapper.MeasureLineHeight()
		maxHeight := lineHeight * 2 // Only allow 2 lines
		result := wrapper.TruncateWithEllipsis(lines, maxHeight)

		if len(result) != 2 {
			t.Errorf("Expected 2 lines, got %d: %v", len(result), result)
			return
		}

		// Last line should end with ellipsis
		lastLine := result[len(result)-1]
		if len(lastLine) < 3 || lastLine[len(lastLine)-3:] != "..." {
			t.Errorf("Expected last line to end with '...', got %q", lastLine)
		}
	})
}

func TestTextWrapper_MeasureLineHeight(t *testing.T) {
	fontFace, err := bitmap.LoadFont("", 10)
	if err != nil {
		t.Skipf("Cannot load font for testing: %v", err)
	}

	wrapper := NewTextWrapper(fontFace, "", 100, WrapModeNormal)
	height := wrapper.MeasureLineHeight()

	if height <= 0 {
		t.Errorf("Expected positive line height, got %d", height)
	}
}

//goland:noinspection GoBoolExpressions
func TestWrapModeConstants(t *testing.T) {
	if WrapModeNormal != "normal" {
		t.Errorf("WrapModeNormal = %q, want 'normal'", WrapModeNormal)
	}
	if WrapModeBreakAll != "break-all" {
		t.Errorf("WrapModeBreakAll = %q, want 'break-all'", WrapModeBreakAll)
	}
}
