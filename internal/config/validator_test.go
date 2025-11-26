package config

import (
	"strings"
	"testing"
)

func TestValidatorConstants(t *testing.T) {
	if MinDeinitializeTimerMs != 1000 {
		t.Errorf("MinDeinitializeTimerMs = %d, want 1000", MinDeinitializeTimerMs)
	}
	if MaxDeinitializeTimerMs != 60000 {
		t.Errorf("MaxDeinitializeTimerMs = %d, want 60000", MaxDeinitializeTimerMs)
	}
	if MinEventBatchSize != 1 {
		t.Errorf("MinEventBatchSize = %d, want 1", MinEventBatchSize)
	}
	if MaxEventBatchSize != 100 {
		t.Errorf("MaxEventBatchSize = %d, want 100", MaxEventBatchSize)
	}
}

func TestValidBackends(t *testing.T) {
	expectedBackends := []string{"", "gamesense", "direct", "any"}

	for _, backend := range expectedBackends {
		if !ValidBackends[backend] {
			t.Errorf("ValidBackends[%q] should be true", backend)
		}
	}

	invalidBackends := []string{"invalid", "GAMESENSE", "Direct", "foo"}
	for _, backend := range invalidBackends {
		if ValidBackends[backend] {
			t.Errorf("ValidBackends[%q] should be false", backend)
		}
	}
}

func TestValidWidgetTypes(t *testing.T) {
	expectedTypes := []string{
		"clock", "cpu", "memory", "network", "disk",
		"keyboard", "keyboard_layout", "volume", "volume_meter",
		"audio_visualizer", "doom",
	}

	for _, wt := range expectedTypes {
		if !ValidWidgetTypes[wt] {
			t.Errorf("ValidWidgetTypes[%q] should be true", wt)
		}
	}

	invalidTypes := []string{"invalid", "CLOCK", "Clock", "timer", ""}
	for _, wt := range invalidTypes {
		if ValidWidgetTypes[wt] {
			t.Errorf("ValidWidgetTypes[%q] should be false", wt)
		}
	}
}

func TestValidateGlobalConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				Backend: "gamesense",
			},
			wantErr: false,
		},
		{
			name: "invalid backend",
			cfg: Config{
				Backend: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid backend",
		},
		{
			name: "deinitialize timer too low",
			cfg: Config{
				Backend:             "gamesense",
				DeinitializeTimerMs: 500,
			},
			wantErr: true,
			errMsg:  "deinitialize_timer_ms",
		},
		{
			name: "deinitialize timer too high",
			cfg: Config{
				Backend:             "gamesense",
				DeinitializeTimerMs: 100000,
			},
			wantErr: true,
			errMsg:  "deinitialize_timer_ms",
		},
		{
			name: "deinitialize timer valid min",
			cfg: Config{
				Backend:             "gamesense",
				DeinitializeTimerMs: MinDeinitializeTimerMs,
			},
			wantErr: false,
		},
		{
			name: "deinitialize timer valid max",
			cfg: Config{
				Backend:             "gamesense",
				DeinitializeTimerMs: MaxDeinitializeTimerMs,
			},
			wantErr: false,
		},
		{
			name: "event batch size too low",
			cfg: Config{
				Backend:        "gamesense",
				EventBatchSize: -1,
			},
			wantErr: true,
			errMsg:  "event_batch_size",
		},
		{
			name: "event batch size too high",
			cfg: Config{
				Backend:        "gamesense",
				EventBatchSize: 200,
			},
			wantErr: true,
			errMsg:  "event_batch_size",
		},
		{
			name: "event batch size valid",
			cfg: Config{
				Backend:        "gamesense",
				EventBatchSize: 50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGlobalConfig(&tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateDisplayConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid display config",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
			},
			wantErr: false,
		},
		{
			name: "zero width",
			cfg: Config{
				Display:       DisplayConfig{Width: 0, Height: 40},
				RefreshRateMs: 100,
			},
			wantErr: true,
			errMsg:  "width must be positive",
		},
		{
			name: "negative width",
			cfg: Config{
				Display:       DisplayConfig{Width: -10, Height: 40},
				RefreshRateMs: 100,
			},
			wantErr: true,
			errMsg:  "width must be positive",
		},
		{
			name: "zero height",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 0},
				RefreshRateMs: 100,
			},
			wantErr: true,
			errMsg:  "height must be positive",
		},
		{
			name: "negative height",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: -5},
				RefreshRateMs: 100,
			},
			wantErr: true,
			errMsg:  "height must be positive",
		},
		{
			name: "zero refresh rate",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 0,
			},
			wantErr: true,
			errMsg:  "refresh_rate_ms must be positive",
		},
		{
			name: "negative refresh rate",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: -50,
			},
			wantErr: true,
			errMsg:  "refresh_rate_ms must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDisplayConfig(&tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateDisplayConfigSupportedResolutions(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid resolutions",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				SupportedResolutions: []ResolutionConfig{
					{Width: 128, Height: 36},
					{Width: 128, Height: 48},
				},
			},
			wantErr: false,
		},
		{
			name: "empty resolutions",
			cfg: Config{
				Display:              DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs:        100,
				SupportedResolutions: []ResolutionConfig{},
			},
			wantErr: false,
		},
		{
			name: "zero width in resolution",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				SupportedResolutions: []ResolutionConfig{
					{Width: 0, Height: 36},
				},
			},
			wantErr: true,
			errMsg:  "supported_resolutions[0]: width must be positive",
		},
		{
			name: "negative height in resolution",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				SupportedResolutions: []ResolutionConfig{
					{Width: 128, Height: -10},
				},
			},
			wantErr: true,
			errMsg:  "supported_resolutions[0]: height must be positive",
		},
		{
			name: "invalid second resolution",
			cfg: Config{
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				SupportedResolutions: []ResolutionConfig{
					{Width: 128, Height: 36},
					{Width: -5, Height: 48},
				},
			},
			wantErr: true,
			errMsg:  "supported_resolutions[1]: width must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDisplayConfig(&tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWidgetType(t *testing.T) {
	tests := []struct {
		name    string
		widget  WidgetConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid clock type",
			widget:  WidgetConfig{Type: "clock"},
			wantErr: false,
		},
		{
			name:    "valid cpu type",
			widget:  WidgetConfig{Type: "cpu"},
			wantErr: false,
		},
		{
			name:    "valid doom type",
			widget:  WidgetConfig{Type: "doom"},
			wantErr: false,
		},
		{
			name:    "empty type",
			widget:  WidgetConfig{Type: ""},
			wantErr: true,
			errMsg:  "type is required",
		},
		{
			name:    "invalid type",
			widget:  WidgetConfig{Type: "invalid"},
			wantErr: true,
			errMsg:  "invalid type",
		},
		{
			name:    "case sensitive type",
			widget:  WidgetConfig{Type: "Clock"},
			wantErr: true,
			errMsg:  "invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWidgetType(0, &tt.widget)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWidgetProperties(t *testing.T) {
	iface := "eth0"
	disk := "C:"

	tests := []struct {
		name    string
		widget  WidgetConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "clock - no required properties",
			widget:  WidgetConfig{Type: "clock", ID: "clock_0"},
			wantErr: false,
		},
		{
			name:    "cpu - no required properties",
			widget:  WidgetConfig{Type: "cpu", ID: "cpu_0"},
			wantErr: false,
		},
		{
			name:    "network - missing interface",
			widget:  WidgetConfig{Type: "network", ID: "network_0"},
			wantErr: true,
			errMsg:  "interface is required",
		},
		{
			name:    "network - empty interface",
			widget:  WidgetConfig{Type: "network", ID: "network_0", Interface: new(string)},
			wantErr: true,
			errMsg:  "interface is required",
		},
		{
			name:    "network - valid interface",
			widget:  WidgetConfig{Type: "network", ID: "network_0", Interface: &iface},
			wantErr: false,
		},
		{
			name:    "disk - missing disk",
			widget:  WidgetConfig{Type: "disk", ID: "disk_0"},
			wantErr: true,
			errMsg:  "disk is required",
		},
		{
			name:    "disk - empty disk",
			widget:  WidgetConfig{Type: "disk", ID: "disk_0", Disk: new(string)},
			wantErr: true,
			errMsg:  "disk is required",
		},
		{
			name:    "disk - valid disk",
			widget:  WidgetConfig{Type: "disk", ID: "disk_0", Disk: &disk},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWidgetProperties(0, &tt.widget)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWidgets(t *testing.T) {
	iface := "eth0"

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid widgets",
			cfg: Config{
				Widgets: []WidgetConfig{
					{Type: "clock", Enabled: BoolPtr(true)},
				},
			},
			wantErr: false,
		},
		{
			name: "no widgets",
			cfg: Config{
				Widgets: []WidgetConfig{},
			},
			wantErr: true,
			errMsg:  "at least one widget",
		},
		{
			name: "disabled widget skips property validation",
			cfg: Config{
				Widgets: []WidgetConfig{
					{Type: "network", Enabled: BoolPtr(false)}, // Missing interface, but disabled
				},
			},
			wantErr: false,
		},
		{
			name: "enabled widget requires properties",
			cfg: Config{
				Widgets: []WidgetConfig{
					{Type: "network", Enabled: BoolPtr(true)}, // Missing interface
				},
			},
			wantErr: true,
			errMsg:  "interface is required",
		},
		{
			name: "multiple widgets with one invalid",
			cfg: Config{
				Widgets: []WidgetConfig{
					{Type: "clock", Enabled: BoolPtr(true)},
					{Type: "network", Enabled: BoolPtr(true), Interface: &iface},
					{Type: "invalid", Enabled: BoolPtr(true)},
				},
			},
			wantErr: true,
			errMsg:  "invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWidgets(&tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWidgetsGeneratesIDs(t *testing.T) {
	cfg := &Config{
		Widgets: []WidgetConfig{
			{Type: "clock"},
			{Type: "cpu"},
			{Type: "clock"},
		},
	}

	err := validateWidgets(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"clock_0", "cpu_0", "clock_1"}
	for i, w := range cfg.Widgets {
		if w.ID != expected[i] {
			t.Errorf("widgets[%d].ID = %q, want %q", i, w.ID, expected[i])
		}
	}
}

func TestValidate(t *testing.T) {
	iface := "eth0"

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "fully valid config",
			cfg: Config{
				Backend:       "gamesense",
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{Type: "clock"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid backend",
			cfg: Config{
				Backend:       "invalid",
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{Type: "clock"},
				},
			},
			wantErr: true,
			errMsg:  "invalid backend",
		},
		{
			name: "invalid display",
			cfg: Config{
				Backend:       "gamesense",
				Display:       DisplayConfig{Width: 0, Height: 40},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{Type: "clock"},
				},
			},
			wantErr: true,
			errMsg:  "width must be positive",
		},
		{
			name: "invalid widget",
			cfg: Config{
				Backend:       "gamesense",
				Display:       DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs: 100,
				Widgets: []WidgetConfig{
					{Type: "invalid"},
				},
			},
			wantErr: true,
			errMsg:  "invalid type",
		},
		{
			name: "complex valid config",
			cfg: Config{
				Backend:             "any",
				Display:             DisplayConfig{Width: 128, Height: 40},
				RefreshRateMs:       50,
				DeinitializeTimerMs: 5000,
				EventBatchSize:      20,
				SupportedResolutions: []ResolutionConfig{
					{Width: 128, Height: 36},
				},
				Widgets: []WidgetConfig{
					{Type: "clock", Enabled: BoolPtr(true)},
					{Type: "network", Enabled: BoolPtr(true), Interface: &iface},
					{Type: "cpu", Enabled: BoolPtr(false)},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
