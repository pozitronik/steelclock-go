package metrics

import "time"

// Metrics providers abstract system metrics collection for widgets.
//
// # Testing Pattern
//
// Widgets store provider instances as unexported fields (e.g., cpuProvider).
// Tests in the same package can inject mock providers after widget creation:
//
//	widget, _ := cpu.New(cfg)
//	widget.cpuProvider = &metrics.MockCPU{
//	    PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
//	        return []float64{75.5}, nil
//	    },
//	}
//	widget.Update() // Uses mock
//
// This pattern enables:
// - Testing specific value scenarios
// - Testing edge cases (0%, 100%, negative, overflow)
// - Testing error handling
//
// See mock.go for available mock implementations.
// See cpu/cpu_mock_test.go for example usage.

// CPUProvider abstracts CPU metrics collection
type CPUProvider interface {
	// Counts returns the number of CPU cores.
	// If logical is true, returns logical cores; otherwise physical cores.
	Counts(logical bool) (int, error)

	// Percent returns CPU usage percentages.
	// If perCore is true, returns a slice with one value per core.
	// If perCore is false, returns a single-element slice with aggregate usage.
	// The interval parameter specifies the sampling duration.
	Percent(interval time.Duration, perCore bool) ([]float64, error)
}

// MemoryProvider abstracts memory metrics collection
type MemoryProvider interface {
	// UsedPercent returns the percentage of memory currently in use.
	UsedPercent() (float64, error)
}

// NetworkProvider abstracts network I/O metrics collection
type NetworkProvider interface {
	// IOCounters returns network I/O statistics for all interfaces.
	IOCounters() ([]NetworkStat, error)
}

// DiskProvider abstracts disk I/O metrics collection
type DiskProvider interface {
	// IOCounters returns disk I/O statistics for all devices.
	IOCounters() (map[string]DiskStat, error)
}
