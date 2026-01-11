//go:build !light

package main

// This file imports all widget packages to ensure their init() functions
// are called and widgets are registered with the factory.
// Add new widget package imports here as widgets are created.
// For light build configuration, see imports_light.go

import (
	// Widget packages (blank imports for registration)
	_ "github.com/pozitronik/steelclock-go/internal/widget/audiovisualizer"
	_ "github.com/pozitronik/steelclock-go/internal/widget/battery"
	_ "github.com/pozitronik/steelclock-go/internal/widget/beefwebwidget"
	_ "github.com/pozitronik/steelclock-go/internal/widget/claudecode"
	_ "github.com/pozitronik/steelclock-go/internal/widget/clipboard"
	_ "github.com/pozitronik/steelclock-go/internal/widget/clock"
	_ "github.com/pozitronik/steelclock-go/internal/widget/cpu"
	_ "github.com/pozitronik/steelclock-go/internal/widget/disk"
	_ "github.com/pozitronik/steelclock-go/internal/widget/doom"
	_ "github.com/pozitronik/steelclock-go/internal/widget/gameoflife"
	_ "github.com/pozitronik/steelclock-go/internal/widget/hackercode"
	_ "github.com/pozitronik/steelclock-go/internal/widget/hyperspace"
	_ "github.com/pozitronik/steelclock-go/internal/widget/keyboard"
	_ "github.com/pozitronik/steelclock-go/internal/widget/keyboardlayout"
	_ "github.com/pozitronik/steelclock-go/internal/widget/matrix"
	_ "github.com/pozitronik/steelclock-go/internal/widget/memory"
	_ "github.com/pozitronik/steelclock-go/internal/widget/network"
	_ "github.com/pozitronik/steelclock-go/internal/widget/screenmirror"
	_ "github.com/pozitronik/steelclock-go/internal/widget/starwarsintro"
	_ "github.com/pozitronik/steelclock-go/internal/widget/telegramcounter"
	_ "github.com/pozitronik/steelclock-go/internal/widget/telegramwidget"
	_ "github.com/pozitronik/steelclock-go/internal/widget/volume"
	_ "github.com/pozitronik/steelclock-go/internal/widget/volumemeter"
	_ "github.com/pozitronik/steelclock-go/internal/widget/weather"
	_ "github.com/pozitronik/steelclock-go/internal/widget/winampwidget"
)
