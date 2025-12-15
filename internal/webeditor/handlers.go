package webeditor

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// registerHandlers sets up all HTTP routes
func (s *Server) registerHandlers(mux *http.ServeMux) {
	// Static file serving (embedded assets)
	fs, err := getFileSystem()
	if err != nil {
		// Fallback: serve a simple error page
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" || r.URL.Path == "/index.html" {
				s.serveIndex(w, r)
				return
			}
			http.NotFound(w, r)
		})
	} else {
		mux.Handle("/", http.FileServer(fs))
	}

	// Preview page redirect (clean URL)
	mux.HandleFunc("/preview", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/preview.html", http.StatusFound)
	})

	// API endpoints
	mux.HandleFunc("/api/schema", s.handleGetSchema)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/load", s.handleLoadConfigByPath)
	mux.HandleFunc("/api/validate", s.handleValidate)
	mux.HandleFunc("/api/profiles", s.handleProfiles)
	mux.HandleFunc("/api/profiles/active", s.handleActiveProfile)
	mux.HandleFunc("/api/profiles/rename", s.handleRenameProfile)

	// Preview endpoints
	mux.HandleFunc("/api/preview", s.handlePreviewInfo)
	mux.HandleFunc("/api/preview/frame", s.handlePreviewFrame)
	mux.HandleFunc("/api/preview/ws", s.handlePreviewWebSocket)
	mux.HandleFunc("/api/preview/override", s.handlePreviewOverride)
}

// serveIndex serves the main HTML page
func (s *Server) serveIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fallbackHTML))
}

// handleGetSchema returns the JSON schema
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read schema from the configured path
	data, err := os.ReadFile(s.schemaPath)
	if err != nil {
		respondError(w, "Schema not found: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// handleConfig handles GET (load) and POST (save) for config
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Simple origin check for POST requests
	if r.Method == http.MethodPost {
		origin := r.Header.Get("Origin")
		if origin != "" && !strings.HasPrefix(origin, "http://127.0.0.1") &&
			!strings.HasPrefix(origin, "http://localhost") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		s.getConfig(w)
	case http.MethodPost:
		s.saveConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getConfig returns the current configuration JSON
func (s *Server) getConfig(w http.ResponseWriter) {
	data, err := s.configProvider.Load()
	if err != nil {
		respondError(w, "Failed to load config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// saveConfig saves configuration (optionally to a specific path)
func (s *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Try to parse as {path, config} structure first
	var wrappedReq struct {
		Path   string          `json:"path"`
		Config json.RawMessage `json:"config"`
	}

	var configData []byte
	var savePath string

	if err := json.Unmarshal(body, &wrappedReq); err == nil && wrappedReq.Config != nil {
		// New format: {path: "...", config: {...}}
		configData = wrappedReq.Config
		savePath = wrappedReq.Path
	} else {
		// Old format: config directly
		configData = body
		savePath = ""
	}

	// Validate JSON syntax
	var cfg interface{}
	if err := json.Unmarshal(configData, &cfg); err != nil {
		respondError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Save to file
	if savePath != "" {
		// Save to specific path
		if err := os.WriteFile(savePath, configData, 0644); err != nil {
			respondError(w, "Failed to save: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Save to active config
		if err := s.configProvider.Save(configData); err != nil {
			respondError(w, "Failed to save: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Configuration saved",
	}

	respondJSON(w, response)
}

// handleLoadConfigByPath loads a config file by path without switching active profile
func (s *Server) handleLoadConfigByPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		respondError(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Read the config file
	data, err := os.ReadFile(req.Path)
	if err != nil {
		respondError(w, "Failed to load config: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// handleValidate validates configuration without saving
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse JSON into config struct
	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		respondJSON(w, map[string]interface{}{
			"valid":  false,
			"errors": []string{"Invalid JSON: " + err.Error()},
		})
		return
	}

	// Validate (defaults are applied when actually loading the config)
	if err := config.Validate(&cfg); err != nil {
		respondJSON(w, map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}

	respondJSON(w, map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	})
}

// handleProfiles handles GET (list) and POST (create) for profiles
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listProfiles(w)
	case http.MethodPost:
		s.createProfile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listProfiles returns list of available profiles
func (s *Server) listProfiles(w http.ResponseWriter) {
	if s.profileProvider == nil {
		respondJSON(w, []ProfileInfo{})
		return
	}

	profiles := s.profileProvider.GetProfiles()
	respondJSON(w, profiles)
}

// createProfile creates a new profile with the given name
func (s *Server) createProfile(w http.ResponseWriter, r *http.Request) {
	// Origin check for POST
	origin := r.Header.Get("Origin")
	if origin != "" && !strings.HasPrefix(origin, "http://127.0.0.1") &&
		!strings.HasPrefix(origin, "http://localhost") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if s.profileProvider == nil {
		respondError(w, "Profile management not available", http.StatusNotImplemented)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondError(w, "Profile name is required", http.StatusBadRequest)
		return
	}

	path, err := s.profileProvider.CreateProfile(req.Name)
	if err != nil {
		respondError(w, "Failed to create profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "Profile created",
	})
}

// handleActiveProfile handles switching active profile
func (s *Server) handleActiveProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Origin check
	origin := r.Header.Get("Origin")
	if origin != "" && !strings.HasPrefix(origin, "http://127.0.0.1") &&
		!strings.HasPrefix(origin, "http://localhost") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if s.profileProvider == nil || s.onProfileSwitch == nil {
		respondError(w, "Profile management not available", http.StatusNotImplemented)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Use the proper profile switch callback which handles:
	// 1. Stopping compositor
	// 2. Loading new config
	// 3. Starting with new profile
	// 4. Updating tray menu
	if err := s.onProfileSwitch(req.Path); err != nil {
		respondError(w, "Failed to switch profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"success": true,
		"message": "Profile switched",
	})
}

// handleRenameProfile handles renaming a profile
func (s *Server) handleRenameProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Origin check
	origin := r.Header.Get("Origin")
	if origin != "" && !strings.HasPrefix(origin, "http://127.0.0.1") &&
		!strings.HasPrefix(origin, "http://localhost") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if s.profileProvider == nil {
		respondError(w, "Profile management not available", http.StatusNotImplemented)
		return
	}

	var req struct {
		Path    string `json:"path"`
		NewName string `json:"new_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		respondError(w, "Profile path is required", http.StatusBadRequest)
		return
	}

	if req.NewName == "" {
		respondError(w, "New name is required", http.StatusBadRequest)
		return
	}

	newPath, err := s.profileProvider.RenameProfile(req.Path, req.NewName)
	if err != nil {
		respondError(w, "Failed to rename profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"success":  true,
		"path":     newPath,
		"new_name": req.NewName,
		"message":  "Profile renamed",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

// respondError sends a JSON error response
func respondError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}

// handlePreviewInfo returns preview configuration and availability
func (s *Server) handlePreviewInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.previewProvider == nil {
		respondJSON(w, map[string]interface{}{
			"available": false,
			"message":   "Preview backend not enabled. Set backend to 'preview' in config.",
		})
		return
	}

	cfg := s.previewProvider.GetPreviewConfig()
	respondJSON(w, map[string]interface{}{
		"available":  true,
		"width":      cfg.Width,
		"height":     cfg.Height,
		"target_fps": cfg.TargetFPS,
	})
}

// handlePreviewFrame returns the current frame as raw bytes (for static preview)
func (s *Server) handlePreviewFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.previewProvider == nil {
		respondError(w, "Preview not available", http.StatusNotImplemented)
		return
	}

	frame, frameNum, timestamp := s.previewProvider.GetCurrentFrame()
	if frame == nil {
		// No frame yet, return empty response with metadata
		respondJSON(w, map[string]interface{}{
			"frame":        nil,
			"frame_number": 0,
			"timestamp":    0,
		})
		return
	}

	cfg := s.previewProvider.GetPreviewConfig()
	respondJSON(w, map[string]interface{}{
		"frame":        frame, // Will be base64 encoded by JSON
		"frame_number": frameNum,
		"timestamp":    timestamp.UnixMilli(),
		"width":        cfg.Width,
		"height":       cfg.Height,
	})
}

// handlePreviewWebSocket upgrades to WebSocket for live preview
func (s *Server) handlePreviewWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.previewProvider == nil {
		http.Error(w, "Preview not available", http.StatusNotImplemented)
		return
	}

	// Delegate to the preview provider's WebSocket handler
	s.previewProvider.HandleWebSocket(w, r)
}

// handlePreviewOverride enables/disables temporary preview backend override
func (s *Server) handlePreviewOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Origin check
	origin := r.Header.Get("Origin")
	if origin != "" && !strings.HasPrefix(origin, "http://127.0.0.1") &&
		!strings.HasPrefix(origin, "http://localhost") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if s.onPreviewOverride == nil {
		respondError(w, "Preview override not supported", http.StatusNotImplemented)
		return
	}

	var req struct {
		Enable bool `json:"enable"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.onPreviewOverride(req.Enable); err != nil {
		respondError(w, "Failed to set preview override: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"success": true,
		"enabled": req.Enable,
	})
}

// fallbackHTML is a minimal HTML page for when embedded assets are not available
const fallbackHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SteelClock Configuration Editor</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        textarea { width: 100%; height: 500px; font-family: monospace; font-size: 14px; }
        button { padding: 10px 20px; font-size: 16px; cursor: pointer; margin: 10px 5px 10px 0; }
        .success { color: green; }
        .error { color: red; }
        #status { margin: 10px 0; padding: 10px; }
    </style>
</head>
<body>
    <h1>SteelClock Configuration Editor</h1>
    <div id="status"></div>
    <div>
        <button onclick="loadConfig()">Reload from File</button>
        <button onclick="saveConfig()">Save & Apply</button>
    </div>
    <textarea id="config"></textarea>
    <script>
        const configEl = document.getElementById('config');
        const statusEl = document.getElementById('status');

        function showStatus(message, isError) {
            statusEl.textContent = message;
            statusEl.className = isError ? 'error' : 'success';
        }

        async function loadConfig() {
            try {
                const res = await fetch('/api/config');
                if (!res.ok) throw new Error(await res.text());
                const json = await res.json();
                configEl.value = JSON.stringify(json, null, 2);
                showStatus('Configuration loaded', false);
            } catch (err) {
                showStatus('Failed to load: ' + err.message, true);
            }
        }

        async function saveConfig() {
            try {
                // Validate JSON before sending
                const parsed = JSON.parse(configEl.value);

                const res = await fetch('/api/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(parsed)
                });

                const result = await res.json();
                if (result.error) {
                    showStatus('Error: ' + result.error, true);
                } else {
                    let msg = result.message || 'Saved successfully';
                    if (result.warning) msg += ' (Warning: ' + result.warning + ')';
                    showStatus(msg, false);
                }
            } catch (err) {
                showStatus('Failed to save: ' + err.message, true);
            }
        }

        // Load config on page load
        loadConfig();
    </script>
</body>
</html>`
