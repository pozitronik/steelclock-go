//go:build darwin

package driver

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation

#include <IOKit/hid/IOHIDManager.h>
#include <IOKit/hid/IOHIDDevice.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

// Helper to get an integer property from a HID device
static int64_t getDeviceIntProperty(IOHIDDeviceRef device, CFStringRef key) {
	CFTypeRef ref = IOHIDDeviceGetProperty(device, key);
	if (ref && CFGetTypeID(ref) == CFNumberGetTypeID()) {
		int64_t value = 0;
		CFNumberGetValue((CFNumberRef)ref, kCFNumberSInt64Type, &value);
		return value;
	}
	return -1;
}

// Helper to get a string property from a HID device
// Caller must free the returned string.
static char* getDeviceStringProperty(IOHIDDeviceRef device, CFStringRef key) {
	CFTypeRef ref = IOHIDDeviceGetProperty(device, key);
	if (ref && CFGetTypeID(ref) == CFStringGetTypeID()) {
		CFIndex length = CFStringGetMaximumSizeForEncoding(
			CFStringGetLength((CFStringRef)ref), kCFStringEncodingUTF8) + 1;
		char *buf = (char *)malloc(length);
		if (buf && CFStringGetCString((CFStringRef)ref, buf, length, kCFStringEncodingUTF8)) {
			return buf;
		}
		free(buf);
	}
	return NULL;
}

// Enumerate HID devices matching optional VID/PID filters.
// Returns count of devices found. Fills arrays with device info.
// Caller must free string arrays.
static int enumerateHID(
	int filterVID, int filterPID,
	int64_t *vids, int64_t *pids,
	char **products, char **manufacturers, char **paths,
	int maxDevices
) {
	IOHIDManagerRef manager = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
	if (!manager) return 0;

	// Build matching dictionary
	CFMutableDictionaryRef matchDict = NULL;
	if (filterVID >= 0 || filterPID >= 0) {
		matchDict = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
			&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
		if (filterVID >= 0) {
			CFNumberRef vidNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &filterVID);
			CFDictionarySetValue(matchDict, CFSTR(kIOHIDVendorIDKey), vidNum);
			CFRelease(vidNum);
		}
		if (filterPID >= 0) {
			CFNumberRef pidNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &filterPID);
			CFDictionarySetValue(matchDict, CFSTR(kIOHIDProductIDKey), pidNum);
			CFRelease(pidNum);
		}
	}

	IOHIDManagerSetDeviceMatching(manager, matchDict);
	if (matchDict) CFRelease(matchDict);

	IOHIDManagerOpen(manager, kIOHIDOptionsTypeNone);

	CFSetRef deviceSet = IOHIDManagerCopyDevices(manager);
	if (!deviceSet) {
		IOHIDManagerClose(manager, kIOHIDOptionsTypeNone);
		CFRelease(manager);
		return 0;
	}

	CFIndex count = CFSetGetCount(deviceSet);
	if (count > maxDevices) count = maxDevices;

	const void **devices = (const void **)malloc(sizeof(void*) * count);
	CFSetGetValues(deviceSet, devices);

	int found = 0;
	for (CFIndex i = 0; i < count && found < maxDevices; i++) {
		IOHIDDeviceRef device = (IOHIDDeviceRef)devices[i];

		vids[found] = getDeviceIntProperty(device, CFSTR(kIOHIDVendorIDKey));
		pids[found] = getDeviceIntProperty(device, CFSTR(kIOHIDProductIDKey));
		products[found] = getDeviceStringProperty(device, CFSTR(kIOHIDProductKey));
		manufacturers[found] = getDeviceStringProperty(device, CFSTR(kIOHIDManufacturerKey));

		// Build a unique path from the transport + location
		char pathBuf[256];
		char *transport = getDeviceStringProperty(device, CFSTR(kIOHIDTransportKey));
		int64_t locationID = getDeviceIntProperty(device, CFSTR(kIOHIDLocationIDKey));
		snprintf(pathBuf, sizeof(pathBuf), "IOHIDDevice_%s_%llx_%llx_%llx",
			transport ? transport : "unknown",
			(long long)locationID,
			(long long)vids[found],
			(long long)pids[found]);
		if (transport) free(transport);

		paths[found] = strdup(pathBuf);
		found++;
	}

	free(devices);
	CFRelease(deviceSet);
	IOHIDManagerClose(manager, kIOHIDOptionsTypeNone);
	CFRelease(manager);

	return found;
}

// Open a HID device by VID and PID (returns the IOHIDDeviceRef as a void pointer)
static void* openHIDDevice(int vid, int pid) {
	IOHIDManagerRef manager = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
	if (!manager) return NULL;

	CFMutableDictionaryRef matchDict = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

	CFNumberRef vidNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &vid);
	CFNumberRef pidNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &pid);
	CFDictionarySetValue(matchDict, CFSTR(kIOHIDVendorIDKey), vidNum);
	CFDictionarySetValue(matchDict, CFSTR(kIOHIDProductIDKey), pidNum);
	CFRelease(vidNum);
	CFRelease(pidNum);

	IOHIDManagerSetDeviceMatching(manager, matchDict);
	CFRelease(matchDict);

	IOHIDManagerOpen(manager, kIOHIDOptionsTypeNone);

	CFSetRef deviceSet = IOHIDManagerCopyDevices(manager);
	if (!deviceSet) {
		IOHIDManagerClose(manager, kIOHIDOptionsTypeNone);
		CFRelease(manager);
		return NULL;
	}

	CFIndex count = CFSetGetCount(deviceSet);
	if (count == 0) {
		CFRelease(deviceSet);
		IOHIDManagerClose(manager, kIOHIDOptionsTypeNone);
		CFRelease(manager);
		return NULL;
	}

	const void **devices = (const void **)malloc(sizeof(void*) * count);
	CFSetGetValues(deviceSet, devices);
	IOHIDDeviceRef device = (IOHIDDeviceRef)devices[0];

	IOReturn result = IOHIDDeviceOpen(device, kIOHIDOptionsTypeSeizeDevice);
	free(devices);
	CFRelease(deviceSet);

	if (result != kIOReturnSuccess) {
		IOHIDManagerClose(manager, kIOHIDOptionsTypeNone);
		CFRelease(manager);
		return NULL;
	}

	// We need to retain the device since the manager owns the set.
	// Store both manager and device together.
	// We'll use a simple struct for this.
	CFRetain(device);

	// Store the manager ref in device property for later cleanup
	// (hack: we leak the manager to keep the device alive)
	// A proper implementation would store both, but for simplicity
	// we keep the manager alive by not releasing it.
	// The caller is responsible for calling closeHIDDevice.

	return (void*)device;
}

static void closeHIDDevice(void *devicePtr) {
	if (devicePtr) {
		IOHIDDeviceRef device = (IOHIDDeviceRef)devicePtr;
		IOHIDDeviceClose(device, kIOHIDOptionsTypeNone);
		CFRelease(device);
	}
}

// Send a feature report to a HID device
// macOS IOHIDDeviceSetReport takes the report ID as a separate parameter
static int sendHIDFeatureReport(void *devicePtr, const uint8_t *data, int length) {
	if (!devicePtr || !data || length <= 0) return -1;

	IOHIDDeviceRef device = (IOHIDDeviceRef)devicePtr;

	// The report ID is the first byte of the data, per convention.
	// On macOS, IOHIDDeviceSetReport takes the report ID separately from the data buffer,
	// so we always strip data[0] as the report ID. Report ID 0 means the device has no
	// report IDs (single-report device per HID spec).
	uint8_t reportID = data[0];
	const uint8_t *reportData = data + 1;
	CFIndex reportLength = length - 1;

	IOReturn result = IOHIDDeviceSetReport(device,
		kIOHIDReportTypeFeature, reportID,
		reportData, reportLength);

	if (result == kIOReturnSuccess) return 0;

	// Try as output report if feature report fails
	result = IOHIDDeviceSetReport(device,
		kIOHIDReportTypeOutput, reportID,
		reportData, reportLength);

	return (result == kIOReturnSuccess) ? 0 : -1;
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

// DeviceHandle wraps a pointer to an IOHIDDeviceRef
type DeviceHandle uintptr

// InvalidHandle represents an invalid device handle
const InvalidHandle DeviceHandle = 0

// maxEnumDevices is the maximum number of devices we enumerate at once
const maxEnumDevices = 64

// findDevicePath finds a HID device by VID, PID, and optional interface
func findDevicePath(vid, pid uint16, targetInterface string) (string, error) {
	devices, err := enumerateHIDDevices(int(vid), int(pid))
	if err != nil {
		return "", err
	}

	for _, dev := range devices {
		if dev.VID == vid && dev.PID == pid {
			return dev.Path, nil
		}
	}

	return "", fmt.Errorf("device VID_%04X PID_%04X interface %s not found", vid, pid, targetInterface)
}

// autoDetectDevice tries to find any known SteelSeries device
func autoDetectDevice(targetInterface string) (string, error) {
	for _, known := range KnownDevices {
		devices, err := enumerateHIDDevices(int(known.VID), int(known.PID))
		if err != nil {
			continue
		}
		for _, dev := range devices {
			if dev.VID == known.VID && dev.PID == known.PID {
				return dev.Path, nil
			}
		}
	}

	return "", fmt.Errorf("no known SteelSeries device found")
}

// openDevice opens a HID device by path
// The path is expected to contain VID/PID info (from our enumeration)
func openDevice(path string) (DeviceHandle, error) {
	// Parse VID and PID from path (format: IOHIDDevice_transport_location_vid_pid)
	vid, pid, err := parseDevicePath(path)
	if err != nil {
		return InvalidHandle, fmt.Errorf("cannot parse device path %q: %w", path, err)
	}

	ptr := C.openHIDDevice(C.int(vid), C.int(pid))
	if ptr == nil {
		return InvalidHandle, fmt.Errorf("failed to open device VID_%04X PID_%04X (check USB permissions)", vid, pid)
	}

	return DeviceHandle(uintptr(ptr)), nil
}

// closeDevice closes a HID device handle
func closeDevice(handle DeviceHandle) error {
	if handle == InvalidHandle {
		return nil
	}
	C.closeHIDDevice(unsafe.Pointer(uintptr(handle)))
	return nil
}

// sendFeatureReport sends a feature report to the HID device
func sendFeatureReport(handle DeviceHandle, data []byte) error {
	if handle == InvalidHandle {
		return fmt.Errorf("invalid handle")
	}

	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	result := C.sendHIDFeatureReport(
		unsafe.Pointer(uintptr(handle)),
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.int(len(data)),
	)

	if result != 0 {
		return fmt.Errorf("failed to send feature report (IOKit error)")
	}

	return nil
}

// EnumerateDevices returns a list of all connected HID devices
func EnumerateDevices() ([]DeviceInfo, error) {
	return enumerateHIDDevices(-1, -1)
}

// enumerateHIDDevices enumerates HID devices, optionally filtered by VID/PID
// Pass -1 for vid or pid to skip that filter
func enumerateHIDDevices(filterVID, filterPID int) ([]DeviceInfo, error) {
	var (
		vids          [maxEnumDevices]C.int64_t
		pids          [maxEnumDevices]C.int64_t
		products      [maxEnumDevices]*C.char
		manufacturers [maxEnumDevices]*C.char
		paths         [maxEnumDevices]*C.char
	)

	count := C.enumerateHID(
		C.int(filterVID), C.int(filterPID),
		&vids[0], &pids[0],
		&products[0], &manufacturers[0], &paths[0],
		C.int(maxEnumDevices),
	)

	var result []DeviceInfo
	for i := 0; i < int(count); i++ {
		dev := DeviceInfo{
			VID: uint16(vids[i]),
			PID: uint16(pids[i]),
		}

		if products[i] != nil {
			dev.ProductName = C.GoString(products[i])
			C.free(unsafe.Pointer(products[i]))
		}
		if manufacturers[i] != nil {
			dev.Manufacturer = C.GoString(manufacturers[i])
			C.free(unsafe.Pointer(manufacturers[i]))
		}
		if paths[i] != nil {
			dev.Path = C.GoString(paths[i])
			C.free(unsafe.Pointer(paths[i]))
		}

		result = append(result, dev)
	}

	return result, nil
}

// parseDevicePath extracts VID and PID from our enumeration path format
// Format: IOHIDDevice_transport_location_vid_pid
func parseDevicePath(path string) (vid, pid int, err error) {
	parts := strings.Split(path, "_")
	if len(parts) < 3 {
		return 0, 0, fmt.Errorf("invalid path format")
	}

	// The last two hex parts are VID and PID
	var v, p int64
	_, err = fmt.Sscanf(parts[len(parts)-2]+"_"+parts[len(parts)-1], "%x_%x", &v, &p)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot parse VID/PID from path: %w", err)
	}

	return int(v), int(p), nil
}
