//go:build windows

package gpu

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	pdh                             = syscall.NewLazyDLL("pdh.dll")
	pdhOpenQuery                    = pdh.NewProc("PdhOpenQueryW")
	pdhCloseQuery                   = pdh.NewProc("PdhCloseQuery")
	pdhAddEnglishCounter            = pdh.NewProc("PdhAddEnglishCounterW")
	pdhCollectQueryData             = pdh.NewProc("PdhCollectQueryData")
	pdhGetFormattedCounterArrayW    = pdh.NewProc("PdhGetFormattedCounterArrayW")
	pdhGetFormattedCounterValue     = pdh.NewProc("PdhGetFormattedCounterValue")
	pdhEnumObjectItemsW             = pdh.NewProc("PdhEnumObjectItemsW")
	pdhGetCounterInfoW              = pdh.NewProc("PdhGetCounterInfoW")
	pdhMakeCounterPathW             = pdh.NewProc("PdhMakeCounterPathW")
	pdhExpandWildCardPathW          = pdh.NewProc("PdhExpandWildCardPathW")
	pdhRemoveCounter                = pdh.NewProc("PdhRemoveCounter")
	pdhGetFormattedCounterArraySize = pdh.NewProc("PdhGetFormattedCounterArrayW")
)

// PDH constants
const (
	pdhFmtDouble      = 0x00000200
	pdhFmtLarge       = 0x00000400
	pdhMoreData       = 0x800007D2
	pdhCstatValidData = 0x00000000
	pdhInvalidData    = 0xC0000BBA
	pdhNoData         = 0x800007D5
)

// PDH_FMT_COUNTERVALUE_ITEM_DOUBLE for array results
type pdhFmtCountervalueItemDouble struct {
	szName   *uint16
	FmtValue pdhFmtCountervalueDouble
}

type pdhFmtCountervalueDouble struct {
	CStatus     uint32
	doubleValue float64
}

type pdhFmtCountervalueLarge struct {
	CStatus    uint32
	largeValue int64
}

// pdhReader implements the Reader interface using Windows PDH API
type pdhReader struct {
	mu            sync.Mutex
	queryHandle   uintptr
	counters      map[string]uintptr // metric name -> counter handle
	adapterCache  []AdapterInfo
	memoryTotals  map[int]map[string]uint64 // adapter -> metric -> total bytes
	initialized   bool
	physRegex     *regexp.Regexp
	engtypeRegex  *regexp.Regexp
}

// newReader creates a new PDH-based GPU metrics reader
func newReader() (Reader, error) {
	r := &pdhReader{
		counters:     make(map[string]uintptr),
		memoryTotals: make(map[int]map[string]uint64),
		physRegex:    regexp.MustCompile(`phys_(\d+)`),
		engtypeRegex: regexp.MustCompile(`engtype_(\w+)`),
	}

	if err := r.initialize(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *pdhReader) initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Open PDH query
	var queryHandle uintptr
	ret, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&queryHandle)))
	if ret != 0 {
		return fmt.Errorf("PdhOpenQuery failed: 0x%x", ret)
	}
	r.queryHandle = queryHandle

	// Add wildcard counters for GPU metrics
	counterPaths := map[string]string{
		"engine":           `\GPU Engine(*)\Utilization Percentage`,
		"memory_dedicated": `\GPU Adapter Memory(*)\Dedicated Usage`,
		"memory_shared":    `\GPU Adapter Memory(*)\Shared Usage`,
	}

	for name, path := range counterPaths {
		pathPtr, _ := syscall.UTF16PtrFromString(path)
		var counterHandle uintptr
		ret, _, _ = pdhAddEnglishCounter.Call(
			r.queryHandle,
			uintptr(unsafe.Pointer(pathPtr)),
			0,
			uintptr(unsafe.Pointer(&counterHandle)),
		)
		if ret != 0 {
			// Counter might not exist on all systems, log but continue
			continue
		}
		r.counters[name] = counterHandle
	}

	if len(r.counters) == 0 {
		pdhCloseQuery.Call(r.queryHandle)
		return fmt.Errorf("no GPU performance counters available")
	}

	// Initial data collection to populate counters
	pdhCollectQueryData.Call(r.queryHandle)

	// Discover adapters
	r.discoverAdapters()

	r.initialized = true
	return nil
}

func (r *pdhReader) discoverAdapters() {
	// Use memory counter to discover adapters (simpler instance names)
	counterHandle, ok := r.counters["memory_dedicated"]
	if !ok {
		counterHandle, ok = r.counters["engine"]
		if !ok {
			return
		}
	}

	// Get array of counter values to extract instance names
	var bufferSize uint32
	var itemCount uint32

	// First call to get buffer size
	ret, _, _ := pdhGetFormattedCounterArrayW.Call(
		counterHandle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)

	if ret != pdhMoreData && ret != 0 {
		return
	}

	if bufferSize == 0 {
		return
	}

	// Allocate buffer and get values
	buffer := make([]byte, bufferSize)
	ret, _, _ = pdhGetFormattedCounterArrayW.Call(
		counterHandle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)

	if ret != 0 {
		return
	}

	// Parse instance names to find unique adapters
	adapterMap := make(map[int]bool)
	itemSize := unsafe.Sizeof(pdhFmtCountervalueItemDouble{})

	for i := uint32(0); i < itemCount; i++ {
		item := (*pdhFmtCountervalueItemDouble)(unsafe.Pointer(&buffer[uintptr(i)*itemSize]))
		if item.szName != nil {
			name := syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(item.szName))[:])
			matches := r.physRegex.FindStringSubmatch(name)
			if len(matches) > 1 {
				var physIndex int
				fmt.Sscanf(matches[1], "%d", &physIndex)
				adapterMap[physIndex] = true
			}
		}
	}

	// Build adapter list
	r.adapterCache = nil
	for idx := range adapterMap {
		r.adapterCache = append(r.adapterCache, AdapterInfo{
			Index: idx,
			Name:  fmt.Sprintf("GPU %d", idx),
		})
	}
}

// GetMetric returns the current value for the specified metric and adapter
func (r *pdhReader) GetMetric(adapter int, metric string) (float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return 0, fmt.Errorf("reader not initialized")
	}

	// Collect fresh data
	ret, _, _ := pdhCollectQueryData.Call(r.queryHandle)
	if ret != 0 {
		return 0, fmt.Errorf("PdhCollectQueryData failed: 0x%x", ret)
	}

	// Determine which counter to use based on metric
	var counterKey string
	var engTypeFilter string

	switch metric {
	case MetricUtilization:
		counterKey = "engine"
		engTypeFilter = "" // All engine types
	case MetricUtilization3D:
		counterKey = "engine"
		engTypeFilter = "3D"
	case MetricUtilizationCopy:
		counterKey = "engine"
		engTypeFilter = "Copy"
	case MetricUtilizationEncode:
		counterKey = "engine"
		engTypeFilter = "VideoEncode"
	case MetricUtilizationDecode:
		counterKey = "engine"
		engTypeFilter = "VideoDecode"
	case MetricMemoryDedicated:
		counterKey = "memory_dedicated"
	case MetricMemoryShared:
		counterKey = "memory_shared"
	default:
		return 0, fmt.Errorf("unknown metric: %s", metric)
	}

	counterHandle, ok := r.counters[counterKey]
	if !ok {
		return 0, fmt.Errorf("counter not available for metric: %s", metric)
	}

	// Get counter array
	var bufferSize uint32
	var itemCount uint32

	ret, _, _ = pdhGetFormattedCounterArrayW.Call(
		counterHandle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)

	if ret != pdhMoreData && ret != 0 {
		return 0, fmt.Errorf("PdhGetFormattedCounterArrayW size query failed: 0x%x", ret)
	}

	if bufferSize == 0 || itemCount == 0 {
		return 0, nil
	}

	buffer := make([]byte, bufferSize)
	ret, _, _ = pdhGetFormattedCounterArrayW.Call(
		counterHandle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)

	if ret != 0 {
		return 0, fmt.Errorf("PdhGetFormattedCounterArrayW failed: 0x%x", ret)
	}

	// Aggregate values for the specified adapter
	var total float64
	var count int
	itemSize := unsafe.Sizeof(pdhFmtCountervalueItemDouble{})

	for i := uint32(0); i < itemCount; i++ {
		item := (*pdhFmtCountervalueItemDouble)(unsafe.Pointer(&buffer[uintptr(i)*itemSize]))
		if item.szName == nil {
			continue
		}

		name := syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(item.szName))[:])

		// Check adapter (phys_N)
		matches := r.physRegex.FindStringSubmatch(name)
		if len(matches) <= 1 {
			continue
		}
		var physIndex int
		fmt.Sscanf(matches[1], "%d", &physIndex)
		if physIndex != adapter {
			continue
		}

		// For engine metrics, check engine type filter
		if engTypeFilter != "" {
			engMatches := r.engtypeRegex.FindStringSubmatch(name)
			if len(engMatches) <= 1 {
				continue
			}
			if !strings.EqualFold(engMatches[1], engTypeFilter) {
				continue
			}
		}

		// Check if value is valid
		if item.FmtValue.CStatus == pdhCstatValidData || item.FmtValue.CStatus == 0 {
			total += item.FmtValue.doubleValue
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	// For memory metrics, return total bytes
	// For utilization metrics, return max value (not average) as that's more representative
	if counterKey == "memory_dedicated" || counterKey == "memory_shared" {
		return total, nil // Already bytes
	}

	// For utilization, take the maximum across all matching engines
	// This is more intuitive than averaging (e.g., if 3D engine is at 100%, GPU is busy)
	maxValue := 0.0
	for i := uint32(0); i < itemCount; i++ {
		item := (*pdhFmtCountervalueItemDouble)(unsafe.Pointer(&buffer[uintptr(i)*itemSize]))
		if item.szName == nil {
			continue
		}

		name := syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(item.szName))[:])

		matches := r.physRegex.FindStringSubmatch(name)
		if len(matches) <= 1 {
			continue
		}
		var physIndex int
		fmt.Sscanf(matches[1], "%d", &physIndex)
		if physIndex != adapter {
			continue
		}

		if engTypeFilter != "" {
			engMatches := r.engtypeRegex.FindStringSubmatch(name)
			if len(engMatches) <= 1 || !strings.EqualFold(engMatches[1], engTypeFilter) {
				continue
			}
		}

		if item.FmtValue.CStatus == pdhCstatValidData || item.FmtValue.CStatus == 0 {
			if item.FmtValue.doubleValue > maxValue {
				maxValue = item.FmtValue.doubleValue
			}
		}
	}

	return maxValue, nil
}

// GetMemoryTotal returns the total memory for the specified metric type
func (r *pdhReader) GetMemoryTotal(adapter int, metricType string) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache first
	if adapterTotals, ok := r.memoryTotals[adapter]; ok {
		if total, ok := adapterTotals[metricType]; ok && total > 0 {
			return total, nil
		}
	}

	// For dedicated memory, we need to query the total
	// Unfortunately, PDH doesn't directly provide total VRAM
	// We'll estimate based on "Total Committed" counter if available
	// or return 0 to show raw bytes instead of percentage

	// Try to get committed memory as a proxy for total
	counterHandle, ok := r.counters["memory_dedicated"]
	if !ok {
		return 0, fmt.Errorf("memory counter not available")
	}

	// Get current values to find committed memory
	var bufferSize uint32
	var itemCount uint32

	ret, _, _ := pdhGetFormattedCounterArrayW.Call(
		counterHandle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)

	if ret != pdhMoreData && ret != 0 {
		return 0, nil
	}

	// For now, return 0 which will cause the widget to display raw utilization
	// In a future enhancement, we could query WMI for total VRAM
	return 0, nil
}

// ListAdapters returns information about available GPU adapters
func (r *pdhReader) ListAdapters() ([]AdapterInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return nil, fmt.Errorf("reader not initialized")
	}

	return r.adapterCache, nil
}

// Close releases PDH resources
func (r *pdhReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		for _, handle := range r.counters {
			pdhRemoveCounter.Call(handle)
		}
		pdhCloseQuery.Call(r.queryHandle)
		r.initialized = false
	}
}
