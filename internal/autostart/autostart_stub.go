//go:build !windows && !linux && !darwin

package autostart

func isEnabled() (bool, error) {
	return false, ErrNotSupported
}

func enable() error {
	return ErrNotSupported
}

func disable() error {
	return ErrNotSupported
}
