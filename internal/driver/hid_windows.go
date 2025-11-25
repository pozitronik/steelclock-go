//go:build windows

package driver

import (
	"fmt"
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

	procHidDSetFeature = modHid.NewProc("HidD_SetFeature")
)

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
	defer procSetupDiDestroyDeviceInfoList.Call(hDevInfo)

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
		procSetupDiGetDeviceInterfaceDetailW.Call(
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

// autoDetectDevice tries to find any known SteelSeries device
func autoDetectDevice(targetInterface string) (string, error) {
	for _, device := range KnownDevices {
		path, err := findDevicePath(device.VID, device.PID, targetInterface)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no known SteelSeries device found")
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

// EnumerateDevices returns a list of all connected SteelSeries HID devices
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
	defer procSetupDiDestroyDeviceInfoList.Call(hDevInfo)

	var devices []DeviceInfo
	var ifaceData spDeviceInterfaceData
	if unsafe.Sizeof(uintptr(0)) == 8 {
		ifaceData.cbSize = 32
	} else {
		ifaceData.cbSize = 28
	}

	// Look for SteelSeries VID
	steelSeriesSubstr := fmt.Sprintf("vid_%04x", SteelSeriesVID)

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
		procSetupDiGetDeviceInterfaceDetailW.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&ifaceData)),
			uintptr(unsafe.Pointer(&detailData)),
			unsafe.Sizeof(detailData),
			uintptr(unsafe.Pointer(&reqSize)),
			0,
		)

		path := syscall.UTF16ToString(detailData.DevicePath[:])
		lPath := strings.ToLower(path)

		// Check if it's a SteelSeries device
		if !strings.Contains(lPath, steelSeriesSubstr) {
			continue
		}

		// Skip system aliases
		if strings.Contains(lPath, "kbd") || strings.Contains(lPath, "col02") {
			continue
		}

		// Extract VID/PID from path
		var vid, pid uint16
		if vidIdx := strings.Index(lPath, "vid_"); vidIdx >= 0 {
			fmt.Sscanf(lPath[vidIdx:], "vid_%04x&pid_%04x", &vid, &pid)
		}

		// Extract interface
		iface := ""
		if idx := strings.Index(lPath, "mi_"); idx >= 0 {
			end := idx + 5 // "mi_XX"
			if end <= len(lPath) {
				iface = lPath[idx:end]
			}
		}

		info := DeviceInfo{
			VID:       vid,
			PID:       pid,
			Path:      path,
			Interface: iface,
		}

		// Try to find product name from known devices
		if known := FindKnownDevice(vid, pid); known != nil {
			info.ProductName = known.Name
			info.Manufacturer = "SteelSeries"
		}

		devices = append(devices, info)
	}

	return devices, nil
}
