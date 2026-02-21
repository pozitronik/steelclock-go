package bluetooth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared/anim"
)

// testConfig returns a valid WidgetConfig for bluetooth widget tests
func testConfig() config.WidgetConfig {
	return config.WidgetConfig{
		Type: "bluetooth",
		Position: config.PositionConfig{
			W: 128,
			H: 40,
		},
		Style: &config.StyleConfig{
			Background: 0,
			Border:     -1,
		},
		Text: &config.TextConfig{
			Size: 10,
			Align: &config.AlignConfig{
				H: "center",
				V: "center",
			},
		},
		Bluetooth: &config.BluetoothConfig{
			Address: "AA:BB:CC:DD:EE:FF",
			APIURL:  "127.0.0.1:8765",
			Format:  "{icon} {name} {battery:20}",
		},
		UpdateInterval: 1.0,
	}
}

func TestNew_ValidConfig(t *testing.T) {
	cfg := testConfig()
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w == nil {
		t.Fatal("New() returned nil widget")
	}
	if w.address != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("address = %q, want %q", w.address, "AA:BB:CC:DD:EE:FF")
	}
	if w.apiURL != "127.0.0.1:8765" {
		t.Errorf("apiURL = %q, want %q", w.apiURL, "127.0.0.1:8765")
	}
	if w.format != "{icon} {name} {battery:20}" {
		t.Errorf("format = %q, want %q", w.format, "{icon} {name} {battery:20}")
	}
	if len(w.tokens) != 5 {
		t.Errorf("tokens count = %d, want 5", len(w.tokens))
	}
}

func TestNew_MissingAddress(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth.Address = ""
	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error for missing address")
	}
}

func TestNew_NilBluetoothConfig(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth = nil
	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error for nil bluetooth config")
	}
}

func TestNew_Colors(t *testing.T) {
	cfg := testConfig()
	on := 200
	off := 50
	cfg.Colors = &config.ColorsConfig{
		On:  &on,
		Off: &off,
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.colorOn != 200 {
		t.Errorf("colorOn = %d, want 200", w.colorOn)
	}
	if w.colorOff != 50 {
		t.Errorf("colorOff = %d, want 50", w.colorOff)
	}
}

func TestUpdate_ConnectedDevice(t *testing.T) {
	batteryLevel := 85
	device := apiResponse{
		Adapter:         &adapterInfo{Available: true, Enabled: true},
		Address:         "AA:BB:CC:DD:EE:FF",
		Name:            "My Headphones",
		DisplayName:     "Headphones BT",
		Type:            "AudioOutput",
		ConnectionState: "Connected",
		IsConnected:     true,
		Battery:         batteryInfo{Level: &batteryLevel, Supported: true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(device)
	}))
	defer server.Close()

	cfg := testConfig()
	cfg.Bluetooth.APIURL = strings.TrimPrefix(server.URL, "http://")

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.apiReachable {
		t.Error("apiReachable should be true")
	}
	if !w.deviceFound {
		t.Error("deviceFound should be true")
	}
	if !w.adapterOk {
		t.Error("adapterOk should be true")
	}
	if !w.connected {
		t.Error("connected should be true")
	}
	if w.connectionState != "Connected" {
		t.Errorf("connectionState = %q, want %q", w.connectionState, "Connected")
	}
	if w.deviceType != "AudioOutput" {
		t.Errorf("deviceType = %q, want %q", w.deviceType, "AudioOutput")
	}
	if w.deviceName != "Headphones BT" {
		t.Errorf("deviceName = %q, want %q (should prefer displayName)", w.deviceName, "Headphones BT")
	}
	if w.batteryLevel == nil || *w.batteryLevel != 85 {
		t.Errorf("batteryLevel = %v, want 85", w.batteryLevel)
	}
	if !w.batterySupport {
		t.Error("batterySupport should be true")
	}
}

func TestUpdate_DeviceNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"device not found","code":404}`))
	}))
	defer server.Close()

	cfg := testConfig()
	cfg.Bluetooth.APIURL = strings.TrimPrefix(server.URL, "http://")

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.apiReachable {
		t.Error("apiReachable should be true (server responded)")
	}
	if w.deviceFound {
		t.Error("deviceFound should be false for 404")
	}
}

func TestUpdate_AdapterOff(t *testing.T) {
	device := apiResponse{
		Adapter:         &adapterInfo{Available: true, Enabled: false},
		Address:         "AA:BB:CC:DD:EE:FF",
		Name:            "My Device",
		Type:            "Mouse",
		ConnectionState: "Disconnected",
		IsConnected:     false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(device)
	}))
	defer server.Close()

	cfg := testConfig()
	cfg.Bluetooth.APIURL = strings.TrimPrefix(server.URL, "http://")

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.adapterOk {
		t.Error("adapterOk should be false when adapter is disabled")
	}
}

func TestUpdate_AdapterFieldAbsent(t *testing.T) {
	// Real bqc API single-device endpoint does not include adapter info
	batteryLevel := 59
	device := apiResponse{
		// Adapter is nil (absent from JSON)
		Address:         "AA:BB:CC:DD:EE:FF",
		Name:            "WH-1000XM6",
		DisplayName:     "WH-1000XM6",
		Type:            "Headset",
		ConnectionState: "Connected",
		IsConnected:     true,
		Battery:         batteryInfo{Level: &batteryLevel, Supported: true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(device)
	}))
	defer server.Close()

	cfg := testConfig()
	cfg.Bluetooth.APIURL = strings.TrimPrefix(server.URL, "http://")

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.adapterOk {
		t.Error("adapterOk should be true when adapter field is absent (API responded successfully)")
	}
	if !w.connected {
		t.Error("connected should be true")
	}
	if w.deviceType != "Headset" {
		t.Errorf("deviceType = %q, want %q", w.deviceType, "Headset")
	}
}

func TestUpdate_APIUnreachable(t *testing.T) {
	cfg := testConfig()
	// Point to a port that is not listening
	cfg.Bluetooth.APIURL = "127.0.0.1:1"

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Should not return error (graceful handling)
	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v, want nil (graceful handling)", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.apiReachable {
		t.Error("apiReachable should be false when API is down")
	}
}

func TestUpdate_DisplayNameFallback(t *testing.T) {
	device := apiResponse{
		Address:         "AA:BB:CC:DD:EE:FF",
		Name:            "BT Device",
		DisplayName:     "", // empty displayName
		Type:            "Mouse",
		ConnectionState: "Connected",
		IsConnected:     true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(device)
	}))
	defer server.Close()

	cfg := testConfig()
	cfg.Bluetooth.APIURL = strings.TrimPrefix(server.URL, "http://")

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.deviceName != "BT Device" {
		t.Errorf("deviceName = %q, want %q (fallback to name when displayName empty)", w.deviceName, "BT Device")
	}
}

func TestRender_NoPanic(t *testing.T) {
	tests := []struct {
		name   string
		format string
		setup  func(w *Widget)
	}{
		{
			name:   "default state",
			format: "{icon} {name} {battery:20}",
			setup: func(_ *Widget) {
				// Default state
			},
		},
		{
			name:   "connected",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.connected = true
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.connectionState = "Connected"
				w.deviceType = "AudioOutput"
				w.deviceName = "Test Headphones"
				level := 75
				w.batteryLevel = &level
				w.batterySupport = true
			},
		},
		{
			name:   "disconnected",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.connected = false
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.connectionState = "Disconnected"
				w.deviceType = "Mouse"
			},
		},
		{
			name:   "adapter off",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.adapterOk = false
				w.apiReachable = true
			},
		},
		{
			name:   "api unreachable",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.apiReachable = false
			},
		},
		{
			name:   "device not found",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.deviceFound = false
				w.apiReachable = true
				w.adapterOk = true
			},
		},
		{
			name:   "transient connecting",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.connectionState = "Connecting"
				w.deviceType = "Keyboard"
			},
		},
		{
			name:   "transient disconnecting",
			format: "{icon} {name} {battery:20}",
			setup: func(w *Widget) {
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.connectionState = "Disconnecting"
				w.deviceType = "Gamepad"
			},
		},
		{
			name:   "level text format",
			format: "{icon} {level}",
			setup: func(w *Widget) {
				w.connected = true
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.deviceType = "AudioOutput"
				level := 42
				w.batteryLevel = &level
				w.batterySupport = true
			},
		},
		{
			name:   "bar format",
			format: "{bar:30}",
			setup: func(w *Widget) {
				w.connected = true
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.deviceType = "AudioOutput"
				level := 90
				w.batteryLevel = &level
				w.batterySupport = true
			},
		},
		{
			name:   "vertical battery format",
			format: "{battery_v:15}",
			setup: func(w *Widget) {
				w.connected = true
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.deviceType = "AudioOutput"
				level := 50
				w.batteryLevel = &level
				w.batterySupport = true
			},
		},
		{
			name:   "state token",
			format: "{name} - {state}",
			setup: func(w *Widget) {
				w.connected = true
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.connectionState = "Connected"
				w.deviceType = "Mouse"
				w.deviceName = "Test Mouse"
			},
		},
		{
			name:   "icon only",
			format: "{icon}",
			setup: func(w *Widget) {
				w.deviceFound = true
				w.adapterOk = true
				w.apiReachable = true
				w.deviceType = "Keyboard"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Bluetooth.Format = tt.format
			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			tt.setup(w)

			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}
			if img == nil {
				t.Error("Render() returned nil image")
			}
		})
	}
}

func TestDeviceTypeIcon(t *testing.T) {
	tests := []struct {
		deviceType string
		wantIcon   string
	}{
		{"AudioOutput", "bt_headphones"},
		{"Headset", "bt_headphones"},
		{"AudioInput", "bt_microphone"},
		{"Keyboard", "bt_keyboard"},
		{"Mouse", "bt_mouse"},
		{"Gamepad", "bt_gamepad"},
		{"Computer", "bt_computer"},
		{"Phone", "bt_phone"},
		{"Unknown", "bt_generic"},
		{"HID", "bt_generic"},
		{"SomethingElse", "bt_generic"},
		{"", "bt_generic"},
	}

	for _, tt := range tests {
		t.Run(tt.deviceType, func(t *testing.T) {
			got := deviceTypeToIcon(tt.deviceType)
			if got != tt.wantIcon {
				t.Errorf("deviceTypeToIcon(%q) = %q, want %q", tt.deviceType, got, tt.wantIcon)
			}
		})
	}
}

func TestNew_LowBatteryThreshold(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth.LowBatteryThreshold = 25

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.lowBatteryThreshold != 25 {
		t.Errorf("lowBatteryThreshold = %d, want 25", w.lowBatteryThreshold)
	}
	if w.batteryBlink == nil {
		t.Fatal("batteryBlink animator should be initialized")
	}
}

func TestRender_LowBatteryBlink(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth.LowBatteryThreshold = 20
	cfg.Bluetooth.Format = "{icon} {name} {battery:20}"

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Use a very short blink interval so the test doesn't wait
	w.batteryBlink = anim.NewBlinkAnimator(anim.BlinkAlways, 1*time.Millisecond)

	// Set up connected state with low battery
	level := 15
	w.connected = true
	w.deviceFound = true
	w.adapterOk = true
	w.apiReachable = true
	w.deviceType = "AudioOutput"
	w.batteryLevel = &level
	w.batterySupport = true

	// Render many times, allowing blink to toggle between calls
	drawnCount := 0
	skippedCount := 0
	for i := 0; i < 40; i++ {
		time.Sleep(2 * time.Millisecond) // allow blink to toggle
		w.batteryBlink.Update(0)

		visible := w.batteryBlink.ShouldRender()
		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img == nil {
			t.Fatal("Render() returned nil image")
		}

		if visible {
			drawnCount++
		} else {
			skippedCount++
		}
	}

	// Both drawn and skipped should happen at least once
	if drawnCount == 0 {
		t.Error("battery should be drawn at least once during blink cycle")
	}
	if skippedCount == 0 {
		t.Error("battery should be skipped at least once during blink cycle")
	}
}

func TestRender_LowBatteryThresholdZero(t *testing.T) {
	cfg := testConfig()
	// Default threshold = 0 means no blinking
	cfg.Bluetooth.LowBatteryThreshold = 0

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	level := 5 // Very low, but threshold is disabled
	w.connected = true
	w.deviceFound = true
	w.adapterOk = true
	w.apiReachable = true
	w.deviceType = "AudioOutput"
	w.batteryLevel = &level
	w.batterySupport = true

	// Render should always succeed regardless of blink state
	for i := 0; i < 10; i++ {
		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img == nil {
			t.Fatal("Render() returned nil image")
		}
	}
}

func TestRender_LowBatteryAboveThreshold(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth.LowBatteryThreshold = 20

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Use a very short blink interval
	w.batteryBlink = anim.NewBlinkAnimator(anim.BlinkAlways, 1*time.Millisecond)

	level := 50 // Above threshold, should never blink
	w.connected = true
	w.deviceFound = true
	w.adapterOk = true
	w.apiReachable = true
	w.deviceType = "AudioOutput"
	w.batteryLevel = &level
	w.batterySupport = true

	// Force blink to hidden state -- battery should still be drawn
	// because level is above threshold
	time.Sleep(2 * time.Millisecond)
	w.batteryBlink.Update(0)
	for w.batteryBlink.ShouldRender() {
		time.Sleep(2 * time.Millisecond)
		w.batteryBlink.Update(0)
	}

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

func TestRender_LowBatteryBlinkFallbackToIcon(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth.LowBatteryThreshold = 20
	cfg.Bluetooth.Format = "{icon} {name}" // no battery shape -> blink falls to icon

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	w.batteryBlink = anim.NewBlinkAnimator(anim.BlinkAlways, 1*time.Millisecond)

	level := 10
	w.connected = true
	w.deviceFound = true
	w.adapterOk = true
	w.apiReachable = true
	w.deviceType = "AudioOutput"
	w.batteryLevel = &level
	w.batterySupport = true

	drawnCount := 0
	skippedCount := 0
	for i := 0; i < 40; i++ {
		time.Sleep(2 * time.Millisecond)
		w.batteryBlink.Update(0)
		visible := w.batteryBlink.ShouldRender()

		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img == nil {
			t.Fatal("Render() returned nil image")
		}
		if visible {
			drawnCount++
		} else {
			skippedCount++
		}
	}

	if drawnCount == 0 {
		t.Error("icon should be drawn at least once during blink cycle")
	}
	if skippedCount == 0 {
		t.Error("icon should be skipped at least once during blink cycle")
	}
}

func TestRender_LowBatteryBlinkFallbackToName(t *testing.T) {
	cfg := testConfig()
	cfg.Bluetooth.LowBatteryThreshold = 20
	cfg.Bluetooth.Format = "{name}" // no icon, no shape -> blink falls to name

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	w.batteryBlink = anim.NewBlinkAnimator(anim.BlinkAlways, 1*time.Millisecond)

	level := 10
	w.connected = true
	w.deviceFound = true
	w.adapterOk = true
	w.apiReachable = true
	w.deviceType = "AudioOutput"
	w.deviceName = "Test Device"
	w.batteryLevel = &level
	w.batterySupport = true

	drawnCount := 0
	skippedCount := 0
	for i := 0; i < 40; i++ {
		time.Sleep(2 * time.Millisecond)
		w.batteryBlink.Update(0)
		visible := w.batteryBlink.ShouldRender()

		img, err := w.Render()
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if img == nil {
			t.Fatal("Render() returned nil image")
		}
		if visible {
			drawnCount++
		} else {
			skippedCount++
		}
	}

	if drawnCount == 0 {
		t.Error("name should be drawn at least once during blink cycle")
	}
	if skippedCount == 0 {
		t.Error("name should be skipped at least once during blink cycle")
	}
}

func TestNew_IconSetSelection(t *testing.T) {
	tests := []struct {
		name        string
		height      int
		wantSetName string
	}{
		{"small widget (8x8 icons)", 8, "bluetooth_8x8"},
		{"medium widget (12x12 icons)", 14, "bluetooth_12x12"},
		{"large widget (16x16 icons)", 40, "bluetooth_16x16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Position.H = tt.height

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if w.iconSet.Name != tt.wantSetName {
				t.Errorf("iconSet.Name = %q, want %q", w.iconSet.Name, tt.wantSetName)
			}
		})
	}
}

func TestRender_FormatVariations(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"name and level", "{name} {level}"},
		{"icon only", "{icon}"},
		{"vertical battery", "{battery_v:15}"},
		{"all tokens", "{icon} {name} {state} {level} {battery:20}"},
		{"bar format", "{icon} {bar:30}"},
		{"vertical bar", "{bar_v:20}"},
		{"mixed separators", "{icon} | {name} - {level}"},
		{"empty format", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Bluetooth.Format = tt.format

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			// Set up connected state
			level := 65
			w.connected = true
			w.deviceFound = true
			w.adapterOk = true
			w.apiReachable = true
			w.connectionState = "Connected"
			w.deviceType = "AudioOutput"
			w.deviceName = "Test Device"
			w.batteryLevel = &level
			w.batterySupport = true

			img, err := w.Render()
			if err != nil {
				t.Errorf("Render() error = %v", err)
			}
			if img == nil {
				t.Error("Render() returned nil image")
			}
		})
	}
}
