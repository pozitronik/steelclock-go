package render

import "regexp"

// TokenType represents the type of format token
type TokenType int

const (
	TokenLiteral    TokenType = iota // Plain text (spaces, separators)
	TokenText                        // Text-based token (name, level, state, temp, etc.)
	TokenIcon                        // Icon token (device type glyph, weather icon, etc.)
	TokenCustomBase                  // Widgets define custom types starting here
)

// Token represents a parsed token from the format string
type Token struct {
	Type    TokenType
	Name    string // Token name without braces (e.g., "icon", "battery")
	Param   string // Optional parameter (e.g., "20" in {battery:20})
	Literal string // For literal tokens, the text content
}

// TokenClassifier maps a token name to its TokenType.
// Widgets provide their own classifier to categorize tokens.
type TokenClassifier func(name string) TokenType

// tokenPattern matches {token} or {token:param} in the format string.
// Compiled once at package level for efficiency.
var tokenPattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?\}`)

// ParseFormatTokens parses a format string into tokens using the provided classifier
// to determine each token's type. The classifier is called for every {name} or {name:param}
// match; literal text between tokens is emitted as TokenLiteral.
func ParseFormatTokens(format string, classify TokenClassifier) []Token {
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

		tokenType := classify(name)
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
