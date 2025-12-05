package widget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewHyperspaceWidget(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.WidgetConfig
		wantErr     bool
		checkValues func(t *testing.T, w *HyperspaceWidget)
	}{
		{
			name: "default configuration",
			cfg: config.WidgetConfig{
				Type:     "hyperspace",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *HyperspaceWidget) {
				if w.starCount != 100 {
					t.Errorf("starCount = %d, want 100", w.starCount)
				}
				if w.mode != "continuous" {
					t.Errorf("mode = %s, want continuous", w.mode)
				}
				// Starts with PhaseStretch for the anticipation effect
				if w.phase != PhaseStretch {
					t.Errorf("phase = %d, want PhaseStretch", w.phase)
				}
			},
		},
		{
			name: "custom star count",
			cfg: config.WidgetConfig{
				Type:     "hyperspace",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Hyperspace: &config.HyperspaceConfig{
					StarCount: 200,
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *HyperspaceWidget) {
				if w.starCount != 200 {
					t.Errorf("starCount = %d, want 200", w.starCount)
				}
				if len(w.stars) != 200 {
					t.Errorf("len(stars) = %d, want 200", len(w.stars))
				}
			},
		},
		{
			name: "cycle mode",
			cfg: config.WidgetConfig{
				Type:     "hyperspace",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Hyperspace: &config.HyperspaceConfig{
					Mode: "cycle",
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *HyperspaceWidget) {
				if w.mode != "cycle" {
					t.Errorf("mode = %s, want cycle", w.mode)
				}
			},
		},
		{
			name: "custom center",
			cfg: config.WidgetConfig{
				Type:     "hyperspace",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Hyperspace: &config.HyperspaceConfig{
					CenterX: intPtr(32),
					CenterY: intPtr(10),
				},
			},
			wantErr: false,
			checkValues: func(t *testing.T, w *HyperspaceWidget) {
				if w.centerX != 32 {
					t.Errorf("centerX = %d, want 32", w.centerX)
				}
				if w.centerY != 10 {
					t.Errorf("centerY = %d, want 10", w.centerY)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewHyperspaceWidget(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHyperspaceWidget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.checkValues != nil {
				tt.checkValues(t, w)
			}
		})
	}
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

func TestHyperspaceWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Hyperspace: &config.HyperspaceConfig{
			StarCount:    50,
			Mode:         "continuous",
			Acceleration: 10.0, // Fast for testing (short stretch duration)
		},
	}

	w, err := NewHyperspaceWidget(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	initialStretchFactor := w.stretchFactor

	// Update multiple times with small delays to ensure time passes
	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Millisecond)
		if err := w.Update(); err != nil {
			t.Fatalf("Update() error: %v", err)
		}
	}

	// Stretch factor should have increased (we're in stretch phase)
	if w.stretchFactor <= initialStretchFactor {
		t.Error("stretchFactor should increase during stretch phase")
	}
}

func TestHyperspaceWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Hyperspace: &config.HyperspaceConfig{
			StarCount: 50,
		},
	}

	w, err := NewHyperspaceWidget(cfg)
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

func TestHyperspaceWidget_PhaseProgression(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Hyperspace: &config.HyperspaceConfig{
			Mode:         "continuous",
			Acceleration: 5.0, // Fast for testing (short stretch/jump duration)
		},
	}

	w, err := NewHyperspaceWidget(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Should start in Stretch phase
	if w.phase != PhaseStretch {
		t.Errorf("Initial phase = %d, want PhaseStretch", w.phase)
	}

	// Run updates to progress through phases
	for i := 0; i < 100; i++ {
		_ = w.Update()
		time.Sleep(10 * time.Millisecond)
	}

	// Should reach hyperspace in continuous mode
	if w.phase != PhaseHyperspace {
		t.Errorf("Final phase = %d, want PhaseHyperspace", w.phase)
	}

	// Stretch factor should be at maximum
	if w.stretchFactor < 0.9 {
		t.Errorf("stretchFactor = %f, want ~1.0", w.stretchFactor)
	}
}

func TestHyperspaceWidget_CycleMode(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Hyperspace: &config.HyperspaceConfig{
			Mode:         "cycle",
			IdleTime:     0.05, // Very short for testing
			TravelTime:   0.05,
			Acceleration: 10.0, // Very fast
		},
	}

	w, err := NewHyperspaceWidget(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Should start in Stretch phase
	if w.phase != PhaseStretch {
		t.Errorf("Initial phase = %d, want PhaseStretch", w.phase)
	}

	// Run many updates to go through full cycle
	for i := 0; i < 200; i++ {
		_ = w.Update()
		time.Sleep(5 * time.Millisecond)
	}

	// In cycle mode, should eventually cycle back to idle or be somewhere in the cycle
	// Just verify it didn't stay in stretch
	if w.phase == PhaseStretch {
		t.Error("Phase should have progressed past Stretch in cycle mode")
	}
}

func TestHyperspaceWidget_InitStar(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
	}

	w, err := NewHyperspaceWidget(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	var s Star
	w.initStar(&s)

	// Check that star is initialized with valid values
	if s.screenX < 0 || s.screenX > float64(w.width) {
		t.Errorf("Star screenX = %f, should be in range [0, %d]", s.screenX, w.width)
	}
	if s.screenY < 0 || s.screenY > float64(w.height) {
		t.Errorf("Star screenY = %f, should be in range [0, %d]", s.screenY, w.height)
	}
	if s.distFromCenter < 0 {
		t.Error("Star distFromCenter should be >= 0")
	}
	if s.baseBrightness == 0 {
		t.Error("Star baseBrightness should not be 0")
	}
}

func TestHyperspaceWidget_DrawStreak(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
	}

	w, err := NewHyperspaceWidget(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Force into hyperspace phase for streak rendering
	w.phase = PhaseHyperspace
	w.stretchFactor = 1.0

	// Render to test streak drawing
	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Empty() {
		t.Error("Rendered image should not be empty")
	}
}

func TestHyperspaceWidget_RadialBlurEffect(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hyperspace",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Hyperspace: &config.HyperspaceConfig{
			StarCount: 20,
		},
	}

	w, err := NewHyperspaceWidget(cfg)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Verify stars have correct direction vectors (pointing away from center)
	for i, s := range w.stars {
		// Direction should point from center toward the star
		dx := s.screenX - float64(w.centerX)
		dy := s.screenY - float64(w.centerY)

		// For stars not at center, direction should be normalized and pointing outward
		if s.distFromCenter > 1.0 {
			// Check direction roughly matches expected
			expectedDirX := dx / s.distFromCenter
			expectedDirY := dy / s.distFromCenter

			tolerance := 0.01
			if absFloat(s.dirX-expectedDirX) > tolerance || absFloat(s.dirY-expectedDirY) > tolerance {
				t.Errorf("Star %d direction mismatch: got (%f,%f), expected (%f,%f)",
					i, s.dirX, s.dirY, expectedDirX, expectedDirY)
			}
		}
	}
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
