//go:build windows

package wca

import (
	"strings"
	"testing"

	"github.com/moutend/go-wca/pkg/wca"
)

// skipIfNoAudioDevice skips the test if no audio device is available
func skipIfNoAudioDevice(t *testing.T) {
	t.Helper()

	err := EnsureCOMInitialized()
	if err != nil {
		t.Skipf("Cannot initialize COM: %v", err)
	}

	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Skipf("Cannot create device enumerator: %v", err)
	}
	defer SafeReleaseMMDeviceEnumerator(&mmde)

	_, err = GetDefaultRenderDevice(mmde)
	if err != nil {
		if strings.Contains(err.Error(), "Element not found") {
			t.Skip("No audio device available (common in CI environments)")
		}
		t.Skipf("Cannot get default audio device: %v", err)
	}
}

// TestEnsureCOMInitialized_Idempotency verifies that EnsureCOMInitialized can be called multiple times safely
func TestEnsureCOMInitialized_Idempotency(t *testing.T) {
	// First call should initialize COM
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("First EnsureCOMInitialized failed: %v", err)
	}

	// Second call should be safe (no-op)
	err = EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("Second EnsureCOMInitialized failed: %v", err)
	}

	// Third call should also be safe
	err = EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("Third EnsureCOMInitialized failed: %v", err)
	}
}

// TestCreateDeviceEnumerator verifies device enumerator creation
func TestCreateDeviceEnumerator(t *testing.T) {
	// Ensure COM is initialized first
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}

	if mmde == nil {
		t.Fatal("CreateDeviceEnumerator returned nil enumerator without error")
	}

	// Cleanup
	SafeReleaseMMDeviceEnumerator(&mmde)
	if mmde != nil {
		t.Error("SafeReleaseMMDeviceEnumerator should set pointer to nil")
	}
}

// TestGetDefaultRenderDevice verifies default audio device retrieval
func TestGetDefaultRenderDevice(t *testing.T) {
	skipIfNoAudioDevice(t)

	// Ensure COM is initialized first
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}
	defer SafeReleaseMMDeviceEnumerator(&mmde)

	// Get default render device
	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		t.Fatalf("GetDefaultRenderDevice failed: %v", err)
	}

	if mmd == nil {
		t.Fatal("GetDefaultRenderDevice returned nil device without error")
	}

	// Cleanup
	SafeReleaseMMDevice(&mmd)
	if mmd != nil {
		t.Error("SafeReleaseMMDevice should set pointer to nil")
	}
}

// TestSafeReleaseMMDeviceEnumerator verifies safe release of device enumerator
func TestSafeReleaseMMDeviceEnumerator(t *testing.T) {
	// Ensure COM is initialized
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device enumerator
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}

	// Release should work
	SafeReleaseMMDeviceEnumerator(&mmde)
	if mmde != nil {
		t.Error("Expected mmde to be nil after SafeRelease")
	}

	// Double release should be safe (no panic)
	SafeReleaseMMDeviceEnumerator(&mmde)

	// Release on nil pointer should be safe
	var nilPtr *wca.IMMDeviceEnumerator
	SafeReleaseMMDeviceEnumerator(&nilPtr)
}

// TestSafeReleaseMMDevice verifies safe release of MM device
func TestSafeReleaseMMDevice(t *testing.T) {
	skipIfNoAudioDevice(t)

	// Ensure COM is initialized
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device enumerator and get device
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}
	defer SafeReleaseMMDeviceEnumerator(&mmde)

	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		t.Fatalf("GetDefaultRenderDevice failed: %v", err)
	}

	// Release should work
	SafeReleaseMMDevice(&mmd)
	if mmd != nil {
		t.Error("Expected mmd to be nil after SafeRelease")
	}

	// Double release should be safe (no panic)
	SafeReleaseMMDevice(&mmd)

	// Release on nil pointer should be safe
	var nilPtr *wca.IMMDevice
	SafeReleaseMMDevice(&nilPtr)
}

// TestSafeReleaseAudioEndpointVolume verifies safe release of audio endpoint volume
func TestSafeReleaseAudioEndpointVolume(t *testing.T) {
	skipIfNoAudioDevice(t)

	// Ensure COM is initialized
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device and activate endpoint volume
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}
	defer SafeReleaseMMDeviceEnumerator(&mmde)

	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		t.Fatalf("GetDefaultRenderDevice failed: %v", err)
	}
	defer SafeReleaseMMDevice(&mmd)

	var aev *wca.IAudioEndpointVolume
	err = mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev)
	if err != nil {
		t.Fatalf("Activate IAudioEndpointVolume failed: %v", err)
	}

	// Release should work
	SafeReleaseAudioEndpointVolume(&aev)
	if aev != nil {
		t.Error("Expected aev to be nil after SafeRelease")
	}

	// Double release should be safe (no panic)
	SafeReleaseAudioEndpointVolume(&aev)

	// Release on nil pointer should be safe
	var nilPtr *wca.IAudioEndpointVolume
	SafeReleaseAudioEndpointVolume(&nilPtr)
}

// TestSafeReleaseAudioMeterInformation verifies safe release of audio meter information
func TestSafeReleaseAudioMeterInformation(t *testing.T) {
	skipIfNoAudioDevice(t)

	// Ensure COM is initialized
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device and activate meter information
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}
	defer SafeReleaseMMDeviceEnumerator(&mmde)

	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		t.Fatalf("GetDefaultRenderDevice failed: %v", err)
	}
	defer SafeReleaseMMDevice(&mmd)

	var ami *wca.IAudioMeterInformation
	err = mmd.Activate(wca.IID_IAudioMeterInformation, wca.CLSCTX_ALL, nil, &ami)
	if err != nil {
		t.Fatalf("Activate IAudioMeterInformation failed: %v", err)
	}

	// Release should work
	SafeReleaseAudioMeterInformation(&ami)
	if ami != nil {
		t.Error("Expected ami to be nil after SafeRelease")
	}

	// Double release should be safe (no panic)
	SafeReleaseAudioMeterInformation(&ami)

	// Release on nil pointer should be safe
	var nilPtr *wca.IAudioMeterInformation
	SafeReleaseAudioMeterInformation(&nilPtr)
}

// TestSafeReleaseAudioClient verifies safe release of audio client
func TestSafeReleaseAudioClient(t *testing.T) {
	skipIfNoAudioDevice(t)

	// Ensure COM is initialized
	err := EnsureCOMInitialized()
	if err != nil {
		t.Fatalf("EnsureCOMInitialized failed: %v", err)
	}

	// Create device and activate audio client
	mmde, err := CreateDeviceEnumerator()
	if err != nil {
		t.Fatalf("CreateDeviceEnumerator failed: %v", err)
	}
	defer SafeReleaseMMDeviceEnumerator(&mmde)

	mmd, err := GetDefaultRenderDevice(mmde)
	if err != nil {
		t.Fatalf("GetDefaultRenderDevice failed: %v", err)
	}
	defer SafeReleaseMMDevice(&mmd)

	var ac *wca.IAudioClient
	err = mmd.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &ac)
	if err != nil {
		t.Fatalf("Activate IAudioClient failed: %v", err)
	}

	// Release should work
	SafeReleaseAudioClient(&ac)
	if ac != nil {
		t.Error("Expected ac to be nil after SafeRelease")
	}

	// Double release should be safe (no panic)
	SafeReleaseAudioClient(&ac)

	// Release on nil pointer should be safe
	var nilPtr *wca.IAudioClient
	SafeReleaseAudioClient(&nilPtr)
}

// TestSafeReleaseAudioCaptureClient_NilSafety verifies safe release behavior with nil pointer
// Note: We can't easily test with a real IAudioCaptureClient without full audio setup
func TestSafeReleaseAudioCaptureClient_NilSafety(t *testing.T) {
	// Should not panic on nil pointer
	var nilPtr *wca.IAudioCaptureClient
	SafeReleaseAudioCaptureClient(&nilPtr)

	// Should not panic on nil double-pointer
	SafeReleaseAudioCaptureClient(nil)
}
