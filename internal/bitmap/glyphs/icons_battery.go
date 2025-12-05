package glyphs

// BatteryIcons8x8 contains battery status icons at 8x8 resolution
var BatteryIcons8x8 = &GlyphSet{
	Name:        "battery_8x8",
	GlyphWidth:  8,
	GlyphHeight: 8,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Lightning bolt for charging (7x8)
		"charging": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{false, false, false, false, true, true, false}, // Row 0:     ##
				{false, false, false, true, true, false, false}, // Row 1:    ##
				{false, false, true, true, false, false, false}, // Row 2:   ##
				{false, true, true, true, true, true, false},    // Row 3:  #####
				{false, false, false, true, true, false, false}, // Row 4:    ##
				{false, false, true, true, false, false, false}, // Row 5:   ##
				{false, true, true, false, false, false, false}, // Row 6:  ##
				{true, true, false, false, false, false, false}, // Row 7: ##
			},
		},
		// Leaf for economy/power saver mode (7x8)
		"economy": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{false, false, false, true, true, true, false},   // Row 0:    ###
				{false, false, true, true, true, true, true},     // Row 1:   #####
				{false, true, true, true, false, true, true},     // Row 2:  ### ##
				{true, true, true, false, false, false, true},    // Row 3: ###   #
				{true, true, false, false, false, true, false},   // Row 4: ##   #
				{true, false, false, false, true, false, false},  // Row 5: #   #
				{false, true, false, true, false, false, false},  // Row 6:  # #
				{false, false, true, false, false, false, false}, // Row 7:   #
			},
		},
		// AC power plug (7x8)
		"ac_power": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{false, true, false, false, false, true, false},  // Row 0:  #   #
				{false, true, false, false, false, true, false},  // Row 1:  #   #
				{false, true, true, true, true, true, false},     // Row 2:  #####
				{false, true, true, true, true, true, false},     // Row 3:  #####
				{false, false, true, true, true, false, false},   // Row 4:   ###
				{false, false, true, true, true, false, false},   // Row 5:   ###
				{false, false, false, true, false, false, false}, // Row 6:    #
				{false, false, false, true, false, false, false}, // Row 7:    #
			},
		},
	},
}

// BatteryIcons12x12 contains battery status icons at 12x12 resolution
var BatteryIcons12x12 = &GlyphSet{
	Name:        "battery_12x12",
	GlyphWidth:  12,
	GlyphHeight: 12,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Lightning bolt for charging (10x12)
		"charging": {
			Width: 10, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, true, true, true, false},  // Row 0:       ###
				{false, false, false, false, false, true, true, true, false, false},  // Row 1:      ###
				{false, false, false, false, true, true, true, false, false, false},  // Row 2:     ###
				{false, false, false, true, true, true, false, false, false, false},  // Row 3:    ###
				{false, false, true, true, true, false, false, false, false, false},  // Row 4:   ###
				{false, true, true, true, true, true, true, true, false, false},      // Row 5: ########
				{false, false, false, false, true, true, true, false, false, false},  // Row 6:     ###
				{false, false, false, true, true, true, false, false, false, false},  // Row 7:    ###
				{false, false, true, true, true, false, false, false, false, false},  // Row 8:   ###
				{false, true, true, true, false, false, false, false, false, false},  // Row 9:  ###
				{true, true, true, false, false, false, false, false, false, false},  // Row 10: ###
				{true, true, false, false, false, false, false, false, false, false}, // Row 11: ##
			},
		},
		// Leaf for economy/power saver mode (10x12)
		"economy": {
			Width: 10, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, true, true, true, false, false},   // Row 0:      ###
				{false, false, false, false, true, true, true, true, true, false},     // Row 1:     #####
				{false, false, false, true, true, true, true, true, true, true},       // Row 2:    #######
				{false, false, true, true, true, true, false, true, true, true},       // Row 3:   ####.###
				{false, true, true, true, true, false, false, false, true, true},      // Row 4:  ####   ##
				{true, true, true, true, false, false, false, true, true, false},      // Row 5: ####   ##
				{true, true, true, false, false, false, true, true, false, false},     // Row 6: ###   ##
				{true, true, false, false, false, true, true, false, false, false},    // Row 7: ##   ##
				{true, false, false, false, true, true, false, false, false, false},   // Row 8: #   ##
				{false, true, false, true, true, false, false, false, false, false},   // Row 9:  # ##
				{false, false, true, true, false, false, false, false, false, false},  // Row 10:   ##
				{false, false, true, false, false, false, false, false, false, false}, // Row 11:   #
			},
		},
		// AC power plug (10x12)
		"ac_power": {
			Width: 10, Height: 12,
			Data: [][]bool{
				{false, false, true, true, false, false, true, true, false, false},   // Row 0:   ##  ##
				{false, false, true, true, false, false, true, true, false, false},   // Row 1:   ##  ##
				{false, false, true, true, false, false, true, true, false, false},   // Row 2:   ##  ##
				{false, true, true, true, true, true, true, true, true, false},       // Row 3:  ########
				{false, true, true, true, true, true, true, true, true, false},       // Row 4:  ########
				{false, true, true, true, true, true, true, true, true, false},       // Row 5:  ########
				{false, false, true, true, true, true, true, true, false, false},     // Row 6:   ######
				{false, false, true, true, true, true, true, true, false, false},     // Row 7:   ######
				{false, false, false, true, true, true, true, false, false, false},   // Row 8:    ####
				{false, false, false, true, true, true, true, false, false, false},   // Row 9:    ####
				{false, false, false, false, true, true, false, false, false, false}, // Row 10:     ##
				{false, false, false, false, true, true, false, false, false, false}, // Row 11:     ##
			},
		},
	},
}

// BatteryIcons16x16 contains battery status icons at 16x16 resolution
var BatteryIcons16x16 = &GlyphSet{
	Name:        "battery_16x16",
	GlyphWidth:  16,
	GlyphHeight: 16,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Lightning bolt for charging (12x16)
		"charging": {
			Width: 12, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, true, true, true, false}, // Row 0:         ###
				{false, false, false, false, false, false, false, true, true, true, true, false},  // Row 1:        ####
				{false, false, false, false, false, false, true, true, true, true, false, false},  // Row 2:       ####
				{false, false, false, false, false, true, true, true, true, false, false, false},  // Row 3:      ####
				{false, false, false, false, true, true, true, true, false, false, false, false},  // Row 4:     ####
				{false, false, false, true, true, true, true, false, false, false, false, false},  // Row 5:    ####
				{false, false, true, true, true, true, false, false, false, false, false, false},  // Row 6:   ####
				{false, true, true, true, true, true, true, true, true, true, true, false},        // Row 7: ###########
				{false, true, true, true, true, true, true, true, true, true, true, false},        // Row 8: ###########
				{false, false, false, false, false, false, true, true, true, true, false, false},  // Row 9:       ####
				{false, false, false, false, false, true, true, true, true, false, false, false},  // Row 10:     ####
				{false, false, false, false, true, true, true, true, false, false, false, false},  // Row 11:    ####
				{false, false, false, true, true, true, true, false, false, false, false, false},  // Row 12:   ####
				{false, false, true, true, true, true, false, false, false, false, false, false},  // Row 13:  ####
				{false, true, true, true, true, false, false, false, false, false, false, false},  // Row 14: ####
				{true, true, true, true, false, false, false, false, false, false, false, false},  // Row 15: ###
			},
		},
		// Leaf for economy/power saver mode (12x16)
		"economy": {
			Width: 12, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, true, true, true, false, false},   // Row 0:        ###
				{false, false, false, false, false, false, true, true, true, true, true, false},     // Row 1:       #####
				{false, false, false, false, false, true, true, true, true, true, true, true},       // Row 2:      #######
				{false, false, false, false, true, true, true, true, true, true, true, true},        // Row 3:     ########
				{false, false, false, true, true, true, true, true, false, true, true, true},        // Row 4:    #####.###
				{false, false, true, true, true, true, true, false, false, false, true, true},       // Row 5:   #####   ##
				{false, true, true, true, true, true, false, false, false, true, true, false},       // Row 6:  #####   ##
				{true, true, true, true, true, false, false, false, true, true, false, false},       // Row 7: #####   ##
				{true, true, true, true, false, false, false, true, true, false, false, false},      // Row 8: ####   ##
				{true, true, true, false, false, false, true, true, false, false, false, false},     // Row 9: ###   ##
				{true, true, false, false, false, true, true, false, false, false, false, false},    // Row 10: ##   ##
				{true, false, false, false, true, true, false, false, false, false, false, false},   // Row 11: #   ##
				{false, true, false, true, true, false, false, false, false, false, false, false},   // Row 12:  # ##
				{false, false, true, true, false, false, false, false, false, false, false, false},  // Row 13:   ##
				{false, false, true, false, false, false, false, false, false, false, false, false}, // Row 14:   #
				{false, true, false, false, false, false, false, false, false, false, false, false}, // Row 15:  #
			},
		},
		// AC power plug (12x16)
		"ac_power": {
			Width: 12, Height: 16,
			Data: [][]bool{
				{false, false, true, true, false, false, false, false, true, true, false, false},   // Row 0:   ##    ##
				{false, false, true, true, false, false, false, false, true, true, false, false},   // Row 1:   ##    ##
				{false, false, true, true, false, false, false, false, true, true, false, false},   // Row 2:   ##    ##
				{false, false, true, true, false, false, false, false, true, true, false, false},   // Row 3:   ##    ##
				{false, true, true, true, true, true, true, true, true, true, true, false},         // Row 4:  ##########
				{false, true, true, true, true, true, true, true, true, true, true, false},         // Row 5:  ##########
				{false, true, true, true, true, true, true, true, true, true, true, false},         // Row 6:  ##########
				{false, true, true, true, true, true, true, true, true, true, true, false},         // Row 7:  ##########
				{false, false, true, true, true, true, true, true, true, true, false, false},       // Row 8:   ########
				{false, false, true, true, true, true, true, true, true, true, false, false},       // Row 9:   ########
				{false, false, false, true, true, true, true, true, true, false, false, false},     // Row 10:    ######
				{false, false, false, true, true, true, true, true, true, false, false, false},     // Row 11:    ######
				{false, false, false, false, true, true, true, true, false, false, false, false},   // Row 12:     ####
				{false, false, false, false, true, true, true, true, false, false, false, false},   // Row 13:     ####
				{false, false, false, false, false, true, true, false, false, false, false, false}, // Row 14:      ##
				{false, false, false, false, false, true, true, false, false, false, false, false}, // Row 15:      ##
			},
		},
	},
}
