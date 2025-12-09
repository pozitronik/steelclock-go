package metrics

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// GopsutilCPU implements CPUProvider using gopsutil
type GopsutilCPU struct{}

// NewGopsutilCPU creates a new gopsutil-based CPU provider
func NewGopsutilCPU() *GopsutilCPU {
	return &GopsutilCPU{}
}

// Counts returns the number of CPU cores
func (g *GopsutilCPU) Counts(logical bool) (int, error) {
	return cpu.Counts(logical)
}

// Percent returns CPU usage percentages
func (g *GopsutilCPU) Percent(interval time.Duration, perCore bool) ([]float64, error) {
	return cpu.Percent(interval, perCore)
}

// GopsutilMemory implements MemoryProvider using gopsutil
type GopsutilMemory struct{}

// NewGopsutilMemory creates a new gopsutil-based memory provider
func NewGopsutilMemory() *GopsutilMemory {
	return &GopsutilMemory{}
}

// UsedPercent returns the percentage of memory in use
func (g *GopsutilMemory) UsedPercent() (float64, error) {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return vmem.UsedPercent, nil
}

// GopsutilNetwork implements NetworkProvider using gopsutil
type GopsutilNetwork struct{}

// NewGopsutilNetwork creates a new gopsutil-based network provider
func NewGopsutilNetwork() *GopsutilNetwork {
	return &GopsutilNetwork{}
}

// IOCounters returns network I/O statistics for all interfaces
func (g *GopsutilNetwork) IOCounters() ([]NetworkStat, error) {
	stats, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	result := make([]NetworkStat, len(stats))
	for i, s := range stats {
		result[i] = NetworkStat{
			Name:      s.Name,
			BytesRecv: s.BytesRecv,
			BytesSent: s.BytesSent,
		}
	}
	return result, nil
}

// GopsutilDisk implements DiskProvider using gopsutil
type GopsutilDisk struct{}

// NewGopsutilDisk creates a new gopsutil-based disk provider
func NewGopsutilDisk() *GopsutilDisk {
	return &GopsutilDisk{}
}

// IOCounters returns disk I/O statistics for all devices
func (g *GopsutilDisk) IOCounters() (map[string]DiskStat, error) {
	stats, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}

	result := make(map[string]DiskStat, len(stats))
	for name, s := range stats {
		result[name] = DiskStat{
			Name:       name,
			ReadBytes:  s.ReadBytes,
			WriteBytes: s.WriteBytes,
		}
	}
	return result, nil
}

// Default provider instances for convenience
var (
	DefaultCPU     CPUProvider     = NewGopsutilCPU()
	DefaultMemory  MemoryProvider  = NewGopsutilMemory()
	DefaultNetwork NetworkProvider = NewGopsutilNetwork()
	DefaultDisk    DiskProvider    = NewGopsutilDisk()
)
