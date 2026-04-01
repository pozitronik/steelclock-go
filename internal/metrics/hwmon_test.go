package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLHMRawValue(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantVal  float64
		wantUnit string
		wantOK   bool
	}{
		// Standard locale-dependent formats (comma as decimal separator)
		{"temperature comma", "74,0 °C", 74.0, "°C", true},
		{"load comma", "6,8 %", 6.8, "%", true},
		{"power comma", "13,9 W", 13.9, "W", true},
		{"clock comma", "3924,0 MHz", 3924.0, "MHz", true},
		{"voltage comma", "1,05 V", 1.05, "V", true},
		{"current comma", "0,50 A", 0.5, "A", true},

		// Dot as decimal separator (English locale)
		{"temperature dot", "74.0 °C", 74.0, "°C", true},
		{"load dot", "6.8 %", 6.8, "%", true},
		{"clock dot", "3924.0 MHz", 3924.0, "MHz", true},

		// No unit (e.g., Factor type)
		{"no unit comma", "45,000", 45.0, "", true},
		{"no unit dot", "1.5", 1.5, "", true},
		{"integer no unit", "100", 100.0, "", true},

		// Zero and boundary values
		{"zero", "0,0 °C", 0.0, "°C", true},
		{"negative", "-10,5 °C", -10.5, "°C", true},
		{"large value", "99999,0 MHz", 99999.0, "MHz", true},

		// Whitespace handling
		{"leading whitespace", "  74,0 °C", 74.0, "°C", true},
		{"trailing whitespace", "74,0 °C  ", 74.0, "°C", true},
		{"extra spaces in unit", "74,0  °C", 74.0, "°C", true},

		// Failure cases
		{"empty string", "", 0, "", false},
		{"whitespace only", "   ", 0, "", false},
		{"non-numeric", "abc °C", 0, "", false},
		{"unit only", "°C", 0, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, unit, ok := parseLHMRawValue(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("parseLHMRawValue(%q) ok = %v, want %v", tt.raw, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if diff := val - tt.wantVal; diff > 0.001 || diff < -0.001 {
				t.Errorf("parseLHMRawValue(%q) val = %f, want %f", tt.raw, val, tt.wantVal)
			}
			if unit != tt.wantUnit {
				t.Errorf("parseLHMRawValue(%q) unit = %q, want %q", tt.raw, unit, tt.wantUnit)
			}
		})
	}
}

func TestCollectSensors(t *testing.T) {
	t.Run("flat tree with sensors", func(t *testing.T) {
		root := lhmNode{
			Text: "Computer",
			Children: []lhmNode{
				{
					Text:     "CPU Temperature",
					SensorID: "/cpu/0/temperature/0",
					Type:     "Temperature",
					Value:    "65,0 °C",
				},
				{
					Text:     "CPU Load",
					SensorID: "/cpu/0/load/0",
					Type:     "Load",
					Value:    "42,5 %",
				},
			},
		}

		var result []HWMonStat
		collectSensors(&root, &result)

		if len(result) != 2 {
			t.Fatalf("got %d sensors, want 2", len(result))
		}
		if result[0].SensorID != "/cpu/0/temperature/0" {
			t.Errorf("result[0].SensorID = %q, want %q", result[0].SensorID, "/cpu/0/temperature/0")
		}
		if result[0].Value != 65.0 {
			t.Errorf("result[0].Value = %f, want 65.0", result[0].Value)
		}
		if result[0].Unit != "°C" {
			t.Errorf("result[0].Unit = %q, want %q", result[0].Unit, "°C")
		}
		if result[1].Type != "Load" {
			t.Errorf("result[1].Type = %q, want %q", result[1].Type, "Load")
		}
	})

	t.Run("nested tree", func(t *testing.T) {
		// Mimics LHM structure: Computer > Hardware > Category > Sensors
		root := lhmNode{
			Text: "Computer",
			Children: []lhmNode{
				{
					Text: "AMD Ryzen 5 3600",
					Children: []lhmNode{
						{
							Text: "Temperatures",
							Children: []lhmNode{
								{
									Text:     "Core (Tctl/Tdie)",
									SensorID: "/amdcpu/0/temperature/2",
									Type:     "Temperature",
									Value:    "55,0 °C",
								},
							},
						},
						{
							Text: "Load",
							Children: []lhmNode{
								{
									Text:     "CPU Total",
									SensorID: "/amdcpu/0/load/0",
									Type:     "Load",
									Value:    "15,0 %",
								},
							},
						},
					},
				},
			},
		}

		var result []HWMonStat
		collectSensors(&root, &result)

		if len(result) != 2 {
			t.Fatalf("got %d sensors, want 2", len(result))
		}
		if result[0].Name != "Core (Tctl/Tdie)" {
			t.Errorf("result[0].Name = %q, want %q", result[0].Name, "Core (Tctl/Tdie)")
		}
		if result[1].Name != "CPU Total" {
			t.Errorf("result[1].Name = %q, want %q", result[1].Name, "CPU Total")
		}
	})

	t.Run("skips nodes without SensorID", func(t *testing.T) {
		root := lhmNode{
			Text: "Computer",
			Children: []lhmNode{
				{Text: "Group node without sensor", Type: "Temperature"},
				{Text: "Sensor", SensorID: "/cpu/0/temp/0", Type: "Temperature", Value: "50,0 °C"},
			},
		}

		var result []HWMonStat
		collectSensors(&root, &result)

		if len(result) != 1 {
			t.Fatalf("got %d sensors, want 1", len(result))
		}
	})

	t.Run("skips nodes without Type", func(t *testing.T) {
		root := lhmNode{
			Text: "Computer",
			Children: []lhmNode{
				{Text: "No type", SensorID: "/cpu/0/temp/0", Value: "50,0 °C"},
				{Text: "Has type", SensorID: "/cpu/0/load/0", Type: "Load", Value: "10,0 %"},
			},
		}

		var result []HWMonStat
		collectSensors(&root, &result)

		if len(result) != 1 {
			t.Fatalf("got %d sensors, want 1", len(result))
		}
	})

	t.Run("skips nodes with unparseable values", func(t *testing.T) {
		root := lhmNode{
			Text: "Computer",
			Children: []lhmNode{
				{Text: "Bad value", SensorID: "/cpu/0/temp/0", Type: "Temperature", Value: "not-a-number"},
				{Text: "Good value", SensorID: "/cpu/0/load/0", Type: "Load", Value: "25,0 %"},
			},
		}

		var result []HWMonStat
		collectSensors(&root, &result)

		if len(result) != 1 {
			t.Fatalf("got %d sensors, want 1 (should skip unparseable)", len(result))
		}
		if result[0].SensorID != "/cpu/0/load/0" {
			t.Errorf("result[0].SensorID = %q, want %q", result[0].SensorID, "/cpu/0/load/0")
		}
	})

	t.Run("empty tree", func(t *testing.T) {
		root := lhmNode{Text: "Computer"}

		var result []HWMonStat
		collectSensors(&root, &result)

		if len(result) != 0 {
			t.Errorf("got %d sensors, want 0 for empty tree", len(result))
		}
	})
}

// sampleLHMResponse returns a realistic LHM/OHM JSON tree for testing.
func sampleLHMResponse() lhmNode {
	return lhmNode{
		Text: "Computer",
		Children: []lhmNode{
			{
				Text: "AMD Ryzen 5 3600",
				Children: []lhmNode{
					{
						Text: "Temperatures",
						Children: []lhmNode{
							{Text: "Core (Tctl/Tdie)", SensorID: "/amdcpu/0/temperature/2", Type: "Temperature", Value: "65,0 °C"},
						},
					},
					{
						Text: "Load",
						Children: []lhmNode{
							{Text: "CPU Total", SensorID: "/amdcpu/0/load/0", Type: "Load", Value: "15,0 %"},
						},
					},
				},
			},
			{
				Text: "NVIDIA GeForce RTX 3070",
				Children: []lhmNode{
					{
						Text: "Temperatures",
						Children: []lhmNode{
							{Text: "GPU Core", SensorID: "/gpu-nvidia/0/temperature/0", Type: "Temperature", Value: "48,0 °C"},
						},
					},
				},
			},
		},
	}
}

func TestLHMHTTPProvider_Sensors(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		root := sampleLHMResponse()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/data.json" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(root)
		}))
		defer server.Close()

		provider := NewLHMHTTPProvider(server.URL)
		sensors, err := provider.Sensors()
		if err != nil {
			t.Fatalf("Sensors() error = %v", err)
		}

		if len(sensors) != 3 {
			t.Fatalf("got %d sensors, want 3", len(sensors))
		}

		// Verify first sensor
		if sensors[0].SensorID != "/amdcpu/0/temperature/2" {
			t.Errorf("sensors[0].SensorID = %q", sensors[0].SensorID)
		}
		if sensors[0].Value != 65.0 {
			t.Errorf("sensors[0].Value = %f, want 65.0", sensors[0].Value)
		}
		if sensors[0].Unit != "°C" {
			t.Errorf("sensors[0].Unit = %q, want °C", sensors[0].Unit)
		}
	})

	t.Run("trailing slash in URL", func(t *testing.T) {
		root := sampleLHMResponse()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/data.json" {
				t.Errorf("unexpected path: %s (double slash?)", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(root)
		}))
		defer server.Close()

		provider := NewLHMHTTPProvider(server.URL + "/")
		sensors, err := provider.Sensors()
		if err != nil {
			t.Fatalf("Sensors() error = %v", err)
		}
		if len(sensors) != 3 {
			t.Fatalf("got %d sensors, want 3", len(sensors))
		}
	})

	t.Run("server returns HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider := NewLHMHTTPProvider(server.URL)
		_, err := provider.Sensors()
		if err == nil {
			t.Fatal("expected error for HTTP 500")
		}
	})

	t.Run("server unreachable", func(t *testing.T) {
		provider := NewLHMHTTPProvider("http://127.0.0.1:1") // port 1 is unlikely to respond
		_, err := provider.Sensors()
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		provider := NewLHMHTTPProvider(server.URL)
		_, err := provider.Sensors()
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("empty sensor tree", func(t *testing.T) {
		root := lhmNode{Text: "Computer"} // no children
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(root)
		}))
		defer server.Close()

		provider := NewLHMHTTPProvider(server.URL)
		_, err := provider.Sensors()
		if err == nil {
			t.Fatal("expected error for empty sensor tree")
		}
	})

	t.Run("all sensors unparseable", func(t *testing.T) {
		root := lhmNode{
			Text: "Computer",
			Children: []lhmNode{
				{Text: "Bad sensor", SensorID: "/cpu/0/temp/0", Type: "Temperature", Value: ""},
				{Text: "Bad sensor 2", SensorID: "/cpu/0/temp/1", Type: "Temperature", Value: "   "},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(root)
		}))
		defer server.Close()

		provider := NewLHMHTTPProvider(server.URL)
		_, err := provider.Sensors()
		if err == nil {
			t.Fatal("expected error when no sensors can be parsed")
		}
	})
}

func TestNewLHMHTTPProvider(t *testing.T) {
	t.Run("trims trailing slash", func(t *testing.T) {
		p := NewLHMHTTPProvider("http://localhost:8085/")
		if p.url != "http://localhost:8085" {
			t.Errorf("url = %q, want trailing slash trimmed", p.url)
		}
	})

	t.Run("preserves URL without trailing slash", func(t *testing.T) {
		p := NewLHMHTTPProvider("http://localhost:8085")
		if p.url != "http://localhost:8085" {
			t.Errorf("url = %q", p.url)
		}
	})

	t.Run("client has timeout", func(t *testing.T) {
		p := NewLHMHTTPProvider("http://localhost:8085")
		if p.client == nil {
			t.Fatal("client is nil")
		}
		if p.client.Timeout == 0 {
			t.Error("client timeout should be non-zero")
		}
	})
}
