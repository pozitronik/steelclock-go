//go:build !windows

package driver

// probeKeyDown has no key-polling implementation off Windows; the probe then
// degrades to a timed sweep (the production target for this diagnostic is Windows).
func probeKeyDown() bool {
	return false
}
