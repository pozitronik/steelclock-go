package glyphs

// WinampIcons8x8 contains 8x8 pixel Winamp playback icons
//
//goland:noinspection DuplicatedCode,DuplicatedCode
var WinampIcons8x8 = &GlyphSet{
	Name:        "winamp_8x8",
	GlyphWidth:  8,
	GlyphHeight: 8,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Play icon (triangle pointing right)
		"play": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, true, false, false, false, false, false, false},
				{false, true, true, false, false, false, false, false},
				{false, true, true, true, false, false, false, false},
				{false, true, true, true, true, false, false, false},
				{false, true, true, true, true, false, false, false},
				{false, true, true, true, false, false, false, false},
				{false, true, true, false, false, false, false, false},
				{false, true, false, false, false, false, false, false},
			},
		},
		// Pause icon (two vertical bars)
		"pause": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
			},
		},
		// Stop icon (square)
		"stop": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false},
			},
		},
	},
}
