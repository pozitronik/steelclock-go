//go:build light

package main

// LIGHT BUILD - Widget Exclusion Configuration
// =============================================
// This is the SINGLE POINT of configuration for light build exclusions.
// Widgets commented out below are EXCLUDED from the light build.
//
// Currently excluded widgets:
//   - telegramcounter (requires external API dependencies)
//   - telegramwidget  (requires external API dependencies)
//
// To modify exclusions, edit the import list below.
// Build with: go build -tags light ./cmd/steelclock

import (
	// Widget packages (blank imports for registration)
	_ "github.com/pozitronik/steelclock-go/internal/widget/audiovisualizer"
	_ "github.com/pozitronik/steelclock-go/internal/widget/battery"
	_ "github.com/pozitronik/steelclock-go/internal/widget/clock"
	_ "github.com/pozitronik/steelclock-go/internal/widget/cpu"
	_ "github.com/pozitronik/steelclock-go/internal/widget/disk"
	_ "github.com/pozitronik/steelclock-go/internal/widget/doom"
	_ "github.com/pozitronik/steelclock-go/internal/widget/gameoflife"
	_ "github.com/pozitronik/steelclock-go/internal/widget/gpu"
	_ "github.com/pozitronik/steelclock-go/internal/widget/hyperspace"
	_ "github.com/pozitronik/steelclock-go/internal/widget/keyboard"
	_ "github.com/pozitronik/steelclock-go/internal/widget/keyboardlayout"
	_ "github.com/pozitronik/steelclock-go/internal/widget/matrix"
	_ "github.com/pozitronik/steelclock-go/internal/widget/memory"
	_ "github.com/pozitronik/steelclock-go/internal/widget/network"
	_ "github.com/pozitronik/steelclock-go/internal/widget/starwarsintro"
	// EXCLUDED: _ "github.com/pozitronik/steelclock-go/internal/widget/telegramcounter"
	// EXCLUDED: _ "github.com/pozitronik/steelclock-go/internal/widget/telegramwidget"
	_ "github.com/pozitronik/steelclock-go/internal/widget/volume"
	_ "github.com/pozitronik/steelclock-go/internal/widget/volumemeter"
	_ "github.com/pozitronik/steelclock-go/internal/widget/weather"
	_ "github.com/pozitronik/steelclock-go/internal/widget/winampwidget"
)
