package webeditor

import "net/http"

// handleReload reloads the active configuration file. It is what Apply uses
// when the app runs without profiles (explicit -config file), where there is
// no profile to switch to.
func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.onReload == nil {
		respondError(w, "Reload not available", http.StatusNotImplemented)
		return
	}

	if err := s.onReload(); err != nil {
		respondError(w, "Failed to reload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"success": true,
		"message": "Configuration reloaded",
	})
}
