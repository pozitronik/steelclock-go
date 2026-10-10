package app

import (
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/webeditor"
)

// newOverrideTestApp returns an app whose devices run the given backends
// (fake backends from device_recovery_test.go), as if the config had started.
func newOverrideTestApp(t *testing.T, cfg *config.Config) *App {
	t.Helper()
	registerFakeBackends()
	plainDevice.present.Store(true)
	reconnectDevice.present.Store(true)

	a := NewApp("unused-config")
	a.lifecycle.isFirstStart = false
	if err := a.lifecycle.Start(cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(a.lifecycle.Stop)
	return a
}

func overrideTestWidgets() []config.WidgetConfig {
	cfg := config.CreateDefault()
	cfg.Widgets[0].Text.Font = "pixel5x7"
	return cfg.Widgets
}

// TestWebClientOverride_RestoresEachDeviceBackend checks that turning the
// preview override on and off brings every device back to its own backend.
func TestWebClientOverride_RestoresEachDeviceBackend(t *testing.T) {
	cfg := config.CreateDefault()
	widgets := overrideTestWidgets()
	cfg.Widgets = nil
	cfg.Devices = []config.DeviceConfig{
		{ID: "a", Backend: "test_recovery_plain", Display: cfg.Display, Widgets: widgets},
		{ID: "b", Backend: "test_recovery_reconnectable", Display: cfg.Display, Widgets: widgets},
	}
	a := newOverrideTestApp(t, cfg)

	if err := a.SetWebClientOverride(true); err != nil {
		t.Fatalf("enable override: %v", err)
	}
	if !a.lifecycle.AllDevicesUseBackend("webclient") {
		t.Fatal("override did not switch all devices to webclient")
	}

	if err := a.SetWebClientOverride(false); err != nil {
		t.Fatalf("disable override: %v", err)
	}
	for i, d := range a.lifecycle.devices {
		if got, want := d.GetCurrentBackend(), cfg.Devices[i].Backend; got != want {
			t.Errorf("device %s restored backend %q, want %q", d.id, got, want)
		}
	}
	if got := a.lifecycle.GetLastGoodConfig(); got != cfg {
		t.Error("last good config after the override is not the original config")
	}
	if a.webclientOverrideActive || a.webclientOverrideConfig != nil {
		t.Error("override state was not cleared")
	}
}

// TestWebClientOverride_RestoresSingleDeviceBackend covers the single-device
// config, where the backend is set at the top level.
func TestWebClientOverride_RestoresSingleDeviceBackend(t *testing.T) {
	cfg := config.CreateDefault()
	cfg.Backend = "test_recovery_plain"
	cfg.Widgets = overrideTestWidgets()
	a := newOverrideTestApp(t, cfg)

	if err := a.SetWebClientOverride(true); err != nil {
		t.Fatalf("enable override: %v", err)
	}
	if err := a.SetWebClientOverride(false); err != nil {
		t.Fatalf("disable override: %v", err)
	}
	if got := a.lifecycle.GetCurrentBackend(); got != "test_recovery_plain" {
		t.Errorf("restored backend %q, want test_recovery_plain", got)
	}
}

// TestWebClientOverride_KeepsPreviewOfWebClientDevice covers a config where
// one device itself uses the webclient backend: after the override ends, the
// editor must still preview that device.
func TestWebClientOverride_KeepsPreviewOfWebClientDevice(t *testing.T) {
	cfg := config.CreateDefault()
	widgets := overrideTestWidgets()
	cfg.Widgets = nil
	cfg.Devices = []config.DeviceConfig{
		{ID: "a", Backend: "test_recovery_plain", Display: cfg.Display, Widgets: widgets},
		{ID: "b", Backend: "webclient", Display: cfg.Display, Widgets: widgets},
	}
	a := newOverrideTestApp(t, cfg)
	a.webEditor = webeditor.NewServer(nil, nil, "", nil, nil)
	a.updateWebClientProviderUnlocked()

	if err := a.SetWebClientOverride(true); err != nil {
		t.Fatalf("enable override: %v", err)
	}
	if ids := strings.Join(a.webEditor.PreviewDeviceIDs(), ","); ids != "a,b" {
		t.Errorf("preview devices during override = %q, want a,b", ids)
	}

	if err := a.SetWebClientOverride(false); err != nil {
		t.Fatalf("disable override: %v", err)
	}
	if ids := strings.Join(a.webEditor.PreviewDeviceIDs(), ","); ids != "b" {
		t.Errorf("preview devices after override = %q, want b", ids)
	}
	if got := a.lifecycle.devices[0].GetCurrentBackend(); got != "test_recovery_plain" {
		t.Errorf("device a restored backend %q, want test_recovery_plain", got)
	}
}

// TestWebClientOverride_DisableWhenInactive is a no-op.
func TestWebClientOverride_DisableWhenInactive(t *testing.T) {
	a := NewApp("unused-config")
	if err := a.SetWebClientOverride(false); err != nil {
		t.Errorf("disable without override: %v", err)
	}
}
