package gamesense

// MockGameSenseAPI is a mock implementation of GameSenseAPI for testing
type MockGameSenseAPI struct {
	// Function handlers for custom behavior
	RegisterGameFunc           func(developer string, deinitializeTimerMs int) error
	BindScreenEventFunc        func(eventName, deviceType string) error
	SendScreenDataFunc         func(eventName string, bitmapData []byte) error
	SendScreenDataMultiResFunc func(eventName string, resolutionData map[string][]byte) error
	SendHeartbeatFunc          func() error
	RemoveGameFunc             func() error
	SupportsMultipleEventsFunc func() bool
	SendMultipleScreenDataFunc func(eventName string, frames [][]byte) error

	// Call tracking
	RegisterGameCalls           int
	BindScreenEventCalls        int
	SendScreenDataCalls         int
	SendScreenDataMultiResCalls int
	SendHeartbeatCalls          int
	RemoveGameCalls             int
	SupportsMultipleEventsCalls int
	SendMultipleScreenDataCalls int

	// Last call arguments
	LastRegisterGameDeveloper string
	LastRegisterGameTimerMs   int
	LastBindEventName         string
	LastBindDeviceType        string
	LastSendDataEventName     string
	LastSendDataBitmap        []byte
}

// RegisterGame implements GameSenseAPI
func (m *MockGameSenseAPI) RegisterGame(developer string, deinitializeTimerMs int) error {
	m.RegisterGameCalls++
	m.LastRegisterGameDeveloper = developer
	m.LastRegisterGameTimerMs = deinitializeTimerMs

	if m.RegisterGameFunc != nil {
		return m.RegisterGameFunc(developer, deinitializeTimerMs)
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
func (m *MockGameSenseAPI) SendScreenData(eventName string, bitmapData []byte) error {
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

// SupportsMultipleEvents implements API
func (m *MockGameSenseAPI) SupportsMultipleEvents() bool {
	m.SupportsMultipleEventsCalls++

	if m.SupportsMultipleEventsFunc != nil {
		return m.SupportsMultipleEventsFunc()
	}
	return false
}

// SendScreenDataMultiRes implements API
func (m *MockGameSenseAPI) SendScreenDataMultiRes(eventName string, resolutionData map[string][]byte) error {
	m.SendScreenDataMultiResCalls++

	if m.SendScreenDataMultiResFunc != nil {
		return m.SendScreenDataMultiResFunc(eventName, resolutionData)
	}
	return nil
}

// SendMultipleScreenData implements API
func (m *MockGameSenseAPI) SendMultipleScreenData(eventName string, frames [][]byte) error {
	m.SendMultipleScreenDataCalls++

	if m.SendMultipleScreenDataFunc != nil {
		return m.SendMultipleScreenDataFunc(eventName, frames)
	}
	return nil
}

// Reset clears all call tracking data
func (m *MockGameSenseAPI) Reset() {
	m.RegisterGameCalls = 0
	m.BindScreenEventCalls = 0
	m.SendScreenDataCalls = 0
	m.SendScreenDataMultiResCalls = 0
	m.SendHeartbeatCalls = 0
	m.RemoveGameCalls = 0
	m.SupportsMultipleEventsCalls = 0
	m.SendMultipleScreenDataCalls = 0

	m.LastRegisterGameDeveloper = ""
	m.LastRegisterGameTimerMs = 0
	m.LastBindEventName = ""
	m.LastBindDeviceType = ""
	m.LastSendDataEventName = ""
	m.LastSendDataBitmap = nil
}

// Ensure MockGameSenseAPI implements GameSenseAPI
var _ API = (*MockGameSenseAPI)(nil)
