package glyphs

// CommonIcons8x8 contains 8×8 pixel common UI icons
var CommonIcons8x8 = &GlyphSet{
	Name:        "common_8x8",
	GlyphWidth:  8,
	GlyphHeight: 8,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"warning": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, false, false, true, true, false, false, false},
				{false, false, true, true, true, true, false, false},
				{false, true, true, false, false, true, true, false},
				{false, true, true, false, false, true, true, false},
				{true, true, true, false, false, true, true, true},
				{true, true, true, true, true, true, true, true},
				{true, true, true, false, false, true, true, true},
				{true, true, true, true, true, true, true, true},
			},
		},
	},
}

// CommonIcons10x10 contains 10×10 pixel common UI icons
var CommonIcons10x10 = &GlyphSet{
	Name:        "common_10x10",
	GlyphWidth:  10,
	GlyphHeight: 10,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"warning": {
			Width: 10, Height: 10,
			Data: [][]bool{
				{false, false, false, false, true, true, false, false, false, false},
				{false, false, false, true, true, true, true, false, false, false},
				{false, false, true, true, true, true, true, true, false, false},
				{false, true, true, true, false, false, true, true, true, false},
				{false, true, true, true, false, false, true, true, true, false},
				{true, true, true, true, false, false, true, true, true, true},
				{true, true, true, true, false, false, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true},
				{true, true, true, true, false, false, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true},
			},
		},
	},
}

// CommonIcons12x12 contains 12×12 pixel common UI icons
var CommonIcons12x12 = &GlyphSet{
	Name:        "common_12x12",
	GlyphWidth:  12,
	GlyphHeight: 12,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"warning": {
			Width: 12, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, true, true, false, false, false, false, false},
				{false, false, false, false, true, true, true, true, false, false, false, false},
				{false, false, false, true, true, true, true, true, true, false, false, false},
				{false, false, true, true, true, true, true, true, true, true, false, false},
				{false, true, true, true, true, false, false, true, true, true, true, false},
				{false, true, true, true, true, false, false, true, true, true, true, false},
				{true, true, true, true, true, false, false, true, true, true, true, true},
				{true, true, true, true, true, false, false, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true},
				{true, true, true, true, true, false, false, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true},
			},
		},
	},
}

// CommonIcons16x16 contains 16×16 pixel common UI icons
var CommonIcons16x16 = &GlyphSet{
	Name:        "common_16x16",
	GlyphWidth:  16,
	GlyphHeight: 16,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"warning": {
			Width: 16, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, true, true, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, true, true, true, true, true, true, false, false, false, false, false},
				{false, false, false, false, true, true, true, true, true, true, true, true, false, false, false, false},
				{false, false, false, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, true, true, true, true, true, false, false, true, true, true, true, true, false, false},
				{false, true, true, true, true, true, true, false, false, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, false, false, true, true, true, true, true, true, false},
				{true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
			},
		},
	},
}
