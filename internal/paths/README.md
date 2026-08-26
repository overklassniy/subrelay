# internal/paths

## Purpose

Resolves application filesystem paths such as the configuration
directory, data directory, and asset locations.

## Contents

- `paths.go` - Provides `ConfigDir`, `SettingsFile`, `LogFile`, and
  `LockFile`. On Windows the config directory lives under
  `%APPDATA%\Subrelay`; on Linux and other Unix-like systems it lives
  under `~/.config/Subrelay`. Paths are resolved once at startup and
  cached.

## Dependencies and interactions

- No internal dependencies.
- Consumed by `subrelay/internal/config` (settings file),
  `subrelay/internal/logging` (log file), and `cmd/subrelay` (lock
  file).
