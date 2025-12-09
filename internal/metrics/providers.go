package metrics

import "time"

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
