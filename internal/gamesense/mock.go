package gamesense

import "sync"

// MockAPI is a mock implementation of API for testing
type MockAPI struct {
	// Function handlers for custom behavior
	RegisterGameFunc    func(developer string) error
	BindScreenEventFunc func(eventName, deviceType string) error
	SendScreenDataFunc  func(eventName string, bitmapData []int) error
	SendHeartbeatFunc   func() error
	RemoveGameFunc      func() error

	// Call tracking
	mu                   sync.Mutex
	RegisterGameCalls    int
	BindScreenEventCalls int
	SendScreenDataCalls  int
	SendHeartbeatCalls   int
	RemoveGameCalls      int

	// Last call arguments
	LastRegisterGameDeveloper string
	LastBindEventName         string
	LastBindDeviceType        string
	LastSendDataEventName     string
	LastSendDataBitmap        []int
}

// NewMockAPI creates a new mock GameSense API
func NewMockAPI() *MockAPI {
	return &MockAPI{}
}

// RegisterGame implements API
func (m *MockAPI) RegisterGame(developer string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RegisterGameCalls++
	m.LastRegisterGameDeveloper = developer

	if m.RegisterGameFunc != nil {
		return m.RegisterGameFunc(developer)
	}
	return nil
}

// BindScreenEvent implements API
func (m *MockAPI) BindScreenEvent(eventName, deviceType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BindScreenEventCalls++
	m.LastBindEventName = eventName
	m.LastBindDeviceType = deviceType

	if m.BindScreenEventFunc != nil {
		return m.BindScreenEventFunc(eventName, deviceType)
	}
	return nil
}

// SendScreenData implements API
func (m *MockAPI) SendScreenData(eventName string, bitmapData []int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SendScreenDataCalls++
	m.LastSendDataEventName = eventName
	m.LastSendDataBitmap = bitmapData

	if m.SendScreenDataFunc != nil {
		return m.SendScreenDataFunc(eventName, bitmapData)
	}
	return nil
}

// SendHeartbeat implements API
func (m *MockAPI) SendHeartbeat() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SendHeartbeatCalls++

	if m.SendHeartbeatFunc != nil {
		return m.SendHeartbeatFunc()
	}
	return nil
}

// RemoveGame implements API
func (m *MockAPI) RemoveGame() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RemoveGameCalls++

	if m.RemoveGameFunc != nil {
		return m.RemoveGameFunc()
	}
	return nil
}

// Reset clears all call tracking data
func (m *MockAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RegisterGameCalls = 0
	m.BindScreenEventCalls = 0
	m.SendScreenDataCalls = 0
	m.SendHeartbeatCalls = 0
	m.RemoveGameCalls = 0

	m.LastRegisterGameDeveloper = ""
	m.LastBindEventName = ""
	m.LastBindDeviceType = ""
	m.LastSendDataEventName = ""
	m.LastSendDataBitmap = nil
}

// GetCallCounts returns the current call counts (thread-safe)
func (m *MockAPI) GetCallCounts() (register, bind, sendData, heartbeat, remove int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.RegisterGameCalls, m.BindScreenEventCalls, m.SendScreenDataCalls, m.SendHeartbeatCalls, m.RemoveGameCalls
}

// Ensure MockAPI implements API
var _ API = (*MockAPI)(nil)
