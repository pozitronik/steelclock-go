package bluetooth

import (
	"regexp"
)

// TokenType represents the type of format token
type TokenType int

const (
	TokenLiteral TokenType = iota // Plain text (spaces, separators)
	TokenText                     // Text-based token (name, level, state)
	TokenIcon                     // Icon token (device type glyph)
	TokenShape                    // Shape token (battery, bar)
)

// Token represents a parsed token from the format string
type Token struct {
	Type    TokenType
	Name    string // Token name without braces (e.g., "icon", "battery")
	Param   string // Optional parameter (e.g., "20" in {battery:20})
	Literal string // For literal tokens, the text content
}

// tokenPattern matches {token} or {token:param} in the format string
var tokenPattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?\}`)

// parseBluetoothFormat parses a format string into tokens
func parseBluetoothFormat(format string) []Token {
	var tokens []Token

	lastEnd := 0
	for _, match := range tokenPattern.FindAllStringSubmatchIndex(format, -1) {
		// Add literal text before this token
		if match[0] > lastEnd {
			tokens = append(tokens, Token{
				Type:    TokenLiteral,
				Literal: format[lastEnd:match[0]],
			})
		}

		// Extract token name and optional parameter
		name := format[match[2]:match[3]]
		param := ""
		if match[4] >= 0 && match[5] >= 0 {
			param = format[match[4]:match[5]]
		}

		tokenType := getBluetoothTokenType(name)
		tokens = append(tokens, Token{
			Type:  tokenType,
			Name:  name,
			Param: param,
		})

		lastEnd = match[1]
	}

	// Add any remaining literal text
	if lastEnd < len(format) {
		tokens = append(tokens, Token{
			Type:    TokenLiteral,
			Literal: format[lastEnd:],
		})
	}

	return tokens
}

// getBluetoothTokenType classifies a token name into its type
func getBluetoothTokenType(name string) TokenType {
	switch name {
	case "icon":
		return TokenIcon
	case "name", "level", "state":
		return TokenText
	case "battery", "battery_h", "battery_v", "bar", "bar_h", "bar_v":
		return TokenShape
	default:
		return TokenLiteral
	}
}

// findBlinkTarget returns the index of the token that should blink for low battery.
// Priority: first shape token -> first icon token -> first name token.
// Returns -1 if no suitable target is found.
func findBlinkTarget(tokens []Token) int {
	firstIcon := -1
	firstName := -1

	for i, t := range tokens {
		switch t.Type {
		case TokenShape:
			return i // Shape tokens have highest priority
		case TokenIcon:
			if firstIcon < 0 {
				firstIcon = i
			}
		case TokenText:
			if t.Name == "name" && firstName < 0 {
				firstName = i
			}
		}
	}

	if firstIcon >= 0 {
		return firstIcon
	}
	return firstName
}
