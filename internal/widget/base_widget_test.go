package widget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewBaseWidget(t *testing.T) {
	tests := []struct {
		name           string
		cfg            config.WidgetConfig
		wantID         string
		wantInterval   time.Duration
		wantPadding    int
		wantAutoHide   bool
		wantBackground uint8
	}{
		{
			name: "basic config",
			cfg: config.WidgetConfig{
				ID: "test-widget",
				Position: config.PositionConfig{
					X: 10, Y: 20, W: 100, H: 50,
				},
				UpdateInterval: 2.0,
			},
			wantID:         "test-widget",
			wantInterval:   2 * time.Second,
			wantPadding:    0,
			wantAutoHide:   false,
			wantBackground: 0,
		},
		{
			name: "default interval",
			cfg: config.WidgetConfig{
				ID:             "test-widget",
				UpdateInterval: 0, // Should default to 1 second
			},
			wantID:       "test-widget",
			wantInterval: 1 * time.Second,
		},
		{
			name: "with style and padding",
			cfg: config.WidgetConfig{
				ID: "styled-widget",
				Style: &config.StyleConfig{
					Background: 128,
					Border:     255,
					Padding:    5,
				},
			},
			wantID:         "styled-widget",
			wantPadding:    5,
			wantBackground: 128,
		},
		{
			name: "with auto-hide",
			cfg: config.WidgetConfig{
				ID: "auto-hide-widget",
				AutoHide: &config.AutoHideConfig{
					Enabled: true,
					Timeout: 3.0,
				},
			},
			wantID:       "auto-hide-widget",
			wantAutoHide: true,
		},
		{
			name: "transparent background",
			cfg: config.WidgetConfig{
				ID: "transparent-widget",
				Style: &config.StyleConfig{
					Background: -1, // Transparent
				},
			},
			wantID:         "transparent-widget",
			wantBackground: 0, // Should render as black
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseWidget(tt.cfg)

			if base.Name() != tt.wantID {
				t.Errorf("Name() = %q, want %q", base.Name(), tt.wantID)
			}

			if tt.wantInterval > 0 && base.GetUpdateInterval() != tt.wantInterval {
				t.Errorf("GetUpdateInterval() = %v, want %v", base.GetUpdateInterval(), tt.wantInterval)
			}

			if base.GetPadding() != tt.wantPadding {
				t.Errorf("GetPadding() = %d, want %d", base.GetPadding(), tt.wantPadding)
			}

			if base.IsAutoHideEnabled() != tt.wantAutoHide {
				t.Errorf("IsAutoHideEnabled() = %v, want %v", base.IsAutoHideEnabled(), tt.wantAutoHide)
			}

			if tt.wantBackground > 0 || tt.cfg.Style != nil {
				if base.GetRenderBackgroundColor() != tt.wantBackground {
					t.Errorf("GetRenderBackgroundColor() = %d, want %d", base.GetRenderBackgroundColor(), tt.wantBackground)
				}
			}
		})
	}
}

func TestBaseWidget_CreateCanvas(t *testing.T) {
	cfg := config.WidgetConfig{
		ID: "test",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Style: &config.StyleConfig{
			Background: 50,
		},
	}

	base := NewBaseWidget(cfg)
	img := base.CreateCanvas()

	if img == nil {
		t.Fatal("CreateCanvas returned nil")
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Canvas dimensions = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}

	// Check background color
	pixel := img.GrayAt(64, 20).Y
	if pixel != 50 {
		t.Errorf("Background color = %d, want 50", pixel)
	}
}

func TestBaseWidget_GetContentArea(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		padding    int
		wantX      int
		wantY      int
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "no padding",
			width:      128,
			height:     40,
			padding:    0,
			wantX:      0,
			wantY:      0,
			wantWidth:  128,
			wantHeight: 40,
		},
		{
			name:       "with padding",
			width:      100,
			height:     50,
			padding:    5,
			wantX:      5,
			wantY:      5,
			wantWidth:  90,
			wantHeight: 40,
		},
		{
			name:       "large padding",
			width:      100,
			height:     100,
			padding:    20,
			wantX:      20,
			wantY:      20,
			wantWidth:  60,
			wantHeight: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				ID: "test",
				Position: config.PositionConfig{
					W: tt.width, H: tt.height,
				},
				Style: &config.StyleConfig{
					Padding: tt.padding,
				},
			}

			base := NewBaseWidget(cfg)
			area := base.GetContentArea()

			if area.X != tt.wantX {
				t.Errorf("ContentArea.X = %d, want %d", area.X, tt.wantX)
			}
			if area.Y != tt.wantY {
				t.Errorf("ContentArea.Y = %d, want %d", area.Y, tt.wantY)
			}
			if area.Width != tt.wantWidth {
				t.Errorf("ContentArea.Width = %d, want %d", area.Width, tt.wantWidth)
			}
			if area.Height != tt.wantHeight {
				t.Errorf("ContentArea.Height = %d, want %d", area.Height, tt.wantHeight)
			}
		})
	}
}

func TestBaseWidget_Dimensions(t *testing.T) {
	cfg := config.WidgetConfig{
		ID: "test",
		Position: config.PositionConfig{
			X: 10, Y: 20, W: 128, H: 40,
		},
	}

	base := NewBaseWidget(cfg)

	if base.Width() != 128 {
		t.Errorf("Width() = %d, want 128", base.Width())
	}

	if base.Height() != 40 {
		t.Errorf("Height() = %d, want 40", base.Height())
	}

	w, h := base.Dimensions()
	if w != 128 || h != 40 {
		t.Errorf("Dimensions() = %d, %d, want 128, 40", w, h)
	}
}

func TestBaseWidget_AutoHide(t *testing.T) {
	cfg := config.WidgetConfig{
		ID: "auto-hide-test",
		AutoHide: &config.AutoHideConfig{
			Enabled: true,
			Timeout: 0.1, // 100ms for fast testing
		},
	}

	base := NewBaseWidget(cfg)

	// Initially should be hidden (never triggered)
	if !base.ShouldHide() {
		t.Error("ShouldHide() should be true initially")
	}

	// Trigger visibility
	base.TriggerAutoHide()

	// Should now be visible
	if base.ShouldHide() {
		t.Error("ShouldHide() should be false after trigger")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should be hidden again
	if !base.ShouldHide() {
		t.Error("ShouldHide() should be true after timeout")
	}
}

func TestBaseWidget_AutoHide_Disabled(t *testing.T) {
	cfg := config.WidgetConfig{
		ID: "no-auto-hide",
		// AutoHide not set = disabled
	}

	base := NewBaseWidget(cfg)

	// Should never hide when auto-hide is disabled
	if base.ShouldHide() {
		t.Error("ShouldHide() should be false when auto-hide is disabled")
	}

	if base.IsAutoHideEnabled() {
		t.Error("IsAutoHideEnabled() should be false")
	}

	// TriggerAutoHide should be a no-op
	base.TriggerAutoHide()
	if base.ShouldHide() {
		t.Error("ShouldHide() should still be false")
	}
}

func TestBaseWidget_ApplyBorder(t *testing.T) {
	tests := []struct {
		name         string
		border       int
		expectBorder bool
	}{
		{
			name:         "border enabled",
			border:       255,
			expectBorder: true,
		},
		{
			name:         "border disabled",
			border:       -1,
			expectBorder: false,
		},
		{
			name:         "border with color 0",
			border:       0,
			expectBorder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				ID: "test",
				Position: config.PositionConfig{
					W: 20, H: 20,
				},
				Style: &config.StyleConfig{
					Background: 128,
					Border:     tt.border,
				},
			}

			base := NewBaseWidget(cfg)
			img := base.CreateCanvas()
			base.ApplyBorder(img)

			// Check top-left corner pixel
			pixel := img.GrayAt(0, 0).Y

			if tt.expectBorder {
				// Border should be drawn (non-background color)
				if pixel == 128 {
					t.Error("Border should have been drawn")
				}
			} else {
				// No border, should be background color
				if pixel != 128 {
					t.Errorf("Border should not have been drawn, pixel = %d", pixel)
				}
			}
		})
	}
}

func TestBaseWidget_GetPosition(t *testing.T) {
	cfg := config.WidgetConfig{
		ID: "test",
		Position: config.PositionConfig{
			X: 10, Y: 20, W: 100, H: 50,
		},
	}

	base := NewBaseWidget(cfg)
	pos := base.GetPosition()

	if pos.X != 10 {
		t.Errorf("Position.X = %d, want 10", pos.X)
	}
	if pos.Y != 20 {
		t.Errorf("Position.Y = %d, want 20", pos.Y)
	}
	if pos.W != 100 {
		t.Errorf("Position.W = %d, want 100", pos.W)
	}
	if pos.H != 50 {
		t.Errorf("Position.H = %d, want 50", pos.H)
	}
}

func TestBaseWidget_GetStyle(t *testing.T) {
	cfg := config.WidgetConfig{
		ID: "test",
		Style: &config.StyleConfig{
			Background: 100,
			Border:     200,
			Padding:    10,
		},
	}

	base := NewBaseWidget(cfg)
	style := base.GetStyle()

	if style.Background != 100 {
		t.Errorf("Style.Background = %d, want 100", style.Background)
	}
	if style.Border != 200 {
		t.Errorf("Style.Border = %d, want 200", style.Border)
	}
	if style.Padding != 10 {
		t.Errorf("Style.Padding = %d, want 10", style.Padding)
	}
}
