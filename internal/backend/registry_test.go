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
	// Clear registry
	for k := range registry {
		delete(registry, k)
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
