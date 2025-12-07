package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewTelegramWidget(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.WidgetConfig
		wantErr bool
	}{
		{
			name: "missing auth config",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Auth:     nil,
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "testhash",
					PhoneNumber: "+1234567890",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with appearance settings",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "testhash",
					PhoneNumber: "+1234567890",
				},
				Appearance: &config.TelegramAppearanceConfig{
					Header: &config.TelegramElementConfig{
						Blink: true,
					},
					Separator: &config.SeparatorConfig{
						Color:     200,
						Thickness: 2,
					},
					Timeout: 10,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewTelegramWidget(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTelegramWidget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w == nil {
				t.Error("NewTelegramWidget() returned nil widget without error")
			}
		})
	}
}

func TestTelegramWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "telegram",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Auth: &config.TelegramAuthConfig{
			APIID:       12345,
			APIHash:     "testhash",
			PhoneNumber: "+1234567890",
		},
	}

	w, err := NewTelegramWidget(cfg)
	if err != nil {
		t.Fatalf("NewTelegramWidget() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestTelegramWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "telegram",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Auth: &config.TelegramAuthConfig{
			APIID:       12345,
			APIHash:     "testhash",
			PhoneNumber: "+1234567890",
		},
	}

	w, err := NewTelegramWidget(cfg)
	if err != nil {
		t.Fatalf("NewTelegramWidget() error = %v", err)
	}

	// Update should not error even when not connected
	if err := w.Update(); err != nil {
		t.Errorf("Update() error = %v", err)
	}
}

func TestTelegramWidget_AppearanceSettings(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name              string
		headerEnabled     *bool
		messageEnabled    *bool
		headerBlink       bool
		separatorColor    int
		timeout           int
		wantHeaderEnabled bool
		wantMsgEnabled    bool
		wantBlink         bool
		wantSepColor      int
		wantTimeout       int
	}{
		{
			name:              "default settings",
			headerEnabled:     nil,
			messageEnabled:    nil,
			headerBlink:       false,
			separatorColor:    0,
			timeout:           0,
			wantHeaderEnabled: true,  // default
			wantMsgEnabled:    true,  // default
			wantBlink:         false, // default
			wantSepColor:      128,   // default
			wantTimeout:       0,     // default
		},
		{
			name:              "header disabled",
			headerEnabled:     &falseVal,
			messageEnabled:    nil,
			headerBlink:       false,
			separatorColor:    0,
			timeout:           0,
			wantHeaderEnabled: false,
			wantMsgEnabled:    true,
			wantBlink:         false,
			wantSepColor:      128,
			wantTimeout:       0,
		},
		{
			name:              "blink enabled",
			headerEnabled:     &trueVal,
			messageEnabled:    nil,
			headerBlink:       true,
			separatorColor:    0,
			timeout:           0,
			wantHeaderEnabled: true,
			wantMsgEnabled:    true,
			wantBlink:         true,
			wantSepColor:      128,
			wantTimeout:       0,
		},
		{
			name:              "custom separator and timeout",
			headerEnabled:     nil,
			messageEnabled:    nil,
			headerBlink:       false,
			separatorColor:    200,
			timeout:           5,
			wantHeaderEnabled: true,
			wantMsgEnabled:    true,
			wantBlink:         false,
			wantSepColor:      200,
			wantTimeout:       5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appearance := &config.TelegramAppearanceConfig{
				Timeout: tt.timeout,
			}

			if tt.headerEnabled != nil || tt.headerBlink {
				appearance.Header = &config.TelegramElementConfig{
					Enabled: tt.headerEnabled,
					Blink:   tt.headerBlink,
				}
			}

			if tt.messageEnabled != nil {
				appearance.Message = &config.TelegramElementConfig{
					Enabled: tt.messageEnabled,
				}
			}

			if tt.separatorColor != 0 {
				appearance.Separator = &config.SeparatorConfig{
					Color:     tt.separatorColor,
					Thickness: 1,
				}
			}

			cfg := config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "testhash",
					PhoneNumber: "+1234567890",
				},
				Appearance: appearance,
			}

			w, err := NewTelegramWidget(cfg)
			if err != nil {
				t.Fatalf("NewTelegramWidget() error = %v", err)
			}

			if w.appearance.Header.Enabled != tt.wantHeaderEnabled {
				t.Errorf("Header.Enabled = %v, want %v", w.appearance.Header.Enabled, tt.wantHeaderEnabled)
			}
			if w.appearance.Message.Enabled != tt.wantMsgEnabled {
				t.Errorf("Message.Enabled = %v, want %v", w.appearance.Message.Enabled, tt.wantMsgEnabled)
			}
			if w.appearance.Header.Blink != tt.wantBlink {
				t.Errorf("Header.Blink = %v, want %v", w.appearance.Header.Blink, tt.wantBlink)
			}
			if w.appearance.Separator.Color != tt.wantSepColor {
				t.Errorf("Separator.Color = %v, want %v", w.appearance.Separator.Color, tt.wantSepColor)
			}
			if w.appearance.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", w.appearance.Timeout, tt.wantTimeout)
			}
		})
	}
}

func TestTelegramWidget_Stop(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "telegram",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Auth: &config.TelegramAuthConfig{
			APIID:       12345,
			APIHash:     "testhash",
			PhoneNumber: "+1234567890",
		},
	}

	w, err := NewTelegramWidget(cfg)
	if err != nil {
		t.Fatalf("NewTelegramWidget() error = %v", err)
	}

	// Stop should not panic
	w.Stop()
}

func TestTelegramWidget_GetAppearance(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "telegram",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Auth: &config.TelegramAuthConfig{
			APIID:       12345,
			APIHash:     "testhash",
			PhoneNumber: "+1234567890",
		},
		Appearance: &config.TelegramAppearanceConfig{
			Timeout: 10,
		},
	}

	w, err := NewTelegramWidget(cfg)
	if err != nil {
		t.Fatalf("NewTelegramWidget() error = %v", err)
	}

	// Test that appearance is set correctly
	if w.appearance.Timeout != 10 {
		t.Errorf("appearance.Timeout = %d, want 10", w.appearance.Timeout)
	}
}

func TestTelegramWidget_AutoHide(t *testing.T) {
	t.Run("widget hidden when auto_hide enabled and no message", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Type:     "telegram",
			Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			Auth: &config.TelegramAuthConfig{
				APIID:       12345,
				APIHash:     "testhash",
				PhoneNumber: "+1234567890",
			},
			AutoHide: &config.AutoHideConfig{
				Enabled: true,
				Timeout: 1.0,
			},
		}

		w, err := NewTelegramWidget(cfg)
		if err != nil {
			t.Fatalf("NewTelegramWidget() error = %v", err)
		}

		// Widget should be hidden initially (no message received)
		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img != nil {
			t.Error("Render() should return nil when auto_hide is enabled and no message received")
		}
	})

	t.Run("widget visible after TriggerAutoHide", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Type:     "telegram",
			Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			Auth: &config.TelegramAuthConfig{
				APIID:       12345,
				APIHash:     "testhash",
				PhoneNumber: "+1234567890",
			},
			AutoHide: &config.AutoHideConfig{
				Enabled: true,
				Timeout: 5.0, // Long timeout so we can check visibility
			},
		}

		w, err := NewTelegramWidget(cfg)
		if err != nil {
			t.Fatalf("NewTelegramWidget() error = %v", err)
		}

		// Trigger auto-hide (simulates message arrival)
		w.TriggerAutoHide()

		// Widget should be visible now
		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img == nil {
			t.Error("Render() should return image after TriggerAutoHide")
		}
	})

	t.Run("widget remains visible when auto_hide disabled", func(t *testing.T) {
		cfg := config.WidgetConfig{
			Type:     "telegram",
			Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
			Auth: &config.TelegramAuthConfig{
				APIID:       12345,
				APIHash:     "testhash",
				PhoneNumber: "+1234567890",
			},
			// No AutoHide config = disabled
		}

		w, err := NewTelegramWidget(cfg)
		if err != nil {
			t.Fatalf("NewTelegramWidget() error = %v", err)
		}

		// Widget should be visible without TriggerAutoHide
		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img == nil {
			t.Error("Render() should return image when auto_hide is disabled")
		}
	})
}
