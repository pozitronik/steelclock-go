package shared

import (
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// ScrollMode type alias for convenience
type ScrollMode = config.ScrollMode

// ScrollMode constants for convenience
const (
	ScrollContinuous = config.ScrollContinuous
	ScrollBounce     = config.ScrollBounce
	ScrollPauseEnds  = config.ScrollPauseEnds
)

// ScrollDirection type alias for convenience
type ScrollDirection = config.ScrollDirection

// ScrollDirection constants for convenience
const (
	ScrollLeft  = config.ScrollLeft
	ScrollRight = config.ScrollRight
	ScrollUp    = config.ScrollUp
	ScrollDown  = config.ScrollDown
)

// ScrollerConfig holds scroll behavior configuration
type ScrollerConfig struct {
	Speed     float64         // pixels per second
	Mode      ScrollMode      // scrolling mode
	Direction ScrollDirection // scroll direction
	Gap       int             // gap between text copies for continuous mode
	PauseMs   int             // pause duration at ends (for bounce/pause_ends)
}

// TextScroller manages scrolling animation state
type TextScroller struct {
	config     ScrollerConfig
	offset     float64
	lastUpdate time.Time
	bounceDir  int       // 1 or -1 for bounce mode
	pauseUntil time.Time // for pause modes
}

// NewTextScroller creates a new text scroller with the given configuration
func NewTextScroller(cfg ScrollerConfig) *TextScroller {
	return &TextScroller{
		config:     cfg,
		offset:     0,
		lastUpdate: time.Now(),
		bounceDir:  1, // Start moving forward
	}
}

// Update advances the scroll position based on elapsed time and content dimensions
// contentSize: width for horizontal scrolling, height for vertical
// containerSize: available space for the content
// Returns the current offset
func (s *TextScroller) Update(contentSize, containerSize int) float64 {
	return s.UpdateWithTime(time.Now(), contentSize, containerSize)
}

// UpdateWithTime advances the scroll position using the provided time
func (s *TextScroller) UpdateWithTime(now time.Time, contentSize, containerSize int) float64 {
	// Don't scroll if content fits in container
	if contentSize <= containerSize {
		s.offset = 0
		return 0
	}

	// Handle pause
	if now.Before(s.pauseUntil) {
		s.lastUpdate = now
		return s.offset
	}

	elapsed := now.Sub(s.lastUpdate).Seconds()
	s.lastUpdate = now

	movement := s.config.Speed * elapsed

	switch s.config.Mode {
	case ScrollContinuous:
		s.updateContinuous(movement, contentSize)
	case ScrollBounce:
		s.updateBounce(movement, contentSize, containerSize, now)
	case ScrollPauseEnds:
		s.updatePauseEnds(movement, contentSize, containerSize, now)
	}

	return s.offset
}

// updateContinuous handles continuous/marquee scrolling
func (s *TextScroller) updateContinuous(movement float64, contentSize int) {
	totalSize := float64(contentSize + s.config.Gap)

	switch s.config.Direction {
	case ScrollLeft, ScrollUp:
		s.offset += movement
		if s.offset >= totalSize {
			s.offset -= totalSize
		}
	case ScrollRight, ScrollDown:
		s.offset -= movement
		if s.offset <= -totalSize {
			s.offset += totalSize
		}
	}
}

// updateBounce handles bounce scrolling (reverse at edges)
func (s *TextScroller) updateBounce(movement float64, contentSize, containerSize int, now time.Time) {
	maxOffset := float64(contentSize - containerSize)

	s.offset += movement * float64(s.bounceDir)

	if s.offset >= maxOffset {
		s.offset = maxOffset
		s.bounceDir = -1
		s.pauseUntil = now.Add(time.Duration(s.config.PauseMs) * time.Millisecond)
	} else if s.offset <= 0 {
		s.offset = 0
		s.bounceDir = 1
		s.pauseUntil = now.Add(time.Duration(s.config.PauseMs) * time.Millisecond)
	}
}

// updatePauseEnds handles scroll with pause at ends then reset
func (s *TextScroller) updatePauseEnds(movement float64, contentSize, containerSize int, now time.Time) {
	maxOffset := float64(contentSize - containerSize)

	switch s.config.Direction {
	case ScrollLeft, ScrollUp:
		s.offset += movement
		if s.offset >= maxOffset {
			s.offset = 0
			s.pauseUntil = now.Add(time.Duration(s.config.PauseMs) * time.Millisecond)
		}
	case ScrollRight, ScrollDown:
		s.offset -= movement
		if s.offset <= -maxOffset {
			s.offset = 0
			s.pauseUntil = now.Add(time.Duration(s.config.PauseMs) * time.Millisecond)
		}
	}
}

// Reset resets the scroll position to the beginning
func (s *TextScroller) Reset() {
	s.offset = 0
	s.bounceDir = 1
	s.pauseUntil = time.Time{}
	s.lastUpdate = time.Now()
}

// GetOffset returns the current scroll offset
func (s *TextScroller) GetOffset() float64 {
	return s.offset
}

// SetOffset sets the scroll offset directly
func (s *TextScroller) SetOffset(offset float64) {
	s.offset = offset
}

// GetConfig returns the scroller configuration
func (s *TextScroller) GetConfig() ScrollerConfig {
	return s.config
}

// SetSpeed changes the scroll speed
func (s *TextScroller) SetSpeed(speed float64) {
	s.config.Speed = speed
}

// IsHorizontal returns true if scrolling horizontally
func (s *TextScroller) IsHorizontal() bool {
	return s.config.Direction == ScrollLeft || s.config.Direction == ScrollRight
}

// IsVertical returns true if scrolling vertically
func (s *TextScroller) IsVertical() bool {
	return s.config.Direction == ScrollUp || s.config.Direction == ScrollDown
}

// IsPaused returns true if the scroller is currently paused
func (s *TextScroller) IsPaused() bool {
	return time.Now().Before(s.pauseUntil)
}
