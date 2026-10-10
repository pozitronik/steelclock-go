package app

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/backend"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// fakeDevice simulates a display device that can be unplugged and plugged back
// in. It backs the test backends registered below.
type fakeDevice struct {
	present atomic.Bool  // the device is connected (probe, create, reconnect)
	creates atomic.Int32 // backend factory calls
}

// fakeClient is a plain backend client (not Reconnectable).
type fakeClient struct {
	removed atomic.Bool
	closed  atomic.Bool
}

func (c *fakeClient) SendScreenData(string, []byte) error                    { return nil }
func (c *fakeClient) SendScreenDataMultiRes(string, map[string][]byte) error { return nil }
func (c *fakeClient) SendMultipleScreenData(string, [][]byte) error          { return nil }
func (c *fakeClient) SendHeartbeat() error                                   { return nil }
func (c *fakeClient) SupportsMultipleEvents() bool                           { return false }
func (c *fakeClient) RegisterGame(string, int) error                         { return nil }
func (c *fakeClient) BindScreenEvent(string, string) error                   { return nil }
func (c *fakeClient) RemoveGame() error                                      { c.removed.Store(true); return nil }
func (c *fakeClient) Close() error                                           { c.closed.Store(true); return nil }

// fakeReconnectableClient can lose its device and reconnect to it.
type fakeReconnectableClient struct {
	fakeClient
	dev        *fakeDevice
	connected  atomic.Bool
	reconnects atomic.Int32
}

func (c *fakeReconnectableClient) IsConnected() bool { return c.connected.Load() }

func (c *fakeReconnectableClient) Reconnect() error {
	if c.connected.Load() {
		return nil
	}
	c.reconnects.Add(1)
	if !c.dev.present.Load() {
		return errors.New("device not found")
	}
	c.connected.Store(true)
	return nil
}

var _ display.Reconnectable = (*fakeReconnectableClient)(nil)

var (
	plainDevice       = &fakeDevice{}
	reconnectDevice   = &fakeDevice{}
	lastReconnectable atomic.Pointer[fakeReconnectableClient]
	registerFakesOnce sync.Once
)

// registerFakeBackends registers explicit-only test backends, so auto-selection
// in other tests never picks them.
func registerFakeBackends() {
	registerFakesOnce.Do(func() {
		backend.RegisterExplicit("test_recovery_plain", func(*config.Config) (display.Backend, error) {
			plainDevice.creates.Add(1)
			if !plainDevice.present.Load() {
				return nil, errors.New("device not found")
			}
			return &fakeClient{}, nil
		})
		backend.RegisterProber("test_recovery_plain", func(*config.Config) bool {
			return plainDevice.present.Load()
		})

		backend.RegisterExplicit("test_recovery_reconnectable", func(*config.Config) (display.Backend, error) {
			reconnectDevice.creates.Add(1)
			if !reconnectDevice.present.Load() {
				return nil, errors.New("device not found")
			}
			c := &fakeReconnectableClient{dev: reconnectDevice}
			c.connected.Store(true)
			lastReconnectable.Store(c)
			return c, nil
		})
	})
}

// newRecoveryTest returns a fresh device instance, a config for the given test
// backend, and the device it drives (reset to present). Supervisor checks run
// only on Wake unless intervalMs is small.
func newRecoveryTest(t *testing.T, backendName string, intervalMs int) (*DeviceInstance, *config.Config, *fakeDevice) {
	t.Helper()
	registerFakeBackends()

	dev := plainDevice
	if backendName == "test_recovery_reconnectable" {
		dev = reconnectDevice
	}
	dev.present.Store(true)
	dev.creates.Store(0)

	cfg := &config.Config{
		GameName:            "test",
		GameDisplayName:     "Test",
		RefreshRateMs:       100,
		ReconnectIntervalMs: intervalMs,
		Backend:             backendName,
		Display:             config.DisplayConfig{Width: 128, Height: 40},
		Widgets: []config.WidgetConfig{{
			ID:       "clock1",
			Type:     "clock",
			Position: config.PositionConfig{W: 128, H: 40},
		}},
	}

	d := NewDeviceInstance("test", make(chan struct{}))
	t.Cleanup(d.Stop)
	return d, cfg, dev
}

// state reads the instance state under its lock.
func state(d *DeviceInstance) (running, waiting bool, client display.Backend) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.comp != nil, d.waiting, d.client
}

// waitForRunning waits until the instance renders again.
func waitForRunning(t *testing.T, d *DeviceInstance) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if running, waiting, _ := state(d); running && !waiting {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("device did not resume")
}

// settle gives the supervisor time to act on a Wake.
func settle() { time.Sleep(100 * time.Millisecond) }

const wakeOnly = 3_600_000 // reconnect interval long enough that only Wake triggers checks

func TestDeviceInstance_StartWithoutDevice_WaitsQuietlyThenResumes(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_plain", wakeOnly)
	dev.present.Store(false)

	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v, want nil while waiting for the device", err)
	}
	if running, waiting, _ := state(d); running || !waiting {
		t.Fatalf("running=%v waiting=%v, want waiting", running, waiting)
	}
	createsAtStart := dev.creates.Load()

	d.Wake()
	settle()
	if running, _, _ := state(d); running {
		t.Fatal("resumed while the device is still missing")
	}
	if got := dev.creates.Load(); got != createsAtStart {
		t.Errorf("backend created %d more time(s) while the probe reports it missing", got-createsAtStart)
	}

	dev.present.Store(true)
	d.Wake()
	waitForRunning(t, d)
	if _, _, client := state(d); client == nil {
		t.Error("client is nil after resuming")
	}
}

func TestDeviceInstance_DeviceLost_RecreatesSameBackend(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_plain", wakeOnly)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	_, _, before := state(d)
	oldClient := before.(*fakeClient)

	dev.present.Store(false)
	d.mu.Lock()
	comp := d.comp
	d.mu.Unlock()
	d.handleDeviceLost(comp)

	running, waiting, client := state(d)
	if running || !waiting || client != nil {
		t.Fatalf("running=%v waiting=%v client=%v, want waiting without a client", running, waiting, client)
	}
	if !oldClient.removed.Load() || !oldClient.closed.Load() {
		t.Error("the lost client was not unregistered and closed")
	}
	d.mu.Lock()
	got := d.wantBackend
	d.mu.Unlock()
	if got != "test_recovery_plain" {
		t.Errorf("wantBackend = %q, want the backend that was lost", got)
	}

	d.Wake()
	settle()
	if running, _, _ := state(d); running {
		t.Fatal("resumed while the device is still missing")
	}

	dev.present.Store(true)
	d.Wake()
	waitForRunning(t, d)
	if _, _, client := state(d); client == nil || client == before {
		t.Errorf("client = %v, want a newly created client", client)
	}
}

func TestDeviceInstance_DeviceLost_ReconnectsKeptClient(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_reconnectable", wakeOnly)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	client := lastReconnectable.Load()

	dev.present.Store(false)
	client.connected.Store(false)
	d.mu.Lock()
	comp := d.comp
	d.mu.Unlock()
	d.handleDeviceLost(comp)

	if _, waiting, kept := state(d); !waiting || kept != client {
		t.Fatalf("waiting=%v client kept=%v, want waiting with the same client", waiting, kept == client)
	}

	d.Wake()
	settle()
	if running, _, _ := state(d); running {
		t.Fatal("resumed while the device is still missing")
	}

	dev.present.Store(true)
	d.Wake()
	waitForRunning(t, d)
	if _, _, now := state(d); now != client {
		t.Error("a reconnectable client should be reconnected, not replaced")
	}
	if got := dev.creates.Load(); got != 1 {
		t.Errorf("backend created %d times, want 1 (only at Start)", got)
	}
}

func TestDeviceInstance_RunningDisconnectedClientIsReconnected(t *testing.T) {
	d, cfg, _ := newRecoveryTest(t, "test_recovery_reconnectable", wakeOnly)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	client := lastReconnectable.Load()
	d.mu.Lock()
	comp := d.comp
	d.mu.Unlock()

	client.connected.Store(false) // unplugged and plugged back in quickly
	d.Wake()
	settle()

	if !client.IsConnected() || client.reconnects.Load() == 0 {
		t.Error("supervisor did not reconnect the running client")
	}
	d.mu.Lock()
	sameComp := d.comp == comp
	d.mu.Unlock()
	if running, waiting, _ := state(d); !running || waiting || !sameComp {
		t.Error("a quick reconnect should keep the running compositor")
	}
}

func TestDeviceInstance_ReusedDisconnectedClientWaitsOnStart(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_reconnectable", wakeOnly)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	client := lastReconnectable.Load()
	d.Stop()

	dev.present.Store(false)
	client.connected.Store(false)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if running, waiting, _ := state(d); running || !waiting {
		t.Fatalf("running=%v waiting=%v, want waiting for the reused client's device", running, waiting)
	}

	dev.present.Store(true)
	d.Wake()
	waitForRunning(t, d)
}

func TestDeviceInstance_TimerDrivesRecovery(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_plain", 20)
	dev.present.Store(false)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	dev.present.Store(true) // no Wake: the interval timer must pick it up
	waitForRunning(t, d)
}

func TestDeviceInstance_StopHaltsRecovery(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_plain", 20)
	dev.present.Store(false)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	d.Stop()
	dev.present.Store(true)
	d.Wake()
	time.Sleep(200 * time.Millisecond) // ten timer intervals

	if running, waiting, _ := state(d); running || waiting {
		t.Errorf("running=%v waiting=%v after Stop, want neither", running, waiting)
	}
}

func TestDeviceInstance_StaleLostCallbackIsIgnored(t *testing.T) {
	d, cfg, _ := newRecoveryTest(t, "test_recovery_plain", wakeOnly)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	d.handleDeviceLost(nil) // not the current compositor

	if running, waiting, _ := state(d); !running || waiting {
		t.Error("a callback from a replaced compositor must not stop the current one")
	}
}

func TestDeviceInstance_ShutdownStopsSupervisor(t *testing.T) {
	d, cfg, dev := newRecoveryTest(t, "test_recovery_plain", 20)
	dev.present.Store(false)
	if err := d.Start(cfg, false); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	d.Shutdown(false)
	dev.present.Store(true)
	time.Sleep(200 * time.Millisecond)

	if running, _, client := state(d); running || client != nil {
		t.Error("device resumed after Shutdown")
	}
}

func TestDeviceInstance_Wake_NeverBlocks(t *testing.T) {
	d := NewDeviceInstance("test", make(chan struct{}))
	for i := 0; i < 10; i++ {
		d.Wake() // no supervisor is draining the channel
	}
}

// TestDeviceInstance_ShutdownReleasesClient checks that a device giving up its
// client (app exit, or the device removed from the config on reload) always
// releases the client's local resources, and unregisters only when asked.
func TestDeviceInstance_ShutdownReleasesClient(t *testing.T) {
	tests := []struct {
		name           string
		unregister     bool
		wantUnregister bool
	}{
		{"keep registration", false, false},
		{"unregister", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeviceInstance("test", make(chan struct{}))
			c := &fakeClient{}
			d.client = c

			d.Shutdown(tt.unregister)

			if !c.closed.Load() {
				t.Error("client was not closed")
			}
			if got := c.removed.Load(); got != tt.wantUnregister {
				t.Errorf("unregistered = %v, want %v", got, tt.wantUnregister)
			}
			if _, _, client := state(d); client != nil {
				t.Errorf("client = %v after shutdown, want nil", client)
			}
		})
	}
}

// TestLifecycleManager_RemovedDeviceReleasesClient checks that a device
// dropped from the config on reload releases its client.
func TestLifecycleManager_RemovedDeviceReleasesClient(t *testing.T) {
	m := NewLifecycleManager()
	removed := NewDeviceInstance("removed", m.retryCancel)
	c := &fakeClient{}
	removed.client = c
	m.devices = []*DeviceInstance{removed}

	m.shutdownOldDevices(nil)

	if !c.closed.Load() {
		t.Error("removed device's client was not closed")
	}
	if c.removed.Load() {
		t.Error("removed device was unregistered on reload")
	}
}
