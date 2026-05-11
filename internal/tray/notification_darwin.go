//go:build darwin

package tray

import (
	"log"
	"os/exec"
	"strings"
)

// ShowNotification displays a native macOS notification using osascript.
func ShowNotification(title, message string) {
	// Escape double quotes for AppleScript
	safeTitle := strings.ReplaceAll(title, `"`, `\"`)
	safeMessage := strings.ReplaceAll(message, `"`, `\"`)

	script := `display notification "` + safeMessage + `" with title "` + safeTitle + `"`
	err := exec.Command("osascript", "-e", script).Run()
	if err != nil {
		log.Printf("Notification (osascript failed): %s - %s: %v", title, message, err)
	}
}
