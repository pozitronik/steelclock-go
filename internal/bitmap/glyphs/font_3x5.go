package glyphs

// Font3x5 is a compact 3×5 pixel monospace font
// Extracted and expanded from internal/widget/doom.go
// Best for small displays where space is limited
var Font3x5 = &GlyphSet{
	Name:        "3x5",
	GlyphWidth:  3,
	GlyphHeight: 5,
	Glyphs: map[rune]*Glyph{
		// Uppercase letters A-Z
		'A': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{true, false, true},
				{true, true, true},
				{true, false, true},
				{true, false, true},
			},
		},
		'B': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, false},
				{true, false, true},
				{true, true, false},
				{true, false, true},
				{true, true, false},
			},
		},
		'C': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, true},
				{true, false, false},
				{true, false, false},
				{true, false, false},
				{false, true, true},
			},
		},
		'D': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, false},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{true, true, false},
			},
		},
		'E': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{true, false, false},
				{true, true, false},
				{true, false, false},
				{true, true, true},
			},
		},
		'F': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{true, false, false},
				{true, true, false},
				{true, false, false},
				{true, false, false},
			},
		},
		'G': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, true},
				{true, false, false},
				{true, false, true},
				{true, false, true},
				{false, true, true},
			},
		},
		'H': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{true, true, true},
				{true, false, true},
				{true, false, true},
			},
		},
		'I': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{false, true, false},
				{false, true, false},
				{false, true, false},
				{true, true, true},
			},
		},
		'J': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, true},
				{false, false, true},
				{false, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'K': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, true, false},
				{true, false, false},
				{true, true, false},
				{true, false, true},
			},
		},
		'L': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, false},
				{true, false, false},
				{true, false, false},
				{true, false, false},
				{true, true, true},
			},
		},
		'M': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, true, true},
				{true, true, true},
				{true, false, true},
				{true, false, true},
			},
		},
		'N': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, true, true},
				{true, true, true},
				{true, false, true},
				{true, false, true},
			},
		},
		'O': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'P': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, false},
				{true, false, true},
				{true, true, false},
				{true, false, false},
				{true, false, false},
			},
		},
		'Q': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{true, false, true},
				{true, false, true},
				{true, true, true},
				{false, false, true},
			},
		},
		'R': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, false},
				{true, false, true},
				{true, true, false},
				{true, false, true},
				{true, false, true},
			},
		},
		'S': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, true},
				{true, false, false},
				{false, true, false},
				{false, false, true},
				{true, true, false},
			},
		},
		'T': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{false, true, false},
				{false, true, false},
				{false, true, false},
				{false, true, false},
			},
		},
		'U': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'V': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'W': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{true, true, true},
				{true, false, true},
			},
		},
		'X': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{false, true, false},
				{true, false, true},
				{true, false, true},
			},
		},
		'Y': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{false, true, false},
				{false, true, false},
				{false, true, false},
			},
		},
		'Z': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{false, false, true},
				{false, true, false},
				{true, false, false},
				{true, true, true},
			},
		},

		// Lowercase letters a-z
		'a': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, false},
				{true, false, true},
				{true, true, true},
				{true, false, true},
			},
		},
		'd': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, true},
				{false, false, true},
				{false, true, true},
				{true, false, true},
				{false, true, true},
			},
		},
		'e': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, false},
				{true, true, true},
				{true, false, false},
				{false, true, false},
			},
		},
		'f': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, true},
				{true, false, false},
				{true, true, false},
				{true, false, false},
				{true, false, false},
			},
		},
		'g': { // descender
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, true},
				{true, false, true},
				{false, true, true},
				{false, false, true},
			},
		},
		'i': { // special - dot above
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{false, false, false},
				{false, true, false},
				{false, true, false},
				{false, true, false},
			},
		},
		'l': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{false, true, false},
				{false, true, false},
				{false, true, false},
				{false, true, false},
			},
		},
		'n': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, true, false},
				{true, false, true},
				{true, false, true},
				{true, false, true},
			},
		},
		'o': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, false},
				{true, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'b': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, false},
				{true, false, false},
				{true, true, false},
				{true, false, true},
				{true, true, false},
			},
		},
		'c': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, true},
				{true, false, false},
				{true, false, false},
				{false, true, true},
			},
		},
		'h': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, false},
				{true, false, false},
				{true, true, false},
				{true, false, true},
				{true, false, true},
			},
		},
		'j': { // descender with dot
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, true},
				{false, false, false},
				{false, false, true},
				{false, false, true},
				{false, true, false},
			},
		},
		'k': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, false},
				{true, false, false},
				{true, false, true},
				{true, true, false},
				{true, false, true},
			},
		},
		'm': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{true, true, true},
				{true, false, true},
				{true, false, true},
			},
		},
		'p': { // descender
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, true, false},
				{true, false, true},
				{true, true, false},
				{true, false, false},
			},
		},
		'q': { // descender
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, true},
				{true, false, true},
				{false, true, true},
				{false, false, true},
			},
		},
		'r': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{true, true, false},
				{true, false, false},
				{true, false, false},
			},
		},
		's': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, true},
				{true, false, false},
				{false, false, true},
				{true, true, false},
			},
		},
		't': { // ascender - full height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{true, true, true},
				{false, true, false},
				{false, true, false},
				{false, false, true},
			},
		},
		'u': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{false, true, true},
			},
		},
		'v': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'w': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{true, false, true},
				{true, true, true},
				{true, false, true},
			},
		},
		'x': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{false, true, false},
				{false, true, false},
				{true, false, true},
			},
		},
		'y': { // descender
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, false, true},
				{true, false, true},
				{false, true, true},
				{false, true, false},
			},
		},
		'z': { // x-height
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{true, true, true},
				{false, false, true},
				{true, false, false},
				{true, true, true},
			},
		},

		// Digits 0-9
		'0': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{true, false, true},
				{true, false, true},
				{true, false, true},
				{false, true, false},
			},
		},
		'1': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{true, true, false},
				{false, true, false},
				{false, true, false},
				{true, true, true},
			},
		},
		'2': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, false},
				{false, false, true},
				{false, true, false},
				{true, false, false},
				{true, true, true},
			},
		},
		'3': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{false, false, true},
				{false, true, true},
				{false, false, true},
				{true, true, true},
			},
		},
		'4': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{true, false, true},
				{true, true, true},
				{false, false, true},
				{false, false, true},
			},
		},
		'5': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{true, false, false},
				{true, true, true},
				{false, false, true},
				{true, true, true},
			},
		},
		'6': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, true},
				{true, false, false},
				{true, true, true},
				{true, false, true},
				{true, true, true},
			},
		},
		'7': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{false, false, true},
				{false, true, false},
				{false, true, false},
				{false, true, false},
			},
		},
		'8': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{true, false, true},
				{true, true, true},
				{true, false, true},
				{true, true, true},
			},
		},
		'9': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, true, true},
				{true, false, true},
				{true, true, true},
				{false, false, true},
				{true, true, false},
			},
		},

		// Basic punctuation and symbols
		' ': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, false, false},
				{false, false, false},
				{false, false, false},
				{false, false, false},
			},
		},
		'!': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, true, false},
				{false, true, false},
				{false, true, false},
				{false, false, false},
				{false, true, false},
			},
		},
		'%': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{true, false, true},
				{false, false, true},
				{false, true, false},
				{true, false, false},
				{true, false, true},
			},
		},
		'.': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, false, false},
				{false, false, false},
				{false, false, false},
				{false, true, false},
			},
		},
		',': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, false, false},
				{false, false, false},
				{false, true, false},
				{true, false, false},
			},
		},
		':': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, true, false},
				{false, false, false},
				{false, true, false},
				{false, false, false},
			},
		},
		'-': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, false},
				{false, false, false},
				{true, true, true},
				{false, false, false},
				{false, false, false},
			},
		},
		'/': {
			Width: 3, Height: 5,
			Data: [][]bool{
				{false, false, true},
				{false, false, true},
				{false, true, false},
				{true, false, false},
				{true, false, false},
			},
		},
	},
	Icons: nil,
}
