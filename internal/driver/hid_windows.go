//go:build windows

package driver

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// DeviceHandle is a Windows HANDLE type
type DeviceHandle syscall.Handle

// InvalidHandle represents an invalid device handle
const InvalidHandle = DeviceHandle(syscall.InvalidHandle)

// Windows API DLLs
var (
	modSetupApi = syscall.NewLazyDLL("setupapi.dll")
	modHid      = syscall.NewLazyDLL("hid.dll")

	procSetupDiGetClassDevsW             = modSetupApi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = modSetupApi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = modSetupApi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = modSetupApi.NewProc("SetupDiDestroyDeviceInfoList")

	procHidDSetFeature        = modHid.NewProc("HidD_SetFeature")
	procHidDGetPreparsedData  = modHid.NewProc("HidD_GetPreparsedData")
	procHidDFreePreparsedData = modHid.NewProc("HidD_FreePreparsedData")
	procHidPGetCaps           = modHid.NewProc("HidP_GetCaps")
)

// hidpStatusSuccess is HIDP_STATUS_SUCCESS returned by HidP_GetCaps.
const hidpStatusSuccess = 0x00110000

// hidpCaps mirrors the Windows HIDP_CAPS structure (all fields USHORT).
// Only the leading fields are used; the trailing counts are kept for correct sizing.
type hidpCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

// readHidCaps reads the HID capabilities of an open device handle. The handle
// only needs query access (AccessRights = 0 is sufficient for HidD_GetPreparsedData).
func readHidCaps(handle DeviceHandle) (hidpCaps, error) {
	var caps hidpCaps
	var preparsed uintptr
	r, _, _ := procHidDGetPreparsedData.Call(uintptr(handle), uintptr(unsafe.Pointer(&preparsed)))
	if r == 0 || preparsed == 0 {
		return caps, fmt.Errorf("HidD_GetPreparsedData failed")
	}
	defer func() { _, _, _ = procHidDFreePreparsedData.Call(preparsed) }()

	st, _, _ := procHidPGetCaps.Call(preparsed, uintptr(unsafe.Pointer(&caps)))
	if st != hidpStatusSuccess {
		return caps, fmt.Errorf("HidP_GetCaps failed (status 0x%X)", st)
	}
	return caps, nil
}

// Windows constants
const (
	digcfPresent         = 0x00000002
	digcfDeviceInterface = 0x00000010

	fileShareRead  = 0x00000001
	fileShareWrite = 0x00000002
	openExisting   = 3
)

// GUID structure for Windows API
type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// HID class GUID
var hidGUID = guid{0x4d1e55b2, 0xf16f, 0x11cf, [8]byte{0x88, 0xcb, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30}}

// spDeviceInterfaceData structure
type spDeviceInterfaceData struct {
	cbSize             uint32
	InterfaceClassGuid guid
	Flags              uint32
	Reserved           uintptr
}

// spDeviceInterfaceDetailData structure
type spDeviceInterfaceDetailData struct {
	cbSize     uint32
	DevicePath [512]uint16
}

// findDevicePath finds a HID device by VID, PID, and interface
func findDevicePath(vid, pid uint16, targetInterface string) (string, error) {
	hDevInfo, _, _ := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&hidGUID)),
		0,
		0,
		digcfPresent|digcfDeviceInterface,
	)
	if hDevInfo == 0 || hDevInfo == ^uintptr(0) {
		return "", fmt.Errorf("SetupDiGetClassDevsW failed")
	}
	defer func() { _, _, _ = procSetupDiDestroyDeviceInfoList.Call(hDevInfo) }()

	var ifaceData spDeviceInterfaceData
	// Set cbSize based on architecture
	if unsafe.Sizeof(uintptr(0)) == 8 {
		ifaceData.cbSize = 32
	} else {
		ifaceData.cbSize = 28
	}

	targetSubstr := fmt.Sprintf("vid_%04x&pid_%04x", vid, pid)
	targetInterface = strings.ToLower(targetInterface)

	for i := 0; ; i++ {
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			hDevInfo,
			0,
			uintptr(unsafe.Pointer(&hidGUID)),
			uintptr(i),
			uintptr(unsafe.Pointer(&ifaceData)),
		)
		if r == 0 {
			break
		}

		var detailData spDeviceInterfaceDetailData
		if unsafe.Sizeof(uintptr(0)) == 8 {
			detailData.cbSize = 8
		} else {
			detailData.cbSize = 5
		}

		var reqSize uint32
		_, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			uintptr(unsafe.Pointer(&detailData)),
			unsafe.Sizeof(detailData),
			uintptr(unsafe.Pointer(&reqSize)),
			0,
		)

		path := syscall.UTF16ToString(detailData.DevicePath[:])
		lPath := strings.ToLower(path)

		// Check if path matches VID/PID and interface
		if strings.Contains(lPath, targetSubstr) && strings.Contains(lPath, targetInterface) {
			// Skip system aliases (keyboard, col02)
			if strings.Contains(lPath, "kbd") || strings.Contains(lPath, "col02") {
				continue
			}
			return path, nil
		}
	}

	return "", fmt.Errorf("device VID_%04X PID_%04X interface %s not found", vid, pid, targetInterface)
}

// openDevice opens a HID device by path
func openDevice(path string) (DeviceHandle, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return InvalidHandle, fmt.Errorf("invalid path: %w", err)
	}

	// Open with AccessRights = 0 to bypass Windows HID driver blocking
	handle, err := syscall.CreateFile(
		pathPtr,
		0, // AccessRights = 0 (important!)
		fileShareRead|fileShareWrite,
		nil,
		openExisting,
		0,
		0,
	)
	if err != nil {
		return InvalidHandle, fmt.Errorf("CreateFile failed: %w", err)
	}

	return DeviceHandle(handle), nil
}

// closeDevice closes a HID device handle
func closeDevice(handle DeviceHandle) error {
	if handle == InvalidHandle {
		return nil
	}
	return syscall.CloseHandle(syscall.Handle(handle))
}

// sendFeatureReport sends a feature report to the HID device
func sendFeatureReport(handle DeviceHandle, data []byte) error {
	if handle == InvalidHandle {
		return fmt.Errorf("invalid handle")
	}

	r, _, err := procHidDSetFeature.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
	)

	if r == 0 {
		return fmt.Errorf("HidD_SetFeature failed: %w", err)
	}

	return nil
}

// parseHidPath extracts the VID, PID and interface (mi_xx) from a Windows HID
// device interface path such as
// `\\?\hid#vid_1038&pid_12cd&mi_04#...`. Fields that are absent yield zero / "".
func parseHidPath(path string) (vid, pid uint16, iface string) {
	lp := strings.ToLower(path)
	if v, ok := parseHex16After(lp, "vid_"); ok {
		vid = v
	}
	if p, ok := parseHex16After(lp, "pid_"); ok {
		pid = p
	}
	if idx := strings.Index(lp, "&mi_"); idx != -1 {
		start := idx + len("&mi_")
		if start+2 <= len(lp) {
			iface = "mi_" + lp[start:start+2]
		}
	}
	return vid, pid, iface
}

// parseHex16After returns the 16-bit hex value of the 4 characters that follow
// marker in s (e.g. marker "vid_" in "...vid_1038...").
func parseHex16After(s, marker string) (uint16, bool) {
	idx := strings.Index(s, marker)
	if idx == -1 {
		return 0, false
	}
	start := idx + len(marker)
	if start+4 > len(s) {
		return 0, false
	}
	v, err := strconv.ParseUint(s[start:start+4], 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(v), true
}

// EnumerateDevices lists all present HID interfaces, parsing VID/PID/interface
// from each device path. SteelSeries (VID 0x1038) interfaces are additionally
// opened read-only to read their HID capabilities (report transport/sizes);
// other devices are never opened. The read-only open coexists with SteelSeries
// GG (or any other app) holding them. Intended for diagnostics.
func EnumerateDevices() ([]DeviceInfo, error) {
	hDevInfo, _, _ := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&hidGUID)),
		0,
		0,
		digcfPresent|digcfDeviceInterface,
	)
	if hDevInfo == 0 || hDevInfo == ^uintptr(0) {
		return nil, fmt.Errorf("SetupDiGetClassDevsW failed")
	}
	defer func() { _, _, _ = procSetupDiDestroyDeviceInfoList.Call(hDevInfo) }()

	var ifaceData spDeviceInterfaceData
	if unsafe.Sizeof(uintptr(0)) == 8 {
		ifaceData.cbSize = 32
	} else {
		ifaceData.cbSize = 28
	}

	var result []DeviceInfo
	for i := 0; ; i++ {
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			hDevInfo,
			0,
			uintptr(unsafe.Pointer(&hidGUID)),
			uintptr(i),
			uintptr(unsafe.Pointer(&ifaceData)),
		)
		if r == 0 {
			break
		}

		var detailData spDeviceInterfaceDetailData
		if unsafe.Sizeof(uintptr(0)) == 8 {
			detailData.cbSize = 8
		} else {
			detailData.cbSize = 5
		}

		var reqSize uint32
		_, _, _ = procSetupDiGetDeviceInterfaceDetailW.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			uintptr(unsafe.Pointer(&detailData)),
			unsafe.Sizeof(detailData),
			uintptr(unsafe.Pointer(&reqSize)),
			0,
		)

		path := syscall.UTF16ToString(detailData.DevicePath[:])
		if path == "" {
			continue
		}

		vid, pid, iface := parseHidPath(path)
		info := DeviceInfo{
			VID:       vid,
			PID:       pid,
			Path:      path,
			Interface: iface,
		}

		// Read HID capabilities for SteelSeries interfaces only, so we learn the
		// real report transport/sizes (e.g. whether the OLED interface supports
		// feature vs output reports) without touching unrelated devices. The
		// read-only open coexists with SteelSeries GG.
		if vid == SteelSeriesVID {
			if handle, oerr := openDevice(path); oerr == nil {
				if caps, cerr := readHidCaps(handle); cerr == nil {
					info.HasCaps = true
					info.UsagePage = caps.UsagePage
					info.Usage = caps.Usage
					info.InputReportLen = int(caps.InputReportByteLength)
					info.OutputReportLen = int(caps.OutputReportByteLength)
					info.FeatureReportLen = int(caps.FeatureReportByteLength)
				}
				_ = closeDevice(handle)
			}
		}

		result = append(result, info)
	}

	return result, nil
}
