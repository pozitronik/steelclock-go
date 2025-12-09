package metrics

import "time"

// MockCPU is a mock implementation of CPUProvider for testing
type MockCPU struct {
	CountsFunc  func(logical bool) (int, error)
	PercentFunc func(interval time.Duration, perCore bool) ([]float64, error)
}

// Counts calls the mock function if set, otherwise returns defaults
func (m *MockCPU) Counts(logical bool) (int, error) {
	if m.CountsFunc != nil {
		return m.CountsFunc(logical)
	}
	return 4, nil // Default: 4 cores
}

// Percent calls the mock function if set, otherwise returns defaults
func (m *MockCPU) Percent(interval time.Duration, perCore bool) ([]float64, error) {
	if m.PercentFunc != nil {
		return m.PercentFunc(interval, perCore)
	}
	if perCore {
		return []float64{25.0, 50.0, 75.0, 100.0}, nil
	}
	return []float64{50.0}, nil
}

// MockMemory is a mock implementation of MemoryProvider for testing
type MockMemory struct {
	UsedPercentFunc func() (float64, error)
}

// UsedPercent calls the mock function if set, otherwise returns default
func (m *MockMemory) UsedPercent() (float64, error) {
	if m.UsedPercentFunc != nil {
		return m.UsedPercentFunc()
	}
	return 65.0, nil // Default: 65% used
}

// MockNetwork is a mock implementation of NetworkProvider for testing
type MockNetwork struct {
	IOCountersFunc func() ([]NetworkStat, error)
}

// IOCounters calls the mock function if set, otherwise returns defaults
func (m *MockNetwork) IOCounters() ([]NetworkStat, error) {
	if m.IOCountersFunc != nil {
		return m.IOCountersFunc()
	}
	return []NetworkStat{
		{Name: "eth0", BytesRecv: 1000000, BytesSent: 500000},
		{Name: "lo", BytesRecv: 100, BytesSent: 100},
	}, nil
}

// MockDisk is a mock implementation of DiskProvider for testing
type MockDisk struct {
	IOCountersFunc func() (map[string]DiskStat, error)
}

// IOCounters calls the mock function if set, otherwise returns defaults
func (m *MockDisk) IOCounters() (map[string]DiskStat, error) {
	if m.IOCountersFunc != nil {
		return m.IOCountersFunc()
	}
	return map[string]DiskStat{
		"sda": {Name: "sda", ReadBytes: 2000000, WriteBytes: 1000000},
		"sdb": {Name: "sdb", ReadBytes: 500000, WriteBytes: 250000},
	}, nil
}
