package webeditor

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
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

	// API endpoints
	mux.HandleFunc("/api/schema", s.handleGetSchema)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/profiles", s.handleProfiles)
	mux.HandleFunc("/api/profiles/active", s.handleActiveProfile)
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

// saveConfig saves configuration and triggers reload
func (s *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Validate JSON syntax
	var cfg interface{}
	if err := json.Unmarshal(body, &cfg); err != nil {
		respondError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Save to file
	if err := s.configProvider.Save(body); err != nil {
		respondError(w, "Failed to save: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger reload
	var warning string
	if s.onReload != nil {
		if err := s.onReload(); err != nil {
			warning = "Config saved but reload failed: " + err.Error()
		}
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Configuration saved and reloaded",
	}
	if warning != "" {
		response["warning"] = warning
	}

	respondJSON(w, response)
}

// handleProfiles returns list of available profiles
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.profileProvider == nil {
		respondJSON(w, []ProfileInfo{})
		return
	}

	profiles := s.profileProvider.GetProfiles()
	respondJSON(w, profiles)
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

	if s.profileProvider == nil {
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

	if err := s.profileProvider.SetActiveProfile(req.Path); err != nil {
		respondError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Trigger reload after profile switch
	if s.onReload != nil {
		if err := s.onReload(); err != nil {
			respondJSON(w, map[string]interface{}{
				"success": true,
				"warning": "Profile switched but reload failed: " + err.Error(),
			})
			return
		}
	}

	respondJSON(w, map[string]interface{}{
		"success": true,
		"message": "Profile switched",
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
