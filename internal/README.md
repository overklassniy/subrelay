# internal

## Purpose

Contains all internal Go packages for the Subrelay application. None of
these packages are intended to be imported by external modules.

## Contents

- `autostart/` - OS-level autostart registration (Windows registry,
  Linux .desktop file).
- `config/` - Application settings persistence and defaults.
- `core/` - sing-box option builder and engine lifecycle management.
- `i18n/` - Russian and English UI translations.
- `logging/` - Ring buffer logger with file output.
- `paths/` - Filesystem path resolution for config, log, and lock files.
- `ports/` - Stable port planning with persistence across restarts.
- `state/` - Runtime snapshot for tray and UI rendering.
- `sub/` - Subscription fetch and VLESS link parser.
- `tray/` - Fyne system tray icon and menu.
- `ui/` - Settings, nodes table, and log viewer windows.
- `update/` - Periodic subscription refresh goroutine.

## Dependencies and interactions

The packages form a layered architecture:

- `paths` is the foundation; `config` and `logging` depend on it.
- `sub` and `ports` depend on `config`.
- `core` depends on `config`, `ports`, and `sub`.
- `state` depends on `core`, `ports`, and `sub`.
- `tray` and `ui` depend on `config`, `i18n`, `state`, and `sub`.
- `update` depends on `config`, `core`, `ports`, `state`, and `sub`.
- `cmd/subrelay` wires all packages together.
