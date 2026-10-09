//go:build !windows && !linux

package hotplug

// startSource is not supported on this platform.
func startSource(uint16, func()) (func(), error) {
	return nil, ErrUnsupported
}
