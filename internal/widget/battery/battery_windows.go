//go:build windows

package battery

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
)

// SYSTEM_POWER_STATUS structure from Windows API
type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// ACLineStatus values
//
//goland:noinspection GoUnusedConst,GoUnusedConst
const (
	acOffline = 0
	acOnline  = 1
	acUnknown = 255
)

// BatteryFlag values
//
//goland:noinspection GoUnusedConst,GoUnusedConst,GoUnusedConst
const (
	batteryHigh      = 1
	batteryLow       = 2
	batteryCritical  = 4
	batteryCharging  = 8
	batteryNoBattery = 128
	batteryUnknown   = 255
)

// SystemStatusFlag values
const (
	// systemStatusBatterySaver indicates that battery saver is on
	systemStatusBatterySaver = 1
)

// getBatteryStatus returns the current battery status on Windows
func getBatteryStatus() (Status, error) {
	var status systemPowerStatus

	ret, _, err := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	if ret == 0 {
		return Status{}, err
	}

	result := Status{
		HasBattery:    status.BatteryFlag != batteryNoBattery && status.BatteryFlag != batteryUnknown,
		IsPluggedIn:   status.ACLineStatus == acOnline,
		IsCharging:    status.BatteryFlag&batteryCharging != 0,
		IsEconomyMode: status.SystemStatusFlag&systemStatusBatterySaver != 0,
	}

	// Battery percentage
	if status.BatteryLifePercent != 255 {
		result.Percentage = int(status.BatteryLifePercent)
	}

	// Time remaining (in seconds from API, convert to minutes)
	if status.BatteryLifeTime != 0xFFFFFFFF {
		result.TimeToEmpty = int(status.BatteryLifeTime / 60)
	}

	return result, nil
}
