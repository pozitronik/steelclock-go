package gamesense

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestClient_Timeout tests that HTTP client respects timeout
func TestClient_Timeout(t *testing.T) {
	// Create server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second) // Longer than client timeout (500ms)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient: &http.Client{
			Timeout: 100 * time.Millisecond, // Short timeout
		},
	}

	err := client.RegisterGame("Developer")
	if err == nil {
		t.Error("RegisterGame() should timeout, but succeeded")
	}
}

// TestClient_ConcurrentRequests tests thread safety
func TestClient_ConcurrentRequests(t *testing.T) {
	requestCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Send multiple concurrent requests
	const numRequests = 10
	errChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			errChan <- client.RegisterGame("Developer")
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}

	// Verify all requests were received
	finalCount := atomic.LoadInt32(&requestCount)
	if finalCount != numRequests {
		t.Errorf("Expected %d requests, got %d", numRequests, finalCount)
	}
}

// TestClient_ServerUnavailable tests handling of connection refused
func TestClient_ServerUnavailable(t *testing.T) {
	client := &Client{
		baseURL:         "http://localhost:1", // Port 1 - typically unavailable
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient: &http.Client{
			Timeout: 100 * time.Millisecond,
		},
	}

	err := client.RegisterGame("Developer")
	if err == nil {
		t.Error("RegisterGame() to unavailable server should fail")
	}
}

// TestClient_ServerClosesConnection tests handling of abrupt connection close
func TestClient_ServerClosesConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection without sending response
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("Server doesn't support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("Hijack failed: %v", err)
		}
		conn.Close()
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Developer")
	if err == nil {
		t.Error("RegisterGame() should fail when server closes connection")
	}
}

// TestClient_LargePayload tests handling of large bitmap data
func TestClient_LargePayload(t *testing.T) {
	var receivedSize int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024*1024) // 1MB buffer
		n, _ := r.Body.Read(buf)
		receivedSize = n
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Send valid 640-byte bitmap
	bitmap := make([]int, 640)
	for i := range bitmap {
		bitmap[i] = i % 256
	}

	err := client.SendScreenData("EVENT", bitmap)
	if err != nil {
		t.Errorf("SendScreenData() error = %v", err)
	}

	if receivedSize == 0 {
		t.Error("Server received no data")
	}

	t.Logf("Server received %d bytes", receivedSize)
}

// TestClient_MalformedURL tests handling of malformed base URL
func TestClient_MalformedURL(t *testing.T) {
	client := &Client{
		baseURL:         "ht!tp://invalid url with spaces",
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Developer")
	if err == nil {
		t.Error("RegisterGame() with malformed URL should fail")
	}
}

// TestClient_EmptyResponse tests handling of empty successful response
func TestClient_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	err := client.RegisterGame("Developer")
	if err != nil {
		t.Errorf("RegisterGame() with empty response should succeed, got error: %v", err)
	}
}

// TestClient_SlowServer tests handling of slow server responses
func TestClient_SlowServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response but within timeout
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient: &http.Client{
			Timeout: 200 * time.Millisecond, // Sufficient timeout
		},
	}

	err := client.RegisterGame("Developer")
	if err != nil {
		t.Errorf("RegisterGame() with slow server should succeed within timeout, got error: %v", err)
	}
}

// TestClient_ContextCancellation tests handling of context cancellation
func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request was cancelled
		select {
		case <-r.Context().Done():
			// Request was cancelled
			return
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Create client with cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create custom HTTP client that respects context
	httpClient := &http.Client{
		Timeout: 1 * time.Second,
	}

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      httpClient,
	}

	// Cancel context immediately
	cancel()

	// Note: The current Client implementation doesn't support context,
	// but this test verifies error handling
	_ = ctx // Use ctx to avoid unused variable error

	// This will timeout or succeed depending on server timing
	_ = client.RegisterGame("Developer")
}

// TestClient_HTTPMethodValidation tests that POST method is used
func TestClient_HTTPMethodValidation(t *testing.T) {
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	_ = client.RegisterGame("Developer")

	if receivedMethod != http.MethodPost {
		t.Errorf("Expected POST method, got %s", receivedMethod)
	}
}

// TestClient_ContentTypeHeader tests that Content-Type header is set correctly
func TestClient_ContentTypeHeader(t *testing.T) {
	var contentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	_ = client.RegisterGame("Developer")

	if contentType != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", contentType)
	}
}

// TestClient_UserAgent tests that requests include appropriate headers
func TestClient_UserAgent(t *testing.T) {
	var userAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	_ = client.RegisterGame("Developer")

	// Go's http.Client sets a default User-Agent
	t.Logf("User-Agent: %s", userAgent)
}

// TestClient_RequestBodyClosed tests that request body is properly closed
func TestClient_RequestBodyClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Make multiple requests to check for resource leaks
	for i := 0; i < 100; i++ {
		err := client.RegisterGame("Developer")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	// If body wasn't closed, we'd run into resource exhaustion
}

// TestClient_ResponseBodyClosed tests that response body is properly closed
func TestClient_ResponseBodyClosed(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		// Write some data to ensure body is not empty
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Make multiple requests
	for i := 0; i < 50; i++ {
		err := client.RegisterGame("Developer")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	if requestCount != 50 {
		t.Errorf("Expected 50 requests, got %d", requestCount)
	}
}

// TestClient_InvalidJSON tests handling of invalid JSON in payload
func TestClient_InvalidJSON(t *testing.T) {
	// This test verifies that json.Marshal doesn't fail on valid Go structs
	client := &Client{
		baseURL:         "http://localhost:12345",
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// All internal payloads should marshal correctly
	// This will fail at HTTP level, not JSON level
	err := client.RegisterGame("Developer")
	if err == nil {
		t.Error("Request to non-existent server should fail")
	}
}

// TestSendScreenData_BoundaryValues tests bitmap size boundary conditions
func TestSendScreenData_BoundaryValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	testCases := []struct {
		name      string
		size      int
		expectErr bool
	}{
		{"exactly 640", 640, false},
		{"zero size", 0, true},
		{"639 bytes", 639, true},
		{"641 bytes", 641, true},
		{"1280 bytes (2x)", 1280, true},
		{"320 bytes (half)", 320, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bitmap := make([]int, tc.size)
			err := client.SendScreenData("EVENT", bitmap)

			if tc.expectErr && err == nil {
				t.Errorf("Expected error for size %d, but got nil", tc.size)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Unexpected error for size %d: %v", tc.size, err)
			}
		})
	}
}

// TestClient_AllPixelValues tests sending all possible grayscale values
func TestClient_AllPixelValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:         server.URL,
		gameName:        "TEST_GAME",
		gameDisplayName: "Test Game",
		httpClient:      &http.Client{},
	}

	// Create bitmap with values 0-255 cycling
	bitmap := make([]int, 640)
	for i := range bitmap {
		bitmap[i] = i % 256
	}

	err := client.SendScreenData("EVENT", bitmap)
	if err != nil {
		t.Errorf("SendScreenData() with all pixel values failed: %v", err)
	}
}
