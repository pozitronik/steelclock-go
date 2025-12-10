package render

import "strings"

// TokenFormatter provides template-based string formatting with named tokens.
// Tokens in the template are replaced with their values.
// Default token format is {tokenName}, but can be customized.
type TokenFormatter struct {
	tokens map[string]string
	prefix string
	suffix string
}

// NewTokenFormatter creates a new TokenFormatter with default {token} format
func NewTokenFormatter() *TokenFormatter {
	return &TokenFormatter{
		tokens: make(map[string]string),
		prefix: "{",
		suffix: "}",
	}
}

// NewTokenFormatterWithFormat creates a TokenFormatter with custom token format
// For example: prefix="{{", suffix="}}" for Mustache-style {{token}} format
func NewTokenFormatterWithFormat(prefix, suffix string) *TokenFormatter {
	return &TokenFormatter{
		tokens: make(map[string]string),
		prefix: prefix,
		suffix: suffix,
	}
}

// Set adds or updates a single token value. Returns self for chaining.
func (f *TokenFormatter) Set(name, value string) *TokenFormatter {
	f.tokens[name] = value
	return f
}

// SetAll adds multiple tokens from a map. Returns self for chaining.
func (f *TokenFormatter) SetAll(tokens map[string]string) *TokenFormatter {
	for name, value := range tokens {
		f.tokens[name] = value
	}
	return f
}

// Get returns the value for a token, or empty string if not found
func (f *TokenFormatter) Get(name string) string {
	return f.tokens[name]
}

// Has checks if a token is defined
func (f *TokenFormatter) Has(name string) bool {
	_, ok := f.tokens[name]
	return ok
}

// Clear removes all tokens
func (f *TokenFormatter) Clear() *TokenFormatter {
	f.tokens = make(map[string]string)
	return f
}

// Format replaces all tokens in the template with their values.
// Unknown tokens are left unchanged.
func (f *TokenFormatter) Format(template string) string {
	if template == "" {
		return ""
	}

	result := template
	for name, value := range f.tokens {
		token := f.prefix + name + f.suffix
		result = strings.ReplaceAll(result, token, value)
	}
	return result
}

// FormatStrict replaces tokens and removes any unmatched tokens from the result.
// This is useful when you want to ensure no {token} patterns remain in output.
func (f *TokenFormatter) FormatStrict(template string) string {
	result := f.Format(template)
	// Remove any remaining tokens that weren't replaced
	return f.removeUnmatchedTokens(result)
}

// removeUnmatchedTokens removes any token patterns that weren't replaced
func (f *TokenFormatter) removeUnmatchedTokens(s string) string {
	result := s
	for {
		startIdx := strings.Index(result, f.prefix)
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(result[startIdx:], f.suffix)
		if endIdx == -1 {
			break
		}
		// Remove the token
		result = result[:startIdx] + result[startIdx+endIdx+len(f.suffix):]
	}
	return result
}

// Count returns the number of defined tokens
func (f *TokenFormatter) Count() int {
	return len(f.tokens)
}

// Clone creates a copy of the formatter with all current tokens
func (f *TokenFormatter) Clone() *TokenFormatter {
	clone := &TokenFormatter{
		tokens: make(map[string]string, len(f.tokens)),
		prefix: f.prefix,
		suffix: f.suffix,
	}
	for k, v := range f.tokens {
		clone.tokens[k] = v
	}
	return clone
}
