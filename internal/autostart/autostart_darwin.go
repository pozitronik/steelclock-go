//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const plistFileName = "com.steelclock.app.plist"

// launchAgentsDir returns ~/Library/LaunchAgents.
func launchAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

// plistFilePath returns the full path to the LaunchAgent plist file.
func plistFilePath() (string, error) {
	dir, err := launchAgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, plistFileName), nil
}

func isEnabled() (bool, error) {
	path, err := plistFilePath()
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

	dir, err := launchAgentsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := plistFilePath()
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.steelclock.app</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
`, exePath, exeDir)

	return os.WriteFile(path, []byte(content), 0644)
}

func disable() error {
	path, err := plistFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil // already removed
	}
	return err
}
