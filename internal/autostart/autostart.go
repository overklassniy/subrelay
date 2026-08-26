// Package autostart manages OS-level autostart registration so the
// application launches on user login. Windows uses the HKCU Run registry
// key; Linux uses a freedesktop.org .desktop file in
// ~/.config/autostart.
package autostart

// Enable registers the current executable for autostart.
//
// Returns:
//   - An error wrapping the OS-specific operation on failure.
func Enable() error {
	return enable()
}

// Disable removes the autostart registration.
//
// Returns:
//   - An error wrapping the OS-specific operation on failure.
func Disable() error {
	return disable()
}

// IsEnabled reports whether autostart is currently registered.
func IsEnabled() bool {
	return isEnabled()
}
