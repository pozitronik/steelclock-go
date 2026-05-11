//go:build darwin

// Package dialog provides native macOS dialogs via osascript.
package dialog

import (
	"fmt"
	"os/exec"
	"strings"
)

// InputBox shows a native macOS input dialog using osascript.
func InputBox(title, prompt string, masked bool) (string, bool) {
	safeTitle := strings.ReplaceAll(title, `"`, `\"`)
	safePrompt := strings.ReplaceAll(prompt, `"`, `\"`)

	var script string
	if masked {
		script = fmt.Sprintf(
			`display dialog "%s" with title "%s" default answer "" with hidden answer`,
			safePrompt, safeTitle,
		)
	} else {
		script = fmt.Sprintf(
			`display dialog "%s" with title "%s" default answer ""`,
			safePrompt, safeTitle,
		)
	}

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		// User cancelled or osascript failed
		return "", false
	}

	// osascript returns "text returned:VALUE"
	result := strings.TrimSpace(string(out))
	prefix := "text returned:"
	if strings.HasPrefix(result, prefix) {
		return result[len(prefix):], true
	}

	return result, true
}

// ShowMessage displays a native macOS message dialog using osascript.
func ShowMessage(title, message string, isError bool) {
	safeTitle := strings.ReplaceAll(title, `"`, `\"`)
	safeMessage := strings.ReplaceAll(message, `"`, `\"`)

	var icon string
	if isError {
		icon = " with icon stop"
	}

	script := fmt.Sprintf(
		`display dialog "%s" with title "%s" buttons {"OK"} default button "OK"%s`,
		safeMessage, safeTitle, icon,
	)

	_ = exec.Command("osascript", "-e", script).Run()
}
