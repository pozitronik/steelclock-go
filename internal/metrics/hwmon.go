package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// LHMHTTPProvider queries LibreHardwareMonitor/OpenHardwareMonitor's built-in
// web server for sensor data. The server exposes a JSON tree at /data.json
// containing all hardware sensors (temperature, voltage, load, power, etc.).
type LHMHTTPProvider struct {
	url    string
	client *http.Client
}

// NewLHMHTTPProvider creates a provider that queries the LHM/OHM HTTP API
// at the given base URL (e.g., "http://localhost:8085").
func NewLHMHTTPProvider(url string) *LHMHTTPProvider {
	return &LHMHTTPProvider{
		url:    strings.TrimRight(url, "/"),
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

// lhmNode represents a node in LHM/OHM's JSON sensor tree.
type lhmNode struct {
	Text     string    `json:"Text"`
	SensorID string    `json:"SensorId"`
	Type     string    `json:"Type"`
	Value    string    `json:"Value"`
	RawValue string    `json:"RawValue"`
	Children []lhmNode `json:"Children"`
}

// Sensors fetches all sensor readings from the LHM/OHM HTTP API.
func (p *LHMHTTPProvider) Sensors() ([]HWMonStat, error) {
	resp, err := p.client.Get(p.url + "/data.json")
	if err != nil {
		return nil, fmt.Errorf("LHM/OHM web server not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LHM/OHM web server returned HTTP %d", resp.StatusCode)
	}

	var root lhmNode
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to parse LHM/OHM JSON: %w", err)
	}

	var result []HWMonStat
	collectSensors(&root, &result)

	if len(result) == 0 {
		return nil, fmt.Errorf("no sensors found in LHM/OHM response")
	}
	return result, nil
}

// collectSensors recursively walks the LHM/OHM sensor tree and collects
// all leaf sensor nodes (those with a non-empty SensorId and Type).
func collectSensors(node *lhmNode, result *[]HWMonStat) {
	if node.SensorID != "" && node.Type != "" {
		value, unit, ok := parseLHMRawValue(node.Value)
		if ok {
			*result = append(*result, HWMonStat{
				SensorID: node.SensorID,
				Name:     node.Text,
				Type:     node.Type,
				Value:    value,
				Unit:     unit,
			})
		}
	}
	for i := range node.Children {
		collectSensors(&node.Children[i], result)
	}
}

// parseLHMRawValue parses a sensor value string from LHM/OHM.
// The format is locale-dependent: "74,0 °C", "6,8 %", "3924,0 MHz".
// Returns the numeric value, the unit string, and whether parsing succeeded.
func parseLHMRawValue(raw string) (float64, string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, "", false
	}

	// Split on first space: number part + unit part
	// Examples: "74,0 °C" -> ["74,0", "°C"]
	//           "45,000"  -> ["45,000"] (no unit, e.g. Factor type)
	parts := strings.SplitN(s, " ", 2)

	numStr := strings.ReplaceAll(parts[0], ",", ".")
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, "", false
	}

	unit := ""
	if len(parts) > 1 {
		unit = strings.TrimSpace(parts[1])
	}

	return val, unit, true
}
