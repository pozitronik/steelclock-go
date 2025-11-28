package glyphs

// CommonIcons12x12 contains warning icon optimized for proper triangle shape
var CommonIcons12x12 = &GlyphSet{
	Name:        "common_12x12",
	GlyphWidth:  12,
	GlyphHeight: 12,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Warning triangle with exclamation mark (12 wide x 10 tall)
		"warning": {
			Width: 12, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, true, true, false, false, false, false, false}, // Верхушка
				{false, false, false, false, true, false, false, true, false, false, false, false},
				{false, false, false, true, false, true, true, false, true, false, false, false}, // Начало знака
				{false, false, false, true, false, true, true, false, true, false, false, false},
				{false, false, true, false, false, true, true, false, false, true, false, false},
				{false, false, true, false, false, false, false, false, false, true, false, false}, // Разрыв
				{false, true, false, false, false, true, true, false, false, false, true, false},   // Точка
				{true, false, false, false, false, false, false, false, false, false, false, true},
				{true, true, true, true, true, true, true, true, true, true, true, true}, // Основание
				{false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}

// CommonIcons16x16 contains warning icon optimized for proper triangle shape
var CommonIcons16x16 = &GlyphSet{
	Name:        "common_16x16",
	GlyphWidth:  16,
	GlyphHeight: 16,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"warning": {
			Width: 16, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, false}, // Верхушка
				{false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false},
				{false, false, false, false, false, true, false, false, false, false, true, false, false, false, false, false},
				{false, false, false, false, false, true, false, true, true, false, true, false, false, false, false, false}, // Палка
				{false, false, false, false, true, false, false, true, true, false, false, true, false, false, false, false},
				{false, false, false, false, true, false, false, true, true, false, false, true, false, false, false, false},
				{false, false, false, true, false, false, false, true, true, false, false, false, true, false, false, false},
				{false, false, false, true, false, false, false, true, true, false, false, false, true, false, false, false},
				{false, false, true, false, false, false, false, true, true, false, false, false, false, true, false, false},
				{false, false, true, false, false, false, false, false, false, false, false, false, false, true, false, false}, // Разрыв
				{false, true, false, false, false, false, false, true, true, false, false, false, false, false, true, false},   // Точка
				{false, true, false, false, false, false, false, true, true, false, false, false, false, false, true, false},
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true}, // Основание
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}

// CommonIcons24x24 contains warning icon optimized for proper triangle shape
var CommonIcons24x24 = &GlyphSet{
	Name:        "common_24x24",
	GlyphWidth:  24,
	GlyphHeight: 24,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"warning": {
			Width: 24, Height: 24,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, false, false, false, false, false, false, false, false, false, false}, // Верхушка
				{false, false, false, false, false, false, false, false, false, true, true, false, false, true, true, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, true, true, false, false, true, true, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, true, true, false, false, false, false, true, true, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, true, true, false, false, true, true, false, false, true, true, false, false, false, false, false, false, false}, // Начало !
				{false, false, false, false, false, false, false, true, true, false, false, true, true, false, false, true, true, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, true, true, false, false, false, true, true, false, false, false, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, false, true, true, false, false, false, true, true, false, false, false, true, true, false, false, false, false, false, false},
				{false, false, false, false, false, true, true, false, false, false, false, true, true, false, false, false, false, true, true, false, false, false, false, false},
				{false, false, false, false, false, true, true, false, false, false, false, true, true, false, false, false, false, true, true, false, false, false, false, false},
				{false, false, false, false, true, true, false, false, false, false, false, true, true, false, false, false, false, false, true, true, false, false, false, false},
				{false, false, false, false, true, true, false, false, false, false, false, true, true, false, false, false, false, false, true, true, false, false, false, false},
				{false, false, false, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, false, false, false}, // Разрыв
				{false, false, false, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, false, false, false},
				{false, false, true, true, false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, false, true, true, false, false}, // Точка
				{false, false, true, true, false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, false, true, true, false, false},
				{false, true, true, false, false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, false, false, true, true, false},
				{false, true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, false},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true}, // Основание
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}
