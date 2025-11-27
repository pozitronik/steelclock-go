package glyphs

// init adds extra ASCII symbols to Font5x7 and Font3x5
func init() {
	// Add extra ASCII symbols to Font5x7
	for r, g := range asciiExtra5x7 {
		Font5x7.Glyphs[r] = g
	}

	// Add extra ASCII symbols to Font3x5
	for r, g := range asciiExtra3x5 {
		Font3x5.Glyphs[r] = g
	}
}

// asciiExtra5x7 contains additional ASCII symbols for 5x7 font
var asciiExtra5x7 = map[rune]*Glyph{
	// Quotation marks
	'"': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	'\'': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	'`': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, false, false},
			{false, false, true, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},

	// Math and comparison operators
	'+': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, false, false, false},
		},
	},
	'=': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	'<': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{false, true, false, false, false},
			{false, false, true, false, false},
			{false, false, false, true, false},
		},
	},
	'>': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, false, false},
			{false, false, true, false, false},
			{false, false, false, true, false},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
		},
	},
	'*': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, true, false, false},
			{true, false, true, false, true},
			{false, true, true, true, false},
			{true, false, true, false, true},
			{false, false, true, false, false},
			{false, false, false, false, false},
		},
	},

	// Special characters
	'#': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, true, false, true, false},
			{true, true, true, true, true},
			{false, true, false, true, false},
			{true, true, true, true, true},
			{false, true, false, true, false},
			{false, true, false, true, false},
		},
	},
	'$': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, true, true, true, true},
			{true, false, true, false, false},
			{false, true, true, true, false},
			{false, false, true, false, true},
			{true, true, true, true, false},
			{false, false, true, false, false},
		},
	},
	'&': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, false, false},
			{true, false, false, true, false},
			{true, false, true, false, false},
			{false, true, false, false, false},
			{true, false, true, false, true},
			{true, false, false, true, false},
			{false, true, true, false, true},
		},
	},
	'@': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, true, true, true},
			{true, false, true, false, true},
			{true, false, true, true, true},
			{true, false, false, false, false},
			{false, true, true, true, true},
		},
	},
	'^': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	'_': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
		},
	},
	'~': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, false, false, false},
			{true, false, true, false, true},
			{false, false, false, true, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	'|': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	'\\': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, false, true, false, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, false, true},
		},
	},

	// Brackets
	'[': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, true, true, true, false},
		},
	},
	']': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, true, true, true, false},
		},
	},
	'{': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, true, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, false, true, true, false},
		},
	},
	'}': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, false, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, true, true, false, false},
		},
	},
}

// asciiExtra3x5 contains additional ASCII symbols for 3x5 font
var asciiExtra3x5 = map[rune]*Glyph{
	// Quotation marks
	'"': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{false, false, false},
			{false, false, false},
			{false, false, false},
		},
	},
	'\'': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{false, true, false},
			{false, false, false},
			{false, false, false},
			{false, false, false},
		},
	},
	'`': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{false, true, false},
			{false, false, false},
			{false, false, false},
			{false, false, false},
		},
	},

	// Math and comparison operators
	'+': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{false, false, false},
		},
	},
	'=': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, true},
			{false, false, false},
			{true, true, true},
			{false, false, false},
		},
	},
	'<': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{false, true, false},
			{true, false, false},
			{false, true, false},
			{false, false, true},
		},
	},
	'>': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{false, true, false},
			{false, false, true},
			{false, true, false},
			{true, false, false},
		},
	},
	'*': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{false, true, false},
			{true, false, true},
			{false, false, false},
		},
	},

	// Special characters
	'#': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, true},
			{true, false, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'$': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{true, true, false},
			{false, true, false},
			{false, true, true},
			{true, true, false},
		},
	},
	'&': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{false, true, false},
			{true, false, true},
			{false, true, true},
		},
	},
	'@': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{true, true, true},
			{true, false, false},
			{false, true, true},
		},
	},
	'^': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{false, false, false},
			{false, false, false},
			{false, false, false},
		},
	},
	'_': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, false, false},
			{false, false, false},
			{false, false, false},
			{true, true, true},
		},
	},
	'~': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, false, true},
			{false, false, false},
			{false, false, false},
		},
	},
	'|': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{false, true, false},
		},
	},
	'\\': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, false, false},
			{false, true, false},
			{false, false, true},
			{false, false, true},
		},
	},

	// Brackets
	'[': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{true, false, false},
			{true, false, false},
			{true, false, false},
			{true, true, false},
		},
	},
	']': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{false, false, true},
			{false, false, true},
			{false, false, true},
			{false, true, true},
		},
	},
	'{': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{false, true, false},
			{true, false, false},
			{false, true, false},
			{false, true, true},
		},
	},
	'}': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{false, true, false},
			{false, false, true},
			{false, true, false},
			{true, true, false},
		},
	},

	// Additional punctuation not in base font
	';': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{false, false, false},
			{false, true, false},
			{true, false, false},
		},
	},
	'?': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{false, false, true},
			{false, true, false},
			{false, false, false},
			{false, true, false},
		},
	},
	'(': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{false, false, true},
		},
	},
	')': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{true, false, false},
		},
	},
}
