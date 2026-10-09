package backend

import (
	"errors"
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// mockBackend implements display.Backend for testing.
type mockBackend struct{}

func (m *mockBackend) SendScreenData(string, []byte) error                    { return nil }
func (m *mockBackend) SendScreenDataMultiRes(string, map[string][]byte) error { return nil }
func (m *mockBackend) SendMultipleScreenData(string, [][]byte) error          { return nil }
func (m *mockBackend) SendHeartbeat() error                                   { return nil }
func (m *mockBackend) SupportsMultipleEvents() bool                           { return false }
func (m *mockBackend) RegisterGame(string, int) error                         { return nil }
func (m *mockBackend) BindScreenEvent(string, string) error                   { return nil }
func (m *mockBackend) RemoveGame() error                                      { return nil }

// saveAndClearRegistry saves the current registry state and clears it for isolated testing.
// Returns a cleanup function that restores the original state.
func saveAndClearRegistry() func() {
	registryMu.Lock()
	saved := make(map[string]registration, len(registry))
	for k, v := range registry {
		saved[k] = v
	}
	savedProbers := make(map[string]Prober, len(probers))
	for k, v := range probers {
		savedProbers[k] = v
	}
	// Clear registry
	for k := range registry {
		delete(registry, k)
	}
	for k := range probers {
		delete(probers, k)
	}
	registryMu.Unlock()

	return func() {
		registryMu.Lock()
		for k := range registry {
			delete(registry, k)
		}
		for k, v := range saved {
			registry[k] = v
		}
		for k := range probers {
			delete(probers, k)
		}
		for k, v := range savedProbers {
			probers[k] = v
		}
		registryMu.Unlock()
	}
}

func TestRegister(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	factory := func(cfg *config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}

	Register("test_backend", factory, 10)

	if !IsRegistered("test_backend") {
		t.Error("test_backend should be registered")
	}
}

func TestIsRegistered(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("exists", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 10)

	if !IsRegistered("exists") {
		t.Error("'exists' should be registered")
	}
	if IsRegistered("not_exists") {
		t.Error("'not_exists' should not be registered")
	}
}

func TestRegisteredTypes(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("beta", func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }, 10)
	Register("alpha", func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }, 20)
	Register("gamma", func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }, 5)

	types := RegisteredTypes()
	if len(types) != 3 {
		t.Fatalf("got %d types, want 3", len(types))
	}

	// Should be sorted alphabetically
	if types[0] != "alpha" || types[1] != "beta" || types[2] != "gamma" {
		t.Errorf("types = %v, want [alpha beta gamma] (sorted)", types)
	}
}

func TestRegisteredTypesList(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("one", func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }, 10)
	Register("two", func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }, 20)

	list := RegisteredTypesList()
	if !strings.Contains(list, "one") || !strings.Contains(list, "two") {
		t.Errorf("list = %q, should contain 'one' and 'two'", list)
	}
	if !strings.Contains(list, ", ") {
		t.Errorf("list = %q, should be comma-separated", list)
	}
}

func TestRegisteredTypes_Empty(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	types := RegisteredTypes()
	if len(types) != 0 {
		t.Errorf("got %d types, want 0 for empty registry", len(types))
	}
}

func TestCreateByName(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("mock", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 10)

	result, err := CreateByName("mock", &config.Config{})
	if err != nil {
		t.Fatalf("CreateByName() error = %v", err)
	}
	if result.Name != "mock" {
		t.Errorf("result.Name = %q, want %q", result.Name, "mock")
	}
	if result.Backend == nil {
		t.Error("result.Backend is nil")
	}
}

func TestCreateByName_Unknown(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	_, err := CreateByName("nonexistent", &config.Config{})
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
	if !strings.Contains(err.Error(), "unknown backend") {
		t.Errorf("error = %q, should mention 'unknown backend'", err.Error())
	}
}

func TestCreateByName_FactoryError(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	expectedErr := errors.New("factory failed")
	Register("failing", func(*config.Config) (display.Backend, error) {
		return nil, expectedErr
	}, 10)

	_, err := CreateByName("failing", &config.Config{})
	if !errors.Is(err, expectedErr) {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

func TestCreate_AutoSelection(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	// Register backends with different priorities
	Register("low_priority", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 100)
	Register("high_priority", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 1)

	cfg := &config.Config{Backend: ""} // empty = auto-select
	result, err := Create(cfg)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Name != "high_priority" {
		t.Errorf("auto-selection picked %q, want %q (highest priority)", result.Name, "high_priority")
	}
}

func TestCreate_ExplicitBackend(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("explicit", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 10)

	cfg := &config.Config{Backend: "explicit"}
	result, err := Create(cfg)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Name != "explicit" {
		t.Errorf("result.Name = %q, want %q", result.Name, "explicit")
	}
}

func TestCreate_AutoFallback(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	// First backend fails, second succeeds
	Register("failing", func(*config.Config) (display.Backend, error) {
		return nil, errors.New("fails")
	}, 1) // higher priority (lower number)
	Register("working", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 10)

	cfg := &config.Config{Backend: ""}
	result, err := Create(cfg)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Name != "working" {
		t.Errorf("fallback should pick 'working', got %q", result.Name)
	}
}

func TestCreate_AllFail(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("fail1", func(*config.Config) (display.Backend, error) {
		return nil, errors.New("fail1")
	}, 1)
	Register("fail2", func(*config.Config) (display.Backend, error) {
		return nil, errors.New("fail2")
	}, 2)

	cfg := &config.Config{Backend: ""}
	_, err := Create(cfg)
	if err == nil {
		t.Fatal("expected error when all backends fail")
	}
	if !strings.Contains(err.Error(), "all backends failed") {
		t.Errorf("error = %q, should mention 'all backends failed'", err.Error())
	}
}

func TestCreate_NoBackends(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	cfg := &config.Config{Backend: ""}
	_, err := Create(cfg)
	if err == nil {
		t.Fatal("expected error when no backends registered")
	}
	if !strings.Contains(err.Error(), "no backends registered") {
		t.Errorf("error = %q, should mention 'no backends registered'", err.Error())
	}
}

func TestCreateExcluding(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("primary", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 1)
	Register("fallback", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 10)

	result, err := CreateExcluding(&config.Config{}, "primary")
	if err != nil {
		t.Fatalf("CreateExcluding() error = %v", err)
	}
	if result.Name != "fallback" {
		t.Errorf("result.Name = %q, want %q (primary was excluded)", result.Name, "fallback")
	}
}

func TestCreateExcluding_AllExcluded(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("only", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	}, 1)

	_, err := CreateExcluding(&config.Config{}, "only")
	if err == nil {
		t.Fatal("expected error when all backends excluded")
	}
}

func TestRegisterExplicit_NotAutoSelected(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("hardware", func(*config.Config) (display.Backend, error) {
		return nil, errors.New("device missing")
	}, 1)
	RegisterExplicit("explicit_only", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	})

	// Auto-selection must fail rather than fall back to the explicit-only backend
	if _, err := Create(&config.Config{Backend: ""}); err == nil {
		t.Fatal("auto-selection picked an explicit-only backend, want error")
	}

	// Failover must not reach it either
	if _, err := CreateExcluding(&config.Config{}, "hardware"); err == nil {
		t.Fatal("failover picked an explicit-only backend, want error")
	}

	// But naming it explicitly still works
	result, err := Create(&config.Config{Backend: "explicit_only"})
	if err != nil {
		t.Fatalf("Create() with explicit name error = %v", err)
	}
	if result.Name != "explicit_only" {
		t.Errorf("result.Name = %q, want %q", result.Name, "explicit_only")
	}
	if !IsRegistered("explicit_only") {
		t.Error("explicit-only backend should still be registered (valid in config)")
	}
}

func TestRegisterExplicit_OnlyExplicitRegistered(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	RegisterExplicit("explicit_only", func(*config.Config) (display.Backend, error) {
		return &mockBackend{}, nil
	})

	_, err := Create(&config.Config{Backend: ""})
	if err == nil {
		t.Fatal("expected error when no auto-selectable backend is registered")
	}
	if !strings.Contains(err.Error(), "no backends registered") {
		t.Errorf("error = %q, should mention 'no backends registered'", err.Error())
	}
}

func TestAvailableByName(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	factory := func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }
	Register("no_probe", factory, 10)
	Register("probe_up", factory, 20)
	Register("probe_down", factory, 30)
	RegisterProber("probe_up", func(*config.Config) bool { return true })
	RegisterProber("probe_down", func(*config.Config) bool { return false })

	tests := []struct {
		name string
		want bool
	}{
		{"unregistered", false},
		{"no_probe", true},
		{"probe_up", true},
		{"probe_down", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AvailableByName(tt.name, &config.Config{}); got != tt.want {
				t.Errorf("AvailableByName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestAvailableByName_PassesConfig(t *testing.T) {
	restore := saveAndClearRegistry()
	defer restore()

	Register("cfg_probe", func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }, 10)
	RegisterProber("cfg_probe", func(cfg *config.Config) bool { return cfg.GameName == "present" })

	if !AvailableByName("cfg_probe", &config.Config{GameName: "present"}) {
		t.Error("probe should see the given config")
	}
	if AvailableByName("cfg_probe", &config.Config{GameName: "absent"}) {
		t.Error("probe should see the given config")
	}
}

func TestAvailable(t *testing.T) {
	factory := func(*config.Config) (display.Backend, error) { return &mockBackend{}, nil }

	tests := []struct {
		name    string
		setup   func()
		backend string
		want    bool
	}{
		{
			name: "explicit backend uses its own probe",
			setup: func() {
				Register("a", factory, 10)
				Register("b", factory, 20)
				RegisterProber("a", func(*config.Config) bool { return false })
			},
			backend: "a",
			want:    false,
		},
		{
			name: "auto: any available auto-selectable backend",
			setup: func() {
				Register("a", factory, 10)
				Register("b", factory, 20)
				RegisterProber("a", func(*config.Config) bool { return false })
				RegisterProber("b", func(*config.Config) bool { return true })
			},
			want: true,
		},
		{
			name: "auto: nothing available",
			setup: func() {
				Register("a", factory, 10)
				RegisterProber("a", func(*config.Config) bool { return false })
			},
			want: false,
		},
		{
			name: "auto: explicit-only backends are not considered",
			setup: func() {
				RegisterExplicit("explicit_only", factory)
			},
			want: false,
		},
		{
			name:    "explicit backend that is not registered",
			setup:   func() {},
			backend: "missing",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := saveAndClearRegistry()
			defer restore()
			tt.setup()

			if got := Available(&config.Config{Backend: tt.backend}); got != tt.want {
				t.Errorf("Available() = %v, want %v", got, tt.want)
			}
		})
	}
}
