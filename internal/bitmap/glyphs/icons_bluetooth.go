package glyphs

// BluetoothIcons8x8 contains Bluetooth device type icons at 8x8 resolution
var BluetoothIcons8x8 = &GlyphSet{
	Name:        "bluetooth_8x8",
	GlyphWidth:  8,
	GlyphHeight: 8,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Bluetooth logo (generic device / HID / Unknown)
		"bt_generic": {
			Width: 5, Height: 8,
			Data: [][]bool{
				{false, false, true, false, false}, // Row 0:   #
				{false, false, true, true, false},  // Row 1:   ##
				{true, false, true, false, true},   // Row 2: # # #
				{false, true, true, true, false},   // Row 3:  ###
				{false, false, true, false, false}, // Row 4:   #
				{false, true, true, true, false},   // Row 5:  ###
				{true, false, true, false, true},   // Row 6: # # #
				{false, false, true, true, false},  // Row 7:   ##
			},
		},
		// Bluetooth logo with X cross (adapter off / API unreachable)
		"bt_off": {
			Width: 8, Height: 8,
			Data: [][]bool{
				{false, false, true, false, false, false, false, false}, // Row 0:   #
				{false, false, true, true, false, false, false, false},  // Row 1:   ##
				{true, false, true, false, true, false, true, false},    // Row 2: # # # #
				{false, true, true, true, false, true, false, true},     // Row 3:  ###  ##
				{false, false, true, false, false, true, false, true},   // Row 4:   #  # #
				{false, true, true, true, false, false, true, false},    // Row 5:  ###  #
				{true, false, true, false, true, false, false, false},   // Row 6: # # #
				{false, false, true, true, false, false, false, false},  // Row 7:   ##
			},
		},
		// Question mark (device not found)
		"bt_unknown": {
			Width: 5, Height: 8,
			Data: [][]bool{
				{false, true, true, true, false},    // Row 0:  ###
				{true, false, false, false, true},   // Row 1: #   #
				{false, false, false, true, false},  // Row 2:    #
				{false, false, true, false, false},  // Row 3:   #
				{false, false, true, false, false},  // Row 4:   #
				{false, false, false, false, false}, // Row 5:
				{false, false, true, false, false},  // Row 6:   #
				{false, false, false, false, false}, // Row 7:
			},
		},
		// Ellipsis dots (transient states)
		"bt_ellipsis": {
			Width: 5, Height: 8,
			Data: [][]bool{
				{false, false, false, false, false}, // Row 0:
				{false, false, false, false, false}, // Row 1:
				{false, false, false, false, false}, // Row 2:
				{false, false, false, false, false}, // Row 3:
				{false, false, false, false, false}, // Row 4:
				{false, false, false, false, false}, // Row 5:
				{true, false, true, false, true},    // Row 6: # # #
				{false, false, false, false, false}, // Row 7:
			},
		},
		// Headphones (AudioOutput, Headset)
		"bt_headphones": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{false, true, true, true, true, true, false},    // Row 0:  #####
				{true, false, false, false, false, false, true}, // Row 1: #     #
				{true, false, false, false, false, false, true}, // Row 2: #     #
				{true, false, false, false, false, false, true}, // Row 3: #     #
				{true, true, false, false, false, true, true},   // Row 4: ##   ##
				{true, true, false, false, false, true, true},   // Row 5: ##   ##
				{true, true, false, false, false, true, true},   // Row 6: ##   ##
				{true, true, false, false, false, true, true},   // Row 7: ##   ##
			},
		},
		// Microphone (AudioInput)
		"bt_microphone": {
			Width: 5, Height: 8,
			Data: [][]bool{
				{false, true, true, true, false},   // Row 0:  ###
				{false, true, true, true, false},   // Row 1:  ###
				{false, true, true, true, false},   // Row 2:  ###
				{true, false, true, false, true},   // Row 3: # # #
				{true, false, true, false, true},   // Row 4: # # #
				{false, true, true, true, false},   // Row 5:  ###
				{false, false, true, false, false}, // Row 6:   #
				{false, true, true, true, false},   // Row 7:  ###
			},
		},
		// Keyboard
		"bt_keyboard": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{false, false, false, false, false, false, false}, // Row 0:
				{true, true, true, true, true, true, true},        // Row 1: #######
				{true, false, true, false, true, false, true},     // Row 2: # # # #
				{true, true, true, true, true, true, true},        // Row 3: #######
				{true, false, true, false, true, false, true},     // Row 4: # # # #
				{true, true, true, true, true, true, true},        // Row 5: #######
				{true, false, true, true, true, false, true},      // Row 6: # ### #
				{true, true, true, true, true, true, true},        // Row 7: #######
			},
		},
		// Mouse
		"bt_mouse": {
			Width: 5, Height: 8,
			Data: [][]bool{
				{false, true, true, true, false},  // Row 0:  ###
				{true, false, true, false, true},  // Row 1: # # #
				{true, false, true, false, true},  // Row 2: # # #
				{true, true, true, true, true},    // Row 3: #####
				{true, false, false, false, true}, // Row 4: #   #
				{true, false, false, false, true}, // Row 5: #   #
				{true, false, false, false, true}, // Row 6: #   #
				{false, true, true, true, false},  // Row 7:  ###
			},
		},
		// Game controller (Gamepad)
		"bt_gamepad": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{false, false, false, false, false, false, false}, // Row 0:
				{false, true, true, true, true, true, false},      // Row 1:  #####
				{true, true, false, true, false, true, true},      // Row 2: ## # ##
				{true, false, true, false, true, false, true},     // Row 3: # # # #
				{true, true, false, false, false, true, true},     // Row 4: ##   ##
				{true, true, true, true, true, true, true},        // Row 5: #######
				{false, true, false, false, false, true, false},   // Row 6:  #   #
				{false, false, false, false, false, false, false}, // Row 7:
			},
		},
		// Computer / Monitor
		"bt_computer": {
			Width: 7, Height: 8,
			Data: [][]bool{
				{true, true, true, true, true, true, true},       // Row 0: #######
				{true, false, false, false, false, false, true},  // Row 1: #     #
				{true, false, false, false, false, false, true},  // Row 2: #     #
				{true, false, false, false, false, false, true},  // Row 3: #     #
				{true, true, true, true, true, true, true},       // Row 4: #######
				{false, false, false, true, false, false, false}, // Row 5:    #
				{false, false, true, true, true, false, false},   // Row 6:   ###
				{false, true, true, true, true, true, false},     // Row 7:  #####
			},
		},
		// Phone
		"bt_phone": {
			Width: 5, Height: 8,
			Data: [][]bool{
				{false, true, true, true, false},  // Row 0:  ###
				{false, true, false, true, false}, // Row 1:  # #
				{false, true, false, true, false}, // Row 2:  # #
				{false, true, false, true, false}, // Row 3:  # #
				{false, true, false, true, false}, // Row 4:  # #
				{false, true, false, true, false}, // Row 5:  # #
				{false, true, true, true, false},  // Row 6:  ###
				{false, true, true, true, false},  // Row 7:  ###
			},
		},
	},
}

// BluetoothIcons12x12 contains Bluetooth device type icons at 12x12 resolution
var BluetoothIcons12x12 = &GlyphSet{
	Name:        "bluetooth_12x12",
	GlyphWidth:  12,
	GlyphHeight: 12,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Bluetooth logo
		"bt_generic": {
			Width: 7, Height: 12,
			Data: [][]bool{
				{false, false, false, true, false, false, false}, // Row 0:    #
				{false, false, false, true, true, false, false},  // Row 1:    ##
				{false, false, false, true, false, true, false},  // Row 2:    # #
				{true, false, false, true, false, false, true},   // Row 3: #  #  #
				{false, true, false, true, false, true, false},   // Row 4:  # # #
				{false, false, true, true, true, false, false},   // Row 5:   ###
				{false, false, true, true, true, false, false},   // Row 6:   ###
				{false, true, false, true, false, true, false},   // Row 7:  # # #
				{true, false, false, true, false, false, true},   // Row 8: #  #  #
				{false, false, false, true, false, true, false},  // Row 9:    # #
				{false, false, false, true, true, false, false},  // Row 10:   ##
				{false, false, false, true, false, false, false}, // Row 11:   #
			},
		},
		// Bluetooth off (with X)
		"bt_off": {
			Width: 11, Height: 12,
			Data: [][]bool{
				{false, false, false, true, false, false, false, false, false, false, false}, // Row 0:    #
				{false, false, false, true, true, false, false, false, false, false, false},  // Row 1:    ##
				{false, false, false, true, false, true, false, false, false, false, false},  // Row 2:    # #
				{true, false, false, true, false, false, true, false, true, false, false},    // Row 3: #  #  # #
				{false, true, false, true, false, true, false, false, false, true, false},    // Row 4:  # # #   #
				{false, false, true, true, true, false, false, false, false, false, true},    // Row 5:   ###    #
				{false, false, true, true, true, false, false, false, false, true, false},    // Row 6:   ###   #
				{false, true, false, true, false, true, false, false, true, false, false},    // Row 7:  # # #  #
				{true, false, false, true, false, false, true, true, false, false, false},    // Row 8: #  #  ##
				{false, false, false, true, false, true, false, false, false, false, false},  // Row 9:    # #
				{false, false, false, true, true, false, false, false, false, false, false},  // Row 10:   ##
				{false, false, false, true, false, false, false, false, false, false, false}, // Row 11:   #
			},
		},
		// Question mark
		"bt_unknown": {
			Width: 7, Height: 12,
			Data: [][]bool{
				{false, false, true, true, true, false, false},    // Row 0:   ###
				{false, true, false, false, false, true, false},   // Row 1:  #   #
				{true, false, false, false, false, false, true},   // Row 2: #     #
				{false, false, false, false, false, false, true},  // Row 3:       #
				{false, false, false, false, false, true, false},  // Row 4:      #
				{false, false, false, false, true, false, false},  // Row 5:     #
				{false, false, false, true, false, false, false},  // Row 6:    #
				{false, false, false, true, false, false, false},  // Row 7:    #
				{false, false, false, false, false, false, false}, // Row 8:
				{false, false, false, true, false, false, false},  // Row 9:    #
				{false, false, false, true, false, false, false},  // Row 10:   #
				{false, false, false, false, false, false, false}, // Row 11:
			},
		},
		// Ellipsis
		"bt_ellipsis": {
			Width: 7, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, false}, // Row 0:
				{false, false, false, false, false, false, false}, // Row 1:
				{false, false, false, false, false, false, false}, // Row 2:
				{false, false, false, false, false, false, false}, // Row 3:
				{false, false, false, false, false, false, false}, // Row 4:
				{false, false, false, false, false, false, false}, // Row 5:
				{false, false, false, false, false, false, false}, // Row 6:
				{false, false, false, false, false, false, false}, // Row 7:
				{true, true, false, false, false, true, true},     // Row 8: ##   ##
				{true, true, false, false, false, true, true},     // Row 9: ##   ##
				{false, false, false, false, false, false, false}, // Row 10:
				{false, false, false, false, false, false, false}, // Row 11:
			},
		},
		// Headphones
		"bt_headphones": {
			Width: 10, Height: 12,
			Data: [][]bool{
				{false, false, false, true, true, true, true, false, false, false},   // Row 0:    ####
				{false, false, true, false, false, false, false, true, false, false}, // Row 1:   #    #
				{false, true, false, false, false, false, false, false, true, false}, // Row 2:  #      #
				{true, false, false, false, false, false, false, false, false, true}, // Row 3: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 4: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 5: #        #
				{true, true, true, false, false, false, false, true, true, true},     // Row 6: ###    ###
				{true, true, true, false, false, false, false, true, true, true},     // Row 7: ###    ###
				{true, true, true, false, false, false, false, true, true, true},     // Row 8: ###    ###
				{true, true, true, false, false, false, false, true, true, true},     // Row 9: ###    ###
				{true, true, true, false, false, false, false, true, true, true},     // Row 10: ###    ###
				{false, true, false, false, false, false, false, false, true, false}, // Row 11:  #      #
			},
		},
		// Microphone
		"bt_microphone": {
			Width: 8, Height: 12,
			Data: [][]bool{
				{false, false, true, true, true, true, false, false},   // Row 0:   ####
				{false, false, true, true, true, true, false, false},   // Row 1:   ####
				{false, false, true, true, true, true, false, false},   // Row 2:   ####
				{false, false, true, true, true, true, false, false},   // Row 3:   ####
				{false, false, true, true, true, true, false, false},   // Row 4:   ####
				{true, false, false, true, true, false, false, true},   // Row 5: #  ##  #
				{true, false, false, true, true, false, false, true},   // Row 6: #  ##  #
				{false, true, false, false, false, false, true, false}, // Row 7:  #    #
				{false, false, true, true, true, true, false, false},   // Row 8:   ####
				{false, false, false, true, true, false, false, false}, // Row 9:    ##
				{false, false, false, true, true, false, false, false}, // Row 10:   ##
				{false, true, true, true, true, true, true, false},     // Row 11:  ######
			},
		},
		// Keyboard
		"bt_keyboard": {
			Width: 11, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false}, // Row 0:
				{true, true, true, true, true, true, true, true, true, true, true},            // Row 1: ###########
				{true, false, false, false, false, false, false, false, false, false, true},   // Row 2: #         #
				{true, false, true, false, true, false, true, false, true, false, true},       // Row 3: # # # # # #
				{true, false, false, false, false, false, false, false, false, false, true},   // Row 4: #         #
				{true, false, true, false, true, false, true, false, true, false, true},       // Row 5: # # # # # #
				{true, false, false, false, false, false, false, false, false, false, true},   // Row 6: #         #
				{true, false, true, false, true, false, true, false, true, false, true},       // Row 7: # # # # # #
				{true, false, false, false, false, false, false, false, false, false, true},   // Row 8: #         #
				{true, false, false, true, true, true, true, true, false, false, true},        // Row 9: #  #####  #
				{true, true, true, true, true, true, true, true, true, true, true},            // Row 10: ###########
				{false, false, false, false, false, false, false, false, false, false, false}, // Row 11:
			},
		},
		// Mouse
		"bt_mouse": {
			Width: 8, Height: 12,
			Data: [][]bool{
				{false, false, true, true, true, true, false, false},   // Row 0:   ####
				{false, true, false, false, false, false, true, false}, // Row 1:  #    #
				{true, false, false, true, true, false, false, true},   // Row 2: #  ##  #
				{true, false, false, true, true, false, false, true},   // Row 3: #  ##  #
				{true, true, true, true, true, true, true, true},       // Row 4: ########
				{true, false, false, false, false, false, false, true}, // Row 5: #      #
				{true, false, false, false, false, false, false, true}, // Row 6: #      #
				{true, false, false, false, false, false, false, true}, // Row 7: #      #
				{true, false, false, false, false, false, false, true}, // Row 8: #      #
				{true, false, false, false, false, false, false, true}, // Row 9: #      #
				{false, true, false, false, false, false, true, false}, // Row 10:  #    #
				{false, false, true, true, true, true, false, false},   // Row 11:   ####
			},
		},
		// Gamepad
		"bt_gamepad": {
			Width: 11, Height: 12,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false}, // Row 0:
				{false, false, true, true, true, true, true, true, true, false, false},        // Row 1:   #######
				{false, true, true, false, false, false, false, false, true, true, false},     // Row 2:  ##     ##
				{true, true, false, true, false, false, false, true, false, true, true},       // Row 3: ## #   # ##
				{true, true, true, true, true, false, true, true, true, true, true},           // Row 4: ##### #####
				{true, true, false, true, false, false, false, true, false, true, true},       // Row 5: ## #   # ##
				{true, true, false, false, false, false, false, false, false, true, true},     // Row 6: ##       ##
				{true, true, true, true, true, true, true, true, true, true, true},            // Row 7: ###########
				{false, true, true, false, false, false, false, false, true, true, false},     // Row 8:  ##     ##
				{false, false, true, false, false, false, false, false, true, false, false},   // Row 9:   #     #
				{false, false, false, false, false, false, false, false, false, false, false}, // Row 10:
				{false, false, false, false, false, false, false, false, false, false, false}, // Row 11:
			},
		},
		// Computer
		"bt_computer": {
			Width: 10, Height: 12,
			Data: [][]bool{
				{true, true, true, true, true, true, true, true, true, true},         // Row 0: ##########
				{true, false, false, false, false, false, false, false, false, true}, // Row 1: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 2: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 3: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 4: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 5: #        #
				{true, false, false, false, false, false, false, false, false, true}, // Row 6: #        #
				{true, true, true, true, true, true, true, true, true, true},         // Row 7: ##########
				{false, false, false, false, true, true, false, false, false, false}, // Row 8:     ##
				{false, false, false, false, true, true, false, false, false, false}, // Row 9:     ##
				{false, false, true, true, true, true, true, true, false, false},     // Row 10:   ######
				{false, true, true, true, true, true, true, true, true, false},       // Row 11:  ########
			},
		},
		// Phone
		"bt_phone": {
			Width: 7, Height: 12,
			Data: [][]bool{
				{false, true, true, true, true, true, false},    // Row 0:  #####
				{false, true, false, false, false, true, false}, // Row 1:  #   #
				{false, true, false, false, false, true, false}, // Row 2:  #   #
				{false, true, false, false, false, true, false}, // Row 3:  #   #
				{false, true, false, false, false, true, false}, // Row 4:  #   #
				{false, true, false, false, false, true, false}, // Row 5:  #   #
				{false, true, false, false, false, true, false}, // Row 6:  #   #
				{false, true, false, false, false, true, false}, // Row 7:  #   #
				{false, true, false, false, false, true, false}, // Row 8:  #   #
				{false, true, true, true, true, true, false},    // Row 9:  #####
				{false, true, false, true, false, true, false},  // Row 10: # # #
				{false, true, true, true, true, true, false},    // Row 11: #####
			},
		},
	},
}

// BluetoothIcons16x16 contains Bluetooth device type icons at 16x16 resolution
var BluetoothIcons16x16 = &GlyphSet{
	Name:        "bluetooth_16x16",
	GlyphWidth:  16,
	GlyphHeight: 16,
	Glyphs:      nil,
	Icons: map[string]*Glyph{
		// Bluetooth logo
		"bt_generic": {
			Width: 9, Height: 16,
			Data: [][]bool{
				{false, false, false, false, true, false, false, false, false}, // Row 0:     #
				{false, false, false, false, true, false, false, false, false}, // Row 1:     #
				{false, false, false, false, true, true, false, false, false},  // Row 2:     ##
				{false, false, false, false, true, false, true, false, false},  // Row 3:     # #
				{true, true, false, false, true, false, false, true, false},    // Row 4: ##  #  #
				{false, true, true, false, true, false, true, false, false},    // Row 5:  ## # #
				{false, false, true, true, true, true, false, false, false},    // Row 6:   ####
				{false, false, false, true, true, false, false, false, false},  // Row 7:    ##
				{false, false, false, true, true, false, false, false, false},  // Row 8:    ##
				{false, false, true, true, true, true, false, false, false},    // Row 9:   ####
				{false, true, true, false, true, false, true, false, false},    // Row 10:  ## # #
				{true, true, false, false, true, false, false, true, false},    // Row 11: ##  #  #
				{false, false, false, false, true, false, true, false, false},  // Row 12:     # #
				{false, false, false, false, true, true, false, false, false},  // Row 13:     ##
				{false, false, false, false, true, false, false, false, false}, // Row 14:     #
				{false, false, false, false, true, false, false, false, false}, // Row 15:     #
			},
		},
		// Bluetooth off (with X)
		"bt_off": {
			Width: 14, Height: 16,
			Data: [][]bool{
				{false, false, false, false, true, false, false, false, false, false, false, false, false, false}, // Row 0:     #
				{false, false, false, false, true, false, false, false, false, false, false, false, false, false}, // Row 1:     #
				{false, false, false, false, true, true, false, false, false, false, false, false, false, false},  // Row 2:     ##
				{false, false, false, false, true, false, true, false, false, false, false, false, false, false},  // Row 3:     # #
				{true, true, false, false, true, false, false, true, false, false, true, false, false, false},     // Row 4: ##  #  #  #
				{false, true, true, false, true, false, true, false, false, false, false, true, false, false},     // Row 5:  ## # #    #
				{false, false, true, true, true, true, false, false, false, false, false, false, true, false},     // Row 6:   ####      #
				{false, false, false, true, true, false, false, false, false, false, false, true, false, false},   // Row 7:    ##      #
				{false, false, false, true, true, false, false, false, false, false, true, false, false, false},   // Row 8:    ##     #
				{false, false, true, true, true, true, false, false, false, true, false, false, false, false},     // Row 9:   ####   #
				{false, true, true, false, true, false, true, false, true, false, false, false, false, false},     // Row 10:  ## # # #
				{true, true, false, false, true, false, false, true, false, false, false, false, false, false},    // Row 11: ##  #  #
				{false, false, false, false, true, false, true, false, false, false, false, false, false, false},  // Row 12:     # #
				{false, false, false, false, true, true, false, false, false, false, false, false, false, false},  // Row 13:     ##
				{false, false, false, false, true, false, false, false, false, false, false, false, false, false}, // Row 14:     #
				{false, false, false, false, true, false, false, false, false, false, false, false, false, false}, // Row 15:     #
			},
		},
		// Question mark
		"bt_unknown": {
			Width: 9, Height: 16,
			Data: [][]bool{
				{false, false, true, true, true, true, true, false, false},      // Row 0:   #####
				{false, true, true, false, false, false, true, true, false},     // Row 1:  ##   ##
				{true, true, false, false, false, false, false, true, true},     // Row 2: ##     ##
				{true, true, false, false, false, false, false, true, true},     // Row 3: ##     ##
				{false, false, false, false, false, false, true, true, false},   // Row 4:       ##
				{false, false, false, false, false, true, true, false, false},   // Row 5:      ##
				{false, false, false, false, true, true, false, false, false},   // Row 6:     ##
				{false, false, false, true, true, false, false, false, false},   // Row 7:    ##
				{false, false, false, true, true, false, false, false, false},   // Row 8:    ##
				{false, false, false, true, true, false, false, false, false},   // Row 9:    ##
				{false, false, false, false, false, false, false, false, false}, // Row 10:
				{false, false, false, false, false, false, false, false, false}, // Row 11:
				{false, false, false, true, true, false, false, false, false},   // Row 12:    ##
				{false, false, false, true, true, false, false, false, false},   // Row 13:    ##
				{false, false, false, false, false, false, false, false, false}, // Row 14:
				{false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Ellipsis
		"bt_ellipsis": {
			Width: 10, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false}, // Row 0:
				{false, false, false, false, false, false, false, false, false, false}, // Row 1:
				{false, false, false, false, false, false, false, false, false, false}, // Row 2:
				{false, false, false, false, false, false, false, false, false, false}, // Row 3:
				{false, false, false, false, false, false, false, false, false, false}, // Row 4:
				{false, false, false, false, false, false, false, false, false, false}, // Row 5:
				{false, false, false, false, false, false, false, false, false, false}, // Row 6:
				{false, false, false, false, false, false, false, false, false, false}, // Row 7:
				{false, false, false, false, false, false, false, false, false, false}, // Row 8:
				{false, false, false, false, false, false, false, false, false, false}, // Row 9:
				{true, true, false, false, false, false, false, false, true, true},     // Row 10: ##      ##
				{true, true, false, false, true, true, false, false, true, true},       // Row 11: ##  ##  ##
				{false, false, false, false, true, true, false, false, false, false},   // Row 12:     ##
				{false, false, false, false, false, false, false, false, false, false}, // Row 13:
				{false, false, false, false, false, false, false, false, false, false}, // Row 14:
				{false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Headphones
		"bt_headphones": {
			Width: 14, Height: 16,
			Data: [][]bool{
				{false, false, false, false, true, true, true, true, true, true, false, false, false, false},       // Row 0:     ######
				{false, false, false, true, true, false, false, false, false, true, true, false, false, false},     // Row 1:    ##    ##
				{false, false, true, true, false, false, false, false, false, false, true, true, false, false},     // Row 2:   ##      ##
				{false, true, true, false, false, false, false, false, false, false, false, true, true, false},     // Row 3:  ##        ##
				{true, true, false, false, false, false, false, false, false, false, false, false, true, true},     // Row 4: ##          ##
				{true, true, false, false, false, false, false, false, false, false, false, false, true, true},     // Row 5: ##          ##
				{true, true, false, false, false, false, false, false, false, false, false, false, true, true},     // Row 6: ##          ##
				{true, true, false, false, false, false, false, false, false, false, false, false, true, true},     // Row 7: ##          ##
				{true, true, true, true, false, false, false, false, false, false, true, true, true, true},         // Row 8: ####      ####
				{true, true, true, true, false, false, false, false, false, false, true, true, true, true},         // Row 9: ####      ####
				{true, true, true, true, false, false, false, false, false, false, true, true, true, true},         // Row 10: ####      ####
				{true, true, true, true, false, false, false, false, false, false, true, true, true, true},         // Row 11: ####      ####
				{true, true, true, true, false, false, false, false, false, false, true, true, true, true},         // Row 12: ####      ####
				{true, true, true, true, false, false, false, false, false, false, true, true, true, true},         // Row 13: ####      ####
				{false, true, true, false, false, false, false, false, false, false, false, true, true, false},     // Row 14:  ##        ##
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Microphone
		"bt_microphone": {
			Width: 10, Height: 16,
			Data: [][]bool{
				{false, false, false, true, true, true, true, false, false, false},     // Row 0:    ####
				{false, false, true, true, true, true, true, true, false, false},       // Row 1:   ######
				{false, false, true, true, true, true, true, true, false, false},       // Row 2:   ######
				{false, false, true, true, true, true, true, true, false, false},       // Row 3:   ######
				{false, false, true, true, true, true, true, true, false, false},       // Row 4:   ######
				{false, false, true, true, true, true, true, true, false, false},       // Row 5:   ######
				{false, false, true, true, true, true, true, true, false, false},       // Row 6:   ######
				{true, false, false, true, true, true, true, false, false, true},       // Row 7: #  ####  #
				{true, false, false, true, true, true, true, false, false, true},       // Row 8: #  ####  #
				{true, true, false, false, false, false, false, false, true, true},     // Row 9: ##      ##
				{false, true, true, false, false, false, false, true, true, false},     // Row 10:  ##    ##
				{false, false, true, true, true, true, true, true, false, false},       // Row 11:   ######
				{false, false, false, false, true, true, false, false, false, false},   // Row 12:     ##
				{false, false, false, false, true, true, false, false, false, false},   // Row 13:     ##
				{false, true, true, true, true, true, true, true, true, false},         // Row 14:  ########
				{false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Keyboard
		"bt_keyboard": {
			Width: 15, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 0:
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},                // Row 1: ###############
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 2: #             #
				{true, false, true, true, false, true, true, false, true, true, false, true, true, false, true},           // Row 3: # ## ## ## ## #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 4: #             #
				{true, false, true, true, false, true, true, false, true, true, false, true, true, false, true},           // Row 5: # ## ## ## ## #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 6: #             #
				{true, false, true, true, false, true, true, false, true, true, false, true, true, false, true},           // Row 7: # ## ## ## ## #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 8: #             #
				{true, false, true, true, false, true, true, false, true, true, false, true, true, false, true},           // Row 9: # ## ## ## ## #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 10: #             #
				{true, false, false, false, true, true, true, true, true, true, true, false, false, false, true},          // Row 11: #   #######   #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 12: #             #
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},                // Row 13: ###############
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 14:
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Mouse
		"bt_mouse": {
			Width: 10, Height: 16,
			Data: [][]bool{
				{false, false, false, true, true, true, true, false, false, false},     // Row 0:    ####
				{false, false, true, true, false, false, true, true, false, false},     // Row 1:   ##  ##
				{false, true, true, false, false, false, false, true, true, false},     // Row 2:  ##    ##
				{true, true, false, false, true, true, false, false, true, true},       // Row 3: ##  ##  ##
				{true, true, false, false, true, true, false, false, true, true},       // Row 4: ##  ##  ##
				{true, true, true, true, true, true, true, true, true, true},           // Row 5: ##########
				{true, true, false, false, false, false, false, false, true, true},     // Row 6: ##      ##
				{true, true, false, false, false, false, false, false, true, true},     // Row 7: ##      ##
				{true, true, false, false, false, false, false, false, true, true},     // Row 8: ##      ##
				{true, true, false, false, false, false, false, false, true, true},     // Row 9: ##      ##
				{true, true, false, false, false, false, false, false, true, true},     // Row 10: ##      ##
				{true, true, false, false, false, false, false, false, true, true},     // Row 11: ##      ##
				{true, true, false, false, false, false, false, false, true, true},     // Row 12: ##      ##
				{false, true, true, false, false, false, false, true, true, false},     // Row 13:  ##    ##
				{false, false, true, true, true, true, true, true, false, false},       // Row 14:   ######
				{false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Gamepad
		"bt_gamepad": {
			Width: 15, Height: 16,
			Data: [][]bool{
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 0:
				{false, false, false, true, true, true, true, true, true, true, true, true, false, false, false},          // Row 1:    #########
				{false, false, true, true, true, false, false, false, false, false, true, true, true, false, false},       // Row 2:   ###     ###
				{false, true, true, false, true, false, false, false, false, false, true, false, true, true, false},       // Row 3:  ## #     # ##
				{true, true, false, true, true, true, false, false, false, true, true, true, false, true, true},           // Row 4: ## ###   ### ##
				{true, true, true, true, true, true, false, false, false, true, false, true, true, true, true},            // Row 5: ######   # ####
				{true, true, false, true, true, true, false, false, false, true, true, true, false, true, true},           // Row 6: ## ###   ### ##
				{true, true, false, false, true, false, false, false, false, false, true, false, false, true, true},       // Row 7: ##  #     #  ##
				{true, true, false, false, false, false, false, false, false, false, false, false, false, true, true},     // Row 8: ##           ##
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},                // Row 9: ###############
				{false, true, true, true, false, false, false, false, false, false, false, true, true, true, false},       // Row 10:  ###       ###
				{false, false, true, false, false, false, false, false, false, false, false, false, true, false, false},   // Row 11:   #         #
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 12:
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 13:
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 14:
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Computer
		"bt_computer": {
			Width: 14, Height: 16,
			Data: [][]bool{
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true},               // Row 0: ##############
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 1: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 2: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 3: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 4: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 5: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 6: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 7: #            #
				{true, false, false, false, false, false, false, false, false, false, false, false, false, true},   // Row 8: #            #
				{true, true, true, true, true, true, true, true, true, true, true, true, true, true},               // Row 9: ##############
				{false, false, false, false, false, true, true, true, true, false, false, false, false, false},     // Row 10:      ####
				{false, false, false, false, false, true, true, true, true, false, false, false, false, false},     // Row 11:      ####
				{false, false, false, true, true, true, true, true, true, true, true, false, false, false},         // Row 12:    ########
				{false, false, true, true, true, true, true, true, true, true, true, true, false, false},           // Row 13:   ##########
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 14:
				{false, false, false, false, false, false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
		// Phone
		"bt_phone": {
			Width: 9, Height: 16,
			Data: [][]bool{
				{false, true, true, true, true, true, true, true, false},        // Row 0:  #######
				{false, true, true, false, false, false, true, true, false},     // Row 1:  ##   ##
				{false, true, false, false, false, false, false, true, false},   // Row 2:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 3:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 4:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 5:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 6:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 7:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 8:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 9:  #     #
				{false, true, false, false, false, false, false, true, false},   // Row 10: #     #
				{false, true, false, false, false, false, false, true, false},   // Row 11: #     #
				{false, true, true, false, false, false, true, true, false},     // Row 12: ##   ##
				{false, true, false, false, true, false, false, true, false},    // Row 13: #  #  #
				{false, true, true, true, true, true, true, true, false},        // Row 14: #######
				{false, false, false, false, false, false, false, false, false}, // Row 15:
			},
		},
	},
}
