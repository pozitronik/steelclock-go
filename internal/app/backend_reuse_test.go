package app

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/backend/webclient"
	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestDeviceInstance_ReloadReusesOrRecreatesClient checks that a restart with
// a new config keeps the backend client only when the settings it was created
// from are unchanged.
func TestDeviceInstance_ReloadReusesOrRecreatesClient(t *testing.T) {
	tests := []struct {
		name       string
		change     func(*config.Config)
		wantReused bool
	}{
		{"unchanged", func(*config.Config) {}, true},
		{"widgets changed", func(c *config.Config) { c.Widgets[0].Position.W = 64 }, true},
		{"refresh rate changed", func(c *config.Config) { c.RefreshRateMs = 50 }, true},
		{"brightness changed", func(c *config.Config) {
			level := 5
			c.DirectDriver = &config.DirectDriverConfig{Brightness: &level}
		}, true},
		{"display height changed", func(c *config.Config) { c.Display.Height = 64 }, false},
		{"direct PID changed", func(c *config.Config) { c.DirectDriver = &config.DirectDriverConfig{PID: "1628"} }, false},
		{"direct interface changed", func(c *config.Config) { c.DirectDriver = &config.DirectDriverConfig{Interface: "mi_03"} }, false},
		{"game name changed", func(c *config.Config) { c.GameName = "OTHER" }, false},
		{"deinitialize timer changed", func(c *config.Config) { c.DeinitializeTimerMs = 5000 }, false},
		{"webclient FPS changed", func(c *config.Config) { c.WebClient = &config.WebClientConfig{TargetFPS: 5} }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, cfg, dev := newRecoveryTest(t, "test_recovery_plain", wakeOnly)
			if err := d.Start(cfg, false); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			_, _, before := state(d)

			next := *cfg
			next.Widgets = append([]config.WidgetConfig(nil), cfg.Widgets...)
			tt.change(&next)
			d.Stop()
			if err := d.Start(&next, false); err != nil {
				t.Fatalf("restart error = %v", err)
			}
			_, _, after := state(d)

			if reused := after == before; reused != tt.wantReused {
				t.Errorf("client reused = %v, want %v", reused, tt.wantReused)
			}
			wantCreates := int32(2)
			if tt.wantReused {
				wantCreates = 1
			}
			if got := dev.creates.Load(); got != wantCreates {
				t.Errorf("backend created %d times, want %d", got, wantCreates)
			}
			if !tt.wantReused && !before.(*fakeClient).removed.Load() {
				t.Error("replaced client was not released")
			}
		})
	}
}

// TestDeviceInstance_ReloadAppliesWebClientSettings reloads a webclient device
// with a new size and frame rate; the browser must get the new settings.
func TestDeviceInstance_ReloadAppliesWebClientSettings(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))
	cfg := &config.Config{
		Backend:   "webclient",
		Display:   config.DisplayConfig{Width: 128, Height: 40},
		WebClient: &config.WebClientConfig{TargetFPS: 30},
	}
	if err := d.ensureClient(cfg); err != nil {
		t.Fatalf("ensureClient() error = %v", err)
	}

	next := *cfg
	next.Display.Height = 64
	next.WebClient = &config.WebClientConfig{TargetFPS: 5}
	if err := d.ensureClient(&next); err != nil {
		t.Fatalf("ensureClient() reload error = %v", err)
	}

	got := d.client.(*webclient.Client).GetConfig()
	if got.Width != 128 || got.Height != 64 || got.TargetFPS != 5 {
		t.Errorf("webclient config = %+v, want 128x64 at 5 FPS", got)
	}
}
