package render

import "testing"

func TestNewTokenFormatter(t *testing.T) {
	f := NewTokenFormatter()
	if f == nil {
		t.Fatal("NewTokenFormatter returned nil")
	}
	if f.Count() != 0 {
		t.Errorf("Expected 0 tokens, got %d", f.Count())
	}
	if f.prefix != "{" || f.suffix != "}" {
		t.Errorf("Expected default format {}, got %s%s", f.prefix, f.suffix)
	}
}

func TestNewTokenFormatterWithFormat(t *testing.T) {
	f := NewTokenFormatterWithFormat("{{", "}}")
	if f == nil {
		t.Fatal("NewTokenFormatterWithFormat returned nil")
	}
	if f.prefix != "{{" || f.suffix != "}}" {
		t.Errorf("Expected format {{}}, got %s%s", f.prefix, f.suffix)
	}
}

func TestTokenFormatter_Set(t *testing.T) {
	f := NewTokenFormatter()

	// Test chaining
	result := f.Set("name", "John").Set("age", "30")
	if result != f {
		t.Error("Set should return self for chaining")
	}

	if f.Count() != 2 {
		t.Errorf("Expected 2 tokens, got %d", f.Count())
	}
}

func TestTokenFormatter_SetAll(t *testing.T) {
	f := NewTokenFormatter()

	tokens := map[string]string{
		"title":  "Hello",
		"artist": "World",
		"album":  "Test",
	}

	result := f.SetAll(tokens)
	if result != f {
		t.Error("SetAll should return self for chaining")
	}

	if f.Count() != 3 {
		t.Errorf("Expected 3 tokens, got %d", f.Count())
	}
}

func TestTokenFormatter_Get(t *testing.T) {
	f := NewTokenFormatter().Set("name", "John")

	if v := f.Get("name"); v != "John" {
		t.Errorf("Expected 'John', got %q", v)
	}

	if v := f.Get("unknown"); v != "" {
		t.Errorf("Expected empty string for unknown token, got %q", v)
	}
}

func TestTokenFormatter_Has(t *testing.T) {
	f := NewTokenFormatter().Set("name", "John")

	if !f.Has("name") {
		t.Error("Expected Has('name') to return true")
	}

	if f.Has("unknown") {
		t.Error("Expected Has('unknown') to return false")
	}
}

func TestTokenFormatter_Clear(t *testing.T) {
	f := NewTokenFormatter().Set("a", "1").Set("b", "2")

	if f.Count() != 2 {
		t.Errorf("Expected 2 tokens before clear, got %d", f.Count())
	}

	result := f.Clear()
	if result != f {
		t.Error("Clear should return self for chaining")
	}

	if f.Count() != 0 {
		t.Errorf("Expected 0 tokens after clear, got %d", f.Count())
	}
}

func TestTokenFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		tokens   map[string]string
		template string
		expected string
	}{
		{
			name:     "empty template",
			tokens:   map[string]string{"name": "John"},
			template: "",
			expected: "",
		},
		{
			name:     "no tokens in template",
			tokens:   map[string]string{"name": "John"},
			template: "Hello World",
			expected: "Hello World",
		},
		{
			name:     "single token",
			tokens:   map[string]string{"name": "John"},
			template: "Hello {name}!",
			expected: "Hello John!",
		},
		{
			name:     "multiple tokens",
			tokens:   map[string]string{"name": "John", "age": "30"},
			template: "{name} is {age} years old",
			expected: "John is 30 years old",
		},
		{
			name:     "repeated token",
			tokens:   map[string]string{"x": "A"},
			template: "{x}{x}{x}",
			expected: "AAA",
		},
		{
			name:     "unknown token unchanged",
			tokens:   map[string]string{"name": "John"},
			template: "Hello {name}, {unknown}!",
			expected: "Hello John, {unknown}!",
		},
		{
			name:     "empty token value",
			tokens:   map[string]string{"name": ""},
			template: "Hello {name}!",
			expected: "Hello !",
		},
		{
			name:     "token with special characters",
			tokens:   map[string]string{"path": "/usr/local/bin"},
			template: "Path: {path}",
			expected: "Path: /usr/local/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTokenFormatter().SetAll(tt.tokens)
			result := f.Format(tt.template)
			if result != tt.expected {
				t.Errorf("Format(%q) = %q, want %q", tt.template, result, tt.expected)
			}
		})
	}
}

func TestTokenFormatter_FormatStrict(t *testing.T) {
	tests := []struct {
		name     string
		tokens   map[string]string
		template string
		expected string
	}{
		{
			name:     "removes unknown tokens",
			tokens:   map[string]string{"name": "John"},
			template: "Hello {name}, {unknown}!",
			expected: "Hello John, !",
		},
		{
			name:     "removes multiple unknown tokens",
			tokens:   map[string]string{},
			template: "{a} and {b} and {c}",
			expected: " and  and ",
		},
		{
			name:     "known tokens replaced",
			tokens:   map[string]string{"name": "John", "age": "30"},
			template: "{name} is {age}",
			expected: "John is 30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTokenFormatter().SetAll(tt.tokens)
			result := f.FormatStrict(tt.template)
			if result != tt.expected {
				t.Errorf("FormatStrict(%q) = %q, want %q", tt.template, result, tt.expected)
			}
		})
	}
}

func TestTokenFormatter_CustomFormat(t *testing.T) {
	f := NewTokenFormatterWithFormat("{{", "}}").Set("name", "John")

	template := "Hello {{name}}!"
	result := f.Format(template)
	expected := "Hello John!"

	if result != expected {
		t.Errorf("Format with custom delimiters: got %q, want %q", result, expected)
	}

	// Original format should not be replaced
	template2 := "Hello {name}!"
	result2 := f.Format(template2)
	if result2 != template2 {
		t.Errorf("Format should not replace {name} with custom delimiters: got %q, want %q", result2, template2)
	}
}

func TestTokenFormatter_Clone(t *testing.T) {
	original := NewTokenFormatter().Set("a", "1").Set("b", "2")
	clone := original.Clone()

	// Check values copied
	if clone.Get("a") != "1" || clone.Get("b") != "2" {
		t.Error("Clone should copy all token values")
	}

	// Modify clone - should not affect original
	clone.Set("a", "modified")
	clone.Set("c", "3")

	if original.Get("a") != "1" {
		t.Error("Modifying clone should not affect original")
	}
	if original.Has("c") {
		t.Error("Adding to clone should not affect original")
	}
}

func TestTokenFormatter_Count(t *testing.T) {
	f := NewTokenFormatter()

	if f.Count() != 0 {
		t.Errorf("Expected 0, got %d", f.Count())
	}

	f.Set("a", "1")
	if f.Count() != 1 {
		t.Errorf("Expected 1, got %d", f.Count())
	}

	f.Set("b", "2").Set("c", "3")
	if f.Count() != 3 {
		t.Errorf("Expected 3, got %d", f.Count())
	}

	// Setting same key updates, doesn't add
	f.Set("a", "updated")
	if f.Count() != 3 {
		t.Errorf("Expected 3 after update, got %d", f.Count())
	}
}

func TestTokenFormatter_Chaining(t *testing.T) {
	// Test full method chaining
	result := NewTokenFormatter().
		Set("greeting", "Hello").
		Set("name", "World").
		SetAll(map[string]string{"punctuation": "!"}).
		Format("{greeting} {name}{punctuation}")

	expected := "Hello World!"
	if result != expected {
		t.Errorf("Chained format: got %q, want %q", result, expected)
	}
}
