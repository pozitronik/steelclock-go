package glyphs

// WinampIcons8x8 contains 8×8 pixel Winamp icons
var WinampIcons8x8 = &GlyphSet{
	Name:        "winamp_8x8",
	GlyphWidth:  8,
	GlyphHeight: 8,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Classic Winamp lightning bolt logo
		"winamp": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, false, false, false, true, true, false, false},
				{false, false, false, true, true, false, false, false},
				{false, false, true, true, true, true, false, false},
				{false, true, true, true, true, false, false, false},
				{false, false, false, true, true, true, true, false},
				{false, false, false, false, true, true, false, false},
				{false, false, false, true, true, false, false, false},
				{false, false, true, true, false, false, false, false},
			},
		},
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

// WinampIcons10x10 contains 10×10 pixel Winamp icons
var WinampIcons10x10 = &GlyphSet{
	Name:        "winamp_10x10",
	GlyphWidth:  10,
	GlyphHeight: 10,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Classic Winamp lightning bolt logo
		"winamp": {
			Width: 10, Height: 10,
			Data: [][]bool{
				{false, false, false, false, false, true, true, false, false, false},
				{false, false, false, false, true, true, true, false, false, false},
				{false, false, false, true, true, true, false, false, false, false},
				{false, false, true, true, true, true, true, false, false, false},
				{false, true, true, true, true, true, true, true, false, false},
				{false, false, false, true, true, true, true, true, true, false},
				{false, false, false, false, true, true, true, true, false, false},
				{false, false, false, false, false, true, true, true, false, false},
				{false, false, false, false, true, true, true, false, false, false},
				{false, false, false, true, true, true, false, false, false, false},
			},
		},
	},
}

// WinampIcons12x12 contains 12×12 pixel Winamp icons
var WinampIcons12x12 = &GlyphSet{
	Name:        "winamp_12x12",
	GlyphWidth:  12,
	GlyphHeight: 12,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Classic Winamp lightning bolt logo
		"winamp": {
			Width: 12, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, true, true, false, false, false, false},
				{false, false, false, false, false, true, true, true, false, false, false, false},
				{false, false, false, false, true, true, true, true, false, false, false, false},
				{false, false, false, true, true, true, true, false, false, false, false, false},
				{false, false, true, true, true, true, true, true, true, false, false, false},
				{false, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, true, true, true, true, true, true, true, true, true, false},
				{false, false, false, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, true, true, true, false, false, false},
				{false, false, false, false, false, true, true, true, false, false, false, false},
				{false, false, false, false, true, true, true, false, false, false, false, false},
			},
		},
	},
}

// WinampIcons16x16 contains 16×16 pixel Winamp icons
var WinampIcons16x16 = &GlyphSet{
	Name:        "winamp_16x16",
	GlyphWidth:  16,
	GlyphHeight: 16,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Classic Winamp lightning bolt logo
		"winamp": {
			Width: 16, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, true, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, false, true, true, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, true, true, true, true, true, false, false, false, false, false, false},
				{false, false, false, false, true, true, true, true, true, false, false, false, false, false, false, false},
				{false, false, false, true, true, true, true, true, true, true, true, false, false, false, false, false},
				{false, false, true, true, true, true, true, true, true, true, true, true, false, false, false, false},
				{false, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, true, true, true, true, true, true, false, false, false, false},
				{false, false, false, false, false, false, false, true, true, true, true, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, true, true, true, false, false, false, false, false},
				{false, false, false, false, false, false, false, true, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, false, true, true, true, false, false, false, false, false, false, false},
			},
		},
	},
}

// GetWinampIconSet returns the appropriate icon set for the given size
func GetWinampIconSet(height int) *GlyphSet {
	switch {
	case height >= 16:
		return WinampIcons16x16
	case height >= 12:
		return WinampIcons12x12
	case height >= 10:
		return WinampIcons10x10
	default:
		return WinampIcons8x8
	}
}
