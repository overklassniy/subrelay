# internal/config

## Purpose

Defines the user-facing application settings, their persistence to
`settings.json`, and the default values applied on first launch.

## Contents

- `settings.go` - Defines `Settings`, `BalancerPorts`, `URLTestSettings`,
  and `HeadersSettings` structs. Provides `Load`, `Save`, `Defaults`,
  `GenerateHWID`, and `Snapshot` (a mutex-free copy of the current
  values, used by the settings window while the lock is held). Settings
  are mutex-protected and edited only through the UI.

## Dependencies and interactions

- Imports `subrelay/internal/paths` for the settings file location.
- Consumed by nearly every other package: `sub` (fetch headers),
  `ports` (port ranges), `core` (builder input), `ui` (settings window),
  `update` (refresh interval), `tray` (balancer ports).
