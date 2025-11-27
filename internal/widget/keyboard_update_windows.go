//go:build windows

package widget

import (
	"syscall"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	getKeyState = user32.NewProc("GetKeyState")
)

const (
	VkCapital = 0x14 // Caps Lock
	VkNumlock = 0x90 // Num Lock
	VkScroll  = 0x91 // Scroll Lock
)

// Update updates the keyboard state
func (w *KeyboardWidget) Update() error {
	caps := isKeyToggled(VkCapital)
	num := isKeyToggled(VkNumlock)
	scroll := isKeyToggled(VkScroll)

	w.mu.Lock()
	w.capsState = caps
	w.numState = num
	w.scrollState = scroll
	w.mu.Unlock()
	return nil
}

// isKeyToggled checks if a toggle key is enabled (Windows only)
func isKeyToggled(vkCode uint32) bool {
	ret, _, _ := getKeyState.Call(uintptr(vkCode))
	// The low-order bit indicates toggle state (1 = on, 0 = off)
	return (ret & 0x1) != 0
}
