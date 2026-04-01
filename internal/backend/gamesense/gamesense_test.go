package gamesense

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBytesToInts(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []int
	}{
		{"empty", []byte{}, []int{}},
		{"single byte", []byte{42}, []int{42}},
		{"zero byte", []byte{0}, []int{0}},
		{"max byte", []byte{255}, []int{255}},
		{"multiple bytes", []byte{0, 128, 255}, []int{0, 128, 255}},
		{"bitmap-like data", []byte{0xFF, 0x00, 0xAA, 0x55}, []int{255, 0, 170, 85}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesToInts(tt.data)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("bytesToInts()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestImageDataKey(t *testing.T) {
	tests := []struct {
		width, height int
		want          string
	}{
		{128, 40, "image-data-128x40"},
		{128, 52, "image-data-128x52"},
		{128, 64, "image-data-128x64"},
		{64, 20, "image-data-64x20"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			c := &Client{displayWidth: tt.width, displayHeight: tt.height}
			got := c.imageDataKey()
			if got != tt.want {
				t.Errorf("imageDataKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpectedBitmapSize(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		want          int
	}{
		{"128x40", 128, 40, 640},       // 5120 bits / 8 = 640 bytes
		{"128x52", 128, 52, 832},       // 6656 bits / 8 = 832 bytes
		{"128x64", 128, 64, 1024},      // 8192 bits / 8 = 1024 bytes
		{"non-byte-aligned", 10, 3, 4}, // 30 bits -> (30+7)/8 = 4 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{displayWidth: tt.width, displayHeight: tt.height}
			got := c.expectedBitmapSize()
			if got != tt.want {
				t.Errorf("expectedBitmapSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGameName(t *testing.T) {
	c := &Client{gameName: "STEELCLOCK"}
	if c.GameName() != "STEELCLOCK" {
		t.Errorf("GameName() = %q, want %q", c.GameName(), "STEELCLOCK")
	}
}

// newTestClient creates a Client pointing at a test HTTP server.
func newTestClient(serverURL string) *Client {
	return &Client{
		baseURL:       serverURL,
		gameName:      "TEST_GAME",
		displayWidth:  128,
		displayHeight: 40,
		httpClient:    http.DefaultClient,
	}
}

func TestRegisterGame(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var received map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/game_metadata" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			json.NewDecoder(r.Body).Decode(&received)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.RegisterGame("TestDev", 15000)
		if err != nil {
			t.Fatalf("RegisterGame() error = %v", err)
		}

		if received["game"] != "TEST_GAME" {
			t.Errorf("game = %v, want TEST_GAME", received["game"])
		}
		if received["developer"] != "TestDev" {
			t.Errorf("developer = %v, want TestDev", received["developer"])
		}
		if received["deinitialize_timer_length_ms"] != 15000.0 {
			t.Errorf("deinitialize_timer = %v, want 15000", received["deinitialize_timer_length_ms"])
		}
	})

	t.Run("no deinitialize timer when zero", func(t *testing.T) {
		var received map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&received)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.RegisterGame("TestDev", 0)
		if err != nil {
			t.Fatalf("RegisterGame() error = %v", err)
		}
		if _, exists := received["deinitialize_timer_length_ms"]; exists {
			t.Error("deinitialize_timer_length_ms should not be present when timer is 0")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.RegisterGame("TestDev", 0)
		if err == nil {
			t.Fatal("expected error for server 500")
		}
	})
}

func TestSendScreenData(t *testing.T) {
	t.Run("correct bitmap size", func(t *testing.T) {
		var receivedPayload map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/game_event" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			json.NewDecoder(r.Body).Decode(&receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		bitmap := make([]byte, c.expectedBitmapSize()) // 640 bytes for 128x40
		bitmap[0] = 0xFF

		err := c.SendScreenData("SCREEN", bitmap)
		if err != nil {
			t.Fatalf("SendScreenData() error = %v", err)
		}

		if receivedPayload["game"] != "TEST_GAME" {
			t.Errorf("game = %v", receivedPayload["game"])
		}
		if receivedPayload["event"] != "SCREEN" {
			t.Errorf("event = %v", receivedPayload["event"])
		}
	})

	t.Run("wrong bitmap size", func(t *testing.T) {
		c := newTestClient("http://unused")
		bitmap := make([]byte, 10) // wrong size

		err := c.SendScreenData("SCREEN", bitmap)
		if err == nil {
			t.Fatal("expected error for wrong bitmap size")
		}
		if !strings.Contains(err.Error(), "invalid bitmap size") {
			t.Errorf("error = %q, should mention 'invalid bitmap size'", err.Error())
		}
	})
}

func TestSendScreenDataMultiRes(t *testing.T) {
	t.Run("empty resolution data", func(t *testing.T) {
		c := newTestClient("http://unused")
		err := c.SendScreenDataMultiRes("SCREEN", map[string][]byte{})
		if err == nil {
			t.Fatal("expected error for empty resolution data")
		}
	})

	t.Run("sends resolution data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		data := map[string][]byte{
			"image-data-128x40": make([]byte, 640),
		}

		err := c.SendScreenDataMultiRes("SCREEN", data)
		if err != nil {
			t.Fatalf("SendScreenDataMultiRes() error = %v", err)
		}
	})
}

func TestSendMultipleScreenData(t *testing.T) {
	t.Run("empty frames", func(t *testing.T) {
		c := newTestClient("http://unused")
		err := c.SendMultipleScreenData("SCREEN", [][]byte{})
		if err != nil {
			t.Fatalf("expected nil error for empty frames, got %v", err)
		}
	})

	t.Run("wrong bitmap size", func(t *testing.T) {
		c := newTestClient("http://unused")
		err := c.SendMultipleScreenData("SCREEN", [][]byte{make([]byte, 10)})
		if err == nil {
			t.Fatal("expected error for wrong bitmap size")
		}
	})

	t.Run("sends last frame", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		bitmapSize := c.expectedBitmapSize()
		frames := [][]byte{
			make([]byte, bitmapSize),
			make([]byte, bitmapSize),
		}

		err := c.SendMultipleScreenData("SCREEN", frames)
		if err != nil {
			t.Fatalf("SendMultipleScreenData() error = %v", err)
		}
	})
}

func TestSendHeartbeat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/game_heartbeat" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.SendHeartbeat()
		if err != nil {
			t.Fatalf("SendHeartbeat() error = %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.SendHeartbeat()
		if err == nil {
			t.Fatal("expected error for server 500")
		}
	})
}

func TestSupportsMultipleEvents(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/supports_multiple_game_events" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		if !c.SupportsMultipleEvents() {
			t.Error("expected true for 200 OK")
		}
	})

	t.Run("not supported (404)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		if c.SupportsMultipleEvents() {
			t.Error("expected false for 404")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		c := newTestClient("http://127.0.0.1:1")
		if c.SupportsMultipleEvents() {
			t.Error("expected false for unreachable server")
		}
	})
}

func TestRemoveGame(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/remove_game" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.RemoveGame()
		if err != nil {
			t.Fatalf("RemoveGame() error = %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.RemoveGame()
		if err == nil {
			t.Fatal("expected error for server 500")
		}
	})
}

func TestBindScreenEvent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var received map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/bind_game_event" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			json.NewDecoder(r.Body).Decode(&received)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := newTestClient(server.URL)
		err := c.BindScreenEvent("SCREEN", "screened-128x40")
		if err != nil {
			t.Fatalf("BindScreenEvent() error = %v", err)
		}

		if received["game"] != "TEST_GAME" {
			t.Errorf("game = %v", received["game"])
		}
		if received["event"] != "SCREEN" {
			t.Errorf("event = %v", received["event"])
		}
	})
}
