package display_test

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/display"
	"github.com/pozitronik/steelclock-go/internal/driver"
	"github.com/pozitronik/steelclock-go/internal/gamesense"
	"github.com/pozitronik/steelclock-go/internal/testutil"
)

// TestInterfaces_Compile verifies that all expected types implement the interfaces.
// This is a compile-time check - if it compiles, the test passes.
func TestInterfaces_Compile(t *testing.T) {
	// This test is primarily a compile-time check.
	// The var _ assignments below will fail to compile if types don't implement interfaces.

	t.Run("gamesense.Client implements Backend", func(t *testing.T) {
		var _ display.Backend = (*gamesense.Client)(nil)
	})

	t.Run("driver.Client implements Backend", func(t *testing.T) {
		var _ display.Backend = (*driver.Client)(nil)
	})

	t.Run("testutil.TestClient implements Backend", func(t *testing.T) {
		var _ display.Backend = (*testutil.TestClient)(nil)
	})
}

// TestBackendInterface_FrameSender verifies FrameSender interface composition
func TestBackendInterface_FrameSender(t *testing.T) {
	var backend display.Backend = testutil.NewTestClient()

	// Verify FrameSender methods exist and can be called
	if err := backend.SendScreenData("event", make([]byte, 640)); err != nil {
		t.Errorf("SendScreenData failed: %v", err)
	}

	resData := map[string][]byte{"image-data-128x40": make([]byte, 640)}
	if err := backend.SendScreenDataMultiRes("event", resData); err != nil {
		t.Errorf("SendScreenDataMultiRes failed: %v", err)
	}

	frames := [][]byte{make([]byte, 640)}
	if err := backend.SendMultipleScreenData("event", frames); err != nil {
		t.Errorf("SendMultipleScreenData failed: %v", err)
	}
}

// TestBackendInterface_HeartbeatSender verifies HeartbeatSender interface
func TestBackendInterface_HeartbeatSender(t *testing.T) {
	var backend display.Backend = testutil.NewTestClient()

	if err := backend.SendHeartbeat(); err != nil {
		t.Errorf("SendHeartbeat failed: %v", err)
	}
}

// TestBackendInterface_BatchCapability verifies BatchCapability interface
func TestBackendInterface_BatchCapability(t *testing.T) {
	// Without multiple events support
	var backend1 display.Backend = testutil.NewTestClient()
	if backend1.SupportsMultipleEvents() {
		t.Error("Default TestClient should not support multiple events")
	}

	// With multiple events support
	var backend2 display.Backend = testutil.NewTestClient(testutil.WithMultipleEventsSupport(true))
	if !backend2.SupportsMultipleEvents() {
		t.Error("TestClient with WithMultipleEventsSupport(true) should support multiple events")
	}
}

// TestBackendInterface_GameRegistrar verifies GameRegistrar interface
func TestBackendInterface_GameRegistrar(t *testing.T) {
	var backend display.Backend = testutil.NewTestClient()

	if err := backend.RegisterGame("Developer", 15000); err != nil {
		t.Errorf("RegisterGame failed: %v", err)
	}

	if err := backend.BindScreenEvent("event", "screened-128x40"); err != nil {
		t.Errorf("BindScreenEvent failed: %v", err)
	}

	if err := backend.RemoveGame(); err != nil {
		t.Errorf("RemoveGame failed: %v", err)
	}
}

// TestClientInterface_Composition verifies Client interface is subset of Backend
func TestClientInterface_Composition(t *testing.T) {
	var backend display.Backend = testutil.NewTestClient()

	// Client is composed of FrameSender + HeartbeatSender + BatchCapability
	// Backend should be usable as Client
	var client display.Client = backend

	// Verify Client methods
	if err := client.SendScreenData("event", make([]byte, 640)); err != nil {
		t.Errorf("SendScreenData via Client failed: %v", err)
	}

	if err := client.SendHeartbeat(); err != nil {
		t.Errorf("SendHeartbeat via Client failed: %v", err)
	}

	_ = client.SupportsMultipleEvents()
}

// TestFrameSenderInterface verifies FrameSender can be used independently
func TestFrameSenderInterface(t *testing.T) {
	var sender display.FrameSender = testutil.NewTestClient()

	if err := sender.SendScreenData("event", make([]byte, 640)); err != nil {
		t.Errorf("SendScreenData failed: %v", err)
	}
}

// TestHeartbeatSenderInterface verifies HeartbeatSender can be used independently
func TestHeartbeatSenderInterface(t *testing.T) {
	var sender display.HeartbeatSender = testutil.NewTestClient()

	if err := sender.SendHeartbeat(); err != nil {
		t.Errorf("SendHeartbeat failed: %v", err)
	}
}

// TestBatchCapabilityInterface verifies BatchCapability can be used independently
func TestBatchCapabilityInterface(t *testing.T) {
	var capability display.BatchCapability = testutil.NewTestClient()
	_ = capability.SupportsMultipleEvents()
}

// TestGameRegistrarInterface verifies GameRegistrar can be used independently
func TestGameRegistrarInterface(t *testing.T) {
	var registrar display.GameRegistrar = testutil.NewTestClient()

	if err := registrar.RegisterGame("Dev", 0); err != nil {
		t.Errorf("RegisterGame failed: %v", err)
	}

	if err := registrar.BindScreenEvent("event", "device"); err != nil {
		t.Errorf("BindScreenEvent failed: %v", err)
	}

	if err := registrar.RemoveGame(); err != nil {
		t.Errorf("RemoveGame failed: %v", err)
	}
}
