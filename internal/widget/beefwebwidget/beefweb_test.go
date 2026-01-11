package beefwebwidget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/beefweb"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// mockClient implements beefweb.Client for testing
type mockClient struct {
	available bool
	state     *beefweb.PlayerState
	err       error
}

func (m *mockClient) IsAvailable() bool {
	return m.available
}

func (m *mockClient) GetState() (*beefweb.PlayerState, error) {
	return m.state, m.err
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.WidgetConfig
		wantErr bool
	}{
		{
			name: "default config",
			cfg: config.WidgetConfig{
				Type: "beefweb",
				Position: config.PositionConfig{
					W: 128,
					H: 40,
				},
			},
			wantErr: false,
		},
		{
			name: "with beefweb config",
			cfg: config.WidgetConfig{
				Type: "beefweb",
				Position: config.PositionConfig{
					W: 128,
					H: 40,
				},
				Beefweb: &config.BeefwebConfig{
					ServerURL: "http://192.168.1.100:8080",
					Placeholder: &config.BeefwebPlaceholderConfig{
						Mode: "hide",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with scroll config",
			cfg: config.WidgetConfig{
				Type: "beefweb",
				Position: config.PositionConfig{
					W: 128,
					H: 40,
				},
				Scroll: &config.ScrollConfig{
					Enabled: true,
					Speed:   50,
					PauseMs: 2000,
				},
			},
			wantErr: false,
		},
		{
			name: "with auto-show config",
			cfg: config.WidgetConfig{
				Type: "beefweb",
				Position: config.PositionConfig{
					W: 128,
					H: 40,
				},
				BeefwebAutoShow: &config.BeefwebAutoShowConfig{
					OnPlay:      true,
					OnPause:     true,
					DurationSec: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "with text format",
			cfg: config.WidgetConfig{
				Type: "beefweb",
				Position: config.PositionConfig{
					W: 128,
					H: 40,
				},
				Text: &config.TextConfig{
					Format: "{artist} - {title} ({duration})",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w == nil {
				t.Error("New() returned nil widget")
			}
		})
	}
}

func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Replace client with mock
	w.client = &mockClient{
		available: true,
		state: &beefweb.PlayerState{
			State: beefweb.StatePlaying,
			Track: &beefweb.TrackInfo{
				Artist:   "Test Artist",
				Title:    "Test Title",
				Album:    "Test Album",
				Duration: 3 * time.Minute,
				Position: 1 * time.Minute,
			},
		},
	}

	err = w.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
}

func TestWidget_Update_Unavailable(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Replace client with mock that is unavailable
	w.client = &mockClient{
		available: false,
	}

	err = w.Update()
	if err != nil {
		t.Errorf("Update() error = %v, want nil (graceful handling)", err)
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render should work even without update
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	// Check dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestWidget_Render_WithTrack(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Set mock client
	w.client = &mockClient{
		available: true,
		state: &beefweb.PlayerState{
			State: beefweb.StatePlaying,
			Track: &beefweb.TrackInfo{
				Artist:   "Test Artist",
				Title:    "Test Title",
				Album:    "Test Album",
				Duration: 3 * time.Minute,
				Position: 1 * time.Minute,
			},
		},
	}

	// Update to get track info
	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "00:00"},
		{"one minute", time.Minute, "01:00"},
		{"one minute thirty seconds", time.Minute + 30*time.Second, "01:30"},
		{"five minutes", 5 * time.Minute, "05:00"},
		{"ten minutes", 10 * time.Minute, "10:00"},
		{"one hour", time.Hour, "60:00"},
		{"negative", -time.Second, "--:--"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestWidget_FormatOutput(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Text: &config.TextConfig{
			Format: "{artist} - {title}",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	state := &beefweb.PlayerState{
		State: beefweb.StatePlaying,
		Track: &beefweb.TrackInfo{
			Artist:   "Pink Floyd",
			Title:    "Comfortably Numb",
			Album:    "The Wall",
			Duration: 6*time.Minute + 24*time.Second,
			Position: 2*time.Minute + 15*time.Second,
		},
	}

	got := w.formatOutput(state)
	want := "Pink Floyd - Comfortably Numb"
	if got != want {
		t.Errorf("formatOutput() = %v, want %v", got, want)
	}
}

func TestWidget_FormatOutput_AllTokens(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Text: &config.TextConfig{
			Format: "{artist}|{title}|{album}|{position}|{duration}|{state}",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	state := &beefweb.PlayerState{
		State: beefweb.StatePlaying,
		Track: &beefweb.TrackInfo{
			Artist:   "Artist",
			Title:    "Title",
			Album:    "Album",
			Duration: 3*time.Minute + 30*time.Second,
			Position: 1*time.Minute + 45*time.Second,
		},
	}

	got := w.formatOutput(state)
	want := "Artist|Title|Album|01:45|03:30|Playing"
	if got != want {
		t.Errorf("formatOutput() = %v, want %v", got, want)
	}
}

func TestWidget_FormatOutput_NilState(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := w.formatOutput(nil)
	if got != "" {
		t.Errorf("formatOutput(nil) = %v, want empty string", got)
	}
}

func TestWidget_FormatOutput_NilTrack(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "beefweb",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	state := &beefweb.PlayerState{
		State: beefweb.StateStopped,
		Track: nil,
	}

	got := w.formatOutput(state)
	if got != "" {
		t.Errorf("formatOutput(nil track) = %v, want empty string", got)
	}
}
