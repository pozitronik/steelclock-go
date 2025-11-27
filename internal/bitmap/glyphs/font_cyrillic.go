package glyphs

// init adds Cyrillic glyphs to Font5x7 and Font3x5
func init() {
	// Add Cyrillic glyphs to Font5x7
	for r, g := range cyrillic5x7 {
		Font5x7.Glyphs[r] = g
	}

	// Add Cyrillic glyphs to Font3x5
	for r, g := range cyrillic3x5 {
		Font3x5.Glyphs[r] = g
	}
}

// cyrillic5x7 contains Russian Cyrillic letters for 5x7 font
// Includes Ukrainian (Ґ, Є, І, Ї) and Belarusian (Ў) extensions
var cyrillic5x7 = map[rune]*Glyph{
	// Russian uppercase А-Я

	'А': { // A
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'Б': { // Be
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, false},
		},
	},
	'В': { // Ve
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, false},
		},
	},
	'Г': { // Ge
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	'Д': { // De
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{true, true, true, true, true},
			{true, false, false, false, true},
		},
	},
	'Е': { // Ye
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, true},
		},
	},
	'Ё': { // Yo
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, true},
		},
	},
	'Ж': { // Zhe
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, true, false, true},
			{true, false, true, false, true},
			{false, true, true, true, false},
			{false, false, true, false, false},
			{false, true, true, true, false},
			{true, false, true, false, true},
			{true, false, true, false, true},
		},
	},
	'З': { // Ze
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, true, true, false},
			{false, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'И': { // I
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, true, true},
			{true, false, true, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'Й': { // Short I
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{true, false, false, false, true},
			{true, false, false, true, true},
			{true, false, true, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'К': { // Ka
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, true, false},
			{true, false, true, false, false},
			{true, true, false, false, false},
			{true, false, true, false, false},
			{true, false, false, true, false},
			{true, false, false, false, true},
		},
	},
	'Л': { // El
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, true, true},
			{false, true, false, false, true},
			{false, true, false, false, true},
			{false, true, false, false, true},
			{false, true, false, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
		},
	},
	'М': { // Em
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, true, false, true, true},
			{true, false, true, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'Н': { // En
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'О': { // O
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'П': { // Pe
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'Р': { // Er
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	'С': { // Es
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'Т': { // Te
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	'У': { // U
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'Ф': { // Ef
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, true, true, true, false},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{false, true, true, true, false},
			{false, false, true, false, false},
		},
	},
	'Х': { // Kha
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, false, true, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'Ц': { // Tse
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, true, true, true, true},
			{false, false, false, false, true},
		},
	},
	'Ч': { // Che
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
		},
	},
	'Ш': { // Sha
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, true, true, true, true},
		},
	},
	'Щ': { // Shcha
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, true, true, true, true},
			{false, false, false, false, true},
		},
	},
	'Ъ': { // Hard sign
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, true, true, true, false},
			{false, true, false, false, true},
			{false, true, false, false, true},
			{false, true, true, true, false},
		},
	},
	'Ы': { // Yery
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, false, true},
			{true, false, false, true, true},
			{true, false, false, false, true},
			{true, true, true, false, true},
		},
	},
	'Ь': { // Soft sign
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, false},
		},
	},
	'Э': { // E reversed
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, true, true, true},
			{false, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'Ю': { // Yu
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, true, false},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, true, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, false, true, false},
		},
	},
	'Я': { // Ya
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, true, false, true},
			{false, true, false, false, true},
			{true, false, false, false, true},
		},
	},

	// Russian lowercase а-я (traditional typography)
	// X-height letters use rows 2-6, ascenders use rows 0-6, descenders use rows 1-6

	'а': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{false, false, false, false, true},
			{false, true, true, true, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
		},
	},
	'б': { // ascender - full height (has top hook)
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, true},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'в': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, true, true, true, false},
		},
	},
	'г': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	'д': { // descender - has legs at bottom
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, true, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{true, true, true, true, true},
			{true, false, false, false, true},
		},
	},
	'е': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{false, true, true, true, false},
		},
	},
	'ё': { // special - diacritics above
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{false, true, true, true, false},
		},
	},
	'ж': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, true, false, true},
			{false, true, true, true, false},
			{false, false, true, false, false},
			{false, true, true, true, false},
			{true, false, true, false, true},
		},
	},
	'з': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{false, false, false, false, true},
			{false, false, true, true, false},
			{false, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'и': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, false, false, true, true},
			{true, false, true, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
		},
	},
	'й': { // special - breve above
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, false, false, true, true},
			{true, false, true, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
		},
	},
	'к': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, true, false},
			{true, false, true, false, false},
			{true, true, false, false, false},
			{true, false, true, false, false},
			{true, false, false, true, false},
		},
	},
	'л': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, true, true, true},
			{false, true, false, false, true},
			{false, true, false, false, true},
			{true, true, false, false, true},
			{true, false, false, false, true},
		},
	},
	'м': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, true, false, true, true},
			{true, false, true, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'н': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'о': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'п': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
		},
	},
	'р': { // descender
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	'с': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, false},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'т': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	'у': { // descender
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'ф': { // ascender - full height (tall structure)
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, true, true, true, false},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{false, true, true, true, false},
			{false, false, true, false, false},
		},
	},
	'х': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{false, true, false, true, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
		},
	},
	'ц': { // descender - hook at bottom
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, true, true, true, true},
			{false, false, false, false, true},
		},
	},
	'ч': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
		},
	},
	'ш': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, true, true, true, true},
		},
	},
	'щ': { // descender - hook at bottom
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, false, true, false, true},
			{true, true, true, true, true},
			{false, false, false, false, true},
		},
	},
	'ъ': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, false, false, false},
			{false, true, false, false, false},
			{false, true, true, true, false},
			{false, true, false, false, true},
			{false, true, true, true, false},
		},
	},
	'ы': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, true},
			{true, true, true, false, true},
			{true, false, false, true, true},
			{true, false, false, false, true},
			{true, true, true, false, true},
		},
	},
	'ь': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{true, true, true, true, false},
		},
	},
	'э': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{false, false, false, false, true},
			{false, false, true, true, true},
			{false, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'ю': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, false, false, true, false},
			{true, false, true, false, true},
			{true, true, true, false, true},
			{true, false, true, false, true},
			{true, false, false, true, false},
		},
	},
	'я': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, true, false, true},
			{false, true, false, false, true},
		},
	},

	// Ukrainian extensions

	'Ґ': { // Ge with upturn
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, true, false},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	'ґ': { // x-height with upturn
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, true, false},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	'Є': { // Ukrainian Ye
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{true, false, false, false, true},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, false},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'є': { // x-height
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, false},
			{false, true, true, true, false},
		},
	},
	'І': { // Ukrainian I
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, true, true, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, true, true, false},
		},
	},
	'і': { // Ukrainian i
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, false, false, false, false},
			{false, true, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, true, true, false},
		},
	},
	'Ї': { // Yi
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, true, true, true, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, true, true, false},
		},
	},
	'ї': { // yi
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, false, false, false, false},
			{false, true, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, true, true, false},
		},
	},

	// Belarusian extension

	'Ў': { // Short U
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, false},
		},
	},
	'ў': { // short u
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, true, true, true, true},
			{false, false, false, false, true},
			{false, true, true, true, false},
		},
	},
}

// cyrillic3x5 contains Russian Cyrillic letters for 3x5 font
// Includes Ukrainian (Ґ, Є, І, Ї) and Belarusian (Ў) extensions
var cyrillic3x5 = map[rune]*Glyph{
	// Russian uppercase А-Я (compact 3x5)

	'А': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{true, true, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'Б': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{true, false, false},
			{true, true, false},
			{true, false, true},
			{true, true, false},
		},
	},
	'В': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{true, false, true},
			{true, true, false},
			{true, false, true},
			{true, true, false},
		},
	},
	'Г': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{true, false, false},
			{true, false, false},
			{true, false, false},
			{true, false, false},
		},
	},
	'Д': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'Е': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{true, false, false},
			{true, true, false},
			{true, false, false},
			{true, true, true},
		},
	},
	'Ё': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, true},
			{true, true, false},
			{true, false, false},
			{true, true, true},
		},
	},
	'Ж': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{true, false, true},
		},
	},
	'З': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{false, false, true},
			{false, true, false},
			{false, false, true},
			{true, true, false},
		},
	},
	'И': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'Й': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'К': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, false},
			{true, false, false},
			{true, true, false},
			{true, false, true},
		},
	},
	'Л': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'М': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'Н': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'О': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{false, true, false},
		},
	},
	'П': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'Р': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{true, false, true},
			{true, true, false},
			{true, false, false},
			{true, false, false},
		},
	},
	'С': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{true, false, false},
			{true, false, false},
			{true, false, false},
			{false, true, true},
		},
	},
	'Т': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{false, true, false},
		},
	},
	'У': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{false, true, true},
			{false, false, true},
			{false, true, false},
		},
	},
	'Ф': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{true, true, true},
			{true, true, true},
			{false, true, false},
		},
	},
	'Х': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{false, true, false},
			{true, false, true},
			{true, false, true},
		},
	},
	'Ц': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{false, false, true},
		},
	},
	'Ч': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{false, true, true},
			{false, false, true},
			{false, false, true},
		},
	},
	'Ш': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, true, true},
		},
	},
	'Щ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, true, true},
			{false, false, true},
		},
	},
	'Ъ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{false, true, false},
			{false, true, true},
			{false, true, true},
			{false, true, true},
		},
	},
	'Ы': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, true, true},
		},
	},
	'Ь': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, false, false},
			{true, true, false},
			{true, false, true},
			{true, true, false},
		},
	},
	'Э': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, false},
			{false, false, true},
			{false, true, true},
			{false, false, true},
			{true, true, false},
		},
	},
	'Ю': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'Я': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{true, false, true},
			{false, true, true},
			{false, true, true},
			{true, false, true},
		},
	},

	// Russian lowercase а-я (3x5)
	// X-height letters use rows 1-4, ascenders use rows 0-4, descenders have tail in row 4

	'а': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, false, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'б': { // ascender - full height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{true, false, false},
			{true, true, false},
			{true, false, true},
			{true, true, false},
		},
	},
	'в': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{true, false, true},
			{true, true, false},
			{true, true, false},
		},
	},
	'г': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, true},
			{true, false, false},
			{true, false, false},
			{true, false, false},
		},
	},
	'д': { // descender - has legs
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, false, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'е': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, true, true},
			{true, false, false},
			{false, true, false},
		},
	},
	'ё': { // special - diacritics above
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{false, true, false},
			{true, true, true},
			{true, false, false},
			{false, true, false},
		},
	},
	'ж': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{false, true, false},
			{true, true, true},
			{true, false, true},
		},
	},
	'з': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{false, false, true},
			{false, true, false},
			{true, true, false},
		},
	},
	'и': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'й': { // special - breve above
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'к': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, true, false},
			{true, true, false},
			{true, false, true},
		},
	},
	'л': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'м': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, true, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'н': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, true, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'о': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, false, true},
			{true, false, true},
			{false, true, false},
		},
	},
	'п': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
		},
	},
	'р': { // descender
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{true, false, true},
			{true, true, false},
			{true, false, false},
		},
	},
	'с': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, true},
			{true, false, false},
			{true, false, false},
			{false, true, true},
		},
	},
	'т': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, true},
			{false, true, false},
			{false, true, false},
			{false, true, false},
		},
	},
	'у': { // descender
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, false, true},
			{false, true, true},
			{false, true, false},
		},
	},
	'ф': { // ascender - full height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{true, true, true},
			{true, true, true},
			{false, true, false},
		},
	},
	'х': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{false, true, false},
			{false, true, false},
			{true, false, true},
		},
	},
	'ц': { // descender - hook at bottom
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{false, false, true},
		},
	},
	'ч': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, false, true},
			{false, true, true},
			{false, false, true},
		},
	},
	'ш': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, false, true},
			{true, true, true},
			{true, true, true},
		},
	},
	'щ': { // descender - hook at bottom
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{false, false, true},
		},
	},
	'ъ': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{false, true, false},
			{false, true, true},
			{false, true, true},
		},
	},
	'ы': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, true, true},
		},
	},
	'ь': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, false},
			{true, true, false},
			{true, false, true},
			{true, true, false},
		},
	},
	'э': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{false, false, true},
			{false, true, true},
			{true, true, false},
		},
	},
	'ю': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, false, true},
			{true, true, true},
			{true, true, true},
			{true, false, true},
		},
	},
	'я': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, true},
			{true, false, true},
			{false, true, true},
			{true, false, true},
		},
	},

	// Ukrainian extensions (3x5)

	'Ґ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{true, true, true},
			{true, false, false},
			{true, false, false},
			{true, false, false},
		},
	},
	'ґ': { // x-height with upturn
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, false, true},
			{true, true, true},
			{true, false, false},
			{true, false, false},
		},
	},
	'Є': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{true, false, false},
			{true, true, false},
			{true, false, false},
			{false, true, true},
		},
	},
	'є': { // x-height
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, true},
			{true, true, false},
			{true, false, false},
			{false, true, true},
		},
	},
	'І': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{true, true, true},
		},
	},
	'і': { // special - dot above
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{false, false, false},
			{false, true, false},
			{false, true, false},
			{false, true, false},
		},
	},
	'Ї': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{true, true, true},
		},
	},
	'ї': { // special - dots above
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{false, false, false},
			{false, true, false},
			{false, true, false},
			{false, true, false},
		},
	},

	// Belarusian extension (3x5)

	'Ў': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{false, true, true},
			{false, false, true},
			{false, true, false},
		},
	},
	'ў': { // special - breve above, descender
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, false, true},
			{true, false, true},
			{false, true, true},
			{false, true, false},
		},
	},
}
