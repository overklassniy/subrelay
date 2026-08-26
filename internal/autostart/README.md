# internal/autostart

## Purpose

Manages OS-level autostart registration so the application launches on
user login.

## Contents

- `autostart.go` - Platform-agnostic interface with `Enable`, `Disable`,
  and `IsEnabled` functions.
- `autostart_windows.go` - Windows implementation using the
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` registry key.
- `autostart_linux.go` - Linux/Unix implementation using a
  freedesktop.org `.desktop` file in `~/.config/autostart`.

## Dependencies and interactions

- On Windows, imports `golang.org/x/sys/windows/registry`.
- Consumed by `subrelay/internal/ui` (settings window autostart
  checkbox) and `cmd/subrelay` (applies autostart on settings change).
