//go:build windows

package gpu

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/yusufpapurcu/wmi"
)

var (
	pdh                          = syscall.NewLazyDLL("pdh.dll")
	pdhOpenQuery                 = pdh.NewProc("PdhOpenQueryW")
	pdhCloseQuery                = pdh.NewProc("PdhCloseQuery")
	pdhAddEnglishCounter         = pdh.NewProc("PdhAddEnglishCounterW")
	pdhCollectQueryData          = pdh.NewProc("PdhCollectQueryData")
	pdhGetFormattedCounterArrayW = pdh.NewProc("PdhGetFormattedCounterArrayW")
)

// PDH constants
const (
	pdhFmtDouble      = 0x00000200
	pdhMoreData       = 0x800007D2
	pdhCstatValidData = 0x00000000
)

// cacheValidDuration controls how long cached collection results remain valid.
// This prevents redundant PdhCollectQueryData calls when multiple GPU widgets
// query different metrics within the same update cycle.
const cacheValidDuration = 100 * time.Millisecond

// PDH_FMT_COUNTERVALUE_ITEM_DOUBLE for array results
type pdhFmtCountervalueItemDouble struct {
	szName   *uint16
	FmtValue pdhFmtCountervalueDouble
}

type pdhFmtCountervalueDouble struct {
	CStatus     uint32
	doubleValue float64
}

// collectionCache holds the aggregated metric values from the last PDH collection.
type collectionCache struct {
	// values maps adapter sequential index -> metric name -> value.
	values    map[int]map[string]float64
	timestamp time.Time
}

// pdhReader implements the Reader interface using Windows PDH API.
//
// Adapter identification uses LUID (Locally Unique Identifier) rather than the
// phys_N field from PDH instance names. All GPUs on a system report phys_0,
// making LUID the only reliable differentiator for multi-GPU systems.
//
// PDH instance name format:
//
//	pid_<N>_luid_0x<HH>_0x<HH>_phys_<N>_eng_<N>_engtype_<TYPE>
//
// LUIDs are discovered during initialization and mapped to sequential indices
// (0, 1, 2, ...) in sorted order for stable adapter numbering.
type pdhReader struct {
	mu            sync.Mutex
	queryHandle   uintptr
	engineCounter uintptr // PDH counter handle for GPU Engine utilization
	adapterCache  []AdapterInfo
	initialized   bool
	luidRegex     *regexp.Regexp // Extracts LUID from PDH instance name
	engtypeRegex  *regexp.Regexp // Extracts engine type (including spaces) from PDH instance name
	luidToIndex   map[string]int // Maps LUID string -> sequential adapter index
	cache         collectionCache
}

// newReader creates a new PDH-based GPU metrics reader.
func newReader() (Reader, error) {
	r := &pdhReader{
		// Matches the LUID portion: "luid_0x00000000_0x0001332d"
		luidRegex: regexp.MustCompile(`luid_(0x[0-9a-fA-F]+_0x[0-9a-fA-F]+)`),
		// Matches everything after "engtype_" to end of string, including spaces.
		// AMD uses "video decode 1", "high priority compute", etc.
		// NVIDIA uses "videodecode", "3d", etc.
		engtypeRegex: regexp.MustCompile(`engtype_(.+)$`),
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

	// Add engine utilization counter (the only currently supported counter category).
	// Memory counters (memory_dedicated, memory_shared) are intentionally omitted:
	// PDH doesn't provide total VRAM counters needed for percentage calculation.
	enginePath := `\GPU Engine(*)\Utilization Percentage`
	pathPtr, _ := syscall.UTF16PtrFromString(enginePath)
	var counterHandle uintptr
	ret, _, _ = pdhAddEnglishCounter.Call(
		r.queryHandle,
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&counterHandle)),
	)
	if ret != 0 {
		_, _, _ = pdhCloseQuery.Call(r.queryHandle)
		return fmt.Errorf("no GPU performance counters available (PdhAddEnglishCounter failed: 0x%x)", ret)
	}
	r.engineCounter = counterHandle

	// Initial data collection to populate counters
	pdhCollectQueryData.Call(r.queryHandle)

	// Discover adapters (with name enrichment)
	r.discoverAdapters()

	r.initialized = true
	return nil
}

// discoverAdapters builds the adapter list from PDH instances using LUID-based
// identification. LUIDs are sorted lexicographically for deterministic adapter
// numbering, then enriched with human-readable names from WMI when available.
func (r *pdhReader) discoverAdapters() {
	luidSet := make(map[string]bool)

	r.forEachCounterInstance(r.engineCounter, func(name string, _ *pdhFmtCountervalueItemDouble) {
		if luid := r.extractLUID(name); luid != "" {
			luidSet[luid] = true
		}
	})

	// Sort LUIDs for deterministic ordering across runs
	luids := make([]string, 0, len(luidSet))
	for luid := range luidSet {
		luids = append(luids, luid)
	}
	sort.Strings(luids)

	// Build LUID -> sequential index mapping
	r.luidToIndex = make(map[string]int, len(luids))
	for i, luid := range luids {
		r.luidToIndex[luid] = i
	}

	// Try to enrich adapter names from WMI.
	// WMI returns adapters ordered by device index, which typically matches
	// the LUID sort order.
	wmiNames := queryAdapterNames()

	r.adapterCache = make([]AdapterInfo, len(luids))
	for i, luid := range luids {
		name := fmt.Sprintf("GPU %d", i)
		if i < len(wmiNames) && wmiNames[i] != "" {
			name = wmiNames[i]
		}
		r.adapterCache[i] = AdapterInfo{
			Index: i,
			Name:  name,
			LUID:  luid,
		}
	}

	log.Printf("[GPU] Discovered %d adapter(s) via LUID:", len(r.adapterCache))
	for _, a := range r.adapterCache {
		log.Printf("[GPU]   %d: %s (LUID: %s)", a.Index, a.Name, a.LUID)
	}
}

// extractLUID extracts the LUID string from a PDH instance name.
// Example input: "pid_1234_luid_0x00000000_0x0001332d_phys_0_eng_0_engtype_3D"
// Returns: "0x00000000_0x0001332d"
func (r *pdhReader) extractLUID(instanceName string) string {
	matches := r.luidRegex.FindStringSubmatch(instanceName)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// forEachCounterInstance iterates over all instances in a PDH counter array.
// The callback receives the instance name and a pointer to the raw counter value item
// (valid only for the duration of the callback).
func (r *pdhReader) forEachCounterInstance(handle uintptr, fn func(name string, item *pdhFmtCountervalueItemDouble)) {
	var bufferSize, itemCount uint32

	ret, _, _ := pdhGetFormattedCounterArrayW.Call(
		handle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		0,
	)
	if (ret != pdhMoreData && ret != 0) || bufferSize == 0 || itemCount == 0 {
		return
	}

	buffer := make([]byte, bufferSize)
	ret, _, _ = pdhGetFormattedCounterArrayW.Call(
		handle,
		pdhFmtDouble,
		uintptr(unsafe.Pointer(&bufferSize)),
		uintptr(unsafe.Pointer(&itemCount)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	if ret != 0 {
		return
	}

	itemSize := unsafe.Sizeof(pdhFmtCountervalueItemDouble{})
	for i := uint32(0); i < itemCount; i++ {
		item := (*pdhFmtCountervalueItemDouble)(unsafe.Pointer(&buffer[uintptr(i)*itemSize]))
		if item.szName == nil {
			continue
		}
		name := syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(item.szName))[:])
		fn(name, item)
	}
}

// collectAndCache performs a single PDH data collection and aggregates all engine
// utilization values per adapter (identified by LUID) and metric type into the cache.
func (r *pdhReader) collectAndCache() error {
	ret, _, _ := pdhCollectQueryData.Call(r.queryHandle)
	if ret != 0 {
		return fmt.Errorf("PdhCollectQueryData failed: 0x%x", ret)
	}

	values := make(map[int]map[string]float64)

	r.forEachCounterInstance(r.engineCounter, func(name string, item *pdhFmtCountervalueItemDouble) {
		if item.FmtValue.CStatus != pdhCstatValidData && item.FmtValue.CStatus != 0 {
			return
		}

		// Identify adapter by LUID
		luid := r.extractLUID(name)
		if luid == "" {
			return
		}
		adapterIdx, known := r.luidToIndex[luid]
		if !known {
			return
		}

		val := item.FmtValue.doubleValue
		if values[adapterIdx] == nil {
			values[adapterIdx] = make(map[string]float64)
		}
		av := values[adapterIdx]

		// Overall utilization = max across all engine types for this adapter
		if val > av[MetricUtilization] {
			av[MetricUtilization] = val
		}

		// Per-engine-type metric
		engMatches := r.engtypeRegex.FindStringSubmatch(name)
		if len(engMatches) > 1 {
			normalized := normalizeEngineType(engMatches[1])
			if metric, ok := engineTypeMetrics[normalized]; ok {
				if val > av[metric] {
					av[metric] = val
				}
			}
		}
	})

	r.cache = collectionCache{values: values, timestamp: time.Now()}
	return nil
}

// GetMetric returns the current value for the specified metric and adapter.
func (r *pdhReader) GetMetric(adapter int, metric string) (float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return 0, fmt.Errorf("reader not initialized")
	}

	// Refresh cache if stale
	if time.Since(r.cache.timestamp) > cacheValidDuration {
		if err := r.collectAndCache(); err != nil {
			return 0, err
		}
	}

	av, ok := r.cache.values[adapter]
	if !ok {
		return 0, nil
	}

	return av[metric], nil
}

// ListAdapters returns information about available GPU adapters.
func (r *pdhReader) ListAdapters() ([]AdapterInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return nil, fmt.Errorf("reader not initialized")
	}

	return r.adapterCache, nil
}

// Close releases PDH resources.
func (r *pdhReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return
	}

	// PdhCloseQuery releases the query and all associated counter handles.
	ret, _, _ := pdhCloseQuery.Call(r.queryHandle)
	if ret != 0 {
		log.Printf("[GPU] PdhCloseQuery failed: 0x%x", ret)
	}

	r.initialized = false
}

// win32VideoController is a WMI result struct for GPU adapter names.
type win32VideoController struct {
	Name string
}

// queryAdapterNames returns GPU adapter names from WMI, ordered by device index.
// Returns nil if the query fails.
func queryAdapterNames() []string {
	var controllers []win32VideoController
	if err := wmi.Query("SELECT Name FROM Win32_VideoController", &controllers); err != nil {
		log.Printf("[GPU] WMI adapter query failed: %v", err)
		return nil
	}

	names := make([]string, len(controllers))
	for i, c := range controllers {
		names[i] = c.Name
	}
	return names
}
