package app

import (
	"testing"
)

// mockSplashClient is a mock client for splash screen testing
type mockSplashClient struct {
	framesSent    int
	lastFrameData []int
	sendErr       error
}

func (m *mockSplashClient) RegisterGame(_ string, _ int) error {
	return nil
}

func (m *mockSplashClient) BindScreenEvent(_, _ string) error {
	return nil
}

func (m *mockSplashClient) SendScreenData(_ string, bitmapData []int) error {
	m.framesSent++
	m.lastFrameData = bitmapData
	return m.sendErr
}

func (m *mockSplashClient) SendScreenDataMultiRes(_ string, _ map[string][]int) error {
	return nil
}

func (m *mockSplashClient) SendHeartbeat() error {
	return nil
}

func (m *mockSplashClient) RemoveGame() error {
	return nil
}

func (m *mockSplashClient) SupportsMultipleEvents() bool {
	return false
}

func (m *mockSplashClient) SendMultipleScreenData(_ string, _ [][]int) error {
	return nil
}

func TestNewSplashRenderer(t *testing.T) {
	client := &mockSplashClient{}
	splash := NewSplashRenderer(client, 128, 40)

	if splash == nil {
		t.Fatal("NewSplashRenderer returned nil")
	}
	if splash.width != 128 {
		t.Errorf("width = %d, want 128", splash.width)
	}
	if splash.height != 40 {
		t.Errorf("height = %d, want 40", splash.height)
	}
}

func TestSplashRenderer_NilClient(t *testing.T) {
	splash := NewSplashRenderer(nil, 128, 40)

	// Should not panic with nil client
	if err := splash.ShowStartupAnimation(); err != nil {
		t.Errorf("ShowStartupAnimation with nil client returned error: %v", err)
	}
	if err := splash.ShowTransitionBanner("Test"); err != nil {
		t.Errorf("ShowTransitionBanner with nil client returned error: %v", err)
	}
	if err := splash.ShowExitMessage(); err != nil {
		t.Errorf("ShowExitMessage with nil client returned error: %v", err)
	}
}

func TestSplashRenderer_RenderStartupFrame(t *testing.T) {
	splash := NewSplashRenderer(nil, 128, 40)

	// Test rendering at different progress points
	testProgress := []float64{0.0, 0.3, 0.5, 0.7, 1.0}
	for _, progress := range testProgress {
		img := splash.renderStartupFrame(progress)
		if img == nil {
			t.Errorf("renderStartupFrame(%f) returned nil", progress)
			continue
		}

		bounds := img.Bounds()
		if bounds.Dx() != 128 || bounds.Dy() != 40 {
			t.Errorf("renderStartupFrame(%f) returned wrong dimensions: %dx%d, want 128x40",
				progress, bounds.Dx(), bounds.Dy())
		}
	}
}

func TestSplashRenderer_RenderTransitionFrame(t *testing.T) {
	splash := NewSplashRenderer(nil, 128, 40)

	profileNames := []string{"Default", "Gaming", "A Very Long Profile Name"}
	testProgress := []float64{0.0, 0.1, 0.5, 0.9, 1.0}

	for _, name := range profileNames {
		for _, progress := range testProgress {
			img := splash.renderTransitionFrame(name, progress)
			if img == nil {
				t.Errorf("renderTransitionFrame(%q, %f) returned nil", name, progress)
				continue
			}

			bounds := img.Bounds()
			if bounds.Dx() != 128 || bounds.Dy() != 40 {
				t.Errorf("renderTransitionFrame(%q, %f) returned wrong dimensions: %dx%d, want 128x40",
					name, progress, bounds.Dx(), bounds.Dy())
			}
		}
	}
}

func TestSplashRenderer_RenderExitFrame(t *testing.T) {
	splash := NewSplashRenderer(nil, 128, 40)

	testProgress := []float64{0.0, 0.2, 0.5, 0.8, 1.0}
	for _, progress := range testProgress {
		img := splash.renderExitFrame(progress)
		if img == nil {
			t.Errorf("renderExitFrame(%f) returned nil", progress)
			continue
		}

		bounds := img.Bounds()
		if bounds.Dx() != 128 || bounds.Dy() != 40 {
			t.Errorf("renderExitFrame(%f) returned wrong dimensions: %dx%d, want 128x40",
				progress, bounds.Dx(), bounds.Dy())
		}
	}
}

func TestSplashRenderer_Abs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
		{100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

//goland:noinspection GoBoolExpressions
func TestSplashAnimationConstants(t *testing.T) {
	// Verify constants are reasonable
	if StartupAnimationDuration <= 0 {
		t.Error("StartupAnimationDuration should be positive")
	}
	if StartupFrameInterval <= 0 {
		t.Error("StartupFrameInterval should be positive")
	}
	if TransitionAnimationDuration <= 0 {
		t.Error("TransitionAnimationDuration should be positive")
	}
	if TransitionFrameInterval <= 0 {
		t.Error("TransitionFrameInterval should be positive")
	}
	if ExitAnimationDuration <= 0 {
		t.Error("ExitAnimationDuration should be positive")
	}
	if ExitFrameInterval <= 0 {
		t.Error("ExitFrameInterval should be positive")
	}

	// Verify frame intervals divide duration reasonably
	startupFrames := int(StartupAnimationDuration / StartupFrameInterval)
	if startupFrames < 10 {
		t.Errorf("Startup animation has too few frames: %d", startupFrames)
	}

	transitionFrames := int(TransitionAnimationDuration / TransitionFrameInterval)
	if transitionFrames < 10 {
		t.Errorf("Transition animation has too few frames: %d", transitionFrames)
	}

	exitFrames := int(ExitAnimationDuration / ExitFrameInterval)
	if exitFrames < 10 {
		t.Errorf("Exit animation has too few frames: %d", exitFrames)
	}
}
