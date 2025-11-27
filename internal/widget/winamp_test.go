package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestNewWinampWidget tests basic widget creation
func TestNewWinampWidget(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		UpdateInterval: 0.2,
	}

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}
	if widget == nil {
		t.Fatal("NewWinampWidget() returned nil")
	}
}

// TestNewWinampWidget_WithTextConfig tests widget creation with text configuration
func TestNewWinampWidget_WithTextConfig(t *testing.T) {
	h := "left"
	v := "top"
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

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	if widget.fontSize != 14 {
		t.Errorf("widget.fontSize = %d, want 14", widget.fontSize)
	}
	if widget.format != "{title} - {status}" {
		t.Errorf("widget.format = %q, want {title} - {status}", widget.format)
	}
	if widget.horizAlign != "left" {
		t.Errorf("widget.horizAlign = %q, want left", widget.horizAlign)
	}
	if widget.vertAlign != "top" {
		t.Errorf("widget.vertAlign = %q, want top", widget.vertAlign)
	}
}

// TestNewWinampWidget_WithScrollConfig tests widget creation with scroll configuration
func TestNewWinampWidget_WithScrollConfig(t *testing.T) {
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

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	if !widget.scrollEnabled {
		t.Error("widget.scrollEnabled = false, want true")
	}
	if widget.scrollDirection != "right" {
		t.Errorf("widget.scrollDirection = %q, want right", widget.scrollDirection)
	}
	if widget.scrollSpeed != 50.0 {
		t.Errorf("widget.scrollSpeed = %f, want 50.0", widget.scrollSpeed)
	}
	if widget.scrollMode != "bounce" {
		t.Errorf("widget.scrollMode = %q, want bounce", widget.scrollMode)
	}
	if widget.scrollPauseMs != 2000 {
		t.Errorf("widget.scrollPauseMs = %d, want 2000", widget.scrollPauseMs)
	}
	if widget.scrollGap != 30 {
		t.Errorf("widget.scrollGap = %d, want 30", widget.scrollGap)
	}
}

// TestNewWinampWidget_WithAutoShowConfig tests widget creation with auto-show configuration
func TestNewWinampWidget_WithAutoShowConfig(t *testing.T) {
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

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	if widget.autoShowOnTrackChange {
		t.Error("widget.autoShowOnTrackChange = true, want false")
	}
	if !widget.autoShowOnPlay {
		t.Error("widget.autoShowOnPlay = false, want true")
	}
	if !widget.autoShowOnPause {
		t.Error("widget.autoShowOnPause = false, want true")
	}
	if !widget.autoShowOnStop {
		t.Error("widget.autoShowOnStop = false, want true")
	}
	if !widget.autoShowOnSeek {
		t.Error("widget.autoShowOnSeek = false, want true")
	}
}

// TestNewWinampWidget_WithPlaceholderConfig tests widget creation with placeholder configuration
func TestNewWinampWidget_WithPlaceholderConfig(t *testing.T) {
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

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	if widget.placeholderMode != "text" {
		t.Errorf("widget.placeholderMode = %q, want text", widget.placeholderMode)
	}
	if widget.placeholderText != "Custom Placeholder" {
		t.Errorf("widget.placeholderText = %q, want Custom Placeholder", widget.placeholderText)
	}
}

// TestWinampWidget_Defaults tests default values
func TestWinampWidget_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	// Check default values
	if widget.format != "{title}" {
		t.Errorf("default format = %q, want {title}", widget.format)
	}
	if widget.fontSize != 12 {
		t.Errorf("default fontSize = %d, want 12", widget.fontSize)
	}
	if widget.horizAlign != "center" {
		t.Errorf("default horizAlign = %q, want center", widget.horizAlign)
	}
	if widget.vertAlign != "center" {
		t.Errorf("default vertAlign = %q, want center", widget.vertAlign)
	}
	if widget.placeholderMode != "icon" {
		t.Errorf("default placeholderMode = %q, want icon", widget.placeholderMode)
	}
	if widget.placeholderText != "No Winamp" {
		t.Errorf("default placeholderText = %q, want No Winamp", widget.placeholderText)
	}
	if widget.scrollEnabled {
		t.Error("default scrollEnabled = true, want false")
	}
	if widget.scrollDirection != "left" {
		t.Errorf("default scrollDirection = %q, want left", widget.scrollDirection)
	}
	if widget.scrollSpeed != 30.0 {
		t.Errorf("default scrollSpeed = %f, want 30.0", widget.scrollSpeed)
	}
	if widget.scrollMode != "continuous" {
		t.Errorf("default scrollMode = %q, want continuous", widget.scrollMode)
	}
	if !widget.autoShowOnTrackChange {
		t.Error("default autoShowOnTrackChange = false, want true")
	}
	if widget.autoShowOnPlay {
		t.Error("default autoShowOnPlay = true, want false")
	}
}

// TestWinampWidget_Update tests the Update method
func TestWinampWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	// Update should not return error
	if err := widget.Update(); err != nil {
		t.Errorf("Update() error = %v, want nil", err)
	}
}

// TestWinampWidget_Render tests the Render method
func TestWinampWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "winamp",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
	}

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	// Render should return an image (placeholder when no Winamp)
	img, err := widget.Render()
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

// TestWinampWidget_Render_TextPlaceholder tests rendering with text placeholder
func TestWinampWidget_Render_TextPlaceholder(t *testing.T) {
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

	widget, err := NewWinampWidget(cfg)
	if err != nil {
		t.Fatalf("NewWinampWidget() error = %v", err)
	}

	img, err := widget.Render()
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
