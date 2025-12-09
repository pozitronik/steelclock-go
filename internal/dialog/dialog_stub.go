//go:build !windows

// Package dialog provides simple input dialogs
package dialog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// InputBox shows an input prompt in the terminal on non-Windows platforms
func InputBox(title, prompt string, masked bool) (string, bool) {
	fmt.Printf("%s\n%s: ", title, prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(input), true
}

// ShowMessage prints a message to stdout on non-Windows platforms
func ShowMessage(title, message string, isError bool) {
	if isError {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", title, message)
	} else {
		fmt.Printf("[%s] %s\n", title, message)
	}
}
