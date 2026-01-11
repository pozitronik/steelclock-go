// Package webeditor provides an embedded web-based configuration editor
package webeditor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// DefaultPort is the default port for the web editor server
const DefaultPort = 8384

// Server manages the embedded web configuration editor
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	port       int

	configProvider    ConfigProvider
	profileProvider   ProfileProvider
	previewProvider   PreviewProvider
	schemaPath        string
	onReload          func() error
	onProfileSwitch   func(path string) error
	onPreviewOverride func(enable bool) error

	mu      sync.Mutex
	running bool
}

// NewServer creates a new web editor server
func NewServer(configProvider ConfigProvider, profileProvider ProfileProvider, schemaPath string, onReload func() error, onProfileSwitch func(path string) error) *Server {
	return &Server{
		configProvider:  configProvider,
		profileProvider: profileProvider,
		schemaPath:      schemaPath,
		onReload:        onReload,
		onProfileSwitch: onProfileSwitch,
	}
}

// SetPreviewProvider sets the preview provider for live preview support
func (s *Server) SetPreviewProvider(provider PreviewProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.previewProvider = provider
}

// SetPreviewOverrideCallback sets the callback for preview override requests
func (s *Server) SetPreviewOverrideCallback(callback func(enable bool) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPreviewOverride = callback
}

// Start starts the HTTP server on the default port (localhost only)
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil // Already running
	}

	// Bind to all interfaces to allow WSL connections
	// Security: The server still validates Origin header on POST requests
	addr := fmt.Sprintf("0.0.0.0:%d", DefaultPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener on port %d: %w", DefaultPort, err)
	}

	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	s.registerHandlers(mux)

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.running = true

	go func() {
		if err := s.httpServer.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Web editor server error: %v", err)
		}
	}()

	log.Printf("Web editor started at http://127.0.0.1:%d", s.port)
	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.running = false
	log.Printf("Web editor stopped")
	return nil
}

// GetURL returns the URL where the editor is accessible
func (s *Server) GetURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// IsRunning returns true if the server is currently running
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
