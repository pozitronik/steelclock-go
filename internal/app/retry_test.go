package app

import (
	"errors"
	"testing"
	"time"
)

func TestRetryWithBackoff_Success(t *testing.T) {
	attempts := 0
	err := RetryWithBackoff(3, nil, func(attempt int) error {
		attempts++
		return nil // Success on first try
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	attempts := 0
	err := RetryWithBackoffConfig(5, 1*time.Millisecond, 10*time.Millisecond, nil, func(attempt int) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on third try
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryWithBackoff_AllAttemptsFail(t *testing.T) {
	attempts := 0
	err := RetryWithBackoffConfig(3, 1*time.Millisecond, 10*time.Millisecond, nil, func(attempt int) error {
		attempts++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Should be wrapped in BackendUnavailableError
	var backendErr *BackendUnavailableError
	if !errors.As(err, &backendErr) {
		t.Errorf("expected BackendUnavailableError, got %T", err)
	}

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryWithBackoff_CancelDuringRetry(t *testing.T) {
	cancel := make(chan struct{})
	attempts := 0

	go func() {
		time.Sleep(5 * time.Millisecond)
		close(cancel)
	}()

	err := RetryWithBackoffConfig(10, 10*time.Millisecond, 100*time.Millisecond, cancel, func(attempt int) error {
		attempts++
		return errors.New("error")
	})

	if err == nil {
		t.Fatal("expected error after cancel, got nil")
	}

	if err.Error() != "retry cancelled" {
		t.Errorf("error = %q, want 'retry cancelled'", err.Error())
	}

	// Should have made at least 1 attempt before cancel
	if attempts < 1 {
		t.Errorf("attempts = %d, want >= 1", attempts)
	}
}

func TestCalculateBackoffDelay(t *testing.T) {
	tests := []struct {
		attempt   int
		baseDelay time.Duration
		maxDelay  time.Duration
		expected  time.Duration
	}{
		{2, 1 * time.Second, 10 * time.Second, 1 * time.Second},   // 1 * 2^0 = 1s
		{3, 1 * time.Second, 10 * time.Second, 2 * time.Second},   // 1 * 2^1 = 2s
		{4, 1 * time.Second, 10 * time.Second, 4 * time.Second},   // 1 * 2^2 = 4s
		{5, 1 * time.Second, 10 * time.Second, 8 * time.Second},   // 1 * 2^3 = 8s
		{6, 1 * time.Second, 10 * time.Second, 10 * time.Second},  // 16s capped to 10s
		{10, 1 * time.Second, 10 * time.Second, 10 * time.Second}, // large value capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := calculateBackoffDelay(tt.attempt, tt.baseDelay, tt.maxDelay)
			if result != tt.expected {
				t.Errorf("calculateBackoffDelay(%d, %v, %v) = %v, want %v",
					tt.attempt, tt.baseDelay, tt.maxDelay, result, tt.expected)
			}
		})
	}
}

//goland:noinspection GoBoolExpressions
func TestRetryConstants(t *testing.T) {
	if DefaultBaseDelay != 1*time.Second {
		t.Errorf("DefaultBaseDelay = %v, want 1s", DefaultBaseDelay)
	}
	if DefaultMaxDelay != 10*time.Second {
		t.Errorf("DefaultMaxDelay = %v, want 10s", DefaultMaxDelay)
	}
}
