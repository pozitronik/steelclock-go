//go:build linux

package widget

// Update updates the keyboard state (Linux stub - always false)
func (w *KeyboardWidget) Update() error {
	w.mu.Lock()
	w.capsState = false
	w.numState = false
	w.scrollState = false
	w.mu.Unlock()
	return nil
}
