// Package paths resolves application filesystem paths such as the
// configuration directory, data directory, and asset locations.
//
// Paths are resolved once at startup and cached for the lifetime of
// the process. On Windows the config directory lives under
// %APPDATA%\Subrelay; on Linux and other Unix-like systems it lives
// under ~/.config/subrelay.
package paths

import (
	"os"
	"path/filepath"
)

// configDir is the resolved configuration directory, cached after the
// first call to ConfigDir.
var configDir string

// ConfigDir returns the absolute path to the application configuration
// directory. The directory is created on first access.
//
// Returns:
//   - The absolute configuration directory path.
//
// Errors:
//   - Returns an error wrapping os.MkdirAll when the directory cannot
//     be created.
func ConfigDir() (string, error) {
	if configDir != "" {
		return configDir, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		base, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(base, ".config")
	}

	dir := filepath.Join(base, appDirName())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	configDir = dir
	return dir, nil
}

// SettingsFile returns the absolute path to settings.json.
//
// Returns:
//   - The absolute settings file path.
//
// Errors:
//   - Returns an error when the config directory cannot be resolved.
func SettingsFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// LogFile returns the absolute path to subrelay.log.
//
// Returns:
//   - The absolute log file path.
//
// Errors:
//   - Returns an error when the config directory cannot be resolved.
func LogFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "subrelay.log"), nil
}

// LockFile returns the absolute path to the single-instance lock file.
// Used only on Unix-like systems; on Windows a named mutex is used
// instead.
//
// Returns:
//   - The absolute lock file path.
//
// Errors:
//   - Returns an error when the config directory cannot be resolved.
func LockFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "subrelay.lock"), nil
}

// appDirName returns the per-OS application directory name. Windows uses
// the capitalized "Subrelay" (matching %APPDATA% conventions); Linux and
// other Unix-like systems use the lowercase "subrelay".
func appDirName() string {
	return "Subrelay"
}
