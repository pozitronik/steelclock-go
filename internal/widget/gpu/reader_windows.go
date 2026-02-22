//go:build windows

package gpu

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/yusufpapurcu/wmi"
)

// ---------------------------------------------------------------------------
// PDH (Performance Data Helper) — metric collection
// ---------------------------------------------------------------------------

var (
	pdhDLL                       = syscall.NewLazyDLL("pdh.dll")
	pdhOpenQuery                 = pdhDLL.NewProc("PdhOpenQueryW")
	pdhCloseQuery                = pdhDLL.NewProc("PdhCloseQuery")
	pdhAddEnglishCounter         = pdhDLL.NewProc("PdhAddEnglishCounterW")
	pdhCollectQueryData          = pdhDLL.NewProc("PdhCollectQueryData")
	pdhGetFormattedCounterArrayW = pdhDLL.NewProc("PdhGetFormattedCounterArrayW")
)

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

// ---------------------------------------------------------------------------
// DXGI (DirectX Graphics Infrastructure) — adapter enumeration
// ---------------------------------------------------------------------------

var (
	dxgiDLL                = syscall.NewLazyDLL("dxgi.dll")
	procCreateDXGIFactory1 = dxgiDLL.NewProc("CreateDXGIFactory1")
)

// IID_IDXGIFactory1 = {770aae78-f26f-4dba-a829-253c83d1b387}
// Stored as a GUID struct in little-endian byte order.
var iidIDXGIFactory1 = [16]byte{
	0x78, 0xae, 0x0a, 0x77, // Data1: 0x770aae78
	0x6f, 0xf2, // Data2: 0xf26f
	0xba, 0x4d, // Data3: 0x4dba
	0xa8, 0x29, 0x25, 0x3c, 0x83, 0xd1, 0xb3, 0x87, // Data4
}

const dxgiAdapterFlagSoftware = 2

// dxgiLUID matches the Windows LUID struct layout: {LowPart uint32, HighPart int32}.
type dxgiLUID struct {
	LowPart  uint32
	HighPart int32
}

// dxgiAdapterDesc1 matches the DXGI_ADAPTER_DESC1 struct layout.
type dxgiAdapterDesc1 struct {
	Description           [128]uint16
	VendorId              uint32
	DeviceId              uint32
	SubSysId              uint32
	Revision              uint32
	DedicatedVideoMemory  uintptr
	DedicatedSystemMemory uintptr
	SharedSystemMemory    uintptr
	AdapterLuid           dxgiLUID
	Flags                 uint32
}

// dxgiAdapterEntry holds information about a hardware GPU adapter from DXGI.
type dxgiAdapterEntry struct {
	Name                 string
	LUID                 string // Formatted as "0xHHHHHHHH_0xHHHHHHHH" to match PDH instance names
	DedicatedVideoMemory uint64 // Total dedicated VRAM in bytes
	SharedSystemMemory   uint64 // Total shared system memory in bytes
}

// formatDXGILUID formats a DXGI LUID to match the PDH instance name format.
// PDH uses "luid_<HighPart>_<LowPart>" with zero-padded 8-digit hex values.
func formatDXGILUID(luid dxgiLUID) string {
	return fmt.Sprintf("0x%08x_0x%08x", uint32(luid.HighPart), luid.LowPart)
}

// comCall invokes a COM method at the given vtable index on the object.
// The object pointer is passed as the first argument (the implicit "this" in COM).
func comCall(obj uintptr, vtblIndex uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	method := *(*uintptr)(unsafe.Pointer(vtbl + vtblIndex*unsafe.Sizeof(uintptr(0))))
	allArgs := append([]uintptr{obj}, args...)
	ret, _, _ := syscall.SyscallN(method, allArgs...)
	return ret
}

// comRelease calls IUnknown::Release (vtable index 2) on a COM object.
func comRelease(obj uintptr) {
	comCall(obj, 2)
}

// queryDXGIAdapters enumerates GPU adapters using DXGI, returning only hardware
// adapters. Software adapters (Microsoft Basic Render Driver / WARP) are filtered
// out via DXGI_ADAPTER_FLAG_SOFTWARE.
//
// Returns nil if DXGI is unavailable or enumeration fails.
func queryDXGIAdapters() []dxgiAdapterEntry {
	var factory uintptr
	ret, _, _ := procCreateDXGIFactory1.Call(
		uintptr(unsafe.Pointer(&iidIDXGIFactory1)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if ret != 0 || factory == 0 {
		log.Printf("[GPU] DXGI: CreateDXGIFactory1 failed: 0x%x", ret)
		return nil
	}
	defer comRelease(factory)

	var adapters []dxgiAdapterEntry
	for i := uintptr(0); ; i++ {
		var adapter uintptr
		// IDXGIFactory1::EnumAdapters1 is at vtable index 12
		if comCall(factory, 12, i, uintptr(unsafe.Pointer(&adapter))) != 0 {
			break // DXGI_ERROR_NOT_FOUND or other error
		}

		var desc dxgiAdapterDesc1
		// IDXGIAdapter1::GetDesc1 is at vtable index 10
		ret := comCall(adapter, 10, uintptr(unsafe.Pointer(&desc)))
		comRelease(adapter)

		if ret != 0 {
			continue
		}

		// Skip software adapters (Microsoft Basic Render Driver, WARP)
		if desc.Flags&dxgiAdapterFlagSoftware != 0 {
			continue
		}

		name := syscall.UTF16ToString(desc.Description[:])
		luid := formatDXGILUID(desc.AdapterLuid)
		adapters = append(adapters, dxgiAdapterEntry{
			Name:                 name,
			LUID:                 luid,
			DedicatedVideoMemory: uint64(desc.DedicatedVideoMemory),
			SharedSystemMemory:   uint64(desc.SharedSystemMemory),
		})
	}

	return adapters
}

// ---------------------------------------------------------------------------
// WMI — fallback adapter name enumeration
// ---------------------------------------------------------------------------

// win32VideoController is a WMI result struct for GPU adapter names.
type win32VideoController struct {
	Name string
}

// queryAdapterNames returns GPU adapter names from WMI, ordered by device index.
// Used as fallback when DXGI enumeration is unavailable.
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

// ---------------------------------------------------------------------------
// pdhReader — main Reader implementation
// ---------------------------------------------------------------------------

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
// Adapter discovery uses DXGI as the primary source: it provides both the adapter
// name and LUID, and has a software adapter flag to filter out phantom adapters
// (e.g., Microsoft Basic Render Driver) that appear in PDH but are not real GPUs.
// Falls back to PDH LUID enumeration + WMI name enrichment if DXGI is unavailable.
type pdhReader struct {
	mu            sync.Mutex
	queryHandle   uintptr
	engineCounter uintptr // PDH counter handle for GPU Engine utilization

	// Memory counters (GPU Adapter Memory category)
	memDedicatedCounter  uintptr // PDH counter handle for dedicated memory usage
	memSharedCounter     uintptr // PDH counter handle for shared memory usage
	memCountersAvailable bool    // True if memory counters were successfully added

	adapterCache []AdapterInfo
	initialized  bool
	luidRegex    *regexp.Regexp // Extracts LUID from PDH instance name
	engtypeRegex *regexp.Regexp // Extracts engine type (including spaces) from PDH instance name
	luidToIndex  map[string]int // Maps LUID string -> sequential adapter index
	cache        collectionCache
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

	// Add engine utilization counter (GPU Engine category).
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

	// Add memory counters (GPU Adapter Memory category).
	// These are instantaneous gauges (not rate-based), so they work with the
	// dual-collection approach without issues. Failure is non-fatal — memory
	// metrics will simply report 0%.
	r.memCountersAvailable = true
	memDedicatedPath := `\GPU Adapter Memory(*)\Dedicated Usage`
	memDedicatedPtr, _ := syscall.UTF16PtrFromString(memDedicatedPath)
	var memDedicatedHandle uintptr
	ret, _, _ = pdhAddEnglishCounter.Call(
		r.queryHandle,
		uintptr(unsafe.Pointer(memDedicatedPtr)),
		0,
		uintptr(unsafe.Pointer(&memDedicatedHandle)),
	)
	if ret != 0 {
		log.Printf("[GPU] Memory counter 'Dedicated Usage' unavailable (0x%x), memory metrics disabled", ret)
		r.memCountersAvailable = false
	} else {
		r.memDedicatedCounter = memDedicatedHandle
	}

	memSharedPath := `\GPU Adapter Memory(*)\Shared Usage`
	memSharedPtr, _ := syscall.UTF16PtrFromString(memSharedPath)
	var memSharedHandle uintptr
	ret, _, _ = pdhAddEnglishCounter.Call(
		r.queryHandle,
		uintptr(unsafe.Pointer(memSharedPtr)),
		0,
		uintptr(unsafe.Pointer(&memSharedHandle)),
	)
	if ret != 0 {
		log.Printf("[GPU] Memory counter 'Shared Usage' unavailable (0x%x), memory metrics disabled", ret)
		r.memCountersAvailable = false
	} else {
		r.memSharedCounter = memSharedHandle
	}

	// Two data collections are required for rate-based counters like
	// "Utilization Percentage". The first establishes a baseline; the second
	// produces valid values and makes instance names available via
	// PdhGetFormattedCounterArrayW.
	pdhCollectQueryData.Call(r.queryHandle)
	time.Sleep(200 * time.Millisecond)
	pdhCollectQueryData.Call(r.queryHandle)

	// Discover adapters
	r.discoverAdapters()

	r.initialized = true
	return nil
}

// discoverAdapters builds the adapter list using two sources:
//
// Primary (DXGI): Enumerates hardware adapters with exact LUID-to-name mapping,
// filtering out software adapters (Microsoft Basic Render Driver) that appear
// in PDH counters but are not real GPUs. Only adapters that also have PDH
// counters are included.
//
// Fallback (PDH + WMI): If DXGI is unavailable, collects all LUIDs from PDH
// counter instances and enriches names from WMI by index. This may include
// phantom software adapters and may have mismatched names when the count of
// PDH LUIDs differs from the count of WMI entries.
func (r *pdhReader) discoverAdapters() {
	// Collect all unique LUIDs from PDH counter instances
	pdhLUIDs := make(map[string]bool)
	var pdhInstanceCount int
	r.forEachCounterInstance(r.engineCounter, func(name string, _ *pdhFmtCountervalueItemDouble) {
		pdhInstanceCount++
		if luid := r.extractLUID(name); luid != "" {
			pdhLUIDs[luid] = true
		}
	})
	log.Printf("[GPU] PDH: %d counter instances, %d unique LUIDs", pdhInstanceCount, len(pdhLUIDs))

	// Try DXGI for authoritative adapter enumeration
	if dxgiAdapters := queryDXGIAdapters(); len(dxgiAdapters) > 0 {
		r.buildFromDXGI(dxgiAdapters, pdhLUIDs)
	} else {
		r.buildFromPDH(pdhLUIDs)
	}

	log.Printf("[GPU] Discovered %d adapter(s):", len(r.adapterCache))
	for _, a := range r.adapterCache {
		log.Printf("[GPU]   %d: %s (LUID: %s)", a.Index, a.Name, a.LUID)
	}
}

// buildFromDXGI builds the adapter list from DXGI hardware adapters, keeping
// only those that also have PDH counters. DXGI provides exact LUID-to-name
// mapping and naturally filters software adapters.
func (r *pdhReader) buildFromDXGI(dxgiAdapters []dxgiAdapterEntry, pdhLUIDs map[string]bool) {
	var adapters []AdapterInfo
	luidToIdx := make(map[string]int)

	for _, da := range dxgiAdapters {
		if !pdhLUIDs[da.LUID] {
			log.Printf("[GPU] DXGI adapter %q (LUID: %s) has no PDH counters, skipping", da.Name, da.LUID)
			continue
		}
		idx := len(adapters)
		luidToIdx[da.LUID] = idx
		adapters = append(adapters, AdapterInfo{
			Index:                idx,
			Name:                 da.Name,
			LUID:                 da.LUID,
			DedicatedVideoMemory: da.DedicatedVideoMemory,
			SharedSystemMemory:   da.SharedSystemMemory,
		})
	}

	r.adapterCache = adapters
	r.luidToIndex = luidToIdx
}

// buildFromPDH builds the adapter list from PDH LUIDs with WMI name enrichment.
// Used as fallback when DXGI is unavailable. LUIDs are sorted lexicographically
// for deterministic ordering.
func (r *pdhReader) buildFromPDH(pdhLUIDs map[string]bool) {
	luids := make([]string, 0, len(pdhLUIDs))
	for luid := range pdhLUIDs {
		luids = append(luids, luid)
	}
	sort.Strings(luids)

	wmiNames := queryAdapterNames()

	r.luidToIndex = make(map[string]int, len(luids))
	r.adapterCache = make([]AdapterInfo, len(luids))
	for i, luid := range luids {
		r.luidToIndex[luid] = i
		name := fmt.Sprintf("GPU %d", i)
		if i < len(wmiNames) && wmiNames[i] != "" {
			name = wmiNames[i]
		}
		r.adapterCache[i] = AdapterInfo{Index: i, Name: name, LUID: luid}
	}
}

// extractLUID extracts the LUID string from a PDH instance name and lowercases it.
// PDH uses uppercase hex (0x000143DE), DXGI uses lowercase (0x000143de);
// lowercasing here ensures both sources produce matching strings.
//
// Example input: "pid_1234_luid_0x00000000_0x0001332D_phys_0_eng_0_engtype_3D"
// Returns: "0x00000000_0x0001332d"
func (r *pdhReader) extractLUID(instanceName string) string {
	matches := r.luidRegex.FindStringSubmatch(instanceName)
	if len(matches) > 1 {
		return strings.ToLower(matches[1])
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

	// Collect memory metrics if counters are available
	if r.memCountersAvailable {
		r.collectMemoryCounter(values, r.memDedicatedCounter, MetricMemoryDedicated, func(a AdapterInfo) uint64 {
			return a.DedicatedVideoMemory
		})
		r.collectMemoryCounter(values, r.memSharedCounter, MetricMemoryShared, func(a AdapterInfo) uint64 {
			return a.SharedSystemMemory
		})
	}

	r.cache = collectionCache{values: values, timestamp: time.Now()}
	return nil
}

// collectMemoryCounter iterates memory counter instances, extracts usage bytes,
// and computes percentage relative to the adapter's total memory.
//
// GPU Adapter Memory instance names use the format:
//
//	luid_0x<HH>_0x<HH>_phys_<N>
//
// (no pid/eng/engtype fields). LUID extraction reuses the existing regex.
// totalGetter returns the total memory for the adapter (e.g., DedicatedVideoMemory).
func (r *pdhReader) collectMemoryCounter(values map[int]map[string]float64, handle uintptr, metric string, totalGetter func(AdapterInfo) uint64) {
	if handle == 0 {
		return
	}

	r.forEachCounterInstance(handle, func(name string, item *pdhFmtCountervalueItemDouble) {
		if item.FmtValue.CStatus != pdhCstatValidData && item.FmtValue.CStatus != 0 {
			return
		}

		luid := r.extractLUID(name)
		if luid == "" {
			return
		}
		adapterIdx, known := r.luidToIndex[luid]
		if !known {
			return
		}

		usageBytes := item.FmtValue.doubleValue
		if adapterIdx >= len(r.adapterCache) {
			return
		}
		totalBytes := totalGetter(r.adapterCache[adapterIdx])
		if totalBytes == 0 {
			return // Avoid division by zero; no total memory info available
		}

		pct := usageBytes / float64(totalBytes) * 100
		if values[adapterIdx] == nil {
			values[adapterIdx] = make(map[string]float64)
		}
		values[adapterIdx][metric] = pct
	})
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
