//go:build !windows

// Package autostart (autostart_linux.go) implements autostart on Linux
// and other Unix-like systems via a freedesktop.org .desktop file in
// ~/.config/autostart.
package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntry = `[Desktop Entry]
Type=Application
Name=Subrelay
Exec=%s
Icon=subrelay
Terminal=false
X-GNOME-Autostart-enabled=true
`

func desktopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart", "subrelay.desktop"), nil
}

func enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(desktopEntry, exe)
	return os.WriteFile(path, []byte(content), 0o644)
}

func disable() error {
	path, err := desktopPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func isEnabled() bool {
	path, err := desktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
