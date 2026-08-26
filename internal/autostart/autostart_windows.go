// Package autostart (autostart_windows.go) implements autostart on
// Windows via the HKCU\Software\Microsoft\Windows\CurrentVersion\Run
// registry key.
package autostart

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

// runKeyPath is the registry path for user-level autostart entries.
const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// appName is the registry value name used for the autostart entry.
const appName = "Subrelay"

func enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(appName, exe)
}

func disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(appName)
}

func isEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(appName)
	return err == nil
}
