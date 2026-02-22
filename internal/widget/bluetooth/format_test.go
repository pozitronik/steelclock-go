package bluetooth

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/shared/render"
)

func TestParseBluetoothFormat(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		wantCount  int
		wantTokens []render.Token
	}{
		{
			name:      "default format",
			format:    "{icon} {name} {battery:20}",
			wantCount: 5,
			wantTokens: []render.Token{
				{Type: render.TokenIcon, Name: "icon"},
				{Type: render.TokenLiteral, Literal: " "},
				{Type: render.TokenText, Name: "name"},
				{Type: render.TokenLiteral, Literal: " "},
				{Type: TokenShape, Name: "battery", Param: "20"},
			},
		},
		{
			name:      "icon only",
			format:    "{icon}",
			wantCount: 1,
			wantTokens: []render.Token{
				{Type: render.TokenIcon, Name: "icon"},
			},
		},
		{
			name:      "name and level",
			format:    "{name} {level}",
			wantCount: 3,
			wantTokens: []render.Token{
				{Type: render.TokenText, Name: "name"},
				{Type: render.TokenLiteral, Literal: " "},
				{Type: render.TokenText, Name: "level"},
			},
		},
		{
			name:      "all text tokens",
			format:    "{name} {level} {state}",
			wantCount: 5,
			wantTokens: []render.Token{
				{Type: render.TokenText, Name: "name"},
				{Type: render.TokenLiteral, Literal: " "},
				{Type: render.TokenText, Name: "level"},
				{Type: render.TokenLiteral, Literal: " "},
				{Type: render.TokenText, Name: "state"},
			},
		},
		{
			name:      "vertical battery",
			format:    "{battery_v:15}",
			wantCount: 1,
			wantTokens: []render.Token{
				{Type: TokenShape, Name: "battery_v", Param: "15"},
			},
		},
		{
			name:      "horizontal bar",
			format:    "{bar_h:30}",
			wantCount: 1,
			wantTokens: []render.Token{
				{Type: TokenShape, Name: "bar_h", Param: "30"},
			},
		},
		{
			name:      "literal text with separator",
			format:    "{icon} | {name}",
			wantCount: 3,
			wantTokens: []render.Token{
				{Type: render.TokenIcon, Name: "icon"},
				{Type: render.TokenLiteral, Literal: " | "},
				{Type: render.TokenText, Name: "name"},
			},
		},
		{
			name:      "unknown token treated as literal",
			format:    "{unknown}",
			wantCount: 1,
			wantTokens: []render.Token{
				{Type: render.TokenLiteral, Name: "unknown"},
			},
		},
		{
			name:      "empty format",
			format:    "",
			wantCount: 0,
		},
		{
			name:      "pure literal text",
			format:    "hello world",
			wantCount: 1,
			wantTokens: []render.Token{
				{Type: render.TokenLiteral, Literal: "hello world"},
			},
		},
		{
			name:      "bar without param",
			format:    "{bar}",
			wantCount: 1,
			wantTokens: []render.Token{
				{Type: TokenShape, Name: "bar"},
			},
		},
		{
			name:      "all shape variants",
			format:    "{battery:20} {battery_h:25} {battery_v:30} {bar:15} {bar_h:20} {bar_v:10}",
			wantCount: 11, // 6 shapes + 5 spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := parseBluetoothFormat(tt.format)
			if len(tokens) != tt.wantCount {
				t.Errorf("parseBluetoothFormat(%q) returned %d tokens, want %d", tt.format, len(tokens), tt.wantCount)
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

func TestGetBluetoothTokenType(t *testing.T) {
	tests := []struct {
		name     string
		wantType render.TokenType
	}{
		{"icon", render.TokenIcon},
		{"name", render.TokenText},
		{"level", render.TokenText},
		{"state", render.TokenText},
		{"battery", TokenShape},
		{"battery_h", TokenShape},
		{"battery_v", TokenShape},
		{"bar", TokenShape},
		{"bar_h", TokenShape},
		{"bar_v", TokenShape},
		{"unknown", render.TokenLiteral},
		{"foo", render.TokenLiteral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBluetoothTokenType(tt.name)
			if got != tt.wantType {
				t.Errorf("getBluetoothTokenType(%q) = %d, want %d", tt.name, got, tt.wantType)
			}
		})
	}
}

func TestFindBlinkTarget(t *testing.T) {
	tests := []struct {
		name   string
		tokens []render.Token
		want   int
	}{
		{
			name:   "shape token first priority",
			tokens: []render.Token{{Type: render.TokenIcon, Name: "icon"}, {Type: render.TokenText, Name: "name"}, {Type: TokenShape, Name: "battery"}},
			want:   2,
		},
		{
			name:   "icon fallback when no shape",
			tokens: []render.Token{{Type: render.TokenIcon, Name: "icon"}, {Type: render.TokenText, Name: "name"}},
			want:   0,
		},
		{
			name:   "name fallback when no shape or icon",
			tokens: []render.Token{{Type: render.TokenText, Name: "name"}, {Type: render.TokenText, Name: "level"}},
			want:   0,
		},
		{
			name:   "no target when only literals",
			tokens: []render.Token{{Type: render.TokenLiteral, Literal: " "}, {Type: render.TokenText, Name: "level"}},
			want:   -1,
		},
		{
			name:   "no target for empty tokens",
			tokens: []render.Token{},
			want:   -1,
		},
		{
			name:   "shape before icon even if icon is first",
			tokens: []render.Token{{Type: render.TokenIcon, Name: "icon"}, {Type: TokenShape, Name: "bar"}},
			want:   1,
		},
		{
			name:   "first shape of multiple",
			tokens: []render.Token{{Type: TokenShape, Name: "battery"}, {Type: TokenShape, Name: "bar"}},
			want:   0,
		},
		{
			name:   "level does not count as name for fallback",
			tokens: []render.Token{{Type: render.TokenText, Name: "level"}, {Type: render.TokenText, Name: "state"}},
			want:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findBlinkTarget(tt.tokens)
			if got != tt.want {
				t.Errorf("findBlinkTarget() = %d, want %d", got, tt.want)
			}
		})
	}
}
