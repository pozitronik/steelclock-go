package bluetooth

import (
	"github.com/pozitronik/steelclock-go/internal/shared/render"
)

// TokenShape extends the shared token type for bluetooth-specific shape tokens
const TokenShape = render.TokenCustomBase

// parseBluetoothFormat parses a format string into tokens
func parseBluetoothFormat(format string) []render.Token {
	return render.ParseFormatTokens(format, getBluetoothTokenType)
}

// getBluetoothTokenType classifies a token name into its type
func getBluetoothTokenType(name string) render.TokenType {
	switch name {
	case "icon":
		return render.TokenIcon
	case "name", "level", "state":
		return render.TokenText
	case "battery", "battery_h", "battery_v", "bar", "bar_h", "bar_v":
		return TokenShape
	default:
		return render.TokenLiteral
	}
}

// findBlinkTarget returns the index of the token that should blink for low battery.
// Priority: first shape token -> first icon token -> first name token.
// Returns -1 if no suitable target is found.
func findBlinkTarget(tokens []render.Token) int {
	firstIcon := -1
	firstName := -1

	for i, t := range tokens {
		switch t.Type {
		case TokenShape:
			return i // Shape tokens have highest priority
		case render.TokenIcon:
			if firstIcon < 0 {
				firstIcon = i
			}
		case render.TokenText:
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
