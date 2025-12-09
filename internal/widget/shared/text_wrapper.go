package shared

import (
	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"golang.org/x/image/font"
)

// TextWrapMode defines how text should be wrapped
type TextWrapMode string

const (
	// WrapModeNormal breaks text on word boundaries (spaces, newlines)
	WrapModeNormal TextWrapMode = "normal"
	// WrapModeBreakAll breaks text at any character
	WrapModeBreakAll TextWrapMode = "break-all"
)

// TextWrapper provides font-aware text wrapping functionality
type TextWrapper struct {
	fontFace font.Face
	fontName string
	maxWidth int
	mode     TextWrapMode
}

// NewTextWrapper creates a new TextWrapper with the specified configuration
func NewTextWrapper(fontFace font.Face, fontName string, maxWidth int, mode TextWrapMode) *TextWrapper {
	if mode == "" {
		mode = WrapModeNormal
	}
	return &TextWrapper{
		fontFace: fontFace,
		fontName: fontName,
		maxWidth: maxWidth,
		mode:     mode,
	}
}

// SetMaxWidth updates the maximum width for wrapping
func (w *TextWrapper) SetMaxWidth(maxWidth int) {
	w.maxWidth = maxWidth
}

// Wrap splits text into lines that fit within maxWidth
func (w *TextWrapper) Wrap(text string) []string {
	if text == "" {
		return nil
	}

	var lines []string
	var currentLine string

	// Helper to measure text width
	measureWidth := func(s string) int {
		width, _ := bitmap.SmartMeasureText(s, w.fontFace, w.fontName)
		return width
	}

	// Helper to add a word to current line or start new line
	addWord := func(word string) {
		if currentLine == "" {
			// Check if word itself fits
			if measureWidth(word) <= w.maxWidth {
				currentLine = word
			} else {
				// Word doesn't fit - break it character by character
				for _, r := range word {
					ch := string(r)
					testLine := currentLine + ch
					if measureWidth(testLine) <= w.maxWidth {
						currentLine = testLine
					} else {
						if currentLine != "" {
							lines = append(lines, currentLine)
						}
						currentLine = ch
					}
				}
			}
		} else {
			testLine := currentLine + " " + word
			if measureWidth(testLine) <= w.maxWidth {
				currentLine = testLine
			} else {
				// Word doesn't fit on current line
				lines = append(lines, currentLine)
				// Check if word fits on new line
				if measureWidth(word) <= w.maxWidth {
					currentLine = word
				} else {
					// Break the word character by character
					currentLine = ""
					for _, r := range word {
						ch := string(r)
						testLine := currentLine + ch
						if measureWidth(testLine) <= w.maxWidth {
							currentLine = testLine
						} else {
							if currentLine != "" {
								lines = append(lines, currentLine)
							}
							currentLine = ch
						}
					}
				}
			}
		}
	}

	if w.mode == WrapModeBreakAll {
		// Break at any character
		for _, r := range text {
			if r == '\n' {
				lines = append(lines, currentLine)
				currentLine = ""
				continue
			}
			ch := string(r)
			testLine := currentLine + ch
			if measureWidth(testLine) <= w.maxWidth {
				currentLine = testLine
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = ch
			}
		}
	} else {
		// Normal word break - break on spaces and newlines
		// Split by newlines first
		paragraphs := SplitByNewlines(text)
		for i, para := range paragraphs {
			if i > 0 {
				// Add previous line before starting new paragraph
				if currentLine != "" {
					lines = append(lines, currentLine)
					currentLine = ""
				}
			}
			// Split paragraph into words
			words := SplitIntoWords(para)
			for _, word := range words {
				if word != "" {
					addWord(word)
				}
			}
		}
	}

	// Don't forget the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// TruncateWithEllipsis truncates lines that don't fit within maxHeight and adds "..." to the last visible line
func (w *TextWrapper) TruncateWithEllipsis(lines []string, maxHeight int) []string {
	if len(lines) == 0 {
		return lines
	}

	_, lineHeight := bitmap.SmartMeasureText("Ag", w.fontFace, w.fontName)
	if lineHeight == 0 {
		lineHeight = 16
	}

	maxLines := maxHeight / lineHeight
	if maxLines <= 0 {
		maxLines = 1
	}

	if len(lines) <= maxLines {
		return lines
	}

	// Truncate to maxLines and add ellipsis to last line
	result := make([]string, maxLines)
	copy(result, lines[:maxLines])
	lastLine := result[maxLines-1]

	ellipsis := "..."
	ellipsisWidth, _ := bitmap.SmartMeasureText(ellipsis, w.fontFace, w.fontName)

	// Remove characters from end until ellipsis fits
	for len(lastLine) > 0 {
		lineWidth, _ := bitmap.SmartMeasureText(lastLine+ellipsis, w.fontFace, w.fontName)
		if lineWidth <= w.maxWidth {
			break
		}
		// Remove last rune
		runes := []rune(lastLine)
		lastLine = string(runes[:len(runes)-1])
	}

	// If the line is too short, try without removing characters
	if lastLine == "" {
		testWidth, _ := bitmap.SmartMeasureText(ellipsis, w.fontFace, w.fontName)
		if testWidth <= w.maxWidth {
			lastLine = ""
		}
	}

	result[maxLines-1] = lastLine + ellipsis

	// Handle edge case where ellipsis alone doesn't fit
	if ellipsisWidth > w.maxWidth && maxLines > 1 {
		result = result[:maxLines-1]
	}

	return result
}

// MeasureLineHeight returns the height of a single line of text
func (w *TextWrapper) MeasureLineHeight() int {
	_, lineHeight := bitmap.SmartMeasureText("Ag", w.fontFace, w.fontName)
	if lineHeight == 0 {
		lineHeight = 16
	}
	return lineHeight
}

// SplitByNewlines splits text by newline characters
func SplitByNewlines(text string) []string {
	var result []string
	current := ""
	for _, r := range text {
		if r == '\n' || r == '\r' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	result = append(result, current)
	return result
}

// SplitIntoWords splits text into words by spaces and tabs
func SplitIntoWords(text string) []string {
	var words []string
	current := ""
	for _, r := range text {
		if r == ' ' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}
