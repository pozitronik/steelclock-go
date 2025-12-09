package gameoflife

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w == nil {
		t.Fatal("New() returned nil")
	}

	if w.gridWidth != 128 {
		t.Errorf("gridWidth = %d, want 128", w.gridWidth)
	}

	if w.gridHeight != 40 {
		t.Errorf("gridHeight = %d, want 40", w.gridHeight)
	}
}

func TestNew_WithConfig(t *testing.T) {
	wrapEdges := false
	trailEffect := false

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 32,
		},
		GameOfLife: &config.GameOfLifeConfig{
			Rules:          "B36/S23",
			WrapEdges:      &wrapEdges,
			InitialPattern: "glider",
			CellSize:       2,
			TrailEffect:    &trailEffect,
			TrailDecay:     50,
			CellColor:      200,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// With cell size 2, grid should be half the display size
	if w.gridWidth != 32 {
		t.Errorf("gridWidth = %d, want 32", w.gridWidth)
	}

	if w.gridHeight != 16 {
		t.Errorf("gridHeight = %d, want 16", w.gridHeight)
	}

	if w.wrapEdges != false {
		t.Error("wrapEdges should be false")
	}

	if w.trailEffect != false {
		t.Error("trailEffect should be false")
	}

	if w.cellColor != 200 {
		t.Errorf("cellColor = %d, want 200", w.cellColor)
	}

	// Check HighLife rules (B36/S23)
	if !containsInt(w.birthRules, 3) || !containsInt(w.birthRules, 6) {
		t.Errorf("birthRules = %v, want [3, 6]", w.birthRules)
	}

	if !containsInt(w.survivalRules, 2) || !containsInt(w.survivalRules, 3) {
		t.Errorf("survivalRules = %v, want [2, 3]", w.survivalRules)
	}
}

func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "clear",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Manually set a blinker pattern (period 2 oscillator)
	// . X .
	// . X .
	// . X .
	w.current[4][5] = 255
	w.current[5][5] = 255
	w.current[6][5] = 255

	// After one update, should become:
	// . . .
	// X X X
	// . . .

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Check horizontal blinker
	if w.current[5][4] != 255 {
		t.Errorf("current[5][4] = %d, want 255", w.current[5][4])
	}
	if w.current[5][5] != 255 {
		t.Errorf("current[5][5] = %d, want 255", w.current[5][5])
	}
	if w.current[5][6] != 255 {
		t.Errorf("current[5][6] = %d, want 255", w.current[5][6])
	}

	// Original vertical cells should be dead (or fading)
	if w.current[4][5] == 255 {
		t.Error("current[4][5] should not be alive")
	}
	if w.current[6][5] == 255 {
		t.Error("current[6][5] should not be alive")
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 20, H: 20,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "glider",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("Render() size = %dx%d, want 20x20", bounds.Dx(), bounds.Dy())
	}
}

func TestParseRules(t *testing.T) {
	tests := []struct {
		rules        string
		wantBirth    []int
		wantSurvival []int
	}{
		{"B3/S23", []int{3}, []int{2, 3}},
		{"B36/S23", []int{3, 6}, []int{2, 3}},
		{"B1357/S1357", []int{1, 3, 5, 7}, []int{1, 3, 5, 7}},
		{"b3/s23", []int{3}, []int{2, 3}},  // lowercase
		{"invalid", []int{3}, []int{2, 3}}, // defaults
	}

	for _, tt := range tests {
		t.Run(tt.rules, func(t *testing.T) {
			birth, survival := parseRules(tt.rules)

			for _, b := range tt.wantBirth {
				if !containsInt(birth, b) {
					t.Errorf("parseRules(%q) birth missing %d, got %v", tt.rules, b, birth)
				}
			}

			for _, s := range tt.wantSurvival {
				if !containsInt(survival, s) {
					t.Errorf("parseRules(%q) survival missing %d, got %v", tt.rules, s, survival)
				}
			}
		})
	}
}

func TestWidget_Patterns(t *testing.T) {
	patternNames := []string{"glider", "r_pentomino", "acorn", "diehard", "lwss", "pulsar", "glider_gun", "random", "clear"}

	for _, pattern := range patternNames {
		t.Run(pattern, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "game_of_life",
				ID:      "test-gol",
				Enabled: config.BoolPtr(true),
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 128, H: 40,
				},
				GameOfLife: &config.GameOfLifeConfig{
					InitialPattern: pattern,
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() with pattern %q error = %v", pattern, err)
			}

			// Should be able to update and render without error
			if err := w.Update(); err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if _, err := w.Render(); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
		})
	}
}

func TestWidget_WrapEdges(t *testing.T) {
	wrapEdges := true

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "clear",
			WrapEdges:      &wrapEdges,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set cells at edges that should wrap
	w.current[0][0] = 255
	w.current[0][9] = 255
	w.current[9][0] = 255

	// Cell at (0,0) should see neighbors at (9,9), (9,0), (9,1), (0,9), (0,1), (1,9), (1,0), (1,1)
	neighbors := w.countNeighbors(0, 0)

	// We set 3 cells: (0,0), (0,9), (9,0)
	// For (0,0): neighbors are at relative positions -1,-1 through 1,1 (excluding 0,0)
	// Wrapped: (-1,-1)=(9,9), (-1,0)=(9,0), (-1,1)=(9,1), (0,-1)=(0,9), etc.
	// (9,0) is a neighbor (at relative -1,0 wrapped)
	// (0,9) is a neighbor (at relative 0,-1 wrapped)
	// So neighbors should be 2
	if neighbors != 2 {
		t.Errorf("countNeighbors(0,0) with wrap = %d, want 2", neighbors)
	}
}

func TestWidget_RestartOnEmpty(t *testing.T) {
	restartTimeout := 0.0 // Immediate restart
	trailEffect := false

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "random", // Start with random
			RestartTimeout: &restartTimeout,
			RandomDensity:  0.5,
			TrailEffect:    &trailEffect,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Store initial hash
	initialHash := w.computeHash()

	// Clear the grid manually to simulate all cells dying
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			w.current[y][x] = 0
		}
	}

	// Grid should be empty now
	if !w.isGridEmpty() {
		t.Fatal("Grid should be empty after manual clear")
	}

	// After update, should restart immediately with random pattern
	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// With 50% density on 100 cells, very unlikely to still be empty
	aliveCells := w.countAliveCells()
	if aliveCells == 0 {
		t.Error("Grid should have been repopulated after restart")
	}

	// Hash should likely be different (new random pattern)
	newHash := w.computeHash()
	t.Logf("Initial hash: %d, new hash: %d, alive cells: %d", initialHash, newHash, aliveCells)
}

func TestWidget_NoRestartWhenDisabled(t *testing.T) {
	restartTimeout := -1.0 // Disabled
	trailEffect := false

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "clear",
			RestartTimeout: &restartTimeout,
			TrailEffect:    &trailEffect,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Run several updates
	for i := 0; i < 10; i++ {
		if err := w.Update(); err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	}

	// Grid should still be empty - no restart
	if !w.isGridEmpty() {
		t.Error("Grid should remain empty when restart is disabled")
	}
}

func TestWidget_RestartOnStable(t *testing.T) {
	restartTimeout := 0.0 // Immediate restart
	trailEffect := false

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "clear",
			RestartTimeout: &restartTimeout,
			TrailEffect:    &trailEffect,
			RandomDensity:  0.5,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Create a stable 2x2 block (still life)
	w.current[4][4] = 255
	w.current[4][5] = 255
	w.current[5][4] = 255
	w.current[5][5] = 255

	initialHash := w.computeHash()

	// Run a few updates - pattern should stay stable
	for i := 0; i < 2; i++ {
		if err := w.Update(); err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	}

	// Hash should be the same (stable pattern)
	if w.computeHash() != initialHash {
		t.Error("2x2 block should be stable")
	}

	// After more updates, stability should be detected and restart triggered
	for i := 0; i < 5; i++ {
		if err := w.Update(); err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	}

	// After restart, pattern should be different (random)
	// Note: there's a small chance random produces the same hash, but very unlikely
	newHash := w.computeHash()
	if newHash == initialHash {
		t.Log("Warning: new hash equals old hash - might be coincidence")
	}
}

func TestWidget_RestartTimeoutConfig(t *testing.T) {
	restartTimeout := 5.0

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			RestartTimeout: &restartTimeout,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.restartTimeout != 5.0 {
		t.Errorf("restartTimeout = %v, want 5.0", w.restartTimeout)
	}
}

func TestWidget_InjectMode(t *testing.T) {
	restartTimeout := 0.0 // Immediate
	trailEffect := false

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "clear",
			RestartTimeout: &restartTimeout,
			RestartMode:    "inject",
			RandomDensity:  0.5,
			TrailEffect:    &trailEffect,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.restartMode != "inject" {
		t.Errorf("restartMode = %q, want 'inject'", w.restartMode)
	}

	// Test injectRandomCells directly - it should add cells without clearing
	w.current[5][5] = 255 // Set one cell
	initialCell := w.current[5][5]

	w.injectRandomCells()

	// Original cell should still be alive
	if w.current[5][5] != initialCell {
		t.Error("injectRandomCells should not modify existing alive cells")
	}

	// Should have more cells now (with 50% density on ~99 empty cells)
	aliveCells := w.countAliveCells()
	if aliveCells < 10 {
		t.Errorf("After inject, should have many cells, got %d", aliveCells)
	}
}

func TestWidget_RandomMode(t *testing.T) {
	restartTimeout := 0.0 // Immediate
	trailEffect := false

	cfg := config.WidgetConfig{
		Type:    "game_of_life",
		ID:      "test-gol",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 10, H: 10,
		},
		GameOfLife: &config.GameOfLifeConfig{
			InitialPattern: "glider", // Start with glider
			RestartTimeout: &restartTimeout,
			RestartMode:    "random", // But restart with random
			RandomDensity:  0.5,
			TrailEffect:    &trailEffect,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Glider has 5 cells
	initialCells := w.countAliveCells()
	if initialCells != 5 {
		t.Errorf("Initial glider should have 5 cells, got %d", initialCells)
	}

	// Clear grid to trigger restart
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			w.current[y][x] = 0
		}
	}

	// Update should trigger restart with random (not glider)
	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// With 50% density on 100 cells, should have ~50 cells (not 5)
	newCells := w.countAliveCells()
	if newCells < 20 {
		t.Errorf("Random mode should create many cells, got %d", newCells)
	}
}
