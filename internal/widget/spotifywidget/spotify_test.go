package spotifywidget

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/spotify"
)

func TestNew_MissingAuth(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("New() without spotify_auth should return error")
	}
}

func TestNew_MissingClientID(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			ClientID: "",
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("New() without client_id should return error")
	}
}

func TestNew_WithValidConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Errorf("New() error = %v", err)
	}
	if w == nil {
		t.Error("New() returned nil widget")
	}
}

func TestNew_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Check defaults
	if w.format != "{artist} - {title}" {
		t.Errorf("format = %v, want {artist} - {title}", w.format)
	}
	if w.placeholderMode != placeholderModeText {
		t.Errorf("placeholderMode = %v, want %v", w.placeholderMode, placeholderModeText)
	}
	if w.placeholderText != "[Not playing]" {
		t.Errorf("placeholderText = %v, want [Not playing]", w.placeholderText)
	}
	if !w.autoShowOnTrackChange {
		t.Error("autoShowOnTrackChange should be true by default")
	}
	if w.autoShowOnPlay {
		t.Error("autoShowOnPlay should be false by default")
	}
	if w.autoShowOnPause {
		t.Error("autoShowOnPause should be false by default")
	}
	if w.autoShowOnStop {
		t.Error("autoShowOnStop should be false by default")
	}
}

func TestNew_CustomConfig(t *testing.T) {
	falseVal := false

	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
		Text: &config.TextConfig{
			Format: "{title} by {artist}",
		},
		Spotify: &config.SpotifyConfig{
			Placeholder: &config.SpotifyPlaceholderConfig{
				Mode: "icon",
				Text: "Custom placeholder",
			},
		},
		SpotifyAutoShow: &config.SpotifyAutoShowConfig{
			OnTrackChange: &falseVal,
			OnPlay:        true,
			OnPause:       true,
			OnStop:        true,
			DurationSec:   10,
		},
		Scroll: &config.ScrollConfig{
			Enabled: true,
			Speed:   50,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.format != "{title} by {artist}" {
		t.Errorf("format = %v, want {title} by {artist}", w.format)
	}
	if w.placeholderMode != placeholderModeIcon {
		t.Errorf("placeholderMode = %v, want %v", w.placeholderMode, placeholderModeIcon)
	}
	if w.autoShowOnTrackChange != falseVal {
		t.Error("autoShowOnTrackChange should be false")
	}
	if !w.autoShowOnPlay {
		t.Error("autoShowOnPlay should be true")
	}
	if !w.autoShowOnPause {
		t.Error("autoShowOnPause should be true")
	}
	if !w.autoShowOnStop {
		t.Error("autoShowOnStop should be true")
	}
	if !w.scrollEnabled {
		t.Error("scrollEnabled should be true")
	}

}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "00:00"},
		{"one second", time.Second, "00:01"},
		{"one minute", time.Minute, "01:00"},
		{"one hour", time.Hour, "60:00"},
		{"mixed", 3*time.Minute + 45*time.Second, "03:45"},
		{"negative", -time.Second, "--:--"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}

	// Check image dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 128 {
		t.Errorf("Image width = %v, want 128", bounds.Dx())
	}
	if bounds.Dy() != 40 {
		t.Errorf("Image height = %v, want 40", bounds.Dy())
	}
}

func TestWidget_FormatOutput(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
		Text: &config.TextConfig{
			Format: "{artist} - {title}",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	state := &spotify.PlayerState{
		State: spotify.StatePlaying,
		Track: &spotify.TrackInfo{
			ID:       "track123",
			Name:     "Test Song",
			Artists:  []string{"Test Artist"},
			Album:    "Test Album",
			Duration: 3 * time.Minute,
			Position: 1*time.Minute + 30*time.Second,
		},
		DeviceName: "Test Device",
		Volume:     75,
	}

	output := w.formatOutput(state)
	expected := "Test Artist - Test Song"
	if output != expected {
		t.Errorf("formatOutput() = %v, want %v", output, expected)
	}
}

func TestWidget_FormatOutput_MultipleArtists(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
		Text: &config.TextConfig{
			Format: "{artists} - {title}",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	state := &spotify.PlayerState{
		State: spotify.StatePlaying,
		Track: &spotify.TrackInfo{
			ID:       "track123",
			Name:     "Collaboration",
			Artists:  []string{"Artist A", "Artist B", "Artist C"},
			Album:    "Test Album",
			Duration: 3 * time.Minute,
			Position: 0,
		},
	}

	output := w.formatOutput(state)
	expected := "Artist A, Artist B, Artist C - Collaboration"
	if output != expected {
		t.Errorf("formatOutput() = %v, want %v", output, expected)
	}
}

func TestWidget_FormatOutput_NilState(t *testing.T) {
	cfg := config.WidgetConfig{
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test with nil state
	output := w.formatOutput(nil)
	if output != "" {
		t.Errorf("formatOutput(nil) = %v, want empty string", output)
	}

	// Test with nil track
	output = w.formatOutput(&spotify.PlayerState{
		State: spotify.StatePlaying,
		Track: nil,
	})
	if output != "" {
		t.Errorf("formatOutput(nil track) = %v, want empty string", output)
	}
}

func TestWidget_Name(t *testing.T) {
	cfg := config.WidgetConfig{
		ID:   "test_spotify",
		Type: "spotify",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		SpotifyAuth: &config.SpotifyAuthConfig{
			Mode:         "manual",
			ClientID:     "test_client_id",
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.Name() != "test_spotify" {
		t.Errorf("Name() = %v, want test_spotify", w.Name())
	}
}
