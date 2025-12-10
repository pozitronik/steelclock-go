package anim

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"time"
)

// TransitionType represents available transition effects
type TransitionType string

const (
	TransitionNone           TransitionType = "none"
	TransitionPushLeft       TransitionType = "push_left"
	TransitionPushRight      TransitionType = "push_right"
	TransitionPushUp         TransitionType = "push_up"
	TransitionPushDown       TransitionType = "push_down"
	TransitionSlideLeft      TransitionType = "slide_left"
	TransitionSlideRight     TransitionType = "slide_right"
	TransitionSlideUp        TransitionType = "slide_up"
	TransitionSlideDown      TransitionType = "slide_down"
	TransitionDissolveFade   TransitionType = "dissolve_fade"
	TransitionDissolvePixel  TransitionType = "dissolve_pixel"
	TransitionDissolveDither TransitionType = "dissolve_dither"
	TransitionBoxIn          TransitionType = "box_in"
	TransitionBoxOut         TransitionType = "box_out"
	TransitionClockWipe      TransitionType = "clock_wipe"
	TransitionRandom         TransitionType = "random"
)

// AllTransitions lists all available transition types (excluding none and random)
var AllTransitions = []TransitionType{
	TransitionPushLeft, TransitionPushRight, TransitionPushUp, TransitionPushDown,
	TransitionSlideLeft, TransitionSlideRight, TransitionSlideUp, TransitionSlideDown,
	TransitionDissolveFade, TransitionDissolvePixel, TransitionDissolveDither,
	TransitionBoxIn, TransitionBoxOut, TransitionClockWipe,
}

// TransitionManager handles frame transitions between old and new content
type TransitionManager struct {
	active         bool
	progress       float64
	startTime      time.Time
	duration       float64 // seconds
	transitionType TransitionType
	oldFrame       *image.Gray
	pixelOrder     []int // pre-shuffled for dissolve_pixel
	width          int
	height         int
}

// NewTransitionManager creates a new transition manager for the given dimensions
func NewTransitionManager(width, height int) *TransitionManager {
	return &TransitionManager{
		width:  width,
		height: height,
	}
}

// Start begins a new transition with the specified type, duration, and old frame
func (t *TransitionManager) Start(transitionType TransitionType, duration float64, oldFrame *image.Gray) {
	if transitionType == TransitionNone {
		return
	}

	t.active = true
	t.progress = 0.0
	t.startTime = time.Now()
	t.duration = duration
	t.oldFrame = oldFrame

	// Select actual transition (handle "random")
	t.transitionType = SelectTransition(transitionType)

	// Pre-generate pixel order for dissolve_pixel
	if t.transitionType == TransitionDissolvePixel {
		t.pixelOrder = GeneratePixelOrder(t.width, t.height)
	}
}

// Update advances the transition based on elapsed time
// Returns true if the transition is still active
func (t *TransitionManager) Update() bool {
	if !t.active {
		return false
	}

	elapsed := time.Since(t.startTime).Seconds()
	t.progress = elapsed / t.duration

	if t.progress >= 1.0 {
		t.progress = 1.0
		t.active = false
		t.oldFrame = nil
		t.pixelOrder = nil
	}

	return t.active
}

// IsActive returns true if a transition is currently in progress (based on stored state).
// Note: For accurate timing in Render(), use IsActiveLive() instead.
func (t *TransitionManager) IsActive() bool {
	return t.active
}

// IsActiveLive returns true if a transition is currently in progress based on elapsed time.
// This is safe to call from read-only contexts (like Render) and provides accurate state
// regardless of how often Update() is called.
func (t *TransitionManager) IsActiveLive() bool {
	if !t.active {
		return false
	}
	elapsed := time.Since(t.startTime).Seconds()
	return elapsed < t.duration
}

// Progress returns the current transition progress (0.0 to 1.0)
func (t *TransitionManager) Progress() float64 {
	return t.progress
}

// LiveProgress calculates and returns the current progress based on elapsed time.
// This is safe to call from read-only contexts (like Render) as it doesn't modify state.
// Returns progress clamped to 0.0-1.0 range.
func (t *TransitionManager) LiveProgress() float64 {
	if !t.active {
		return 0.0
	}
	elapsed := time.Since(t.startTime).Seconds()
	progress := elapsed / t.duration
	if progress > 1.0 {
		progress = 1.0
	}
	return progress
}

// OldFrame returns the captured old frame (may be nil)
func (t *TransitionManager) OldFrame() *image.Gray {
	return t.oldFrame
}

// Type returns the current active transition type
func (t *TransitionManager) Type() TransitionType {
	return t.transitionType
}

// PixelOrder returns the pre-generated pixel order for dissolve_pixel
func (t *TransitionManager) PixelOrder() []int {
	return t.pixelOrder
}

// Apply composites old and new frames to dst based on stored progress.
// Note: For accurate timing in Render(), use ApplyLive() instead.
func (t *TransitionManager) Apply(dst, newFrame *image.Gray) {
	if t.oldFrame == nil {
		CopyGrayImage(dst, newFrame)
		return
	}
	ApplyTransition(dst, t.oldFrame, newFrame, t.progress, t.transitionType, t.pixelOrder)
}

// ApplyLive composites old and new frames using live progress calculated from elapsed time.
// This is safe to call from read-only contexts (like Render) and provides smooth transitions
// regardless of how often Update() is called.
func (t *TransitionManager) ApplyLive(dst, newFrame *image.Gray) {
	if t.oldFrame == nil {
		CopyGrayImage(dst, newFrame)
		return
	}
	progress := t.LiveProgress()
	ApplyTransition(dst, t.oldFrame, newFrame, progress, t.transitionType, t.pixelOrder)
}

// Reset clears the transition state
func (t *TransitionManager) Reset() {
	t.active = false
	t.progress = 0.0
	t.oldFrame = nil
	t.pixelOrder = nil
}

// Cancel stops an in-progress transition
func (t *TransitionManager) Cancel() {
	t.Reset()
}

// SelectTransition returns the actual transition type, handling "random"
func SelectTransition(requested TransitionType) TransitionType {
	if requested != TransitionRandom {
		return requested
	}
	return AllTransitions[rand.Intn(len(AllTransitions))]
}

// GeneratePixelOrder creates a shuffled list of pixel indices for dissolve_pixel
func GeneratePixelOrder(width, height int) []int {
	total := width * height
	order := make([]int, total)
	for i := 0; i < total; i++ {
		order[i] = i
	}
	// Fisher-Yates shuffle
	for i := total - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	return order
}

// ApplyTransition composites old and new frames based on transition type and progress
func ApplyTransition(dst, oldFrame, newFrame *image.Gray, progress float64, transitionType TransitionType, pixelOrder []int) {
	switch transitionType {
	case TransitionNone:
		// Instant switch at 50%
		if progress < 0.5 {
			CopyGrayImage(dst, oldFrame)
		} else {
			CopyGrayImage(dst, newFrame)
		}

	case TransitionPushLeft:
		applyPushTransition(dst, oldFrame, newFrame, progress, -1, 0)
	case TransitionPushRight:
		applyPushTransition(dst, oldFrame, newFrame, progress, 1, 0)
	case TransitionPushUp:
		applyPushTransition(dst, oldFrame, newFrame, progress, 0, -1)
	case TransitionPushDown:
		applyPushTransition(dst, oldFrame, newFrame, progress, 0, 1)

	case TransitionSlideLeft:
		applySlideTransition(dst, oldFrame, newFrame, progress, -1, 0)
	case TransitionSlideRight:
		applySlideTransition(dst, oldFrame, newFrame, progress, 1, 0)
	case TransitionSlideUp:
		applySlideTransition(dst, oldFrame, newFrame, progress, 0, -1)
	case TransitionSlideDown:
		applySlideTransition(dst, oldFrame, newFrame, progress, 0, 1)

	case TransitionDissolveFade:
		applyDissolveFade(dst, oldFrame, newFrame, progress)

	case TransitionDissolvePixel:
		applyDissolvePixel(dst, oldFrame, newFrame, progress, pixelOrder)

	case TransitionDissolveDither:
		applyDissolveDither(dst, oldFrame, newFrame, progress)

	case TransitionBoxIn:
		applyBoxTransition(dst, oldFrame, newFrame, progress, true)
	case TransitionBoxOut:
		applyBoxTransition(dst, oldFrame, newFrame, progress, false)

	case TransitionClockWipe:
		applyClockWipe(dst, oldFrame, newFrame, progress)

	default:
		// Unknown transition, just copy new frame
		CopyGrayImage(dst, newFrame)
	}
}

// CopyGrayImage copies src to dst
func CopyGrayImage(dst, src *image.Gray) {
	copy(dst.Pix, src.Pix)
}

// applyPushTransition pushes old frame out while new frame comes in
func applyPushTransition(dst, oldFrame, newFrame *image.Gray, progress float64, dirX, dirY int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate offset based on progress
	offsetX := int(float64(width) * progress * float64(dirX))
	offsetY := int(float64(height) * progress * float64(dirY))

	// Draw old frame (being pushed out)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x - offsetX
			srcY := y - offsetY
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dst.SetGray(x+bounds.Min.X, y+bounds.Min.Y, oldFrame.GrayAt(srcX+bounds.Min.X, srcY+bounds.Min.Y))
			}
		}
	}

	// Draw new frame (coming in)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// New frame enters from opposite side
			srcX := x - offsetX + width*dirX
			srcY := y - offsetY + height*dirY
			dstX := x + bounds.Min.X
			dstY := y + bounds.Min.Y
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dst.SetGray(dstX, dstY, newFrame.GrayAt(srcX+bounds.Min.X, srcY+bounds.Min.Y))
			}
		}
	}
}

// applySlideTransition slides new frame over old frame
func applySlideTransition(dst, oldFrame, newFrame *image.Gray, progress float64, dirX, dirY int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// First, copy old frame
	CopyGrayImage(dst, oldFrame)

	// Calculate new frame position (slides in from edge)
	var startX, startY int
	if dirX < 0 {
		startX = width - int(float64(width)*progress)
	} else if dirX > 0 {
		startX = int(float64(width)*progress) - width
	}
	if dirY < 0 {
		startY = height - int(float64(height)*progress)
	} else if dirY > 0 {
		startY = int(float64(height)*progress) - height
	}

	// Draw new frame on top
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x - startX
			srcY := y - startY
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				dstX := x + bounds.Min.X
				dstY := y + bounds.Min.Y
				dst.SetGray(dstX, dstY, newFrame.GrayAt(srcX+bounds.Min.X, srcY+bounds.Min.Y))
			}
		}
	}
}

// applyDissolveFade crossfades between old and new frames
func applyDissolveFade(dst, oldFrame, newFrame *image.Gray, progress float64) {
	bounds := dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldVal := float64(oldFrame.GrayAt(x, y).Y)
			newVal := float64(newFrame.GrayAt(x, y).Y)
			blended := uint8(oldVal*(1-progress) + newVal*progress)
			dst.SetGray(x, y, color.Gray{Y: blended})
		}
	}
}

// applyDissolvePixel randomly switches pixels from old to new
func applyDissolvePixel(dst, oldFrame, newFrame *image.Gray, progress float64, pixelOrder []int) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	total := len(pixelOrder)
	threshold := int(float64(total) * progress)

	// Copy old frame first
	CopyGrayImage(dst, oldFrame)

	// Replace pixels up to threshold with new frame
	for i := 0; i < threshold && i < total; i++ {
		idx := pixelOrder[i]
		x := idx % width
		y := idx / width
		dst.SetGray(x+bounds.Min.X, y+bounds.Min.Y, newFrame.GrayAt(x+bounds.Min.X, y+bounds.Min.Y))
	}
}

// applyDissolveDither uses ordered dithering pattern for transition
func applyDissolveDither(dst, oldFrame, newFrame *image.Gray, progress float64) {
	bounds := dst.Bounds()

	// 8x8 Bayer dithering matrix (values 0-63)
	bayer8x8 := [8][8]float64{
		{0, 32, 8, 40, 2, 34, 10, 42},
		{48, 16, 56, 24, 50, 18, 58, 26},
		{12, 44, 4, 36, 14, 46, 6, 38},
		{60, 28, 52, 20, 62, 30, 54, 22},
		{3, 35, 11, 43, 1, 33, 9, 41},
		{51, 19, 59, 27, 49, 17, 57, 25},
		{15, 47, 7, 39, 13, 45, 5, 37},
		{63, 31, 55, 23, 61, 29, 53, 21},
	}

	threshold := progress * 64.0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ditherVal := bayer8x8[y%8][x%8]
			if ditherVal < threshold {
				dst.SetGray(x, y, newFrame.GrayAt(x, y))
			} else {
				dst.SetGray(x, y, oldFrame.GrayAt(x, y))
			}
		}
	}
}

// applyBoxTransition reveals new frame through expanding/contracting box
func applyBoxTransition(dst, oldFrame, newFrame *image.Gray, progress float64, boxIn bool) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	centerX := width / 2
	centerY := height / 2

	// Calculate box dimensions
	var boxW, boxH int
	if boxIn {
		// Box shrinks from edges, revealing new content
		boxW = int(float64(width) * (1 - progress) / 2)
		boxH = int(float64(height) * (1 - progress) / 2)
	} else {
		// Box expands from center, revealing new content
		boxW = int(float64(width) * progress / 2)
		boxH = int(float64(height) * progress / 2)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dstX := x + bounds.Min.X
			dstY := y + bounds.Min.Y

			// Check if pixel is inside the box
			inBox := x >= centerX-boxW && x < centerX+boxW && y >= centerY-boxH && y < centerY+boxH

			if boxIn {
				// Box shrinks: outside box = new, inside box = old
				if inBox {
					dst.SetGray(dstX, dstY, oldFrame.GrayAt(dstX, dstY))
				} else {
					dst.SetGray(dstX, dstY, newFrame.GrayAt(dstX, dstY))
				}
			} else {
				// Box expands: inside box = new, outside box = old
				if inBox {
					dst.SetGray(dstX, dstY, newFrame.GrayAt(dstX, dstY))
				} else {
					dst.SetGray(dstX, dstY, oldFrame.GrayAt(dstX, dstY))
				}
			}
		}
	}
}

// applyClockWipe reveals new frame through clockwise radial sweep from 12 o'clock
func applyClockWipe(dst, oldFrame, newFrame *image.Gray, progress float64) {
	bounds := dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	centerX := float64(width) / 2
	centerY := float64(height) / 2

	// Sweep angle: 0 = 12 o'clock, progress 1.0 = full circle (360 degrees)
	sweepAngle := progress * 2 * math.Pi

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dstX := x + bounds.Min.X
			dstY := y + bounds.Min.Y

			// Calculate angle from center to this pixel
			// atan2 returns angle from positive X axis, so adjust for 12 o'clock start
			dx := float64(x) - centerX
			dy := float64(y) - centerY

			// Angle from 12 o'clock (top), clockwise
			// atan2(dx, -dy) gives angle from top, positive clockwise
			pixelAngle := math.Atan2(dx, -dy)
			if pixelAngle < 0 {
				pixelAngle += 2 * math.Pi
			}

			// If pixel angle is less than sweep angle, show new frame
			if pixelAngle < sweepAngle {
				dst.SetGray(dstX, dstY, newFrame.GrayAt(dstX, dstY))
			} else {
				dst.SetGray(dstX, dstY, oldFrame.GrayAt(dstX, dstY))
			}
		}
	}
}
