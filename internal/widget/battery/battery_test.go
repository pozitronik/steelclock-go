package battery

import (
	"testing"
	"time"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestGetColorForLevel(t *testing.T) {
	w := &Widget{
		criticalThreshold: 10,
		lowThreshold:      20,
		colorNormal:       255,
		colorLow:          200,
		colorCritical:     150,
	}

	tests := []struct {
		name       string
		percentage int
		wantColor  uint8
	}{
		{"critical at threshold", 10, 150},
		{"critical below threshold", 5, 150},
		{"critical at zero", 0, 150},
		{"low at threshold", 20, 200},
		{"low between thresholds", 15, 200},
		{"normal above low", 21, 255},
		{"normal at 50", 50, 255},
		{"normal at 100", 100, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.getColorForLevel(tt.percentage)
			if got != tt.wantColor {
				t.Errorf("getColorForLevel(%d) = %d, want %d", tt.percentage, got, tt.wantColor)
			}
		})
	}
}

func TestShouldShowIndicator(t *testing.T) {
	w := &Widget{}

	t.Run("inactive state never shown", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeAlways}
		if w.shouldShowIndicator(state, false) {
			t.Error("should not show when inactive")
		}
	})

	t.Run("always mode", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeAlways}
		if !w.shouldShowIndicator(state, true) {
			t.Error("always mode should show when active")
		}
	})

	t.Run("never mode", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeNever}
		if w.shouldShowIndicator(state, true) {
			t.Error("never mode should not show")
		}
	})

	t.Run("blink mode", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeBlink}
		if !w.shouldShowIndicator(state, true) {
			t.Error("blink mode should show when active")
		}
	})

	t.Run("notify mode within duration", func(t *testing.T) {
		state := &indicatorState{
			mode:        indicatorModeNotify,
			notifyUntil: time.Now().Add(1 * time.Hour),
		}
		if !w.shouldShowIndicator(state, true) {
			t.Error("notify mode should show within duration")
		}
	})

	t.Run("notify mode after expiry", func(t *testing.T) {
		state := &indicatorState{
			mode:        indicatorModeNotify,
			notifyUntil: time.Now().Add(-1 * time.Hour),
		}
		if w.shouldShowIndicator(state, true) {
			t.Error("notify mode should not show after expiry")
		}
	})

	t.Run("notify_blink mode within duration", func(t *testing.T) {
		state := &indicatorState{
			mode:        indicatorModeNotifyBlink,
			notifyUntil: time.Now().Add(1 * time.Hour),
		}
		if !w.shouldShowIndicator(state, true) {
			t.Error("notify_blink mode should show within duration")
		}
	})

	t.Run("notify_blink mode after expiry", func(t *testing.T) {
		state := &indicatorState{
			mode:        indicatorModeNotifyBlink,
			notifyUntil: time.Now().Add(-1 * time.Hour),
		}
		if w.shouldShowIndicator(state, true) {
			t.Error("notify_blink mode should not show after expiry")
		}
	})
}

func TestShouldBlinkIndicator(t *testing.T) {
	w := &Widget{}

	t.Run("always mode does not blink", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeAlways}
		// Not a blink mode, should never return true
		if w.shouldBlinkIndicator(state) {
			t.Error("always mode should not blink")
		}
	})

	t.Run("never mode does not blink", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeNever}
		if w.shouldBlinkIndicator(state) {
			t.Error("never mode should not blink")
		}
	})

	t.Run("notify mode does not blink", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeNotify}
		if w.shouldBlinkIndicator(state) {
			t.Error("notify mode should not blink")
		}
	})

	// Blink modes depend on time.Now().Second()%2, so we test behavior is deterministic
	// for both blink and notify_blink
	t.Run("blink mode returns boolean", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeBlink}
		// Just verify it runs without panic and returns a bool (time-dependent)
		_ = w.shouldBlinkIndicator(state)
	})

	t.Run("notify_blink mode returns boolean", func(t *testing.T) {
		state := &indicatorState{mode: indicatorModeNotifyBlink}
		_ = w.shouldBlinkIndicator(state)
	})
}

func TestSelectBatteryIconSet(t *testing.T) {
	tests := []struct {
		name   string
		height int
	}{
		{"tiny height 8", 8},
		{"small height 13", 13},
		{"medium height 14", 14},
		{"medium height 21", 21},
		{"large height 22", 22},
		{"large height 40", 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iconSet := selectBatteryIconSet(tt.height)
			if iconSet == nil {
				t.Fatalf("selectBatteryIconSet(%d) returned nil", tt.height)
			}
		})
	}
}

func TestFormatMinutes(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{0, "-"},
		{-5, "-"},
		{30, "30m"},
		{59, "59m"},
		{60, "1h 0m"},
		{90, "1h 30m"},
		{125, "2h 5m"},
		{1440, "24h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatMinutes(tt.minutes)
			if got != tt.want {
				t.Errorf("formatMinutes(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestGetVisibleStatusIcon(t *testing.T) {
	t.Run("charging has highest priority", func(t *testing.T) {
		w := &Widget{
			chargingState: indicatorState{mode: indicatorModeAlways},
			pluggedState:  indicatorState{mode: indicatorModeAlways},
			economyState:  indicatorState{mode: indicatorModeAlways},
		}
		status := Status{IsCharging: true, IsPluggedIn: true, IsEconomyMode: true}

		icon, state := w.getVisibleStatusIcon(status)
		if icon != "charging" {
			t.Errorf("icon = %q, want 'charging' (highest priority)", icon)
		}
		if state == nil {
			t.Error("state should not be nil")
		}
	})

	t.Run("economy when not charging", func(t *testing.T) {
		w := &Widget{
			chargingState: indicatorState{mode: indicatorModeAlways},
			pluggedState:  indicatorState{mode: indicatorModeAlways},
			economyState:  indicatorState{mode: indicatorModeAlways},
		}
		status := Status{IsCharging: false, IsPluggedIn: true, IsEconomyMode: true}

		icon, _ := w.getVisibleStatusIcon(status)
		if icon != "economy" {
			t.Errorf("icon = %q, want 'economy'", icon)
		}
	})

	t.Run("plugged when not charging and not economy", func(t *testing.T) {
		w := &Widget{
			chargingState: indicatorState{mode: indicatorModeAlways},
			pluggedState:  indicatorState{mode: indicatorModeAlways},
			economyState:  indicatorState{mode: indicatorModeAlways},
		}
		status := Status{IsCharging: false, IsPluggedIn: true, IsEconomyMode: false}

		icon, _ := w.getVisibleStatusIcon(status)
		if icon != "ac_power" {
			t.Errorf("icon = %q, want 'ac_power'", icon)
		}
	})

	t.Run("no icon when nothing active", func(t *testing.T) {
		w := &Widget{
			chargingState: indicatorState{mode: indicatorModeAlways},
			pluggedState:  indicatorState{mode: indicatorModeAlways},
			economyState:  indicatorState{mode: indicatorModeAlways},
		}
		status := Status{IsCharging: false, IsPluggedIn: false, IsEconomyMode: false}

		icon, state := w.getVisibleStatusIcon(status)
		if icon != "" {
			t.Errorf("icon = %q, want empty", icon)
		}
		if state != nil {
			t.Error("state should be nil")
		}
	})

	t.Run("never mode hides indicator", func(t *testing.T) {
		w := &Widget{
			chargingState: indicatorState{mode: indicatorModeNever},
			pluggedState:  indicatorState{mode: indicatorModeAlways},
			economyState:  indicatorState{mode: indicatorModeNever},
		}
		status := Status{IsCharging: true, IsPluggedIn: true, IsEconomyMode: true}

		icon, _ := w.getVisibleStatusIcon(status)
		// Charging is "never", economy is "never", plugged is "always"
		if icon != "ac_power" {
			t.Errorf("icon = %q, want 'ac_power' (charging and economy hidden)", icon)
		}
	})
}

func TestExpandFormat(t *testing.T) {
	w := &Widget{
		chargingState:     indicatorState{mode: indicatorModeAlways},
		pluggedState:      indicatorState{mode: indicatorModeAlways},
		economyState:      indicatorState{mode: indicatorModeAlways},
		criticalThreshold: 10,
		lowThreshold:      20,
	}

	tests := []struct {
		name   string
		format string
		status Status
		want   string
	}{
		{
			"simple percent",
			"{percent}%",
			Status{Percentage: 85},
			"85%",
		},
		{
			"pct alias",
			"{pct}%",
			Status{Percentage: 42},
			"42%",
		},
		{
			"time to empty",
			"TTL: {time_left}",
			Status{Percentage: 50, TimeToEmpty: 90},
			"TTL: 1h 30m",
		},
		{
			"time to full while charging",
			"ETA: {time}",
			Status{Percentage: 50, IsCharging: true, TimeToFull: 45, TimeToEmpty: 120},
			"ETA: 45m",
		},
		{
			"level normal",
			"{level}",
			Status{Percentage: 50},
			"normal",
		},
		{
			"level low",
			"{level}",
			Status{Percentage: 15},
			"low",
		},
		{
			"level critical",
			"{level}",
			Status{Percentage: 5},
			"critical",
		},
		{
			"charging status text",
			"{percent}% {status}",
			Status{Percentage: 60, IsCharging: true},
			"60% CHG",
		},
		{
			"empty tokens collapse spaces",
			"{percent}% {status}",
			Status{Percentage: 60, IsCharging: false},
			"60%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.expandFormat(tt.format, tt.status)
			if got != tt.want {
				t.Errorf("expandFormat(%q, ...) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

func TestNew_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "battery",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.displayMode != "icon" {
		t.Errorf("displayMode = %q, want 'icon'", w.displayMode)
	}
	if !w.showPercentage {
		t.Error("showPercentage should default to true")
	}
	if w.lowThreshold != 20 {
		t.Errorf("lowThreshold = %d, want 20", w.lowThreshold)
	}
	if w.criticalThreshold != 10 {
		t.Errorf("criticalThreshold = %d, want 10", w.criticalThreshold)
	}
	if w.textFormat != "{percent}%" {
		t.Errorf("textFormat = %q, want '{percent}%%'", w.textFormat)
	}
	if w.chargingState.mode != indicatorModeAlways {
		t.Errorf("chargingState.mode = %q, want 'always'", w.chargingState.mode)
	}
	if w.economyState.mode != indicatorModeBlink {
		t.Errorf("economyState.mode = %q, want 'blink'", w.economyState.mode)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	showPct := false
	cfg := config.WidgetConfig{
		Type:    "battery",
		Enabled: config.BoolPtr(true),
		Mode:    "text",
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Battery: &config.BatteryConfig{
			ShowPercentage:    &showPct,
			Orientation:       "vertical",
			LowThreshold:      30,
			CriticalThreshold: 15,
		},
		Text: &config.TextConfig{
			Format: "{percent}% - {status_full}",
		},
		PowerStatus: &config.PowerStatusConfig{
			ShowCharging:   "notify",
			ShowPlugged:    "never",
			ShowEconomy:    "always",
			NotifyDuration: 120,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if w.displayMode != "text" {
		t.Errorf("displayMode = %q, want 'text'", w.displayMode)
	}
	if w.showPercentage {
		t.Error("showPercentage should be false")
	}
	if w.orientation != "vertical" {
		t.Errorf("orientation = %q, want 'vertical'", w.orientation)
	}
	if w.lowThreshold != 30 {
		t.Errorf("lowThreshold = %d, want 30", w.lowThreshold)
	}
	if w.criticalThreshold != 15 {
		t.Errorf("criticalThreshold = %d, want 15", w.criticalThreshold)
	}
	if w.textFormat != "{percent}% - {status_full}" {
		t.Errorf("textFormat = %q", w.textFormat)
	}
	if w.chargingState.mode != "notify" {
		t.Errorf("chargingState.mode = %q, want 'notify'", w.chargingState.mode)
	}
	if w.pluggedState.mode != "never" {
		t.Errorf("pluggedState.mode = %q, want 'never'", w.pluggedState.mode)
	}
	if w.economyState.mode != "always" {
		t.Errorf("economyState.mode = %q, want 'always'", w.economyState.mode)
	}
	if w.chargingState.notifyDuration != 120*time.Second {
		t.Errorf("notifyDuration = %v, want 120s", w.chargingState.notifyDuration)
	}
}

func TestRender_NoData(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "battery",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestRender_NoBattery(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "battery",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w.mu.Lock()
	w.hasData = true
	w.currentStatus = Status{HasBattery: false}
	w.mu.Unlock()

	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestRender_AllModes(t *testing.T) {
	modes := []string{"icon", "text", "bar", "gauge", "graph"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:    "battery",
				Enabled: config.BoolPtr(true),
				Mode:    mode,
				Position: config.PositionConfig{
					X: 0, Y: 0, W: 64, H: 40,
				},
			}

			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			w.mu.Lock()
			w.hasData = true
			w.currentStatus = Status{
				HasBattery:  true,
				Percentage:  75,
				IsCharging:  true,
				IsPluggedIn: true,
			}
			w.mu.Unlock()

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
