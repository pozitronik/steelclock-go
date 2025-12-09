package metrics

import (
	"errors"
	"testing"
	"time"
)

func TestMockCPU_Counts(t *testing.T) {
	t.Run("default returns 4 cores", func(t *testing.T) {
		mock := &MockCPU{}
		count, err := mock.Counts(true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if count != 4 {
			t.Errorf("expected 4 cores, got %d", count)
		}
	})

	t.Run("custom function", func(t *testing.T) {
		mock := &MockCPU{
			CountsFunc: func(logical bool) (int, error) {
				if logical {
					return 8, nil
				}
				return 4, nil
			},
		}

		count, _ := mock.Counts(true)
		if count != 8 {
			t.Errorf("expected 8 logical cores, got %d", count)
		}

		count, _ = mock.Counts(false)
		if count != 4 {
			t.Errorf("expected 4 physical cores, got %d", count)
		}
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("cpu error")
		mock := &MockCPU{
			CountsFunc: func(logical bool) (int, error) {
				return 0, expectedErr
			},
		}

		_, err := mock.Counts(true)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestMockCPU_Percent(t *testing.T) {
	t.Run("default per core", func(t *testing.T) {
		mock := &MockCPU{}
		percents, err := mock.Percent(100*time.Millisecond, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(percents) != 4 {
			t.Errorf("expected 4 values, got %d", len(percents))
		}
	})

	t.Run("default aggregate", func(t *testing.T) {
		mock := &MockCPU{}
		percents, err := mock.Percent(100*time.Millisecond, false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(percents) != 1 {
			t.Errorf("expected 1 value, got %d", len(percents))
		}
		if percents[0] != 50.0 {
			t.Errorf("expected 50.0, got %f", percents[0])
		}
	})

	t.Run("custom function", func(t *testing.T) {
		mock := &MockCPU{
			PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
				if perCore {
					return []float64{10.0, 20.0}, nil
				}
				return []float64{15.0}, nil
			},
		}

		percents, err := mock.Percent(50*time.Millisecond, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(percents) != 2 {
			t.Errorf("expected 2 values, got %d", len(percents))
		}
		if percents[0] != 10.0 || percents[1] != 20.0 {
			t.Errorf("expected [10.0, 20.0], got %v", percents)
		}

		percents, _ = mock.Percent(50*time.Millisecond, false)
		if len(percents) != 1 || percents[0] != 15.0 {
			t.Errorf("expected [15.0], got %v", percents)
		}
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("percent error")
		mock := &MockCPU{
			PercentFunc: func(interval time.Duration, perCore bool) ([]float64, error) {
				return nil, expectedErr
			},
		}

		_, err := mock.Percent(100*time.Millisecond, true)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestMockMemory_UsedPercent(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		mock := &MockMemory{}
		percent, err := mock.UsedPercent()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if percent != 65.0 {
			t.Errorf("expected 65.0, got %f", percent)
		}
	})

	t.Run("custom function", func(t *testing.T) {
		mock := &MockMemory{
			UsedPercentFunc: func() (float64, error) {
				return 80.5, nil
			},
		}
		percent, _ := mock.UsedPercent()
		if percent != 80.5 {
			t.Errorf("expected 80.5, got %f", percent)
		}
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("memory error")
		mock := &MockMemory{
			UsedPercentFunc: func() (float64, error) {
				return 0, expectedErr
			},
		}

		_, err := mock.UsedPercent()
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestMockNetwork_IOCounters(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		mock := &MockNetwork{}
		stats, err := mock.IOCounters()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(stats) != 2 {
			t.Errorf("expected 2 interfaces, got %d", len(stats))
		}
	})

	t.Run("custom function", func(t *testing.T) {
		mock := &MockNetwork{
			IOCountersFunc: func() ([]NetworkStat, error) {
				return []NetworkStat{
					{Name: "wlan0", BytesRecv: 5000, BytesSent: 3000},
				}, nil
			},
		}
		stats, _ := mock.IOCounters()
		if len(stats) != 1 {
			t.Errorf("expected 1 interface, got %d", len(stats))
		}
		if stats[0].Name != "wlan0" {
			t.Errorf("expected wlan0, got %s", stats[0].Name)
		}
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("network error")
		mock := &MockNetwork{
			IOCountersFunc: func() ([]NetworkStat, error) {
				return nil, expectedErr
			},
		}

		_, err := mock.IOCounters()
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestMockDisk_IOCounters(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		mock := &MockDisk{}
		stats, err := mock.IOCounters()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(stats) != 2 {
			t.Errorf("expected 2 disks, got %d", len(stats))
		}
		if _, ok := stats["sda"]; !ok {
			t.Error("expected sda in stats")
		}
	})

	t.Run("custom function", func(t *testing.T) {
		mock := &MockDisk{
			IOCountersFunc: func() (map[string]DiskStat, error) {
				return map[string]DiskStat{
					"nvme0n1": {Name: "nvme0n1", ReadBytes: 10000, WriteBytes: 5000},
				}, nil
			},
		}
		stats, _ := mock.IOCounters()
		if len(stats) != 1 {
			t.Errorf("expected 1 disk, got %d", len(stats))
		}
		if _, ok := stats["nvme0n1"]; !ok {
			t.Error("expected nvme0n1 in stats")
		}
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("disk error")
		mock := &MockDisk{
			IOCountersFunc: func() (map[string]DiskStat, error) {
				return nil, expectedErr
			},
		}

		_, err := mock.IOCounters()
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}

// Test that interfaces are properly implemented
func TestInterfaceImplementation(t *testing.T) {
	// These will fail at compile time if interfaces aren't implemented
	var _ CPUProvider = &MockCPU{}
	var _ CPUProvider = &GopsutilCPU{}

	var _ MemoryProvider = &MockMemory{}
	var _ MemoryProvider = &GopsutilMemory{}

	var _ NetworkProvider = &MockNetwork{}
	var _ NetworkProvider = &GopsutilNetwork{}

	var _ DiskProvider = &MockDisk{}
	var _ DiskProvider = &GopsutilDisk{}
}

// Integration tests for gopsutil implementations
// These tests call real system APIs and are skipped in short mode

func TestGopsutilCPU_Counts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cpu := NewGopsutilCPU()
	count, err := cpu.Counts(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 core, got %d", count)
	}
}

func TestGopsutilCPU_Percent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cpu := NewGopsutilCPU()

	// Test aggregate
	percents, err := cpu.Percent(10*time.Millisecond, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(percents) != 1 {
		t.Errorf("expected 1 aggregate value, got %d", len(percents))
	}
	if percents[0] < 0 || percents[0] > 100 {
		t.Errorf("expected percentage between 0-100, got %f", percents[0])
	}

	// Test per-core
	percents, err = cpu.Percent(10*time.Millisecond, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(percents) < 1 {
		t.Errorf("expected at least 1 per-core value, got %d", len(percents))
	}
}

func TestGopsutilMemory_UsedPercent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	mem := NewGopsutilMemory()
	percent, err := mem.UsedPercent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if percent < 0 || percent > 100 {
		t.Errorf("expected percentage between 0-100, got %f", percent)
	}
}

func TestGopsutilNetwork_IOCounters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	net := NewGopsutilNetwork()
	stats, err := net.IOCounters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Most systems have at least loopback interface
	if len(stats) < 1 {
		t.Errorf("expected at least 1 network interface, got %d", len(stats))
	}

	// Verify struct fields are populated
	for _, stat := range stats {
		if stat.Name == "" {
			t.Error("expected interface name to be non-empty")
		}
	}
}

func TestGopsutilDisk_IOCounters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	disk := NewGopsutilDisk()
	stats, err := disk.IOCounters()
	if err != nil {
		// Some systems may not support disk I/O counters
		t.Skipf("disk IOCounters not supported: %v", err)
	}
	// Note: Some systems may have no disk stats, so we just verify no error
	for name, stat := range stats {
		if stat.Name == "" {
			t.Errorf("expected disk %s to have name populated", name)
		}
	}
}
