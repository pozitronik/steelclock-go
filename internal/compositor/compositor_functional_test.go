package compositor

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/layout"
	"github.com/pozitronik/steelclock-go/internal/testutil"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

// renderingWidget is a mock widget that renders visible content
type renderingWidget struct {
	name           string
	updateInterval time.Duration
	position       config.PositionConfig
	style          config.StyleConfig
	pattern        string // "checker", "solid", "border"
}

func newRenderingWidget(name string, x, y, w, h int, pattern string) *renderingWidget {
	return &renderingWidget{
		name:           name,
		updateInterval: 100 * time.Millisecond,
		position:       config.PositionConfig{X: x, Y: y, W: w, H: h},
		style:          config.StyleConfig{Background: 0},
		pattern:        pattern,
	}
}

func (r *renderingWidget) Name() string                       { return r.name }
func (r *renderingWidget) Update() error                      { return nil }
func (r *renderingWidget) GetUpdateInterval() time.Duration   { return r.updateInterval }
func (r *renderingWidget) GetPosition() config.PositionConfig { return r.position }
func (r *renderingWidget) GetStyle() config.StyleConfig       { return r.style }

func (r *renderingWidget) Render() (image.Image, error) {
	img := image.NewGray(image.Rect(0, 0, r.position.W, r.position.H))

	switch r.pattern {
	case "solid":
		for y := 0; y < r.position.H; y++ {
			for x := 0; x < r.position.W; x++ {
				img.Set(x, y, color.White)
			}
		}
	case "checker":
		for y := 0; y < r.position.H; y++ {
			for x := 0; x < r.position.W; x++ {
				if (x+y)%2 == 0 {
					img.Set(x, y, color.White)
				}
			}
		}
	case "border":
		for x := 0; x < r.position.W; x++ {
			img.Set(x, 0, color.White)
			img.Set(x, r.position.H-1, color.White)
		}
		for y := 0; y < r.position.H; y++ {
			img.Set(0, y, color.White)
			img.Set(r.position.W-1, y, color.White)
		}
	}

	return img, nil
}

// TestFunctional_TestClientIntegration demonstrates using testutil.TestClient
func TestFunctional_TestClientIntegration(t *testing.T) {
	// Create test client with frame capture
	client := testutil.NewTestClient(
		testutil.WithDimensions(128, 40),
		testutil.WithMaxFrames(50),
	)

	// Create a widget that renders visible content
	widgets := []widget.Widget{
		newRenderingWidget("solid-widget", 0, 0, 128, 40, "solid"),
	}

	displayCfg := config.DisplayConfig{Width: 128, Height: 40, Background: 0}
	layoutMgr := layout.NewManager(displayCfg, widgets)

	// Disable frame deduplication for this test as we're testing frame capture, not deduplication
	dedupDisabled := false
	cfg := &config.Config{
		RefreshRateMs:     50,
		FrameDedupEnabled: &dedupDisabled,
		Display:           displayCfg,
	}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Start and capture frames
	_ = comp.Start()
	err := client.WaitForFrames(5, 2*time.Second)
	comp.Stop()

	if err != nil {
		t.Fatalf("Failed to capture frames: %v", err)
	}

	// Verify frames were captured
	if client.FrameCount() < 5 {
		t.Errorf("Expected at least 5 frames, got %d", client.FrameCount())
	}

	// Verify frames are not blank
	lastFrame := client.LastFrame()
	if lastFrame == nil {
		t.Fatal("No frame captured")
	}

	if testutil.IsBlankFrame(lastFrame.Data) {
		t.Error("Frame should not be blank with solid widget")
	}

	pixelCount := testutil.CountSetPixels(lastFrame.Data)
	t.Logf("Frame has %d set pixels (expected ~5120 for full white)", pixelCount)

	// Full white frame should have all pixels set
	if !testutil.IsFullFrame(lastFrame.Data) {
		t.Error("Solid widget should produce full white frame")
	}
}

// TestFunctional_FrameComparison demonstrates frame comparison capabilities
func TestFunctional_FrameComparison(t *testing.T) {
	client := testutil.NewTestClient()

	// Widget with checkerboard pattern
	widgets := []widget.Widget{
		newRenderingWidget("checker", 0, 0, 128, 40, "checker"),
	}

	displayCfg := config.DisplayConfig{Width: 128, Height: 40, Background: 0}
	layoutMgr := layout.NewManager(displayCfg, widgets)
	cfg := &config.Config{RefreshRateMs: 50, Display: displayCfg}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frames := client.Frames()
	if len(frames) < 2 {
		t.Skip("Not enough frames")
	}

	// Consecutive frames should be identical (static pattern)
	diff := testutil.CompareFrames(frames[0].Data, frames[1].Data)

	if !diff.Identical {
		t.Errorf("Static pattern frames should be identical, got %d different pixels", diff.DifferentPixels)
		if testing.Verbose() {
			t.Log(testutil.FrameDiffToASCII(frames[0].Data, frames[1].Data))
		}
	}

	// Checkerboard should have ~50% pixels set
	pixelCount := testutil.CountSetPixels(frames[0].Data)
	totalPixels := 128 * 40
	ratio := float64(pixelCount) / float64(totalPixels)
	t.Logf("Checkerboard has %.1f%% pixels set", ratio*100)

	if ratio < 0.4 || ratio > 0.6 {
		t.Errorf("Checkerboard should have ~50%% pixels set, got %.1f%%", ratio*100)
	}
}

// TestFunctional_RegionComparison demonstrates region-based comparison
func TestFunctional_RegionComparison(t *testing.T) {
	client := testutil.NewTestClient()

	// Two widgets in different positions
	widgets := []widget.Widget{
		newRenderingWidget("left", 0, 0, 64, 40, "solid"),
		newRenderingWidget("right", 64, 0, 64, 40, "border"),
	}

	displayCfg := config.DisplayConfig{Width: 128, Height: 40, Background: 0}
	layoutMgr := layout.NewManager(displayCfg, widgets)
	cfg := &config.Config{RefreshRateMs: 50, Display: displayCfg}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(3, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame captured")
	}

	// Create expected blank frame for comparison
	blank := make([]byte, 640)

	// Left half should be fully set (solid white)
	leftDiff := testutil.CompareRegion(frame.Data, blank, 0, 0, 64, 40)
	t.Logf("Left region: %d pixels different from blank", leftDiff.DifferentPixels)

	// Right half should have border only (not fully set)
	rightDiff := testutil.CompareRegion(frame.Data, blank, 64, 0, 64, 40)
	t.Logf("Right region: %d pixels different from blank", rightDiff.DifferentPixels)

	// Left (solid) should have more pixels than right (border only)
	if leftDiff.DifferentPixels <= rightDiff.DifferentPixels {
		t.Error("Solid region should have more pixels than border region")
	}
}

// TestFunctional_ErrorInjection demonstrates error injection
func TestFunctional_ErrorInjection(t *testing.T) {
	client := testutil.NewTestClient()

	widgets := []widget.Widget{
		newRenderingWidget("test", 0, 0, 128, 40, "solid"),
	}

	displayCfg := config.DisplayConfig{Width: 128, Height: 40, Background: 0}
	layoutMgr := layout.NewManager(displayCfg, widgets)
	cfg := &config.Config{RefreshRateMs: 50, Display: displayCfg}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	// Inject errors for first 5 sends
	testErr := &testError{msg: "injected error"}
	client.SetSendError(testErr, 5)

	_ = comp.Start()

	// Wait enough time for several frame attempts
	time.Sleep(400 * time.Millisecond)

	comp.Stop()

	// Frames should still be captured after errors cleared
	t.Logf("Captured %d frames (first 5 send attempts failed)", client.FrameCount())

	// Call tracking
	calls := client.Calls()
	sendCalls := 0
	for _, c := range calls {
		if c.Method == "SendScreenData" || c.Method == "SendScreenDataMultiRes" {
			sendCalls++
		}
	}
	t.Logf("Total send method calls: %d", sendCalls)
}

// TestFunctional_TimingVerification demonstrates timing analysis
func TestFunctional_TimingVerification(t *testing.T) {
	client := testutil.NewTestClient()

	widgets := []widget.Widget{
		newRenderingWidget("test", 0, 0, 128, 40, "solid"),
	}

	displayCfg := config.DisplayConfig{Width: 128, Height: 40, Background: 0}
	layoutMgr := layout.NewManager(displayCfg, widgets)

	// 100ms refresh rate = 10 FPS
	cfg := &config.Config{RefreshRateMs: 100, Display: displayCfg}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(10, 3*time.Second)
	comp.Stop()

	stats := client.CalculateTimingStats()
	t.Logf("Timing stats: %s", stats.String())

	// Verify frame rate is approximately correct (within 50% tolerance)
	err := client.VerifyFrameRate(100, 50)
	if err != nil {
		t.Logf("Frame rate verification: %v", err)
		// Don't fail - timing can be variable in tests
	}

	if stats.AverageFPS < 5 || stats.AverageFPS > 15 {
		t.Logf("FPS outside expected range (5-15): %.1f", stats.AverageFPS)
	}
}

// TestFunctional_PixelInspection demonstrates individual pixel inspection
func TestFunctional_PixelInspection(t *testing.T) {
	client := testutil.NewTestClient()

	// Border widget - pixels at edges only
	widgets := []widget.Widget{
		newRenderingWidget("border", 0, 0, 128, 40, "border"),
	}

	displayCfg := config.DisplayConfig{Width: 128, Height: 40, Background: 0}
	layoutMgr := layout.NewManager(displayCfg, widgets)
	cfg := &config.Config{RefreshRateMs: 50, Display: displayCfg}

	comp := NewCompositor(client, layoutMgr, widgets, cfg)

	_ = comp.Start()
	_ = client.WaitForFrames(1, 2*time.Second)
	comp.Stop()

	frame := client.LastFrame()
	if frame == nil {
		t.Fatal("No frame")
	}

	// Check corner pixels (should be set)
	corners := []struct{ x, y int }{
		{0, 0}, {127, 0}, {0, 39}, {127, 39},
	}
	for _, c := range corners {
		pixel := testutil.GetPixel(frame.Data, c.x, c.y)
		if pixel != 1 {
			t.Errorf("Corner pixel (%d,%d) should be set, got %d", c.x, c.y, pixel)
		}
	}

	// Check center pixel (should not be set)
	centerPixel := testutil.GetPixel(frame.Data, 64, 20)
	if centerPixel != 0 {
		t.Errorf("Center pixel should not be set, got %d", centerPixel)
	}

	// Check first row
	row0 := testutil.GetRow(frame.Data, 0)
	setInRow := 0
	for _, p := range row0 {
		if p == 1 {
			setInRow++
		}
	}
	t.Logf("First row has %d set pixels (expected 128 for border)", setInRow)
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
