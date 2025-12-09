package app

import (
	"fmt"
	"log"
	"time"
)

const (
	// DefaultBaseDelay is the initial delay between retry attempts
	DefaultBaseDelay = 1 * time.Second
	// DefaultMaxDelay is the maximum delay between retry attempts
	DefaultMaxDelay = 10 * time.Second
)

// RetryWithBackoff executes an operation with exponential backoff retry logic.
// It attempts the operation up to maxAttempts times, with increasing delays between attempts.
// The cancel channel can be used to cancel the retry loop.
// Returns nil on success, or a BackendUnavailableError if all attempts fail.
func RetryWithBackoff(maxAttempts int, cancel <-chan struct{}, operation func(attempt int) error) error {
	return RetryWithBackoffConfig(maxAttempts, DefaultBaseDelay, DefaultMaxDelay, cancel, operation)
}

// RetryWithBackoffConfig executes an operation with configurable exponential backoff.
func RetryWithBackoffConfig(maxAttempts int, baseDelay, maxDelay time.Duration, cancel <-chan struct{}, operation func(attempt int) error) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Wait before retry (except for first attempt)
		if attempt > 1 {
			delay := calculateBackoffDelay(attempt, baseDelay, maxDelay)
			log.Printf("Retrying in %v... (attempt %d/%d)", delay, attempt, maxAttempts)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-cancel:
				log.Println("Retry cancelled")
				return fmt.Errorf("retry cancelled")
			}
		}

		// Attempt the operation
		if err := operation(attempt); err != nil {
			if attempt == maxAttempts {
				return &BackendUnavailableError{Err: err}
			}
			continue
		}

		// Success
		return nil
	}

	return &BackendUnavailableError{Err: fmt.Errorf("failed after %d attempts", maxAttempts)}
}

// calculateBackoffDelay calculates the delay for a given attempt using exponential backoff.
// The delay doubles with each attempt, capped at maxDelay.
func calculateBackoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-2)
	// attempt 2 -> baseDelay * 1
	// attempt 3 -> baseDelay * 2
	// attempt 4 -> baseDelay * 4
	// etc.
	multiplier := uint(1) << uint(attempt-2)
	delay := time.Duration(float64(baseDelay) * float64(multiplier))

	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}
