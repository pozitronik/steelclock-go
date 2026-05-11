//go:build darwin

package battery

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// getBatteryStatus returns the current battery status on macOS using ioreg
func getBatteryStatus() (Status, error) {
	result := Status{}

	// Use ioreg to read AppleSmartBattery properties
	out, err := exec.Command("ioreg", "-rc", "AppleSmartBattery").Output()
	if err != nil {
		// No battery (e.g., desktop Mac)
		result.HasBattery = false
		return result, nil
	}

	output := string(out)

	// Check if battery exists
	if !strings.Contains(output, "AppleSmartBattery") {
		result.HasBattery = false
		return result, nil
	}

	result.HasBattery = true

	// Parse battery properties from ioreg output
	result.Percentage = readIoregInt(output, "CurrentCapacity")
	maxCapacity := readIoregInt(output, "MaxCapacity")

	// If CurrentCapacity is raw, calculate percentage
	if maxCapacity > 0 && result.Percentage > 0 && maxCapacity != 100 {
		result.Percentage = result.Percentage * 100 / maxCapacity
	}

	// Clamp percentage
	if result.Percentage > 100 {
		result.Percentage = 100
	}
	if result.Percentage < 0 {
		result.Percentage = 0
	}

	// Charging state
	result.IsCharging = readIoregBool(output, "IsCharging")
	result.IsPluggedIn = readIoregBool(output, "ExternalConnected")

	// Time to empty/full (ioreg reports in minutes)
	timeToEmpty := readIoregInt(output, "AvgTimeToEmpty")
	if timeToEmpty > 0 && timeToEmpty < 65535 { // 65535 means not discharging
		result.TimeToEmpty = timeToEmpty
	}

	timeToFull := readIoregInt(output, "AvgTimeToFull")
	if timeToFull > 0 && timeToFull < 65535 { // 65535 means not charging
		result.TimeToFull = timeToFull
	}

	// Check Low Power Mode
	result.IsEconomyMode = isPowerSavingMode()

	return result, nil
}

// isPowerSavingMode checks if Low Power Mode is active on macOS
func isPowerSavingMode() bool {
	out, err := exec.Command("pmset", "-g").Output()
	if err != nil {
		return false
	}
	// Look for "lowpowermode 1" in pmset output
	return strings.Contains(string(out), "lowpowermode            1") ||
		strings.Contains(string(out), "lowpowermode\t\t\t1")
}

var ioregIntRegex = regexp.MustCompile(`=\s*(\d+)`)
var ioregBoolRegex = regexp.MustCompile(`=\s*(Yes|No)`)

// readIoregInt reads an integer property from ioreg output
func readIoregInt(output, key string) int {
	// Find the line containing the key
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "\""+key+"\"") {
			matches := ioregIntRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				val, err := strconv.Atoi(matches[1])
				if err == nil {
					return val
				}
			}
		}
	}
	return 0
}

// readIoregBool reads a boolean property from ioreg output
func readIoregBool(output, key string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "\""+key+"\"") {
			matches := ioregBoolRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1] == "Yes"
			}
		}
	}
	return false
}
