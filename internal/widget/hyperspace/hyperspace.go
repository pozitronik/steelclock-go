package hyperspace

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("hyperspace", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Phase represents the current phase of the hyperspace effect
type Phase int

const (
	PhaseIdle       Phase = iota
	PhaseStretch          // The "anticipation" - stars begin stretching
	PhaseJump             // The "burst" - rapid acceleration into lightspeed
	PhaseHyperspace       // Full hyperspace tunnel effect
	PhaseExit             // Deceleration back to normal
)

// Movement mode constants
const (
	modeContinuous = "continuous"
	modeCycle      = "cycle"
)

// Star represents a single star in the hyperspace effect
type Star struct {
	screenX, screenY float64 // Current screen position
	distFromCenter   float64 // Current distance from vanishing point
	dirX, dirY       float64 // Normalized direction vector (away from center)
	baseBrightness   uint8   // Base brightness (varies for depth)
	twinklePhase     float64 // For subtle twinkling in idle
	speed            float64 // Individual speed variation
}

// Widget displays the Star Wars hyperspace/lightspeed effect
type Widget struct {
	*widget.BaseWidget
	mu sync.Mutex

	// Configuration
	starCount       int
	maxStreakLength float64 // Maximum streak length at full hyperspace
	stretchDuration float64 // Duration of the stretch phase (seconds)
	jumpDuration    float64 // Duration of the jump/burst phase (seconds)
	centerX         int     // Vanishing point X
	centerY         int     // Vanishing point Y
	starColor       uint8
	mode            string  // "continuous" or "cycle"
	idleTime        float64 // Seconds in idle phase (cycle mode)
	travelTime      float64 // Seconds in hyperspace (cycle mode)
	starSpeed       float64 // Speed of star movement during hyperspace

	// State
	stars         []Star
	phase         Phase
	phaseStart    time.Time
	stretchFactor float64 // Current stretch amount (0.0 = dots, 1.0 = full streaks)
	rng           *rand.Rand
	frameCount    int

	// Drift effect for idle phase (spaceship turning)
	driftDirX     float64   // Current drift direction X
	driftDirY     float64   // Current drift direction Y
	driftSpeed    float64   // Drift speed in pixels per frame
	driftChangeAt time.Time // When to change drift direction

	// Display dimensions
	width  int
	height int
}

// New creates a new hyperspace effect widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	pos := base.GetPosition()

	// Default configuration
	starCount := 100
	maxStreakLength := 80.0 // Maximum streak length in pixels
	stretchDuration := 0.8  // Phase 2: anticipation stretch
	jumpDuration := 0.5     // Phase 3: the burst into hyperspace
	centerX := pos.W / 2
	centerY := pos.H / 2
	starColor := uint8(255)
	mode := modeContinuous
	idleTime := 5.0
	travelTime := 3.0
	starSpeed := 3.0 // Pixels per frame during hyperspace

	if cfg.Hyperspace != nil {
		if cfg.Hyperspace.StarCount > 0 {
			starCount = cfg.Hyperspace.StarCount
		}
		if cfg.Hyperspace.TrailLength > 0 {
			maxStreakLength = cfg.Hyperspace.TrailLength * 50.0 // Scale config value
		}
		if cfg.Hyperspace.CenterX != nil {
			centerX = *cfg.Hyperspace.CenterX
		}
		if cfg.Hyperspace.CenterY != nil {
			centerY = *cfg.Hyperspace.CenterY
		}
		if cfg.Hyperspace.StarColor > 0 {
			starColor = uint8(cfg.Hyperspace.StarColor)
		}
		if cfg.Hyperspace.Mode != "" {
			mode = cfg.Hyperspace.Mode
		}
		if cfg.Hyperspace.IdleTime > 0 {
			idleTime = cfg.Hyperspace.IdleTime
		}
		if cfg.Hyperspace.TravelTime > 0 {
			travelTime = cfg.Hyperspace.TravelTime
		}
		if cfg.Hyperspace.Acceleration > 0 {
			// Use acceleration to control stretch/jump duration
			stretchDuration = 1.0 / cfg.Hyperspace.Acceleration
			jumpDuration = 0.5 / cfg.Hyperspace.Acceleration
		}
		if cfg.Hyperspace.Speed > 0 {
			starSpeed = cfg.Hyperspace.Speed * 10.0 // Scale config value
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	w := &Widget{
		BaseWidget:      base,
		starCount:       starCount,
		maxStreakLength: maxStreakLength,
		stretchDuration: stretchDuration,
		jumpDuration:    jumpDuration,
		centerX:         centerX,
		centerY:         centerY,
		starColor:       starColor,
		mode:            mode,
		idleTime:        idleTime,
		travelTime:      travelTime,
		starSpeed:       starSpeed,
		width:           pos.W,
		height:          pos.H,
		rng:             rng,
		stretchFactor:   0.0,
		phase:           PhaseIdle,
		phaseStart:      time.Now(),
	}

	// Initialize stars at random screen positions
	w.stars = make([]Star, starCount)
	for i := range w.stars {
		w.initStar(&w.stars[i])
	}

	// Start with the stretch phase (anticipation before jump)
	w.phase = PhaseStretch
	w.phaseStart = time.Now()

	// Initialize drift for idle phase
	w.driftSpeed = 0.15 // Slow drift
	w.randomizeDriftDirection()

	return w, nil
}

// randomizeDriftDirection sets a new random drift direction
func (w *Widget) randomizeDriftDirection() {
	angle := w.rng.Float64() * 2 * math.Pi
	w.driftDirX = math.Cos(angle)
	w.driftDirY = math.Sin(angle)
	// Change direction every 2-5 seconds
	w.driftChangeAt = time.Now().Add(time.Duration(2+w.rng.Float64()*3) * time.Second)
}

// applyDrift moves all stars slowly in the drift direction (spaceship turning effect)
func (w *Widget) applyDrift() {
	for i := range w.stars {
		s := &w.stars[i]

		// Move star in drift direction
		s.screenX += w.driftDirX * w.driftSpeed
		s.screenY += w.driftDirY * w.driftSpeed

		// Wrap around screen edges
		if s.screenX < 0 {
			s.screenX += float64(w.width)
		} else if s.screenX >= float64(w.width) {
			s.screenX -= float64(w.width)
		}
		if s.screenY < 0 {
			s.screenY += float64(w.height)
		} else if s.screenY >= float64(w.height) {
			s.screenY -= float64(w.height)
		}

		// Recalculate direction from center (needed for when jump starts)
		dx := s.screenX - float64(w.centerX)
		dy := s.screenY - float64(w.centerY)
		s.distFromCenter = math.Sqrt(dx*dx + dy*dy)
		if s.distFromCenter > 0.1 {
			s.dirX = dx / s.distFromCenter
			s.dirY = dy / s.distFromCenter
		}
	}
}

// initStar initializes a star at a random screen position
func (w *Widget) initStar(s *Star) {
	// Random position on screen
	s.screenX = w.rng.Float64() * float64(w.width)
	s.screenY = w.rng.Float64() * float64(w.height)

	// Calculate direction vector from center (vanishing point) to star
	dx := s.screenX - float64(w.centerX)
	dy := s.screenY - float64(w.centerY)
	s.distFromCenter = math.Sqrt(dx*dx + dy*dy)

	// Normalize direction (avoid division by zero)
	if s.distFromCenter > 0.1 {
		s.dirX = dx / s.distFromCenter
		s.dirY = dy / s.distFromCenter
	} else {
		// Star at center - give it a random direction
		angle := w.rng.Float64() * 2 * math.Pi
		s.dirX = math.Cos(angle)
		s.dirY = math.Sin(angle)
		s.distFromCenter = 1.0
	}

	// Brightness varies for depth perception (some dim, some bright)
	s.baseBrightness = uint8(float64(w.starColor) * (0.5 + w.rng.Float64()*0.5))

	// Random twinkle phase
	s.twinklePhase = w.rng.Float64() * math.Pi * 2

	// Random speed variation (0.7 to 1.3)
	s.speed = 0.7 + w.rng.Float64()*0.6
}

// respawnStarAtCenter respawns a star near the center for continuous flow
func (w *Widget) respawnStarAtCenter(s *Star) {
	// Spawn near center with random direction
	angle := w.rng.Float64() * 2 * math.Pi
	s.dirX = math.Cos(angle)
	s.dirY = math.Sin(angle)

	// Start very close to center
	startDist := 1.0 + w.rng.Float64()*3.0
	s.screenX = float64(w.centerX) + s.dirX*startDist
	s.screenY = float64(w.centerY) + s.dirY*startDist
	s.distFromCenter = startDist

	// Randomize brightness and speed
	s.baseBrightness = uint8(float64(w.starColor) * (0.5 + w.rng.Float64()*0.5))
	s.speed = 0.7 + w.rng.Float64()*0.6
}

// moveStars moves all stars outward from center (rushing past effect)
// speedFactor is 0.0 to 1.0, controlling how fast stars move
func (w *Widget) moveStars(speedFactor float64) {
	maxDist := math.Sqrt(float64(w.width*w.width+w.height*w.height)) / 2

	for i := range w.stars {
		s := &w.stars[i]

		// Move star outward along its direction
		movement := w.starSpeed * s.speed * speedFactor
		s.screenX += s.dirX * movement
		s.screenY += s.dirY * movement
		s.distFromCenter += movement

		// If star went off the screen, respawn at center
		if s.screenX < -20 || s.screenX > float64(w.width)+20 ||
			s.screenY < -20 || s.screenY > float64(w.height)+20 ||
			s.distFromCenter >= maxDist {
			w.respawnStarAtCenter(s)
		}
	}
}

// Update advances the animation
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(w.phaseStart).Seconds()
	w.frameCount++

	// Handle phase transitions
	switch w.phase {
	case PhaseIdle:
		// Stars are static points with slow drift (spaceship turning effect)
		w.stretchFactor = 0.0

		// Apply drift to all stars
		w.applyDrift()

		// Occasionally change drift direction for variety
		if now.After(w.driftChangeAt) {
			w.randomizeDriftDirection()
		}

		if w.mode == modeCycle && elapsed >= w.idleTime {
			w.phase = PhaseStretch
			w.phaseStart = now
		}

	case PhaseStretch:
		// Phase 2: Anticipation - stars begin to stretch outward
		// Use ease-in curve (slow start, accelerating)
		progress := elapsed / w.stretchDuration
		if progress > 1.0 {
			progress = 1.0
		}
		// Quadratic ease-in: starts slow, accelerates
		w.stretchFactor = progress * progress * 0.3 // Max 30% stretch in this phase

		if elapsed >= w.stretchDuration {
			w.phase = PhaseJump
			w.phaseStart = now
		}

	case PhaseJump:
		// Phase 3: The burst - exponential acceleration to full lightspeed
		progress := elapsed / w.jumpDuration
		if progress > 1.0 {
			progress = 1.0
		}
		// Exponential ease-in: snaps to full speed
		// Starting from 0.3 (end of stretch) to 1.0 (full hyperspace)
		expProgress := math.Pow(progress, 3) // Cubic for even more dramatic snap
		w.stretchFactor = 0.3 + expProgress*0.7

		// Start moving stars as we accelerate
		w.moveStars(expProgress)

		if elapsed >= w.jumpDuration {
			w.stretchFactor = 1.0
			w.phase = PhaseHyperspace
			w.phaseStart = now
		}

	case PhaseHyperspace:
		// Full hyperspace - maximum stretch
		w.stretchFactor = 1.0

		// Move stars outward from center (rushing past effect)
		w.moveStars(1.0)

		// In cycle mode, transition to exit after travel time
		if w.mode == modeCycle && elapsed >= w.travelTime {
			w.phase = PhaseExit
			w.phaseStart = now
		}

	case PhaseExit:
		// Decelerate - streaks shrink back to points
		progress := elapsed / (w.stretchDuration + w.jumpDuration)
		if progress > 1.0 {
			progress = 1.0
		}
		// Ease-out: fast start, slow end
		w.stretchFactor = 1.0 - (progress * progress)

		if progress >= 1.0 {
			w.stretchFactor = 0.0
			w.phase = PhaseIdle
			w.phaseStart = now
		}
	}

	return nil
}

// Render draws the hyperspace effect
func (w *Widget) Render() (image.Image, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create canvas with background
	img := w.CreateCanvas()

	// Draw stars
	for i := range w.stars {
		s := &w.stars[i]
		w.drawStar(img, s)
	}

	// Draw border if enabled
	w.ApplyBorder(img)

	return img, nil
}

// drawStar renders a single star with its streak effect
func (w *Widget) drawStar(img *image.Gray, s *Star) {
	// Calculate streak length based on:
	// 1. Current stretch factor (phase-dependent)
	// 2. Distance from center (edge stars have longer streaks)
	// 3. Maximum streak length config

	// Normalize distance (0 at center, 1 at max distance which is corner)
	maxDist := math.Sqrt(float64(w.width*w.width+w.height*w.height)) / 2
	normalizedDist := s.distFromCenter / maxDist

	// Streak length: longer for stars farther from center (radial blur effect)
	streakLength := w.stretchFactor * normalizedDist * w.maxStreakLength

	// Calculate star brightness (brighter during hyperspace)
	brightness := s.baseBrightness
	if w.phase == PhaseHyperspace || w.phase == PhaseJump {
		// Boost brightness during hyperspace
		boosted := float64(brightness) * 1.3
		if boosted > 255 {
			boosted = 255
		}
		brightness = uint8(boosted)
	} else if w.phase == PhaseIdle {
		// Subtle twinkling in idle phase
		twinkle := 0.8 + 0.2*math.Sin(s.twinklePhase+float64(w.frameCount)*0.1)
		brightness = uint8(float64(s.baseBrightness) * twinkle)
	}

	if streakLength < 1.0 {
		// Just a dot - no streak
		x, y := int(s.screenX), int(s.screenY)
		if x >= 0 && x < w.width && y >= 0 && y < w.height {
			img.Set(x, y, color.Gray{Y: brightness})
		}
		return
	}

	// Draw streak: line from star position extending outward (away from center)
	// The streak extends FROM the star position OUTWARD (away from vanishing point)
	startX := s.screenX
	startY := s.screenY
	endX := s.screenX + s.dirX*streakLength
	endY := s.screenY + s.dirY*streakLength

	w.drawStreak(img, startX, startY, endX, endY, brightness)
}

// drawStreak draws a radial streak with gradient (bright at head, dim at tail)
func (w *Widget) drawStreak(img *image.Gray, x1, y1, x2, y2 float64, brightness uint8) {
	// Calculate line parameters
	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 1.0 {
		// Too short, just draw a point
		ix, iy := int(x1), int(y1)
		if ix >= 0 && ix < w.width && iy >= 0 && iy < w.height {
			img.Set(ix, iy, color.Gray{Y: brightness})
		}
		return
	}

	// Normalize direction
	dx /= length
	dy /= length

	// Draw pixels along the line with brightness gradient
	steps := int(length) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := x1 + dx*float64(i)
		y := y1 + dy*float64(i)

		ix, iy := int(x), int(y)
		if ix < 0 || ix >= w.width || iy < 0 || iy >= w.height {
			continue
		}

		// Gradient: brightest at the star (head), fading toward the tail
		// Head is at (x1,y1), tail is at (x2,y2)
		fadeFactor := 1.0 - t*0.8 // Fade to 20% at the end
		pixelBrightness := uint8(float64(brightness) * fadeFactor)

		// Additive blending - brighter pixels win
		existing := img.GrayAt(ix, iy).Y
		if pixelBrightness > existing {
			img.Set(ix, iy, color.Gray{Y: pixelBrightness})
		}
	}

	// Draw the head (star point) brighter and slightly larger
	headX, headY := int(x1), int(y1)
	if headX >= 0 && headX < w.width && headY >= 0 && headY < w.height {
		img.Set(headX, headY, color.Gray{Y: brightness})
	}
}
