//go:build !windows && !linux

package driver

import "fmt"

// DeviceHandle is a placeholder for Unix systems
type DeviceHandle int

// InvalidHandle represents an invalid device handle
const InvalidHandle DeviceHandle = -1

// ErrNotSupported is returned when the direct driver is not available on this platform
var ErrNotSupported = fmt.Errorf("direct USB HID driver is not supported on this platform")

// findDevicePath is not supported on Unix
func findDevicePath(vid, pid uint16, targetInterface string) (string, error) {
	return "", ErrNotSupported
}

// autoDetectDevice is not supported on Unix
func autoDetectDevice(targetInterface string) (string, error) {
	return "", ErrNotSupported
}

// openDevice is not supported on Unix
func openDevice(path string) (DeviceHandle, error) {
	return InvalidHandle, ErrNotSupported
}

// closeDevice is not supported on Unix
func closeDevice(handle DeviceHandle) error {
	return nil
}

// sendFeatureReport is not supported on Unix
func sendFeatureReport(handle DeviceHandle, data []byte) error {
	return ErrNotSupported
}

// EnumerateDevices is not supported on Unix
func EnumerateDevices() ([]DeviceInfo, error) {
	return nil, ErrNotSupported
}
