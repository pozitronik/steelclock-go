package tray

import (
	"testing"
)

// TestShowNotification tests that ShowNotification doesn't panic
// The actual behavior is platform-specific and tested by build tags
func TestShowNotification(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		message string
	}{
		{
			name:    "normal notification",
			title:   "Test Title",
			message: "Test Message",
		},
		{
			name:    "empty title",
			title:   "",
			message: "Test Message",
		},
		{
			name:    "empty message",
			title:   "Test Title",
			message: "",
		},
		{
			name:    "both empty",
			title:   "",
			message: "",
		},
		{
			name:    "long title",
			title:   "This is a very long title that might exceed normal notification limits",
			message: "Short message",
		},
		{
			name:    "long message",
			title:   "Short",
			message: "This is a very long message that might exceed normal notification limits and should be handled gracefully by the notification system without causing any issues or crashes",
		},
		{
			name:    "special characters",
			title:   "Test & <Title> \"with\" 'quotes'",
			message: "Message with\nnewlines\tand\ttabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ShowNotification panicked: %v", r)
				}
			}()

			ShowNotification(tt.title, tt.message)
		})
	}
}

// TestShowNotification_Concurrent tests that concurrent notifications don't cause issues
func TestShowNotification_Concurrent(t *testing.T) {
	const numGoroutines = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()

			ShowNotification("Concurrent Test", "Message from goroutine")
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
