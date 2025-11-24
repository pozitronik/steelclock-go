//go:build !windows

package tray

import "log"

// ShowNotification is a no-op on Unix systems (notifications not implemented)
func ShowNotification(title, message string) {
	log.Printf("Notification (not shown on Unix): %s - %s", title, message)
}
