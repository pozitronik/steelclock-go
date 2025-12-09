//go:build linux

package battery

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const powerSupplyPath = "/sys/class/power_supply"

// ACPI platform profile path (modern kernels)
const acpiPlatformProfilePath = "/sys/firmware/acpi/platform_profile"

// getBatteryStatus returns the current battery status on Linux
func getBatteryStatus() (Status, error) {
	result := Status{}

	// Find battery device
	entries, err := os.ReadDir(powerSupplyPath)
	if err != nil {
		return result, err
	}

	var batteryPath string
	var acPath string

	for _, entry := range entries {
		name := entry.Name()
		typePath := filepath.Join(powerSupplyPath, name, "type")
		typeData, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}

		deviceType := strings.TrimSpace(string(typeData))
		if deviceType == "Battery" && batteryPath == "" {
			batteryPath = filepath.Join(powerSupplyPath, name)
		} else if deviceType == "Mains" || deviceType == "AC" {
			acPath = filepath.Join(powerSupplyPath, name)
		}
	}

	// Check AC status
	if acPath != "" {
		onlineData, err := os.ReadFile(filepath.Join(acPath, "online"))
		if err == nil {
			result.IsPluggedIn = strings.TrimSpace(string(onlineData)) == "1"
		}
	}

	// No battery found
	if batteryPath == "" {
		result.HasBattery = false
		return result, nil
	}

	result.HasBattery = true

	// Read battery percentage
	// Try capacity first (percentage directly)
	capacityData, err := os.ReadFile(filepath.Join(batteryPath, "capacity"))
	if err == nil {
		percentage, err := strconv.Atoi(strings.TrimSpace(string(capacityData)))
		if err == nil {
			result.Percentage = percentage
		}
	} else {
		// Fall back to calculating from energy_now/energy_full or charge_now/charge_full
		energyNow := readIntFile(filepath.Join(batteryPath, "energy_now"))
		energyFull := readIntFile(filepath.Join(batteryPath, "energy_full"))
		if energyNow > 0 && energyFull > 0 {
			result.Percentage = energyNow * 100 / energyFull
		} else {
			chargeNow := readIntFile(filepath.Join(batteryPath, "charge_now"))
			chargeFull := readIntFile(filepath.Join(batteryPath, "charge_full"))
			if chargeNow > 0 && chargeFull > 0 {
				result.Percentage = chargeNow * 100 / chargeFull
			}
		}
	}

	// Clamp percentage
	if result.Percentage > 100 {
		result.Percentage = 100
	}
	if result.Percentage < 0 {
		result.Percentage = 0
	}

	// Read charging status
	statusData, err := os.ReadFile(filepath.Join(batteryPath, "status"))
	if err == nil {
		status := strings.TrimSpace(string(statusData))
		result.IsCharging = status == "Charging"
		// If status is "Full" or "Not charging", we're on AC but not charging
		if status == "Full" || status == "Not charging" {
			result.IsPluggedIn = true
		}
	}

	// Try to read time to empty/full
	timeToEmpty := readIntFile(filepath.Join(batteryPath, "time_to_empty_now"))
	if timeToEmpty > 0 {
		result.TimeToEmpty = timeToEmpty / 60 // Convert seconds to minutes
	}

	timeToFull := readIntFile(filepath.Join(batteryPath, "time_to_full_now"))
	if timeToFull > 0 {
		result.TimeToFull = timeToFull / 60 // Convert seconds to minutes
	}

	// Check for power saving / economy mode
	result.IsEconomyMode = isPowerSavingMode()

	return result, nil
}

// isPowerSavingMode checks if power saving mode is active on Linux
func isPowerSavingMode() bool {
	// Try ACPI platform profile first (works on many modern laptops)
	profileData, err := os.ReadFile(acpiPlatformProfilePath)
	if err == nil {
		profile := strings.TrimSpace(string(profileData))
		// Common power saving profile names
		if profile == "low-power" || profile == "power-saver" || profile == "quiet" {
			return true
		}
	}

	// Try to check energy_performance_preference (Intel P-state or AMD)
	epps, err := filepath.Glob("/sys/devices/system/cpu/cpufreq/policy*/energy_performance_preference")
	if err == nil && len(epps) > 0 {
		// Check the first CPU policy
		eppData, err := os.ReadFile(epps[0])
		if err == nil {
			epp := strings.TrimSpace(string(eppData))
			if epp == "power" || epp == "balance_power" {
				return true
			}
		}
	}

	// Try scaling_governor as a fallback
	governors, err := filepath.Glob("/sys/devices/system/cpu/cpufreq/policy*/scaling_governor")
	if err == nil && len(governors) > 0 {
		govData, err := os.ReadFile(governors[0])
		if err == nil {
			gov := strings.TrimSpace(string(govData))
			if gov == "powersave" {
				return true
			}
		}
	}

	return false
}

// readIntFile reads an integer from a file, returns 0 on error
func readIntFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return val
}
