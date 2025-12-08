package shared

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockConnectable is a mock implementation of Connectable for testing
type mockConnectable struct {
	mu           sync.Mutex
	connected    bool
	connectError error
	connectDelay time.Duration
	connectCount int
}

func (m *mockConnectable) Connect(ctx context.Context) error {
	m.mu.Lock()
	m.connectCount++
	delay := m.connectDelay
	err := m.connectError
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err != nil {
		return err
	}

	m.mu.Lock()
	m.connected = true
	m.mu.Unlock()
	return nil
}

func (m *mockConnectable) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockConnectable) setConnected(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = v
}

func (m *mockConnectable) getConnectCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connectCount
}

func TestNewConnectionManager(t *testing.T) {
	mock := &mockConnectable{}
	cm := NewConnectionManager(mock, 5*time.Second, 30*time.Second)

	if cm == nil {
		t.Fatal("NewConnectionManager returned nil")
	}
	if cm.reconnectInterval != 5*time.Second {
		t.Errorf("reconnectInterval = %v, want 5s", cm.reconnectInterval)
	}
	if cm.connectionTimeout != 30*time.Second {
		t.Errorf("connectionTimeout = %v, want 30s", cm.connectionTimeout)
	}
}

func TestConnectionManager_Update_AlreadyConnected(t *testing.T) {
	mock := &mockConnectable{connected: true}
	cm := NewConnectionManager(mock, 0, 30*time.Second)

	connecting := cm.Update()

	if connecting {
		t.Error("Update() = true when already connected, want false")
	}
	if mock.getConnectCount() != 0 {
		t.Errorf("Connect() called %d times, want 0", mock.getConnectCount())
	}
}

func TestConnectionManager_Update_InitiatesConnection(t *testing.T) {
	mock := &mockConnectable{connectDelay: 50 * time.Millisecond}
	cm := NewConnectionManager(mock, 0, 30*time.Second)

	connecting := cm.Update()

	if !connecting {
		t.Error("Update() = false, want true (connecting)")
	}
	if !cm.IsConnecting() {
		t.Error("IsConnecting() = false after Update(), want true")
	}

	// Wait for connection to complete (Connect() delay + IsConnected() polling interval)
	time.Sleep(250 * time.Millisecond)

	if cm.IsConnecting() {
		t.Error("IsConnecting() = true after delay, want false")
	}
	if !mock.IsConnected() {
		t.Error("mock.IsConnected() = false, want true")
	}
	if mock.getConnectCount() != 1 {
		t.Errorf("Connect() called %d times, want 1", mock.getConnectCount())
	}
}

func TestConnectionManager_Update_RespectsReconnectInterval(t *testing.T) {
	mock := &mockConnectable{}
	cm := NewConnectionManager(mock, 200*time.Millisecond, 30*time.Second)

	// First call should initiate connection
	cm.Update()
	// Wait for connection to complete (includes IsConnected() polling)
	time.Sleep(150 * time.Millisecond)
	mock.setConnected(false) // Simulate disconnect

	// Immediate second call should not reconnect (interval not passed)
	connecting := cm.Update()
	if connecting {
		t.Error("Update() should not reconnect before interval")
	}

	// Wait for interval to pass
	time.Sleep(200 * time.Millisecond)

	// Now should reconnect
	connecting = cm.Update()
	if !connecting {
		t.Error("Update() should reconnect after interval")
	}
}

func TestConnectionManager_Update_HandlesError(t *testing.T) {
	expectedErr := errors.New("connection failed")
	mock := &mockConnectable{connectError: expectedErr}
	cm := NewConnectionManager(mock, 0, 30*time.Second)

	var receivedErr error
	cm.SetErrorCallback(func(err error) {
		receivedErr = err
	})

	cm.Update()
	time.Sleep(20 * time.Millisecond) // Let goroutine complete

	if cm.GetError() == nil {
		t.Error("GetError() = nil, want error")
	}
	if !errors.Is(cm.GetError(), expectedErr) {
		t.Errorf("GetError() = %v, want %v", cm.GetError(), expectedErr)
	}
	if receivedErr == nil {
		t.Error("error callback not called")
	}
}

func TestConnectionManager_ClearError(t *testing.T) {
	mock := &mockConnectable{connectError: errors.New("error")}
	cm := NewConnectionManager(mock, 0, 30*time.Second)

	cm.Update()
	time.Sleep(20 * time.Millisecond)

	if cm.GetError() == nil {
		t.Fatal("GetError() = nil before ClearError")
	}

	cm.ClearError()

	if cm.GetError() != nil {
		t.Error("GetError() != nil after ClearError")
	}
}

func TestConnectionManager_ResetConnectionTimer(t *testing.T) {
	mock := &mockConnectable{}
	cm := NewConnectionManager(mock, 1*time.Hour, 30*time.Second) // Long interval

	// First connection
	cm.Update()
	// Wait for connection to complete (includes IsConnected() polling)
	time.Sleep(150 * time.Millisecond)
	mock.setConnected(false)

	// Should not reconnect (interval is 1 hour)
	if cm.Update() {
		t.Error("Update() should not reconnect before interval")
	}

	// Reset timer
	cm.ResetConnectionTimer()

	// Should now reconnect immediately
	if !cm.Update() {
		t.Error("Update() should reconnect after ResetConnectionTimer")
	}
}

func TestConnectionManager_IsConnected(t *testing.T) {
	mock := &mockConnectable{connected: true}
	cm := NewConnectionManager(mock, 0, 30*time.Second)

	if !cm.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}

	mock.setConnected(false)

	if cm.IsConnected() {
		t.Error("IsConnected() = true after disconnect, want false")
	}
}

func TestConnectionManager_ConcurrentAccess(t *testing.T) {
	mock := &mockConnectable{connectDelay: 10 * time.Millisecond}
	cm := NewConnectionManager(mock, 0, 30*time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.Update()
			_ = cm.IsConnecting()
			_ = cm.IsConnected()
			_ = cm.GetError()
		}()
	}
	wg.Wait()

	// Should have only initiated one connection despite concurrent calls
	// Wait for connection to complete (includes IsConnected() polling)
	time.Sleep(200 * time.Millisecond)
	if mock.getConnectCount() > 1 {
		t.Errorf("Connect() called %d times, want 1", mock.getConnectCount())
	}
}

func TestConnectionManager_IsInitialState(t *testing.T) {
	mock := &mockConnectable{
		connectDelay: 50 * time.Millisecond,
	}

	cm := NewConnectionManager(mock, 100*time.Millisecond, 5*time.Second)

	// Before any Update call, should be in initial state
	if !cm.IsInitialState() {
		t.Error("IsInitialState() = false initially, want true")
	}

	// After Update initiates connection, should no longer be initial state
	cm.Update()
	if cm.IsInitialState() {
		t.Error("IsInitialState() = true after Update, want false")
	}

	// Wait for connection to complete
	time.Sleep(100 * time.Millisecond)
	cm.Update()

	// Still should not be initial state even after connection completes
	if cm.IsInitialState() {
		t.Error("IsInitialState() = true after connection complete, want false")
	}
}
