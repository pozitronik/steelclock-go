package starwarsintro

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.WidgetConfig
		wantErr     bool
		checkValues func(t *testing.T, w *Widget)
	}{
		{
			name: "default configuration",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				// Check default phases enabled
				if !w.preIntroEnabled {
					t.Error("preIntroEnabled should be true by default")
				}
				if !w.logoEnabled {
					t.Error("logoEnabled should be true by default")
				}
				if !w.starsEnabled {
					t.Error("starsEnabled should be true by default")
				}
				// Check default values
				if w.scrollSpeed != 0.5 {
					t.Errorf("scrollSpeed = %f, want 0.5", w.scrollSpeed)
				}
				if w.perspective != 0.7 {
					t.Errorf("perspective = %f, want 0.7", w.perspective)
				}
				if w.slant != 60.0 {
					t.Errorf("slant = %f, want 60.0", w.slant)
				}
				if !w.loop {
					t.Error("loop should be true by default")
				}
				// Should start in pre-intro phase
				if w.phase != PhasePreIntroFadeIn {
					t.Errorf("phase = %d, want PhasePreIntroFadeIn", w.phase)
				}
			},
		},
		{
			name: "pre-intro disabled starts at logo",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					PreIntro: &config.StarWarsPreIntroConfig{
						Enabled: boolPtr(false),
					},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.preIntroEnabled {
					t.Error("preIntroEnabled should be false")
				}
				if w.phase != PhaseLogoHold {
					t.Errorf("phase = %d, want PhaseLogoHold", w.phase)
				}
			},
		},
		{
			name: "both pre-intro and logo disabled starts at crawl",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					PreIntro: &config.StarWarsPreIntroConfig{
						Enabled: boolPtr(false),
					},
					Logo: &config.StarWarsLogoConfig{
						Enabled: boolPtr(false),
					},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.phase != PhaseCrawl {
					t.Errorf("phase = %d, want PhaseCrawl", w.phase)
				}
			},
		},
		{
			name: "custom pre-intro settings",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					PreIntro: &config.StarWarsPreIntroConfig{
						Text:    "Custom pre-intro",
						Color:   100,
						FadeIn:  3.0,
						Hold:    4.0,
						FadeOut: 2.0,
					},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.preIntroText != "Custom pre-intro" {
					t.Errorf("preIntroText = %s, want 'Custom pre-intro'", w.preIntroText)
				}
				if w.preIntroColor != 100 {
					t.Errorf("preIntroColor = %d, want 100", w.preIntroColor)
				}
				if w.preIntroFadeIn != 3.0 {
					t.Errorf("preIntroFadeIn = %f, want 3.0", w.preIntroFadeIn)
				}
			},
		},
		{
			name: "custom logo settings",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					Logo: &config.StarWarsLogoConfig{
						Text:           "CUSTOM\nLOGO",
						Color:          200,
						HoldBefore:     1.0,
						ShrinkDuration: 5.0,
						FinalScale:     0.2,
					},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.logoText != "CUSTOM\nLOGO" {
					t.Errorf("logoText = %s, want 'CUSTOM\\nLOGO'", w.logoText)
				}
				if len(w.logoLines) != 2 {
					t.Errorf("logoLines count = %d, want 2", len(w.logoLines))
				}
				if w.logoColor != 200 {
					t.Errorf("logoColor = %d, want 200", w.logoColor)
				}
				if w.logoShrinkDuration != 5.0 {
					t.Errorf("logoShrinkDuration = %f, want 5.0", w.logoShrinkDuration)
				}
			},
		},
		{
			name: "custom stars settings",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					Stars: &config.StarWarsStarsConfig{
						Count:      100,
						Brightness: 150,
					},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.starsCount != 100 {
					t.Errorf("starsCount = %d, want 100", w.starsCount)
				}
				if len(w.stars) != 100 {
					t.Errorf("len(stars) = %d, want 100", len(w.stars))
				}
				if w.starsBrightness != 150 {
					t.Errorf("starsBrightness = %d, want 150", w.starsBrightness)
				}
			},
		},
		{
			name: "stars disabled",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					Stars: &config.StarWarsStarsConfig{
						Enabled: boolPtr(false),
					},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.starsEnabled {
					t.Error("starsEnabled should be false")
				}
			},
		},
		{
			name: "custom slant angle",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					Slant: 25.0,
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.slant != 25.0 {
					t.Errorf("slant = %f, want 25.0", w.slant)
				}
			},
		},
		{
			name: "custom text",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					Text: []string{"Line 1", "Line 2"},
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if len(w.lines) != 2 {
					t.Errorf("lines count = %d, want 2", len(w.lines))
				}
				if w.lines[0] != "Line 1" {
					t.Errorf("lines[0] = %s, want 'Line 1'", w.lines[0])
				}
			},
		},
		{
			name: "loop disabled",
			cfg: config.WidgetConfig{
				Type:     "starwars_intro",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				StarWarsIntro: &config.StarWarsIntroConfig{
					Loop: boolPtr(false),
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *Widget) {
				if w.loop {
					t.Error("loop should be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.checkValues != nil {
				tt.checkValues(t, w)
			}
		})
	}
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

func TestWidget_Update_PreIntroPhase(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			PreIntro: &config.StarWarsPreIntroConfig{
				FadeIn:  0.01,
				Hold:    0.01,
				FadeOut: 0.01,
			},
			Logo: &config.StarWarsLogoConfig{
				Enabled: boolPtr(false),
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Should start in pre-intro fade in
	if w.phase != PhasePreIntroFadeIn {
		t.Errorf("Initial phase = %d, want PhasePreIntroFadeIn", w.phase)
	}

	// Run updates to progress through pre-intro phases
	for i := 0; i < 50; i++ {
		_ = w.Update()
		time.Sleep(5 * time.Millisecond)
	}

	// Should have progressed to crawl (since logo is disabled)
	if w.phase != PhaseCrawl {
		t.Errorf("Phase = %d, want PhaseCrawl", w.phase)
	}
}

func TestWidget_Update_LogoPhase(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			PreIntro: &config.StarWarsPreIntroConfig{
				Enabled: boolPtr(false),
			},
			Logo: &config.StarWarsLogoConfig{
				HoldBefore:     0.01,
				ShrinkDuration: 0.05,
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Should start in logo hold (pre-intro disabled)
	if w.phase != PhaseLogoHold {
		t.Errorf("Initial phase = %d, want PhaseLogoHold", w.phase)
	}

	// Initial logo scale should be 1.0
	if w.logoScale != 1.0 {
		t.Errorf("Initial logoScale = %f, want 1.0", w.logoScale)
	}

	// Run updates to progress through logo shrink
	for i := 0; i < 50; i++ {
		_ = w.Update()
		time.Sleep(5 * time.Millisecond)
	}

	// Should have progressed to crawl
	if w.phase != PhaseCrawl {
		t.Errorf("Phase = %d, want PhaseCrawl", w.phase)
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWidget_RenderAllPhases(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Test rendering in each phase
	phases := []int{
		PhasePreIntroFadeIn,
		PhasePreIntroHold,
		PhasePreIntroFadeOut,
		PhaseLogoHold,
		PhaseLogoShrink,
		PhaseCrawl,
		PhasePauseEnd,
	}

	for _, phase := range phases {
		w.phase = phase
		w.phaseStart = time.Now()

		img, err := w.Render()
		if err != nil {
			t.Errorf("Render() error in phase %d: %v", phase, err)
			continue
		}

		bounds := img.Bounds()
		if bounds.Empty() {
			t.Errorf("Rendered image is empty in phase %d", phase)
		}
	}
}

func TestWidget_PhaseProgression(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			PreIntro: &config.StarWarsPreIntroConfig{
				FadeIn:  0.01,
				Hold:    0.01,
				FadeOut: 0.01,
			},
			Logo: &config.StarWarsLogoConfig{
				HoldBefore:     0.01,
				ShrinkDuration: 0.02,
			},
			Text:        []string{"A", "B"},
			ScrollSpeed: 100.0, // Very fast
			LineSpacing: 4,
			PauseAtEnd:  0.01,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Track phase changes
	seenPhases := make(map[int]bool)
	seenPhases[w.phase] = true

	// Run updates to progress through all phases
	for i := 0; i < 200; i++ {
		_ = w.Update()
		seenPhases[w.phase] = true
		time.Sleep(5 * time.Millisecond)
	}

	// Should have seen multiple phases
	if len(seenPhases) < 3 {
		t.Errorf("Only saw %d phases, expected at least 3", len(seenPhases))
	}
}

func TestWidget_Stars(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			Stars: &config.StarWarsStarsConfig{
				Count:      75,
				Brightness: 180,
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Verify stars are initialized
	if len(w.stars) != 75 {
		t.Errorf("len(stars) = %d, want 75", len(w.stars))
	}

	// Verify stars are within bounds
	for i, star := range w.stars {
		if star.x < 0 || star.x >= w.width {
			t.Errorf("Star %d x=%d out of bounds [0, %d)", i, star.x, w.width)
		}
		if star.y < 0 || star.y >= w.height {
			t.Errorf("Star %d y=%d out of bounds [0, %d)", i, star.y, w.height)
		}
		if star.brightness == 0 {
			t.Errorf("Star %d has zero brightness", i)
		}
	}
}

func TestWidget_LogoLines(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			Logo: &config.StarWarsLogoConfig{
				Text: "LINE1\nLINE2\nLINE3",
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	if len(w.logoLines) != 3 {
		t.Errorf("len(logoLines) = %d, want 3", len(w.logoLines))
	}

	expected := []string{"LINE1", "LINE2", "LINE3"}
	for i, line := range w.logoLines {
		if line != expected[i] {
			t.Errorf("logoLines[%d] = %s, want %s", i, line, expected[i])
		}
	}
}

func TestWidget_TotalHeight(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			Text:        []string{"A", "B", "C", "D"},
			LineSpacing: 10,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Total height should be lines * lineSpacing
	expectedHeight := float64(4 * 10)
	if w.totalHeight != expectedHeight {
		t.Errorf("totalHeight = %f, want %f", w.totalHeight, expectedHeight)
	}
}

func TestWidget_GlyphSetInitialized(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	if w.glyphSet == nil {
		t.Error("glyphSet should not be nil")
	}
}

func TestWidget_GetCharWidth(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Test getCharWidth for known character
	width := w.getCharWidth('A')
	if width <= 0 {
		t.Errorf("getCharWidth('A') = %d, should be positive", width)
	}

	// Test getCharWidth for unknown character (should return default)
	unknownWidth := w.getCharWidth('\u9999') // Unlikely to be in font
	if unknownWidth <= 0 {
		t.Errorf("getCharWidth for unknown char = %d, should be positive (default)", unknownWidth)
	}
}

func TestWidget_NoLoop(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "starwars_intro",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		StarWarsIntro: &config.StarWarsIntroConfig{
			PreIntro: &config.StarWarsPreIntroConfig{
				Enabled: boolPtr(false),
			},
			Logo: &config.StarWarsLogoConfig{
				Enabled: boolPtr(false),
			},
			Text:        []string{"A"},
			ScrollSpeed: 100.0, // Very fast
			LineSpacing: 4,
			Loop:        boolPtr(false),
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Run updates until text scrolls off
	for i := 0; i < 50; i++ {
		_ = w.Update()
		time.Sleep(10 * time.Millisecond)
	}

	// With loop disabled, should stay in crawl phase
	if w.phase == PhasePauseEnd {
		t.Error("Should not enter end pause phase when loop is disabled")
	}
}
