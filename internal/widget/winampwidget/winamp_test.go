package winampwidget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared/anim"
)

// TestNew tests basic widget creation
func TestNew(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		UpdateInterval: 0.2,
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w == nil {
		t.Fatal("New() returned nil")
	}
}

// TestNew_WithTextConfig tests widget creation with text configuration
func TestNew_WithTextConfig(t *testing.T) {
	h := config.AlignLeft
	v := config.AlignTop
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Text: &config.TextConfig{
			Size:   14,
			Format: "{title} - {status}",
			Align: &config.AlignConfig{
				H: h,
				V: v,
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.fontSize != 14 {
		t.Errorf("widget.fontSize = %d, want 14", w.fontSize)
	}
	if w.format != "{title} - {status}" {
		t.Errorf("widget.format = %q, want {title} - {status}", w.format)
	}
	if w.horizAlign != config.AlignLeft {
		t.Errorf("widget.horizAlign = %q, want left", w.horizAlign)
	}
	if w.vertAlign != config.AlignTop {
		t.Errorf("widget.vertAlign = %q, want top", w.vertAlign)
	}
}

// TestNew_WithScrollConfig tests widget creation with scroll configuration
func TestNew_WithScrollConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Scroll: &config.ScrollConfig{
			Enabled:   true,
			Direction: "right",
			Speed:     50.0,
			Mode:      "bounce",
			PauseMs:   2000,
			Gap:       30,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !w.scrollEnabled {
		t.Error("widget.scrollEnabled = false, want true")
	}
	scrollCfg := w.scroller.GetConfig()
	if scrollCfg.Direction != anim.ScrollRight {
		t.Errorf("scrollCfg.Direction = %q, want right", scrollCfg.Direction)
	}
	if scrollCfg.Speed != 50.0 {
		t.Errorf("scrollCfg.Speed = %f, want 50.0", scrollCfg.Speed)
	}
	if scrollCfg.Mode != anim.ScrollBounce {
		t.Errorf("scrollCfg.Mode = %q, want bounce", scrollCfg.Mode)
	}
	if scrollCfg.PauseMs != 2000 {
		t.Errorf("scrollCfg.PauseMs = %d, want 2000", scrollCfg.PauseMs)
	}
	if w.scrollGap != 30 {
		t.Errorf("widget.scrollGap = %d, want 30", w.scrollGap)
	}
}

// TestNew_WithAutoShowConfig tests widget creation with auto-show configuration
func TestNew_WithAutoShowConfig(t *testing.T) {
	onTrackChange := false
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		AutoShow: &config.WinampAutoShowConfig{
			OnTrackChange: &onTrackChange,
			OnPlay:        true,
			OnPause:       true,
			OnStop:        true,
			OnSeek:        true,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.autoShowOnTrackChange {
		t.Error("widget.autoShowOnTrackChange = true, want false")
	}
	if !w.autoShowOnPlay {
		t.Error("widget.autoShowOnPlay = false, want true")
	}
	if !w.autoShowOnPause {
		t.Error("widget.autoShowOnPause = false, want true")
	}
	if !w.autoShowOnStop {
		t.Error("widget.autoShowOnStop = false, want true")
	}
	if !w.autoShowOnSeek {
		t.Error("widget.autoShowOnSeek = false, want true")
	}
}

// TestNew_WithPlaceholderConfig tests widget creation with placeholder configuration
func TestNew_WithPlaceholderConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Winamp: &config.WinampConfig{
			Placeholder: &config.WinampPlaceholderConfig{
				Mode: "text",
				Text: "Custom Placeholder",
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.placeholderMode != "text" {
		t.Errorf("widget.placeholderMode = %q, want text", w.placeholderMode)
	}
	if w.placeholderText != "Custom Placeholder" {
		t.Errorf("widget.placeholderText = %q, want Custom Placeholder", w.placeholderText)
	}
}

// TestWidget_Defaults tests default values
func TestWidget_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Check default values
	if w.format != "{title}" {
		t.Errorf("default format = %q, want {title}", w.format)
	}
	if w.fontSize != 12 {
		t.Errorf("default fontSize = %d, want 12", w.fontSize)
	}
	if w.horizAlign != "center" {
		t.Errorf("default horizAlign = %q, want center", w.horizAlign)
	}
	if w.vertAlign != "center" {
		t.Errorf("default vertAlign = %q, want center", w.vertAlign)
	}
	if w.placeholderMode != "icon" {
		t.Errorf("default placeholderMode = %q, want icon", w.placeholderMode)
	}
	if w.placeholderText != "No Winamp" {
		t.Errorf("default placeholderText = %q, want No Winamp", w.placeholderText)
	}
	if w.scrollEnabled {
		t.Error("default scrollEnabled = true, want false")
	}
	defaultScrollCfg := w.scroller.GetConfig()
	if defaultScrollCfg.Direction != anim.ScrollLeft {
		t.Errorf("default scrollDirection = %q, want left", defaultScrollCfg.Direction)
	}
	if defaultScrollCfg.Speed != 30.0 {
		t.Errorf("default scrollSpeed = %f, want 30.0", defaultScrollCfg.Speed)
	}
	if defaultScrollCfg.Mode != anim.ScrollContinuous {
		t.Errorf("default scrollMode = %q, want continuous", defaultScrollCfg.Mode)
	}
	if !w.autoShowOnTrackChange {
		t.Error("default autoShowOnTrackChange = false, want true")
	}
	if w.autoShowOnPlay {
		t.Error("default autoShowOnPlay = true, want false")
	}
}

// TestWidget_Update tests the Update method
func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update should not return error
	if err := w.Update(); err != nil {
		t.Errorf("Update() error = %v, want nil", err)
	}
}

// TestWidget_Render tests the Render method
func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render should return an image (placeholder when no Winamp)
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}

	// Should return an image with correct dimensions
	if img != nil {
		bounds := img.Bounds()
		if bounds.Dx() != 128 || bounds.Dy() != 40 {
			t.Errorf("Render() bounds = %v, want (128, 40)", bounds)
		}
	}
}

// TestWidget_Render_TextPlaceholder tests rendering with text placeholder
func TestWidget_Render_TextPlaceholder(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Winamp: &config.WinampConfig{
			Placeholder: &config.WinampPlaceholderConfig{
				Mode: "text",
				Text: "No Music",
			},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

// TestFormatTime tests the formatTime helper function
func TestFormatTime(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "00:00"},
		{30, "00:30"},
		{60, "01:00"},
		{90, "01:30"},
		{600, "10:00"},
		{3661, "61:01"},
		{-1, "--:--"},
		{-100, "--:--"},
	}

	for _, tt := range tests {
		got := formatTime(tt.seconds)
		if got != tt.expected {
			t.Errorf("formatTime(%d) = %q, want %q", tt.seconds, got, tt.expected)
		}
	}
}
