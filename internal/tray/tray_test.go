package tray

import (
	"testing"
	"time"
)

// TestNewManager tests that NewManager creates a valid manager
func TestNewManager(t *testing.T) {
	configPath := "/test/config.json"
	reloadFunc := func() error { return nil }
	exitFunc := func() {}

	mgr := NewManager(configPath, reloadFunc, exitFunc)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.configPath != configPath {
		t.Errorf("configPath = %s, want %s", mgr.configPath, configPath)
	}

	if mgr.onReload == nil {
		t.Error("onReload is nil")
	}

	if mgr.onExit == nil {
		t.Error("onExit is nil")
	}

	if mgr.readyChan == nil {
		t.Error("readyChan is nil")
	}
}

// TestOnReady tests that OnReady sets the callback
func TestOnReady(t *testing.T) {
	mgr := NewManager("/test/config.json", func() error { return nil }, func() {})

	callbackCalled := false
	callback := func() {
		callbackCalled = true
	}

	mgr.OnReady(callback)

	if mgr.onReadyCallback == nil {
		t.Error("onReadyCallback was not set")
	}

	// Call the callback directly to verify it works
	mgr.onReadyCallback()

	if !callbackCalled {
		t.Error("callback was not called")
	}
}

// TestWaitReady tests that WaitReady blocks until ready channel is closed
func TestWaitReady(t *testing.T) {
	mgr := NewManager("/test/config.json", func() error { return nil }, func() {})

	// Test that WaitReady blocks
	readyReceived := false
	done := make(chan bool)

	go func() {
		mgr.WaitReady()
		readyReceived = true
		done <- true
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Should still be blocking
	select {
	case <-done:
		t.Error("WaitReady returned before ready channel was closed")
	default:
		// Expected - still blocking
	}

	// Now close the ready channel
	close(mgr.readyChan)

	// Wait for goroutine to finish (with timeout)
	select {
	case <-done:
		if !readyReceived {
			t.Error("readyReceived was not set to true")
		}
	case <-time.After(1 * time.Second):
		t.Error("WaitReady did not return after ready channel was closed")
	}
}

// TestWaitReady_AlreadyClosed tests WaitReady when channel is already closed
func TestWaitReady_AlreadyClosed(t *testing.T) {
	mgr := NewManager("/test/config.json", func() error { return nil }, func() {})

	// Close the channel before waiting
	close(mgr.readyChan)

	// This should return immediately
	done := make(chan bool)
	go func() {
		mgr.WaitReady()
		done <- true
	}()

	select {
	case <-done:
		// Expected - returned immediately
	case <-time.After(1 * time.Second):
		t.Error("WaitReady did not return immediately when channel already closed")
	}
}

// TestOnReady_MultipleCallbacks tests that setting callback multiple times works
func TestOnReady_MultipleCallbacks(t *testing.T) {
	mgr := NewManager("/test/config.json", func() error { return nil }, func() {})

	firstCalled := false
	secondCalled := false

	mgr.OnReady(func() {
		firstCalled = true
	})

	mgr.OnReady(func() {
		secondCalled = true
	})

	// Only the last callback should be set
	mgr.onReadyCallback()

	if firstCalled {
		t.Error("first callback should not have been called (was overwritten)")
	}

	if !secondCalled {
		t.Error("second callback should have been called")
	}
}

// TestOnReady_NilCallback tests that nil callback is handled
func TestOnReady_NilCallback(t *testing.T) {
	mgr := NewManager("/test/config.json", func() error { return nil }, func() {})

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("OnReady with nil callback panicked: %v", r)
		}
	}()

	mgr.OnReady(nil)

	if mgr.onReadyCallback != nil {
		t.Error("onReadyCallback should be nil")
	}
}

// TestManagerCallbacks tests that callbacks are stored correctly
func TestManagerCallbacks(t *testing.T) {
	reloadCalled := false
	exitCalled := false

	reloadFunc := func() error {
		reloadCalled = true
		return nil
	}

	exitFunc := func() {
		exitCalled = true
	}

	mgr := NewManager("/test/config.json", reloadFunc, exitFunc)

	// Call the reload callback
	if mgr.onReload != nil {
		_ = mgr.onReload()
	}

	if !reloadCalled {
		t.Error("reload callback was not called")
	}

	// Call the exit callback
	if mgr.onExit != nil {
		mgr.onExit()
	}

	if !exitCalled {
		t.Error("exit callback was not called")
	}
}
