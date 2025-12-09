//go:build !windows

package wca

import "fmt"

// EnsureCOMInitialized is a no-op on non-Windows platforms.
// COM is a Windows-specific technology.
func EnsureCOMInitialized() error {
	return fmt.Errorf("COM is not supported on this platform")
}

// CreateDeviceEnumerator is not available on non-Windows platforms.
func CreateDeviceEnumerator() (interface{}, error) {
	return nil, fmt.Errorf("device enumerator is not supported on this platform")
}

// GetDefaultRenderDevice is not available on non-Windows platforms.
func GetDefaultRenderDevice(mmde interface{}) (interface{}, error) {
	return nil, fmt.Errorf("audio devices are not supported on this platform")
}

// SafeReleaseAudioEndpointVolume is a no-op on non-Windows platforms.
func SafeReleaseAudioEndpointVolume(ptr interface{}) {}

// SafeReleaseAudioMeterInformation is a no-op on non-Windows platforms.
func SafeReleaseAudioMeterInformation(ptr interface{}) {}

// SafeReleaseMMDevice is a no-op on non-Windows platforms.
func SafeReleaseMMDevice(ptr interface{}) {}

// SafeReleaseMMDeviceEnumerator is a no-op on non-Windows platforms.
func SafeReleaseMMDeviceEnumerator(ptr interface{}) {}

// SafeReleaseAudioClient is a no-op on non-Windows platforms.
func SafeReleaseAudioClient(ptr interface{}) {}

// SafeReleaseAudioCaptureClient is a no-op on non-Windows platforms.
func SafeReleaseAudioCaptureClient(ptr interface{}) {}
