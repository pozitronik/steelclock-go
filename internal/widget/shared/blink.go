package shared

import (
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// BlinkMode type aliases for convenience
type BlinkMode = config.BlinkMode

// Blink mode constants for convenience
const (
	BlinkNever       = config.BlinkNever
	BlinkAlways      = config.BlinkAlways
	BlinkProgressive = config.BlinkProgressive
)

// BlinkAnimator handles toggling blink state at intervals
type BlinkAnimator struct {
	state        bool
	lastToggle   time.Time
	mode         BlinkMode
	baseInterval time.Duration // interval for "always" mode
}

// NewBlinkAnimator creates a new blink animator with the specified mode and base interval
func NewBlinkAnimator(mode BlinkMode, baseInterval time.Duration) *BlinkAnimator {
	return &BlinkAnimator{
		state:        true, // Start visible
		lastToggle:   time.Now(),
		mode:         mode,
		baseInterval: baseInterval,
	}
}

// Update advances the blink state based on elapsed time
// For progressive mode, pass the intensity value (e.g., unread count)
// Returns true if state changed
func (b *BlinkAnimator) Update(intensity int) bool {
	interval := b.GetInterval(intensity)
	if interval <= 0 {
		return false
	}

	if time.Since(b.lastToggle) >= interval {
		b.state = !b.state
		b.lastToggle = time.Now()
		return true
	}
	return false
}

// UpdateWithTime advances the blink state using the provided time
// Useful when caller already has the current time
func (b *BlinkAnimator) UpdateWithTime(now time.Time, intensity int) bool {
	interval := b.GetInterval(intensity)
	if interval <= 0 {
		return false
	}

	if now.Sub(b.lastToggle) >= interval {
		b.state = !b.state
		b.lastToggle = now
		return true
	}
	return false
}

// State returns the current blink state (true = visible, false = hidden)
func (b *BlinkAnimator) State() bool {
	return b.state
}

// ShouldRender returns true if content should be rendered (respects blink state)
// For "never" mode, always returns true
func (b *BlinkAnimator) ShouldRender() bool {
	if b.mode == BlinkNever {
		return true
	}
	return b.state
}

// GetInterval returns the current blink interval based on mode and intensity
// Returns 0 if blinking is disabled
func (b *BlinkAnimator) GetInterval(intensity int) time.Duration {
	switch b.mode {
	case BlinkAlways:
		return b.baseInterval
	case BlinkProgressive:
		if intensity <= 0 {
			return 0 // No blinking when intensity is 0
		}
		// Scale from baseInterval (1 msg) to baseInterval/10 (10+ msgs)
		// Formula: interval = baseInterval - (intensity-1) * (baseInterval/10)
		// Clamped to [baseInterval/10, baseInterval]
		minInterval := b.baseInterval / 10
		reduction := time.Duration(intensity-1) * (b.baseInterval / 10)
		interval := b.baseInterval - reduction
		if interval < minInterval {
			interval = minInterval
		}
		return interval
	default: // BlinkNever
		return 0
	}
}

// SetMode changes the blink mode
func (b *BlinkAnimator) SetMode(mode BlinkMode) {
	b.mode = mode
}

// Reset resets the blink state to visible
func (b *BlinkAnimator) Reset() {
	b.state = true
	b.lastToggle = time.Now()
}

// Mode returns the current blink mode
func (b *BlinkAnimator) Mode() BlinkMode {
	return b.mode
}
