package glyphs

// TelegramIconsDot contains simplified dot icon for very small displays
var TelegramIconsDot = &GlyphSet{
	Name:        "telegram_dot",
	GlyphWidth:  7,
	GlyphHeight: 7,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		"telegram": {
			Width: 7, Height: 7,
			Data: [][]bool{
				{false, false, false, false, false, true, false},
				{false, false, false, false, true, true, true},
				{false, false, false, false, false, true, false},
				{false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false},
			},
		},
	},
}

// TelegramIcons8x8 contains simplified Telegram paper airplane icon
var TelegramIcons8x8 = &GlyphSet{
	Name:        "telegram_8x8",
	GlyphWidth:  8,
	GlyphHeight: 8,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Paper airplane with basic shape preserved
		"telegram": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, true, true},
				{false, false, false, true, true, true, true, true},
				{false, true, true, true, true, false, true, true},
				{true, true, true, true, false, true, true, true},
				{false, false, false, false, true, true, true, false},
				{false, false, false, false, false, true, true, false},
				{false, false, false, false, false, false, true, false},
			},
		},
	},
}

// TelegramIcons12x12 contains Telegram paper airplane icon
var TelegramIcons12x12 = &GlyphSet{
	Name:        "telegram_12x12",
	GlyphWidth:  12,
	GlyphHeight: 12,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Paper airplane with fold line visible
		"telegram": {
			Width: 12, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, true, true},
				{false, false, false, false, false, false, false, false, true, true, true, true},
				{false, false, false, false, false, false, true, true, true, true, true, true},
				{false, false, false, true, true, true, true, true, false, true, true, true},
				{false, true, true, true, true, true, false, false, true, true, true, true},
				{true, true, true, true, true, false, false, true, true, true, true, false},
				{false, false, false, false, false, false, true, true, true, true, true, false},
				{false, false, false, false, false, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}

// TelegramIcons16x16 contains Telegram paper airplane icon with more detail
var TelegramIcons16x16 = &GlyphSet{
	Name:        "telegram_16x16",
	GlyphWidth:  16,
	GlyphHeight: 16,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Paper airplane with fold detail
		"telegram": {
			Width: 16, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true},
				{false, false, false, false, false, false, true, true, true, true, true, false, true, true, true, false},
				{false, false, false, true, true, true, true, true, true, false, false, true, true, true, true, false},
				{false, true, true, true, true, true, true, true, false, false, true, true, true, true, true, false},
				{true, true, true, true, true, true, false, false, false, true, true, true, true, true, true, false},
				{true, true, true, true, true, false, false, false, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, false, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}

// TelegramIcons24x24 contains detailed Telegram paper airplane icon
var TelegramIcons24x24 = &GlyphSet{
	Name:        "telegram_24x24",
	GlyphWidth:  24,
	GlyphHeight: 24,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Detailed paper airplane with fold line visible
		"telegram": {
			Width: 24, Height: 24,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, false, false, true, true, true, true, true, false},
				{false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, false},
				{false, false, false, true, true, true, true, true, true, true, true, true, true, false, false, false, true, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, true, true, true, true, true, false, false, false, true, true, true, true, true, true, true, true, false},
				{true, true, true, true, true, true, true, true, true, true, false, false, false, false, true, true, true, true, true, true, true, true, true, false},
				{false, true, true, true, true, true, true, true, true, false, false, false, false, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, true, true, true, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}

// TelegramIcons32x32 contains large detailed Telegram paper airplane icon
var TelegramIcons32x32 = &GlyphSet{
	Name:        "telegram_32x32",
	GlyphWidth:  32,
	GlyphHeight: 32,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Large paper airplane with fold line and inner detail
		"telegram": {
			Width: 32, Height: 32,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, false, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, false, false, true, true, true, true, true, true, true, true, true, false},
				{false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{true, true, true, true, true, true, true, true, true, true, true, true, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, true, true, true, true, true, true, true, true, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, true, true, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, true, true, true, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, true, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, true, true, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, true, true, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, true, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
			},
		},
	},
}

// GetTelegramIcon returns the appropriate Telegram icon glyph set for the given size
func GetTelegramIcon(size int) *GlyphSet {
	switch {
	case size >= 32:
		return TelegramIcons32x32
	case size >= 24:
		return TelegramIcons24x24
	case size >= 16:
		return TelegramIcons16x16
	case size >= 12:
		return TelegramIcons12x12
	case size >= 9:
		return TelegramIcons8x8
	default:
		return TelegramIconsDot
	}
}
