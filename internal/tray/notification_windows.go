//go:build windows

package tray

import (
	"log"

	"github.com/go-toast/toast"
)

// ShowNotification displays a Windows toast notification
func ShowNotification(title, message string) {
	notification := toast.Notification{
		AppID:   "SteelClock",
		Title:   title,
		Message: message,
		Icon:    "", // Could use icon path if needed
	}

	err := notification.Push()
	if err != nil {
		log.Printf("Failed to show notification: %v", err)
	}
}
