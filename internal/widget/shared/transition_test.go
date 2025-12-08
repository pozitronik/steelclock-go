package shared

import (
	"image"
	"image/color"
	"testing"
	"time"
)

func TestNewTransitionManager(t *testing.T) {
	tm := NewTransitionManager(128, 40)

	if tm == nil {
		t.Fatal("NewTransitionManager returned nil")
	}
	if tm.width != 128 {
		t.Errorf("width = %d, want 128", tm.width)
	}
	if tm.height != 40 {
		t.Errorf("height = %d, want 40", tm.height)
	}
	if tm.IsActive() {
		t.Error("IsActive() = true, want false initially")
	}
}

func TestTransitionManager_Start(t *testing.T) {
	tm := NewTransitionManager(128, 40)
	oldFrame := createTestFrame(128, 40, 100)

	tm.Start(TransitionDissolveFade, 0.5, oldFrame)

	if !tm.IsActive() {
		t.Error("IsActive() = false after Start, want true")
	}
	if tm.Progress() != 0.0 {
		t.Errorf("Progress() = %f, want 0.0", tm.Progress())
	}
	if tm.OldFrame() != oldFrame {
		t.Error("OldFrame() does not match provided frame")
	}
	if tm.Type() != TransitionDissolveFade {
		t.Errorf("Type() = %s, want %s", tm.Type(), TransitionDissolveFade)
	}
}

func TestTransitionManager_StartNone(t *testing.T) {
	tm := NewTransitionManager(128, 40)
	oldFrame := createTestFrame(128, 40, 100)

	tm.Start(TransitionNone, 0.5, oldFrame)

	if tm.IsActive() {
		t.Error("IsActive() = true after Start with TransitionNone, want false")
	}
}

func TestTransitionManager_StartRandom(t *testing.T) {
	tm := NewTransitionManager(128, 40)
	oldFrame := createTestFrame(128, 40, 100)

	tm.Start(TransitionRandom, 0.5, oldFrame)

	if !tm.IsActive() {
		t.Error("IsActive() = false after Start with TransitionRandom, want true")
	}
	// Should have selected a real transition type
	if tm.Type() == TransitionRandom || tm.Type() == TransitionNone {
		t.Errorf("Type() = %s, should be a concrete transition type", tm.Type())
	}
}

func TestTransitionManager_StartDissolvePixel(t *testing.T) {
	tm := NewTransitionManager(10, 10)
	oldFrame := createTestFrame(10, 10, 100)

	tm.Start(TransitionDissolvePixel, 0.5, oldFrame)

	if tm.PixelOrder() == nil {
		t.Error("PixelOrder() = nil for dissolve_pixel transition")
	}
	if len(tm.PixelOrder()) != 100 {
		t.Errorf("len(PixelOrder()) = %d, want 100", len(tm.PixelOrder()))
	}
}

func TestTransitionManager_Update(t *testing.T) {
	tm := NewTransitionManager(128, 40)
	oldFrame := createTestFrame(128, 40, 100)

	// Use a very short duration for testing
	tm.Start(TransitionDissolveFade, 0.01, oldFrame)

	// Should be active initially
	if !tm.IsActive() {
		t.Error("IsActive() = false after Start, want true")
	}

	// Wait for transition to complete
	time.Sleep(20 * time.Millisecond)
	tm.Update()

	// Should be inactive after duration
	if tm.IsActive() {
		t.Error("IsActive() = true after duration, want false")
	}
	if tm.Progress() != 1.0 {
		t.Errorf("Progress() = %f, want 1.0", tm.Progress())
	}
	if tm.OldFrame() != nil {
		t.Error("OldFrame() should be nil after completion")
	}
}

func TestTransitionManager_Reset(t *testing.T) {
	tm := NewTransitionManager(128, 40)
	oldFrame := createTestFrame(128, 40, 100)

	tm.Start(TransitionDissolvePixel, 1.0, oldFrame)
	tm.Reset()

	if tm.IsActive() {
		t.Error("IsActive() = true after Reset, want false")
	}
	if tm.Progress() != 0.0 {
		t.Errorf("Progress() = %f after Reset, want 0.0", tm.Progress())
	}
	if tm.OldFrame() != nil {
		t.Error("OldFrame() should be nil after Reset")
	}
	if tm.PixelOrder() != nil {
		t.Error("PixelOrder() should be nil after Reset")
	}
}

func TestTransitionManager_Cancel(t *testing.T) {
	tm := NewTransitionManager(128, 40)
	oldFrame := createTestFrame(128, 40, 100)

	tm.Start(TransitionDissolveFade, 1.0, oldFrame)
	tm.Cancel()

	if tm.IsActive() {
		t.Error("IsActive() = true after Cancel, want false")
	}
}

func TestSelectTransition(t *testing.T) {
	// Specific transitions should return themselves
	if got := SelectTransition(TransitionDissolveFade); got != TransitionDissolveFade {
		t.Errorf("SelectTransition(dissolve_fade) = %s, want dissolve_fade", got)
	}

	// Random should return a valid transition
	for i := 0; i < 10; i++ {
		got := SelectTransition(TransitionRandom)
		if got == TransitionRandom || got == TransitionNone {
			t.Errorf("SelectTransition(random) returned %s", got)
		}
	}
}

func TestGeneratePixelOrder(t *testing.T) {
	order := GeneratePixelOrder(10, 10)

	if len(order) != 100 {
		t.Errorf("len(order) = %d, want 100", len(order))
	}

	// Check all indices are present
	seen := make(map[int]bool)
	for _, idx := range order {
		if idx < 0 || idx >= 100 {
			t.Errorf("invalid index %d in order", idx)
		}
		if seen[idx] {
			t.Errorf("duplicate index %d in order", idx)
		}
		seen[idx] = true
	}
}

func TestCopyGrayImage(t *testing.T) {
	src := createTestFrame(10, 10, 128)
	dst := createTestFrame(10, 10, 0)

	CopyGrayImage(dst, src)

	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if dst.GrayAt(x, y).Y != 128 {
				t.Errorf("dst[%d,%d] = %d, want 128", x, y, dst.GrayAt(x, y).Y)
			}
		}
	}
}

func TestApplyTransition_None(t *testing.T) {
	oldFrame := createTestFrame(10, 10, 0)
	newFrame := createTestFrame(10, 10, 255)
	dst := createTestFrame(10, 10, 128)

	// At progress 0.4, should show old frame
	ApplyTransition(dst, oldFrame, newFrame, 0.4, TransitionNone, nil)
	if dst.GrayAt(5, 5).Y != 0 {
		t.Errorf("at progress 0.4, pixel = %d, want 0 (old)", dst.GrayAt(5, 5).Y)
	}

	// At progress 0.6, should show new frame
	ApplyTransition(dst, oldFrame, newFrame, 0.6, TransitionNone, nil)
	if dst.GrayAt(5, 5).Y != 255 {
		t.Errorf("at progress 0.6, pixel = %d, want 255 (new)", dst.GrayAt(5, 5).Y)
	}
}

func TestApplyTransition_DissolveFade(t *testing.T) {
	oldFrame := createTestFrame(10, 10, 0)
	newFrame := createTestFrame(10, 10, 200)
	dst := createTestFrame(10, 10, 128)

	// At progress 0.5, should be blended
	ApplyTransition(dst, oldFrame, newFrame, 0.5, TransitionDissolveFade, nil)

	expected := uint8(100) // 0 * 0.5 + 200 * 0.5 = 100
	if dst.GrayAt(5, 5).Y != expected {
		t.Errorf("at progress 0.5, pixel = %d, want %d", dst.GrayAt(5, 5).Y, expected)
	}
}

func TestApplyTransition_DissolvePixel(t *testing.T) {
	oldFrame := createTestFrame(10, 10, 0)
	newFrame := createTestFrame(10, 10, 255)
	dst := createTestFrame(10, 10, 128)
	pixelOrder := GeneratePixelOrder(10, 10)

	// At progress 0.0, all pixels should be old
	ApplyTransition(dst, oldFrame, newFrame, 0.0, TransitionDissolvePixel, pixelOrder)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if dst.GrayAt(x, y).Y != 0 {
				t.Errorf("at progress 0.0, pixel[%d,%d] = %d, want 0", x, y, dst.GrayAt(x, y).Y)
			}
		}
	}

	// At progress 1.0, all pixels should be new
	ApplyTransition(dst, oldFrame, newFrame, 1.0, TransitionDissolvePixel, pixelOrder)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if dst.GrayAt(x, y).Y != 255 {
				t.Errorf("at progress 1.0, pixel[%d,%d] = %d, want 255", x, y, dst.GrayAt(x, y).Y)
			}
		}
	}
}

func TestApplyTransition_AllTypes(t *testing.T) {
	// Ensure all transition types can be applied without panic
	oldFrame := createTestFrame(20, 20, 50)
	newFrame := createTestFrame(20, 20, 200)
	dst := createTestFrame(20, 20, 0)
	pixelOrder := GeneratePixelOrder(20, 20)

	allTypes := []TransitionType{
		TransitionNone,
		TransitionPushLeft, TransitionPushRight, TransitionPushUp, TransitionPushDown,
		TransitionSlideLeft, TransitionSlideRight, TransitionSlideUp, TransitionSlideDown,
		TransitionDissolveFade, TransitionDissolvePixel, TransitionDissolveDither,
		TransitionBoxIn, TransitionBoxOut, TransitionClockWipe,
	}

	for _, tt := range allTypes {
		t.Run(string(tt), func(t *testing.T) {
			for progress := 0.0; progress <= 1.0; progress += 0.25 {
				ApplyTransition(dst, oldFrame, newFrame, progress, tt, pixelOrder)
			}
		})
	}
}

func TestTransitionManager_Apply(t *testing.T) {
	tm := NewTransitionManager(10, 10)
	oldFrame := createTestFrame(10, 10, 0)
	newFrame := createTestFrame(10, 10, 200)
	dst := createTestFrame(10, 10, 128)

	tm.Start(TransitionDissolveFade, 1.0, oldFrame)

	// Simulate half progress
	tm.startTime = time.Now().Add(-500 * time.Millisecond)
	tm.Update()

	tm.Apply(dst, newFrame)

	// At ~50% progress, should be blended
	pixel := dst.GrayAt(5, 5).Y
	if pixel < 80 || pixel > 120 {
		t.Errorf("at ~50%% progress, pixel = %d, expected around 100", pixel)
	}
}

func TestTransitionManager_IsActiveLive(t *testing.T) {
	tm := NewTransitionManager(10, 10)
	oldFrame := createTestFrame(10, 10, 100)

	// Before starting, should return false
	if tm.IsActiveLive() {
		t.Error("IsActiveLive() = true before Start, want false")
	}

	// After starting with long duration, should return true
	tm.Start(TransitionDissolveFade, 10.0, oldFrame) // 10 second duration
	if !tm.IsActiveLive() {
		t.Error("IsActiveLive() = false after Start with long duration, want true")
	}

	// After duration elapsed, should return false
	tm.Reset()
	tm.Start(TransitionDissolveFade, 0.01, oldFrame) // 10ms duration
	time.Sleep(50 * time.Millisecond)                // Wait for it to expire
	if tm.IsActiveLive() {
		t.Error("IsActiveLive() = true after duration elapsed, want false")
	}

	// When not active, should return false
	tm.Cancel()
	if tm.IsActiveLive() {
		t.Error("IsActiveLive() = true after Cancel, want false")
	}
}

func TestTransitionManager_LiveProgress(t *testing.T) {
	tm := NewTransitionManager(10, 10)
	oldFrame := createTestFrame(10, 10, 100)

	// Before starting, should return 0.0
	if tm.LiveProgress() != 0.0 {
		t.Errorf("LiveProgress() = %f before Start, want 0.0", tm.LiveProgress())
	}

	// Just after starting, should be close to 0
	tm.Start(TransitionDissolveFade, 1.0, oldFrame)
	progress := tm.LiveProgress()
	if progress < 0.0 || progress > 0.1 {
		t.Errorf("LiveProgress() = %f just after Start, expected near 0.0", progress)
	}

	// After some time, should have increased
	time.Sleep(100 * time.Millisecond)
	progress = tm.LiveProgress()
	if progress < 0.05 || progress > 0.3 {
		t.Errorf("LiveProgress() = %f after 100ms with 1s duration, expected ~0.1", progress)
	}

	// After full duration, should be capped at 1.0
	tm.Reset()
	tm.Start(TransitionDissolveFade, 0.01, oldFrame) // 10ms duration
	time.Sleep(50 * time.Millisecond)
	progress = tm.LiveProgress()
	if progress != 1.0 {
		t.Errorf("LiveProgress() = %f after duration exceeded, want 1.0", progress)
	}

	// When not active, should return 0.0
	tm.Cancel()
	if tm.LiveProgress() != 0.0 {
		t.Errorf("LiveProgress() = %f after Cancel, want 0.0", tm.LiveProgress())
	}
}

func TestTransitionManager_ApplyLive(t *testing.T) {
	tm := NewTransitionManager(10, 10)
	oldFrame := createTestFrame(10, 10, 0)
	newFrame := createTestFrame(10, 10, 200)
	dst := createTestFrame(10, 10, 128)

	// Without starting, should copy new frame
	tm.ApplyLive(dst, newFrame)
	if dst.GrayAt(5, 5).Y != 200 {
		t.Errorf("ApplyLive without oldFrame: pixel = %d, want 200", dst.GrayAt(5, 5).Y)
	}

	// Start a transition
	tm.Start(TransitionDissolveFade, 1.0, oldFrame)

	// Just after starting, should be mostly old frame
	tm.ApplyLive(dst, newFrame)
	pixel := dst.GrayAt(5, 5).Y
	if pixel > 50 {
		t.Errorf("ApplyLive just after start: pixel = %d, expected closer to 0", pixel)
	}

	// Wait a bit and apply again
	time.Sleep(200 * time.Millisecond)
	tm.ApplyLive(dst, newFrame)
	pixel = dst.GrayAt(5, 5).Y
	// Should be somewhere between old and new (0 to 200)
	if pixel < 10 || pixel > 80 {
		t.Errorf("ApplyLive after 200ms: pixel = %d, expected blend ~20-60", pixel)
	}
}

func TestTransitionManager_Apply_NoOldFrame(t *testing.T) {
	tm := NewTransitionManager(10, 10)
	newFrame := createTestFrame(10, 10, 200)
	dst := createTestFrame(10, 10, 0)

	// Apply without starting a transition (no oldFrame)
	tm.Apply(dst, newFrame)

	// Should just copy newFrame to dst
	if dst.GrayAt(5, 5).Y != 200 {
		t.Errorf("Apply without oldFrame: pixel = %d, want 200", dst.GrayAt(5, 5).Y)
	}
}

// createTestFrame creates a grayscale image filled with the given value
func createTestFrame(w, h int, value uint8) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, w, h))
	c := color.Gray{Y: value}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
