package render

import "testing"

// testClassifier is a simple classifier for testing purposes.
// It classifies "icon" as TokenIcon, "name"/"value" as TokenText,
// and everything else as TokenLiteral (unknown).
func testClassifier(name string) TokenType {
	switch name {
	case "icon":
		return TokenIcon
	case "name", "value", "temp":
		return TokenText
	default:
		return TokenLiteral
	}
}

func TestParseFormatTokens(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		wantCount  int
		wantTokens []Token
	}{
		{
			name:      "empty format",
			format:    "",
			wantCount: 0,
		},
		{
			name:      "pure literal text",
			format:    "hello world",
			wantCount: 1,
			wantTokens: []Token{
				{Type: TokenLiteral, Literal: "hello world"},
			},
		},
		{
			name:      "single token",
			format:    "{icon}",
			wantCount: 1,
			wantTokens: []Token{
				{Type: TokenIcon, Name: "icon"},
			},
		},
		{
			name:      "token with param",
			format:    "{icon:24}",
			wantCount: 1,
			wantTokens: []Token{
				{Type: TokenIcon, Name: "icon", Param: "24"},
			},
		},
		{
			name:      "mixed literal and tokens",
			format:    "{icon} {name} {value}",
			wantCount: 5,
			wantTokens: []Token{
				{Type: TokenIcon, Name: "icon"},
				{Type: TokenLiteral, Literal: " "},
				{Type: TokenText, Name: "name"},
				{Type: TokenLiteral, Literal: " "},
				{Type: TokenText, Name: "value"},
			},
		},
		{
			name:      "literal before and after token",
			format:    "Hello {name} World",
			wantCount: 3,
			wantTokens: []Token{
				{Type: TokenLiteral, Literal: "Hello "},
				{Type: TokenText, Name: "name"},
				{Type: TokenLiteral, Literal: " World"},
			},
		},
		{
			name:      "unknown token classified as literal",
			format:    "{unknown}",
			wantCount: 1,
			wantTokens: []Token{
				{Type: TokenLiteral, Name: "unknown"},
			},
		},
		{
			name:      "adjacent tokens no gap",
			format:    "{icon}{name}",
			wantCount: 2,
			wantTokens: []Token{
				{Type: TokenIcon, Name: "icon"},
				{Type: TokenText, Name: "name"},
			},
		},
		{
			name:      "literal separator between tokens",
			format:    "{icon} | {name}",
			wantCount: 3,
			wantTokens: []Token{
				{Type: TokenIcon, Name: "icon"},
				{Type: TokenLiteral, Literal: " | "},
				{Type: TokenText, Name: "name"},
			},
		},
		{
			name:      "multiple params",
			format:    "{icon:large} {temp:raw}",
			wantCount: 3,
			wantTokens: []Token{
				{Type: TokenIcon, Name: "icon", Param: "large"},
				{Type: TokenLiteral, Literal: " "},
				{Type: TokenText, Name: "temp", Param: "raw"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ParseFormatTokens(tt.format, testClassifier)
			if len(tokens) != tt.wantCount {
				t.Errorf("ParseFormatTokens(%q) returned %d tokens, want %d", tt.format, len(tokens), tt.wantCount)
				for i, tok := range tokens {
					t.Logf("  token[%d]: type=%d name=%q param=%q literal=%q", i, tok.Type, tok.Name, tok.Param, tok.Literal)
				}
				return
			}
			for i, want := range tt.wantTokens {
				if i >= len(tokens) {
					break
				}
				got := tokens[i]
				if got.Type != want.Type {
					t.Errorf("token[%d].Type = %d, want %d", i, got.Type, want.Type)
				}
				if want.Name != "" && got.Name != want.Name {
					t.Errorf("token[%d].Name = %q, want %q", i, got.Name, want.Name)
				}
				if want.Param != "" && got.Param != want.Param {
					t.Errorf("token[%d].Param = %q, want %q", i, got.Param, want.Param)
				}
				if want.Literal != "" && got.Literal != want.Literal {
					t.Errorf("token[%d].Literal = %q, want %q", i, got.Literal, want.Literal)
				}
			}
		})
	}
}
