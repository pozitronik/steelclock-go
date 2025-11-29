//go:build linux

package widget

import (
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// LED bit positions for evdev
const (
	LedNuml    = 0x00
	LedCapsl   = 0x01
	LedScrolll = 0x02
)

// EVIOCGLED ioctl number: _IOR('E', 0x19, sizeof(int))
// 'E' = 0x45, direction = 2 (read), size depends on architecture
// For getting LED state: (2 << 30) | (size << 16) | ('E' << 8) | 0x19
func eviocgledIoctl(size uintptr) uintptr {
	return (2 << 30) | (size << 16) | (0x45 << 8) | 0x19
}

// Track if we've logged the input group warning
var inputGroupWarningLogged bool

// Update updates the keyboard state by reading LED states
func (w *KeyboardWidget) Update() error {
	// Use evdev - the only reliable method for reading keyboard LED state
	caps, num, scroll := readLEDStatesFromEvdev()

	if !evdevAvailable && !inputGroupWarningLogged {
		inputGroupWarningLogged = true
		log.Printf("[KEYBOARD] WARNING: Cannot read keyboard LED state. Add your user to the 'input' group:")
		log.Printf("[KEYBOARD]   sudo usermod -aG input $USER")
		log.Printf("[KEYBOARD] Then log out and log back in.")
	}

	w.mu.Lock()
	w.capsState = caps
	w.numState = num
	w.scrollState = scroll
	w.mu.Unlock()

	return nil
}

// Cached keyboard event device path
var (
	keyboardEventDevice string
	eventDeviceSearched bool
)

// findKeyboardEventDevice finds the event device for a USB keyboard with LEDs
func findKeyboardEventDevice() string {
	if eventDeviceSearched {
		return keyboardEventDevice
	}
	eventDeviceSearched = true

	// Read /proc/bus/input/devices to find keyboards with LED support
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		return ""
	}

	// Parse the file to find USB keyboards with LED support
	// Format: blocks separated by empty lines, each block has I:, N:, P:, S:, U:, H:, B: lines
	lines := strings.Split(string(data), "\n")

	var currentName string
	var currentHandlers string
	var currentHasLeds bool
	var currentIsUSB bool
	var bestDevice string
	var bestInputNum int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// End of block - check if this is a good keyboard
			if currentHasLeds && currentIsUSB {
				// Extract event device from handlers
				for _, handler := range strings.Fields(currentHandlers) {
					if strings.HasPrefix(handler, "event") {
						// Extract input number to prefer higher numbers (more recently connected)
						numStr := strings.TrimPrefix(handler, "event")
						num, _ := strconv.Atoi(numStr)
						if num > bestInputNum {
							bestInputNum = num
							bestDevice = "/dev/input/" + handler
							log.Printf("[KEYBOARD] Found keyboard: %s -> %s", currentName, bestDevice)
						}
					}
				}
			}
			// Reset for next block
			currentName = ""
			currentHandlers = ""
			currentHasLeds = false
			currentIsUSB = false
			continue
		}

		if strings.HasPrefix(line, "N: Name=") {
			currentName = strings.Trim(strings.TrimPrefix(line, "N: Name="), "\"")
		} else if strings.HasPrefix(line, "H: Handlers=") {
			currentHandlers = strings.TrimPrefix(line, "H: Handlers=")
			// Check if it has "leds" handler
			if strings.Contains(currentHandlers, "leds") {
				currentHasLeds = true
			}
		} else if strings.HasPrefix(line, "I: Bus=") {
			// USB bus is 0003
			if strings.Contains(line, "Bus=0003") {
				currentIsUSB = true
			}
		}
	}

	keyboardEventDevice = bestDevice
	return keyboardEventDevice
}

// Track if evdev is available
var evdevAvailable = true

// readLEDStatesFromEvdev reads LED states directly from the keyboard's event device
func readLEDStatesFromEvdev() (caps, num, scroll bool) {
	if !evdevAvailable {
		return false, false, false
	}

	device := findKeyboardEventDevice()
	if device == "" {
		evdevAvailable = false
		return false, false, false
	}

	fd, err := os.Open(device)
	if err != nil {
		// Permission denied or other error - evdev not usable
		evdevAvailable = false
		return false, false, false
	}
	defer fd.Close()

	// EVIOCGLED returns a bitmask of LED states
	// We need enough bytes to hold at least LED_SCROLLL (bit 2) = 1 byte
	var leds [1]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), eviocgledIoctl(1), uintptr(unsafe.Pointer(&leds[0])))
	if errno != 0 {
		evdevAvailable = false
		return false, false, false
	}

	// LED bits: NUML=0, CAPSL=1, SCROLLL=2
	num = leds[0]&(1<<LedNuml) != 0
	caps = leds[0]&(1<<LedCapsl) != 0
	scroll = leds[0]&(1<<LedScrolll) != 0

	return caps, num, scroll
}
