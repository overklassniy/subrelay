# internal/logging

## Purpose

Provides a shared logger that writes structured lines to both an
in-memory ring buffer (consumed by the log viewer window) and a log file
on disk.

## Contents

- `logger.go` - Defines `Logger` and `Line`. The ring buffer keeps the
  most recent lines so the UI can display them without re-reading the
  file. The file logger appends every line for persistent
  troubleshooting. Provides `Global` (singleton), `New`, `Info`, `Warn`,
  `Error`, `Lines`, `Clear` (empties the in-memory ring buffer only,
  used by the log viewer's Clear action), `Subscribe`, and
  `Unsubscribe`.

## Dependencies and interactions

- Imports `subrelay/internal/paths` for the log file location.
- Consumed by `subrelay/internal/core` (engine lifecycle messages),
  `subrelay/internal/update` (refresh cycle messages),
  `subrelay/internal/ui` (log viewer window), and `cmd/subrelay`
  (startup/shutdown).
