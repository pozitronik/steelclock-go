//go:build windows

package driver

import "syscall"

var (
	modUser32            = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = modUser32.NewProc("GetAsyncKeyState")
)

// vkSpace is the Windows virtual-key code for the spacebar.
const vkSpace = 0x20

// probeKeyDown reports whether the spacebar is currently pressed. It reads global
// key state (no window focus required), so the user can watch the OLED and tap
// space regardless of which window is active.
func probeKeyDown() bool {
	r, _, _ := procGetAsyncKeyState.Call(uintptr(vkSpace))
	return r&0x8000 != 0
}
