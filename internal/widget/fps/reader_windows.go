//go:build windows

package fps

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// errRTSSGone reports that the mapped segment is no longer a valid RTSS
// segment (RTSS exited or replaced it); the caller should reopen it.
var errRTSSGone = errors.New("RTSS shared memory is no longer valid")

var (
	modKernel32          = syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMappingW = modKernel32.NewProc("OpenFileMappingW")
	procMapViewOfFile    = modKernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile  = modKernel32.NewProc("UnmapViewOfFile")
	procGetTickCount     = modKernel32.NewProc("GetTickCount")

	modUser32                    = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow      = modUser32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
)

const (
	fileMapRead      = 0x0004
	rtssMappingName  = "RTSSSharedMemoryV2"
	rtssMaxProcesses = 256 // matches RTSS_SHARED_MEMORY::arrApp[256]

	rtssSignature    = 0x52545353 // 'RTSS' while the segment is valid
	rtssMinVersion   = 0x00020000 // v2.0: first layout with dwAppArrOffset etc.
	rtssFgPIDVersion = 0x00020010 // v2.16: adds dwLastForegroundAppProcessID
	rtssStaleAfterMs = 2000       // minimum age before a measurement window is stale

	// Header field byte offsets. Stable since RTSS shared memory v2.0 — the
	// header itself carries dwAppArrOffset/dwAppEntrySize/dwAppArrSize so app
	// entries never need to be located by a hardcoded struct size.
	// See RTSSSharedMemory.h (RTSS SDK) for the authoritative layout.
	offSignature                  = 0
	offVersion                    = 4
	offAppEntrySize               = 8
	offAppArrOffset               = 12
	offAppArrSize                 = 16
	offLastForegroundAppProcessID = 68 // valid for shared memory v2.16+

	// rtssHeaderSize covers every header field read above.
	rtssHeaderSize = offLastForegroundAppProcessID + 4

	// RTSS_SHARED_MEMORY_APP_ENTRY field byte offsets, relative to the
	// entry's own base. These are the struct's leading fields, stable since
	// v2.0 regardless of how much trailing data newer RTSS versions add.
	entryOffProcessID = 0
	entryOffName      = 4
	entryOffNameLen   = 260 // MAX_PATH
	entryOffTime0     = 268 // start of the once-per-second measurement period (ms, GetTickCount-style)
	entryOffTime1     = 272 // end of that measurement period (ms)
	entryOffFrames    = 276 // frames rendered during (Time1 - Time0)
	entryOffFrameTime = 280 // most recent single-frame time, in microseconds
)

// getForegroundProcessID returns the process ID owning the current OS
// foreground window. This is more reliable than RTSS's own
// dwLastForegroundAppProcessID field, which can be stale right after RTSS
// (re)starts — e.g. after an update — until the next focus change.
func getForegroundProcessID() uint32 {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

// rtssReader reads current framerate from RTSS's (RivaTuner Statistics
// Server) shared memory segment. RTSS is the framerate-capture engine behind
// MSI Afterburner and is widely used by other on-screen-display tools as a
// stable, already-hooked source of per-process FPS without implementing our
// own DirectX/OpenGL/Vulkan present hooks.
type rtssReader struct {
	// base is the start of the mapped view. It is kept as an unsafe.Pointer
	// (not a uintptr) so field addresses are derived with unsafe.Add, which
	// keeps pointer provenance intact for the checkptr instrumentation that
	// -race enables.
	base unsafe.Pointer
	// size is the readable length of the view in bytes. Header-derived
	// offsets are checked against it, since reading past the view is an
	// access violation, which is fatal rather than a recoverable panic.
	size uintptr
	mu   sync.Mutex
}

// newRTSSReader opens and maps the RTSS shared memory segment. Fails if RTSS
// isn't running — the caller should retry periodically, since RTSS may start
// later (e.g. when a game launches) or may not be needed until then.
func newRTSSReader() (*rtssReader, error) {
	namePtr, err := syscall.UTF16PtrFromString(rtssMappingName)
	if err != nil {
		return nil, err
	}
	h, _, _ := procOpenFileMappingW.Call(fileMapRead, 0, uintptr(unsafe.Pointer(namePtr)))
	if h == 0 {
		return nil, fmt.Errorf("RTSS shared memory not found (is RTSS/MSI Afterburner running?)")
	}
	handle := syscall.Handle(h)
	defer func() { _ = syscall.CloseHandle(handle) }() // MapViewOfFile keeps its own reference

	addr, _, _ := procMapViewOfFile.Call(h, fileMapRead, 0, 0, 0)
	if addr == 0 {
		return nil, fmt.Errorf("failed to map RTSS shared memory")
	}
	var info windows.MemoryBasicInformation
	if err := windows.VirtualQuery(addr, &info, unsafe.Sizeof(info)); err != nil {
		_, _, _ = procUnmapViewOfFile.Call(addr)
		return nil, fmt.Errorf("failed to query RTSS shared memory size: %w", err)
	}
	// The view is mapped outside the Go heap, so converting its address is safe.
	return &rtssReader{base: unsafe.Pointer(addr), size: info.RegionSize}, nil
}

func readU32(base unsafe.Pointer, off uintptr) uint32 {
	return *(*uint32)(unsafe.Add(base, off))
}

func readCString(base unsafe.Pointer, maxLen int) string {
	b := make([]byte, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		c := *(*byte)(unsafe.Add(base, i))
		if c == 0 {
			break
		}
		b = append(b, c)
	}
	return string(b)
}

// GetFPS returns the current framerate and process name of the active
// foreground game. Returns (0, "", nil) when RTSS is running but nothing
// relevant is currently hooked and rendering — not an error condition.
func (r *rtssReader) GetFPS() (fps float64, processName string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.base == nil || r.size < rtssHeaderSize {
		return 0, "", errRTSSGone
	}
	// Per the RTSS SDK, only a segment carrying the 'RTSS' signature and a
	// v2.0+ layout may be read; anything else must be reopened.
	version := readU32(r.base, offVersion)
	if readU32(r.base, offSignature) != rtssSignature || version < rtssMinVersion {
		return 0, "", errRTSSGone
	}

	appEntrySize := uintptr(readU32(r.base, offAppEntrySize))
	appArrOffset := uintptr(readU32(r.base, offAppArrOffset))
	appArrSize := readU32(r.base, offAppArrSize)
	if appArrSize > rtssMaxProcesses {
		appArrSize = rtssMaxProcesses
	}
	var rtssFgPID uint32
	if version >= rtssFgPIDVersion {
		rtssFgPID = readU32(r.base, offLastForegroundAppProcessID)
	}
	osFgPID := getForegroundProcessID()
	tick, _, _ := procGetTickCount.Call()
	now := uint32(tick)

	// Every entry must at least hold the fields read below.
	if appEntrySize < entryOffFrameTime+4 || appArrSize == 0 {
		return 0, "", nil
	}
	// The entries read below must lie inside the view; a header claiming
	// otherwise is corrupt or truncated, so reopen rather than read past it.
	if uint64(appArrOffset)+uint64(appArrSize)*uint64(appEntrySize) > uint64(r.size) {
		return 0, "", errRTSSGone
	}

	var (
		osMatchFPS, rtssMatchFPS, latestFPS       float64
		osMatchBase, rtssMatchBase, latestBase    unsafe.Pointer
		osMatchFound, rtssMatchFound, latestFound bool
		latestTime1                               uint32
	)

	for i := uint32(0); i < appArrSize; i++ {
		entryBase := unsafe.Add(r.base, appArrOffset+uintptr(i)*appEntrySize)
		pid := readU32(entryBase, entryOffProcessID)
		if pid == 0 {
			continue
		}
		// dwFrameTime (a single frame's time) is only used here as an "is
		// this entry actively rendering right now" signal — it's noisy
		// frame-to-frame. The displayed value uses the smoothed once-per-
		// second dwFrames/(Time1-Time0) window RTSS itself documents for
		// this purpose, matching what RTSS's own OSD and similar overlay
		// tools (Steam, etc.) show rather than a single-frame spike.
		if readU32(entryBase, entryOffFrameTime) == 0 {
			continue
		}
		time0 := readU32(entryBase, entryOffTime0)
		time1 := readU32(entryBase, entryOffTime1)
		frames := readU32(entryBase, entryOffFrames)
		if time0 == 0 || time1 <= time0 {
			continue
		}
		// RTSS closes a measurement window about once per averaging interval
		// while the app presents frames; a window that stopped advancing means
		// the app is minimized, paused or hung and its FPS is stale.
		if now-time1 > max(rtssStaleAfterMs, 2*(time1-time0)) {
			continue
		}
		entryFPS := 1000.0 * float64(frames) / float64(time1-time0)

		if osFgPID != 0 && pid == osFgPID {
			osMatchFPS, osMatchBase, osMatchFound = entryFPS, entryBase, true
		}
		if rtssFgPID != 0 && pid == rtssFgPID {
			rtssMatchFPS, rtssMatchBase, rtssMatchFound = entryFPS, entryBase, true
		}
		if !latestFound || time1 > latestTime1 {
			latestFPS, latestBase, latestTime1, latestFound = entryFPS, entryBase, time1, true
		}
	}

	// Prefer matching the OS's actual foreground window — more reliable than
	// RTSS's own tracking, which can be stale right after RTSS (re)starts
	// (e.g. after an update) until the next focus change. Fall back to RTSS's
	// own foreground field, then to whichever hooked app rendered most
	// recently, rather than an arbitrary array-order pick. The process name is
	// decoded only for the entry actually selected, not every candidate.
	switch {
	case osMatchFound:
		return osMatchFPS, readCString(unsafe.Add(osMatchBase, entryOffName), entryOffNameLen), nil
	case rtssMatchFound:
		return rtssMatchFPS, readCString(unsafe.Add(rtssMatchBase, entryOffName), entryOffNameLen), nil
	case latestFound:
		return latestFPS, readCString(unsafe.Add(latestBase, entryOffName), entryOffNameLen), nil
	default:
		return 0, "", nil
	}
}

// Close unmaps the shared memory segment.
func (r *rtssReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.base != nil {
		_, _, _ = procUnmapViewOfFile.Call(uintptr(r.base))
		r.base = nil
		r.size = 0
	}
}

// newReader opens the RTSS shared memory segment as a Reader.
func newReader() (Reader, error) {
	return newRTSSReader()
}
