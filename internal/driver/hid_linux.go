//go:build linux

package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// DeviceHandle is a Linux file descriptor
type DeviceHandle int

// InvalidHandle represents an invalid device handle
const InvalidHandle DeviceHandle = -1

// IOCTL constants for HID
// HIDIOCSFEATURE(len) = _IOC(_IOC_WRITE|_IOC_READ, 'H', 0x06, len)
// = ((3 << 30) | (len << 16) | ('H' << 8) | 0x06)
const (
	iocWrite    = 1
	iocRead     = 2
	hidIOCType  = 'H'
	hidIOCSFeat = 0x06
)

// hidiocsfeature calculates the ioctl request number for HIDIOCSFEATURE
func hidiocsfeature(length int) uintptr {
	dir := uintptr(iocWrite | iocRead)
	return (dir << 30) | (uintptr(length) << 16) | (uintptr(hidIOCType) << 8) | hidIOCSFeat
}

// sysHidrawPath is the sysfs path for hidraw devices
const sysHidrawPath = "/sys/class/hidraw"

// devPath is the device path prefix
const devPath = "/dev"

// hidrawDevice contains parsed information about a hidraw device
type hidrawDevice struct {
	name       string // e.g., "hidraw0"
	path       string // e.g., "/dev/hidraw0"
	vid        uint16
	pid        uint16
	hidName    string // Device name from HID_NAME
	interface_ string // Interface identifier from path
}

// parseUevent parses the uevent file to extract VID and PID
// Format of HID_ID: bus_type:vendor_id:product_id (all hex, 8 chars each)
func parseUevent(ueventPath string) (vid, pid uint16, hidName string, err error) {
	data, err := os.ReadFile(ueventPath)
	if err != nil {
		return 0, 0, "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "HID_ID=") {
			// Format: HID_ID=0003:00001038:00001612
			parts := strings.Split(strings.TrimPrefix(line, "HID_ID="), ":")
			if len(parts) >= 3 {
				v, err1 := strconv.ParseUint(parts[1], 16, 16)
				p, err2 := strconv.ParseUint(parts[2], 16, 16)
				if err1 == nil && err2 == nil {
					vid = uint16(v)
					pid = uint16(p)
				}
			}
		} else if strings.HasPrefix(line, "HID_NAME=") {
			hidName = strings.TrimPrefix(line, "HID_NAME=")
		}
	}

	if vid == 0 && pid == 0 {
		return 0, 0, "", fmt.Errorf("could not parse VID/PID from uevent")
	}

	return vid, pid, hidName, nil
}

// getInterfaceFromPath tries to extract interface identifier from device path
// Looks for patterns like "input0", "input1" in the sysfs path
func getInterfaceFromPath(sysPath string) string {
	// Read the device path symlink to get full path
	devicePath, err := filepath.EvalSymlinks(filepath.Join(sysPath, "device"))
	if err != nil {
		return ""
	}

	// Look for inputN pattern in path
	parts := strings.Split(devicePath, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "input") {
			// Convert input0 -> mi_00, input1 -> mi_01, etc.
			numStr := strings.TrimPrefix(part, "input")
			if num, err := strconv.Atoi(numStr); err == nil {
				return fmt.Sprintf("mi_%02d", num)
			}
		}
	}

	return ""
}

// enumerateHidrawDevices lists all hidraw devices with their VID/PID
func enumerateHidrawDevices() ([]hidrawDevice, error) {
	entries, err := os.ReadDir(sysHidrawPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w (is hidraw module loaded?)", sysHidrawPath, err)
	}

	var devices []hidrawDevice
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "hidraw") {
			continue
		}

		sysDevPath := filepath.Join(sysHidrawPath, entry.Name())
		ueventPath := filepath.Join(sysDevPath, "device", "uevent")

		vid, pid, hidName, err := parseUevent(ueventPath)
		if err != nil {
			continue // Skip devices we can't parse
		}

		iface := getInterfaceFromPath(sysDevPath)

		devices = append(devices, hidrawDevice{
			name:       entry.Name(),
			path:       filepath.Join(devPath, entry.Name()),
			vid:        vid,
			pid:        pid,
			hidName:    hidName,
			interface_: iface,
		})
	}

	return devices, nil
}

// findDevicePath finds a HID device by VID, PID, and interface
func findDevicePath(vid, pid uint16, targetInterface string) (string, error) {
	devices, err := enumerateHidrawDevices()
	if err != nil {
		return "", err
	}

	targetInterface = strings.ToLower(targetInterface)

	for _, dev := range devices {
		if dev.vid == vid && dev.pid == pid {
			// Check interface if specified
			if targetInterface != "" && dev.interface_ != "" {
				if strings.ToLower(dev.interface_) != targetInterface {
					continue
				}
			}
			return dev.path, nil
		}
	}

	return "", fmt.Errorf("device VID_%04X PID_%04X interface %s not found", vid, pid, targetInterface)
}

// autoDetectDevice tries to find any known SteelSeries device
func autoDetectDevice(targetInterface string) (string, error) {
	devices, err := enumerateHidrawDevices()
	if err != nil {
		return "", err
	}

	targetInterface = strings.ToLower(targetInterface)

	// Try each known device
	for _, known := range KnownDevices {
		for _, dev := range devices {
			if dev.vid == known.VID && dev.pid == known.PID {
				// Check interface if specified
				if targetInterface != "" && dev.interface_ != "" {
					if strings.ToLower(dev.interface_) != targetInterface {
						continue
					}
				}
				return dev.path, nil
			}
		}
	}

	return "", fmt.Errorf("no known SteelSeries device found")
}

// openDevice opens a HID device by path
func openDevice(path string) (DeviceHandle, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return InvalidHandle, fmt.Errorf("permission denied opening %s (try adding udev rule or run as root): %w", path, err)
		}
		return InvalidHandle, fmt.Errorf("failed to open %s: %w", path, err)
	}
	return DeviceHandle(fd), nil
}

// closeDevice closes a HID device handle
func closeDevice(handle DeviceHandle) error {
	if handle == InvalidHandle {
		return nil
	}
	return syscall.Close(int(handle))
}

// sendFeatureReport sends a feature report to the HID device
func sendFeatureReport(handle DeviceHandle, data []byte) error {
	if handle == InvalidHandle {
		return fmt.Errorf("invalid handle")
	}

	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	// Calculate ioctl request for this data length
	req := hidiocsfeature(len(data))

	// Perform ioctl
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(handle),
		req,
		uintptr(unsafe.Pointer(&data[0])),
	)

	if errno != 0 {
		return fmt.Errorf("HIDIOCSFEATURE ioctl failed: %v", errno)
	}

	return nil
}

// EnumerateDevices returns a list of all connected HID devices
func EnumerateDevices() ([]DeviceInfo, error) {
	devices, err := enumerateHidrawDevices()
	if err != nil {
		return nil, err
	}

	var result []DeviceInfo
	for _, dev := range devices {
		result = append(result, DeviceInfo{
			VID:         dev.vid,
			PID:         dev.pid,
			Path:        dev.path,
			ProductName: dev.hidName,
			Interface:   dev.interface_,
		})
	}

	return result, nil
}
