package widget

import (
	"image"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

func init() {
	Register("game_of_life", func(cfg config.WidgetConfig) (Widget, error) {
		return NewGameOfLifeWidget(cfg)
	})
}

// Restart mode constants
const (
	restartModeReset  = "reset"
	restartModeInject = "inject"
	restartModeRandom = "random"
)

// GameOfLifeWidget implements Conway's Game of Life cellular automaton
type GameOfLifeWidget struct {
	*BaseWidget
	mu sync.Mutex

	// Configuration
	birthRules     []int   // Neighbor counts that cause birth
	survivalRules  []int   // Neighbor counts that cause survival
	wrapEdges      bool    // Toroidal topology
	cellSize       int     // Pixels per cell
	trailEffect    bool    // Enable fading trail
	trailDecay     int     // Decay amount per frame
	cellColor      uint8   // Alive cell color
	randomDensity  float64 // Initial density for random pattern
	initialPattern string  // Pattern to use on restart
	restartTimeout float64 // Seconds to wait before restart (-1 = never, 0 = immediate)
	restartMode    string  // "reset", "inject", or "random"

	// Grid state
	gridWidth  int
	gridHeight int
	current    [][]uint8 // Current cell states (0=dead, 255=alive, 1-254=fading)
	next       [][]uint8 // Next generation buffer

	// Restart tracking
	stableFrames int       // Count of frames with no change
	restartAt    time.Time // When to restart (zero = not scheduled)
	lastHash     uint64    // Hash of last frame for detecting stability

	rng *rand.Rand
}

// Predefined patterns (relative coordinates)
var gameOfLifePatterns = map[string][][2]int{
	"glider": {
		{1, 0}, {2, 1}, {0, 2}, {1, 2}, {2, 2},
	},
	"r_pentomino": {
		{1, 0}, {2, 0}, {0, 1}, {1, 1}, {1, 2},
	},
	"acorn": {
		{1, 0}, {3, 1}, {0, 2}, {1, 2}, {4, 2}, {5, 2}, {6, 2},
	},
	"diehard": {
		{6, 0}, {0, 1}, {1, 1}, {1, 2}, {5, 2}, {6, 2}, {7, 2},
	},
	"lwss": { // Lightweight spaceship
		{1, 0}, {4, 0}, {0, 1}, {0, 2}, {4, 2}, {0, 3}, {1, 3}, {2, 3}, {3, 3},
	},
	"pulsar": {
		// Top section
		{2, 0}, {3, 0}, {4, 0}, {8, 0}, {9, 0}, {10, 0},
		{0, 2}, {5, 2}, {7, 2}, {12, 2},
		{0, 3}, {5, 3}, {7, 3}, {12, 3},
		{0, 4}, {5, 4}, {7, 4}, {12, 4},
		{2, 5}, {3, 5}, {4, 5}, {8, 5}, {9, 5}, {10, 5},
		// Bottom section (mirrored)
		{2, 7}, {3, 7}, {4, 7}, {8, 7}, {9, 7}, {10, 7},
		{0, 8}, {5, 8}, {7, 8}, {12, 8},
		{0, 9}, {5, 9}, {7, 9}, {12, 9},
		{0, 10}, {5, 10}, {7, 10}, {12, 10},
		{2, 12}, {3, 12}, {4, 12}, {8, 12}, {9, 12}, {10, 12},
	},
	"glider_gun": { // Gosper glider gun
		{24, 0},
		{22, 1}, {24, 1},
		{12, 2}, {13, 2}, {20, 2}, {21, 2}, {34, 2}, {35, 2},
		{11, 3}, {15, 3}, {20, 3}, {21, 3}, {34, 3}, {35, 3},
		{0, 4}, {1, 4}, {10, 4}, {16, 4}, {20, 4}, {21, 4},
		{0, 5}, {1, 5}, {10, 5}, {14, 5}, {16, 5}, {17, 5}, {22, 5}, {24, 5},
		{10, 6}, {16, 6}, {24, 6},
		{11, 7}, {15, 7},
		{12, 8}, {13, 8},
	},
}

// NewGameOfLifeWidget creates a new Game of Life widget
func NewGameOfLifeWidget(cfg config.WidgetConfig) (*GameOfLifeWidget, error) {
	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	// Default configuration
	birthRules := []int{3}       // Standard: born with 3 neighbors
	survivalRules := []int{2, 3} // Standard: survive with 2 or 3 neighbors
	wrapEdges := true
	cellSize := 1
	trailEffect := true
	trailDecay := 30
	cellColor := uint8(255)
	randomDensity := 0.3
	initialPattern := restartModeRandom
	restartTimeout := 3.0           // Default: restart after 3 seconds
	restartMode := restartModeReset // Default: reset to initial pattern

	if cfg.GameOfLife != nil {
		if cfg.GameOfLife.Rules != "" {
			birthRules, survivalRules = parseRules(cfg.GameOfLife.Rules)
		}
		if cfg.GameOfLife.WrapEdges != nil {
			wrapEdges = *cfg.GameOfLife.WrapEdges
		}
		if cfg.GameOfLife.CellSize > 0 {
			cellSize = cfg.GameOfLife.CellSize
			if cellSize > 4 {
				cellSize = 4
			}
		}
		if cfg.GameOfLife.TrailEffect != nil {
			trailEffect = *cfg.GameOfLife.TrailEffect
		}
		if cfg.GameOfLife.TrailDecay > 0 {
			trailDecay = cfg.GameOfLife.TrailDecay
		}
		if cfg.GameOfLife.CellColor > 0 {
			cellColor = uint8(cfg.GameOfLife.CellColor)
		}
		if cfg.GameOfLife.RandomDensity > 0 {
			randomDensity = cfg.GameOfLife.RandomDensity
		}
		if cfg.GameOfLife.InitialPattern != "" {
			initialPattern = cfg.GameOfLife.InitialPattern
		}
		if cfg.GameOfLife.RestartTimeout != nil {
			restartTimeout = *cfg.GameOfLife.RestartTimeout
		}
		if cfg.GameOfLife.RestartMode != "" {
			restartMode = cfg.GameOfLife.RestartMode
		}
	}

	// Calculate grid dimensions
	gridWidth := pos.W / cellSize
	gridHeight := pos.H / cellSize

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	w := &GameOfLifeWidget{
		BaseWidget:     base,
		birthRules:     birthRules,
		survivalRules:  survivalRules,
		wrapEdges:      wrapEdges,
		cellSize:       cellSize,
		trailEffect:    trailEffect,
		trailDecay:     trailDecay,
		cellColor:      cellColor,
		randomDensity:  randomDensity,
		initialPattern: initialPattern,
		restartTimeout: restartTimeout,
		restartMode:    restartMode,
		gridWidth:      gridWidth,
		gridHeight:     gridHeight,
		rng:            rng,
	}

	// Initialize grids
	w.current = make([][]uint8, gridHeight)
	w.next = make([][]uint8, gridHeight)
	for y := 0; y < gridHeight; y++ {
		w.current[y] = make([]uint8, gridWidth)
		w.next[y] = make([]uint8, gridWidth)
	}

	// Set initial pattern
	w.setPattern(initialPattern)

	return w, nil
}

// parseRules parses rules in "B3/S23" or "B36/S23" format
func parseRules(rules string) ([]int, []int) {
	birth := []int{3}
	survival := []int{2, 3}

	rules = strings.ToUpper(rules)
	parts := strings.Split(rules, "/")

	if len(parts) == 2 {
		// Parse birth rules
		if strings.HasPrefix(parts[0], "B") {
			birth = parseDigits(parts[0][1:])
		}
		// Parse survival rules
		if strings.HasPrefix(parts[1], "S") {
			survival = parseDigits(parts[1][1:])
		}
	}

	return birth, survival
}

// parseDigits extracts individual digits from a string
func parseDigits(s string) []int {
	var result []int
	for _, c := range s {
		if n, err := strconv.Atoi(string(c)); err == nil && n >= 0 && n <= 8 {
			result = append(result, n)
		}
	}
	return result
}

// injectRandomCells adds random cells to the existing grid without clearing it
func (w *GameOfLifeWidget) injectRandomCells() {
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			// Only add to empty cells
			if w.current[y][x] == 0 && w.rng.Float64() < w.randomDensity {
				w.current[y][x] = w.cellColor
			}
		}
	}
}

// performRestart handles restart based on restart_mode
func (w *GameOfLifeWidget) performRestart() {
	switch w.restartMode {
	case restartModeInject:
		// Add new cells to existing grid
		w.injectRandomCells()
	case restartModeRandom:
		// Always use random pattern
		w.setPattern(restartModeRandom)
	default: // restartModeReset
		// Reset to initial pattern
		w.setPattern(w.initialPattern)
	}
}

// setPattern initializes the grid with a pattern
func (w *GameOfLifeWidget) setPattern(pattern string) {
	// Clear grid
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			w.current[y][x] = 0
		}
	}

	if pattern == restartModeRandom {
		// Random initialization
		for y := 0; y < w.gridHeight; y++ {
			for x := 0; x < w.gridWidth; x++ {
				if w.rng.Float64() < w.randomDensity {
					w.current[y][x] = w.cellColor
				}
			}
		}
		return
	}

	if pattern == "clear" {
		return
	}

	// Try predefined patterns
	coords, ok := gameOfLifePatterns[pattern]
	if !ok {
		// Default to random if pattern not found
		w.setPattern(restartModeRandom)
		return
	}

	// Calculate center offset
	centerX := w.gridWidth / 2
	centerY := w.gridHeight / 2

	// Find pattern bounds for centering
	minX, minY, maxX, maxY := 0, 0, 0, 0
	for _, c := range coords {
		if c[0] > maxX {
			maxX = c[0]
		}
		if c[1] > maxY {
			maxY = c[1]
		}
	}

	offsetX := centerX - (maxX-minX)/2
	offsetY := centerY - (maxY-minY)/2

	// Place pattern
	for _, c := range coords {
		x := offsetX + c[0]
		y := offsetY + c[1]
		if x >= 0 && x < w.gridWidth && y >= 0 && y < w.gridHeight {
			w.current[y][x] = w.cellColor
		}
	}
}

// countNeighbors counts alive neighbors for a cell
func (w *GameOfLifeWidget) countNeighbors(x, y int) int {
	count := 0

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}

			nx, ny := x+dx, y+dy

			if w.wrapEdges {
				// Wrap around edges (toroidal)
				if nx < 0 {
					nx = w.gridWidth - 1
				} else if nx >= w.gridWidth {
					nx = 0
				}
				if ny < 0 {
					ny = w.gridHeight - 1
				} else if ny >= w.gridHeight {
					ny = 0
				}
			} else {
				// Bounded - skip if outside
				if nx < 0 || nx >= w.gridWidth || ny < 0 || ny >= w.gridHeight {
					continue
				}
			}

			// Cell is alive if brightness is at maximum
			if w.current[ny][nx] == w.cellColor {
				count++
			}
		}
	}

	return count
}

// contains checks if a slice contains a value
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// computeHash computes a simple hash of the grid state (only alive cells)
func (w *GameOfLifeWidget) computeHash() uint64 {
	var hash uint64 = 5381
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			if w.current[y][x] == w.cellColor {
				// DJB2 hash variant
				hash = ((hash << 5) + hash) + uint64(y*w.gridWidth+x)
			}
		}
	}
	return hash
}

// countAliveCells returns the number of fully alive cells
func (w *GameOfLifeWidget) countAliveCells() int {
	count := 0
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			if w.current[y][x] == w.cellColor {
				count++
			}
		}
	}
	return count
}

// isGridEmpty returns true if all cells are dead (including fading)
func (w *GameOfLifeWidget) isGridEmpty() bool {
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			if w.current[y][x] > 0 {
				return false
			}
		}
	}
	return true
}

// Update advances the simulation one generation
func (w *GameOfLifeWidget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if restart is scheduled
	if !w.restartAt.IsZero() && time.Now().After(w.restartAt) {
		w.performRestart()
		w.restartAt = time.Time{}
		w.stableFrames = 0
		w.lastHash = 0
		return nil
	}

	// Skip update if waiting for restart
	if !w.restartAt.IsZero() {
		return nil
	}

	// Calculate next generation
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			neighbors := w.countNeighbors(x, y)
			isAlive := w.current[y][x] == w.cellColor

			if isAlive {
				// Currently alive
				if contains(w.survivalRules, neighbors) {
					w.next[y][x] = w.cellColor // Survives
				} else {
					// Dies - start fading or go to 0
					if w.trailEffect {
						w.next[y][x] = w.cellColor - 1 // Start fading
					} else {
						w.next[y][x] = 0
					}
				}
			} else if w.current[y][x] > 0 {
				// Fading cell - continue fading
				if w.current[y][x] > uint8(w.trailDecay) {
					w.next[y][x] = w.current[y][x] - uint8(w.trailDecay)
				} else {
					w.next[y][x] = 0
				}
			} else {
				// Currently dead
				if contains(w.birthRules, neighbors) {
					w.next[y][x] = w.cellColor // Birth
				} else {
					w.next[y][x] = 0
				}
			}
		}
	}

	// Swap buffers
	w.current, w.next = w.next, w.current

	// Check for restart conditions (only if restart is enabled)
	if w.restartTimeout >= 0 {
		currentHash := w.computeHash()
		aliveCells := w.countAliveCells()

		// Detect end conditions:
		// 1. All cells dead (and no fading trails)
		// 2. Pattern is stable (same hash for multiple frames)
		shouldRestart := false

		if aliveCells == 0 && w.isGridEmpty() {
			// All cells dead
			shouldRestart = true
		} else if currentHash == w.lastHash {
			// Pattern unchanged - might be stable
			w.stableFrames++
			// Consider stable after 3 identical frames
			if w.stableFrames >= 3 {
				shouldRestart = true
			}
		} else {
			w.stableFrames = 0
		}

		w.lastHash = currentHash

		if shouldRestart {
			if w.restartTimeout == 0 {
				// Immediate restart
				w.performRestart()
				w.stableFrames = 0
				w.lastHash = 0
			} else {
				// Schedule restart
				w.restartAt = time.Now().Add(time.Duration(w.restartTimeout * float64(time.Second)))
			}
		}
	}

	return nil
}

// Render draws the current state
func (w *GameOfLifeWidget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create canvas with background
	img := w.CreateCanvas()

	// Draw cells
	for y := 0; y < w.gridHeight; y++ {
		for x := 0; x < w.gridWidth; x++ {
			brightness := w.current[y][x]
			if brightness == 0 {
				continue
			}

			// Draw cell (potentially multiple pixels if cellSize > 1)
			px := x * w.cellSize
			py := y * w.cellSize

			bitmap.DrawFilledRectangle(img, px, py, w.cellSize, w.cellSize, brightness)
		}
	}

	// Draw border if enabled
	w.ApplyBorder(img)

	return img, nil
}
