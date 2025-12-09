package gamesense

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// findCorePropsPathFunc is a variable for dependency injection in tests
var findCorePropsPathFunc = findCorePropsPath

// defaultFallbackPath is the hardcoded fallback path for coreProps.json
// This is a variable to allow tests to override it
var defaultFallbackPath = filepath.Join("C:", "ProgramData", "SteelSeries", "SteelSeries Engine 3", "coreProps.json")

// DiscoverServer finds the SteelSeries GameSense server address
func DiscoverServer() (string, error) {
	corePropsPath, err := findCorePropsPathFunc()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(corePropsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read coreProps.json: %w", err)
	}

	var props struct {
		Address string `json:"address"`
	}

	if err := json.Unmarshal(data, &props); err != nil {
		return "", fmt.Errorf("failed to parse coreProps.json: %w", err)
	}

	if props.Address == "" {
		return "", fmt.Errorf("no 'address' field in coreProps.json")
	}

	// Validate address format (should be "host:port")
	parts := strings.Split(props.Address, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid address format: %s", props.Address)
	}

	if _, err := strconv.Atoi(parts[1]); err != nil {
		return "", fmt.Errorf("invalid port number: %s", parts[1])
	}

	return props.Address, nil
}

// findCorePropsPath locates the coreProps.json file
func findCorePropsPath() (string, error) {
	// Windows: %PROGRAMDATA%\SteelSeries\SteelSeries Engine 3\coreProps.json
	programData := os.Getenv("PROGRAMDATA")
	if programData != "" {
		path := filepath.Join(programData, "SteelSeries", "SteelSeries Engine 3", "coreProps.json")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Fallback: C:\ProgramData\... (uses variable to allow test override)
	if _, err := os.Stat(defaultFallbackPath); err == nil {
		return defaultFallbackPath, nil
	}

	//goland:noinspection ALL
	return "", fmt.Errorf("cannot find coreProps.json - is SteelSeries Engine 3 installed?")
}
