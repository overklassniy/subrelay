# internal/state

## Purpose

Holds the runtime snapshot of the application: parsed nodes, port
assignments, engine state, last update time, last error, and the last
built configuration as JSON. The tray and UI windows read this snapshot
to render status, node lists, and the log viewer's config dump without
touching the engine directly.

## Contents

- `state.go` - Defines `Snapshot` (immutable point-in-time view,
  including `ConfigJSON`) and `Manager` (mutex-protected holder with
  `Update`, `SetEngineState`, `SetError`, `ClearError`, and `Snapshot`
  methods). `Update` populates `ConfigJSON` from `BuildResult.JSON()`.

## Dependencies and interactions

- Imports `subrelay/internal/core` for `core.State` and
  `core.BuildResult`.
- Imports `subrelay/internal/ports` and `subrelay/internal/sub` for
  snapshot data types.
- Consumed by `subrelay/internal/tray` (menu rebuild) and
  `subrelay/internal/ui` (nodes table).
