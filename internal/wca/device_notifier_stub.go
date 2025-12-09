//go:build !windows

package wca

import "fmt"

// DeviceNotifier stub for non-Windows platforms
type DeviceNotifier struct{}

// GetDeviceNotifier returns an error on non-Windows platforms
// Device change notifications are only supported on Windows via IMMNotificationClient
func GetDeviceNotifier() (*DeviceNotifier, error) {
	return nil, fmt.Errorf("device notifications are not supported on this platform")
}

// Subscribe is a no-op on non-Windows platforms
func (dn *DeviceNotifier) Subscribe() <-chan struct{} {
	return nil
}

// Unsubscribe is a no-op on non-Windows platforms
func (dn *DeviceNotifier) Unsubscribe(_ <-chan struct{}) {}

// Stop is a no-op on non-Windows platforms
func (dn *DeviceNotifier) Stop() {}
