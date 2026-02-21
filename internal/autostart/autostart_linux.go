//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopFileName = "steelclock.desktop"

// desktopDir returns ~/.config/autostart.
func desktopDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart"), nil
}

// desktopFilePath returns the full path to the .desktop file.
func desktopFilePath() (string, error) {
	dir, err := desktopDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, desktopFileName), nil
}

func isEnabled() (bool, error) {
	path, err := desktopFilePath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func enable() error {
	exePath, exeDir, err := getAppPaths()
	if err != nil {
		return err
	}

	dir, err := desktopDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := desktopFilePath()
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=SteelClock
Exec=%s
Path=%s
X-GNOME-Autostart-enabled=true
Comment=SteelSeries OLED Display Manager
`, exePath, exeDir)

	return os.WriteFile(path, []byte(content), 0644)
}

func disable() error {
	path, err := desktopFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil // already removed
	}
	return err
}
