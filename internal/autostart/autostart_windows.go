//go:build windows

package autostart

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const (
	registryKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName       = "SteelClock"
)

func isEnabled() (bool, error) {
	exePath, _, err := getAppPaths()
	if err != nil {
		return false, err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, nil // key doesn't exist — not enabled
	}
	defer func() { _ = key.Close() }()

	val, _, err := key.GetStringValue(valueName)
	if err != nil {
		return false, nil // value doesn't exist — not enabled
	}

	return val == exePath, nil
}

func enable() error {
	exePath, _, err := getAppPaths()
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = key.Close() }()

	return key.SetStringValue(valueName, exePath)
}

func disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil // key doesn't exist — nothing to disable
	}
	defer func() { _ = key.Close() }()

	err = key.DeleteValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return nil // already removed
	}
	return err
}
