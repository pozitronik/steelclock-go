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
			name: "missing telegram config",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Telegram: nil,
			},
			wantErr: true,
		},
		{
			name: "missing auth config",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Telegram: &config.TelegramConfig{
					Auth: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Telegram: &config.TelegramConfig{
					Auth: &config.TelegramAuthConfig{
						APIID:       12345,
						APIHash:     "testhash",
						PhoneNumber: "+1234567890",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with display settings",
			cfg: config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Telegram: &config.TelegramConfig{
					Auth: &config.TelegramAuthConfig{
						APIID:       12345,
						APIHash:     "testhash",
						PhoneNumber: "+1234567890",
					},
					Display: &config.TelegramDisplayConfig{
						Mode:           "ticker",
						MaxMessages:    10,
						TruncateLength: 100,
						ScrollSpeed:    2.0,
					},
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
		Telegram: &config.TelegramConfig{
			Auth: &config.TelegramAuthConfig{
				APIID:       12345,
				APIHash:     "testhash",
				PhoneNumber: "+1234567890",
			},
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
		Telegram: &config.TelegramConfig{
			Auth: &config.TelegramAuthConfig{
				APIID:       12345,
				APIHash:     "testhash",
				PhoneNumber: "+1234567890",
			},
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

func TestTelegramWidget_DisplayModes(t *testing.T) {
	modes := []string{"last_message", "unread_count", "ticker", "notification"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Telegram: &config.TelegramConfig{
					Auth: &config.TelegramAuthConfig{
						APIID:       12345,
						APIHash:     "testhash",
						PhoneNumber: "+1234567890",
					},
					Display: &config.TelegramDisplayConfig{
						Mode: mode,
					},
				},
			}

			w, err := NewTelegramWidget(cfg)
			if err != nil {
				t.Fatalf("NewTelegramWidget() error = %v", err)
			}

			if w.displayMode != mode {
				t.Errorf("displayMode = %s, want %s", w.displayMode, mode)
			}
		})
	}
}

func TestTelegramWidget_Stop(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "telegram",
		Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
		Telegram: &config.TelegramConfig{
			Auth: &config.TelegramAuthConfig{
				APIID:       12345,
				APIHash:     "testhash",
				PhoneNumber: "+1234567890",
			},
		},
	}

	w, err := NewTelegramWidget(cfg)
	if err != nil {
		t.Fatalf("NewTelegramWidget() error = %v", err)
	}

	// Stop should not panic
	w.Stop()
}

func TestTelegramWidget_FormatHeader(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name       string
		showSender bool
		showChat   bool
		showTime   bool
		wantEmpty  bool
	}{
		{
			name:       "all enabled",
			showSender: true,
			showChat:   true,
			showTime:   true,
			wantEmpty:  false,
		},
		{
			name:       "all disabled",
			showSender: false,
			showChat:   false,
			showTime:   false,
			wantEmpty:  true,
		},
		{
			name:       "only sender",
			showSender: true,
			showChat:   false,
			showTime:   false,
			wantEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:     "telegram",
				Position: config.PositionConfig{X: 0, Y: 0, W: 128, H: 40},
				Telegram: &config.TelegramConfig{
					Auth: &config.TelegramAuthConfig{
						APIID:       12345,
						APIHash:     "testhash",
						PhoneNumber: "+1234567890",
					},
					Display: &config.TelegramDisplayConfig{
						ShowSender: &trueVal,
						ShowChat:   &trueVal,
						ShowTime:   &trueVal,
					},
				},
			}

			// Override display settings
			if !tt.showSender {
				cfg.Telegram.Display.ShowSender = &falseVal
			}
			if !tt.showChat {
				cfg.Telegram.Display.ShowChat = &falseVal
			}
			if !tt.showTime {
				cfg.Telegram.Display.ShowTime = &falseVal
			}

			w, err := NewTelegramWidget(cfg)
			if err != nil {
				t.Fatalf("NewTelegramWidget() error = %v", err)
			}

			if w.showSender != tt.showSender {
				t.Errorf("showSender = %v, want %v", w.showSender, tt.showSender)
			}
			if w.showChat != tt.showChat {
				t.Errorf("showChat = %v, want %v", w.showChat, tt.showChat)
			}
			if w.showTime != tt.showTime {
				t.Errorf("showTime = %v, want %v", w.showTime, tt.showTime)
			}
		})
	}
}
