package gamesense

// MockGameSenseAPI is a mock implementation of GameSenseAPI for testing
type MockGameSenseAPI struct {
	// Function handlers for custom behavior
	RegisterGameFunc    func(developer string) error
	BindScreenEventFunc func(eventName, deviceType string) error
	SendScreenDataFunc  func(eventName string, bitmapData []int) error
	SendHeartbeatFunc   func() error
	RemoveGameFunc      func() error

	// Call tracking
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

// RegisterGame implements GameSenseAPI
func (m *MockGameSenseAPI) RegisterGame(developer string) error {
	m.RegisterGameCalls++
	m.LastRegisterGameDeveloper = developer

	if m.RegisterGameFunc != nil {
		return m.RegisterGameFunc(developer)
	}
	return nil
}

// BindScreenEvent implements GameSenseAPI
func (m *MockGameSenseAPI) BindScreenEvent(eventName, deviceType string) error {
	m.BindScreenEventCalls++
	m.LastBindEventName = eventName
	m.LastBindDeviceType = deviceType

	if m.BindScreenEventFunc != nil {
		return m.BindScreenEventFunc(eventName, deviceType)
	}
	return nil
}

// SendScreenData implements GameSenseAPI
func (m *MockGameSenseAPI) SendScreenData(eventName string, bitmapData []int) error {
	m.SendScreenDataCalls++
	m.LastSendDataEventName = eventName
	m.LastSendDataBitmap = bitmapData

	if m.SendScreenDataFunc != nil {
		return m.SendScreenDataFunc(eventName, bitmapData)
	}
	return nil
}

// SendHeartbeat implements GameSenseAPI
func (m *MockGameSenseAPI) SendHeartbeat() error {
	m.SendHeartbeatCalls++

	if m.SendHeartbeatFunc != nil {
		return m.SendHeartbeatFunc()
	}
	return nil
}

// RemoveGame implements GameSenseAPI
func (m *MockGameSenseAPI) RemoveGame() error {
	m.RemoveGameCalls++

	if m.RemoveGameFunc != nil {
		return m.RemoveGameFunc()
	}
	return nil
}

// Reset clears all call tracking data
func (m *MockGameSenseAPI) Reset() {
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

// Ensure MockGameSenseAPI implements GameSenseAPI
var _ API = (*MockGameSenseAPI)(nil)
