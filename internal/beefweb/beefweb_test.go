package beefweb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPlaybackStateString(t *testing.T) {
	tests := []struct {
		state PlaybackState
		want  string
	}{
		{StateStopped, "Stopped"},
		{StatePlaying, "Playing"},
		{StatePaused, "Paused"},
		{PlaybackState(99), "Stopped"}, // Unknown state defaults to Stopped
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("PlaybackState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"default URL", ""},
		{"custom URL", "http://192.168.1.100:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.baseURL)
			if client == nil {
				t.Error("New() returned nil")
			}
		})
	}
}

func TestHTTPClient_IsAvailable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"server available", http.StatusOK, true},
		{"server not found", http.StatusNotFound, false},
		{"server error", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := New(server.URL)
			if got := client.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPClient_IsAvailable_ServerDown(t *testing.T) {
	// Use an invalid URL to simulate server being down
	client := New("http://localhost:1")
	if client.IsAvailable() {
		t.Error("IsAvailable() = true for unreachable server, want false")
	}
}

func TestHTTPClient_IsAvailable_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL)

	// First call
	client.IsAvailable()
	if callCount != 1 {
		t.Errorf("First call: callCount = %d, want 1", callCount)
	}

	// Second call should be cached
	client.IsAvailable()
	if callCount != 1 {
		t.Errorf("Second call (cached): callCount = %d, want 1", callCount)
	}
}

func TestHTTPClient_GetState(t *testing.T) {
	tests := []struct {
		name      string
		response  map[string]interface{}
		wantState PlaybackState
		wantTrack bool
		wantErr   bool
	}{
		{
			name: "playing with track",
			response: map[string]interface{}{
				"player": map[string]interface{}{
					"playbackState": "playing",
					"activeItem": map[string]interface{}{
						"index":    5,
						"position": 135.5,
						"duration": 384.2,
						"columns":  []string{"Pink Floyd", "Comfortably Numb", "The Wall"},
					},
					"volume": map[string]interface{}{
						"value":   -10.0,
						"min":     -100.0,
						"max":     0.0,
						"isMuted": false,
					},
				},
			},
			wantState: StatePlaying,
			wantTrack: true,
			wantErr:   false,
		},
		{
			name: "paused",
			response: map[string]interface{}{
				"player": map[string]interface{}{
					"playbackState": "paused",
					"activeItem": map[string]interface{}{
						"index":    0,
						"position": 0,
						"duration": 0,
						"columns":  []string{"Artist", "Title", "Album"},
					},
					"volume": map[string]interface{}{
						"value":   0.0,
						"min":     -100.0,
						"max":     0.0,
						"isMuted": false,
					},
				},
			},
			wantState: StatePaused,
			wantTrack: true,
			wantErr:   false,
		},
		{
			name: "stopped",
			response: map[string]interface{}{
				"player": map[string]interface{}{
					"playbackState": "stopped",
					"activeItem": map[string]interface{}{
						"index":    -1,
						"position": 0,
						"duration": 0,
						"columns":  []string{},
					},
					"volume": map[string]interface{}{
						"value":   0.0,
						"min":     -100.0,
						"max":     0.0,
						"isMuted": false,
					},
				},
			},
			wantState: StateStopped,
			wantTrack: false,
			wantErr:   false,
		},
		{
			name: "muted",
			response: map[string]interface{}{
				"player": map[string]interface{}{
					"playbackState": "playing",
					"activeItem": map[string]interface{}{
						"index":    0,
						"position": 10.0,
						"duration": 200.0,
						"columns":  []string{"Artist", "Title", "Album"},
					},
					"volume": map[string]interface{}{
						"value":   -100.0,
						"min":     -100.0,
						"max":     0.0,
						"isMuted": true,
					},
				},
			},
			wantState: StatePlaying,
			wantTrack: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := New(server.URL)
			state, err := client.GetState()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if state.State != tt.wantState {
				t.Errorf("GetState() state = %v, want %v", state.State, tt.wantState)
			}

			hasTrack := state.Track != nil
			if hasTrack != tt.wantTrack {
				t.Errorf("GetState() hasTrack = %v, want %v", hasTrack, tt.wantTrack)
			}
		})
	}
}

func TestHTTPClient_GetState_TrackInfo(t *testing.T) {
	response := map[string]interface{}{
		"player": map[string]interface{}{
			"playbackState": "playing",
			"activeItem": map[string]interface{}{
				"index":    5,
				"position": 135.5,
				"duration": 384.2,
				"columns":  []string{"Pink Floyd", "Comfortably Numb", "The Wall"},
			},
			"volume": map[string]interface{}{
				"value":   -50.0,
				"min":     -100.0,
				"max":     0.0,
				"isMuted": false,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := New(server.URL)
	state, err := client.GetState()
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}

	if state.Track == nil {
		t.Fatal("GetState() Track is nil")
	}

	if state.Track.Artist != "Pink Floyd" {
		t.Errorf("Track.Artist = %v, want %v", state.Track.Artist, "Pink Floyd")
	}

	if state.Track.Title != "Comfortably Numb" {
		t.Errorf("Track.Title = %v, want %v", state.Track.Title, "Comfortably Numb")
	}

	if state.Track.Album != "The Wall" {
		t.Errorf("Track.Album = %v, want %v", state.Track.Album, "The Wall")
	}

	if state.Track.Index != 5 {
		t.Errorf("Track.Index = %v, want %v", state.Track.Index, 5)
	}

	expectedPosition := 135*time.Second + 500*time.Millisecond
	if state.Track.Position != expectedPosition {
		t.Errorf("Track.Position = %v, want %v", state.Track.Position, expectedPosition)
	}

	expectedDuration := 384*time.Second + 200*time.Millisecond
	if state.Track.Duration != expectedDuration {
		t.Errorf("Track.Duration = %v, want %v", state.Track.Duration, expectedDuration)
	}
}

func TestHTTPClient_GetState_Volume(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		min        float64
		max        float64
		wantVolume float64
	}{
		{"full volume", 0.0, -100.0, 0.0, 1.0},
		{"half volume", -50.0, -100.0, 0.0, 0.5},
		{"zero volume", -100.0, -100.0, 0.0, 0.0},
		{"quarter volume", -75.0, -100.0, 0.0, 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := map[string]interface{}{
				"player": map[string]interface{}{
					"playbackState": "playing",
					"activeItem": map[string]interface{}{
						"index":    0,
						"position": 0,
						"duration": 0,
						"columns":  []string{"A", "B", "C"},
					},
					"volume": map[string]interface{}{
						"value":   tt.value,
						"min":     tt.min,
						"max":     tt.max,
						"isMuted": false,
					},
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := New(server.URL)
			state, err := client.GetState()
			if err != nil {
				t.Fatalf("GetState() error = %v", err)
			}

			if state.Volume != tt.wantVolume {
				t.Errorf("Volume = %v, want %v", state.Volume, tt.wantVolume)
			}
		})
	}
}

func TestHTTPClient_GetState_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.GetState()
	if err == nil {
		t.Error("GetState() error = nil, want error for server error")
	}
}

func TestHTTPClient_GetState_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := New(server.URL)
	_, err := client.GetState()
	if err == nil {
		t.Error("GetState() error = nil, want error for invalid JSON")
	}
}

func TestNormalizeVolume(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		min   float64
		max   float64
		want  float64
	}{
		{"full volume", 0.0, -100.0, 0.0, 1.0},
		{"zero volume", -100.0, -100.0, 0.0, 0.0},
		{"half volume", -50.0, -100.0, 0.0, 0.5},
		{"invalid range", 0.0, 0.0, 0.0, 0.0},
		{"below min", -150.0, -100.0, 0.0, 0.0},
		{"above max", 50.0, -100.0, 0.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeVolume(tt.value, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("normalizeVolume(%v, %v, %v) = %v, want %v", tt.value, tt.min, tt.max, got, tt.want)
			}
		})
	}
}
