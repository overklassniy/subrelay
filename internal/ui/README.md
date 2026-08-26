# internal/ui

## Purpose

Implements the Fyne dialog windows for settings, first-run wizard, nodes
table, and log viewer. All configuration is edited through these
windows; no manual config files are expected.

## Contents

- `settings_win.go` - Settings window with the fields grouped into tabs
  (general, ports, URL test, headers). Numeric and URL fields are
  validated inline via `fyne.StringValidator`s, and a shared Save button
  is disabled while any field across any tab is invalid. The General tab
  includes a "Check for updates" button that triggers the GitHub release
  check. Also provides the first-run wizard dialog.
- `nodes_win.go` - Nodes table window showing each node's name,
  transport type, RU checkbox (override), and SOCKS/HTTP ports with copy
  buttons, under a labeled header row. Includes a search field and a
  Refresh button that triggers a full subscription refresh and then
  reloads the table in place.
- `logs_win.go` - Log viewer window displaying the ring buffer contents
  from the shared logger, with a level filter, manual/auto-refresh,
  clear (in-memory buffer only), open log file, and dump the last built
  sing-box configuration to disk.

## Dependencies and interactions

- Imports `fyne.io/fyne/v2` widgets, containers, dialogs, and
  `fyne.io/fyne/v2/data/validation` for field validators.
- Imports `subrelay/internal/autostart` for the autostart toggle.
- Imports `subrelay/internal/config`, `subrelay/internal/i18n`,
  `subrelay/internal/logging`, `subrelay/internal/paths`,
  `subrelay/internal/state`, `subrelay/internal/sub` for window content.
- Consumed by `cmd/subrelay` (main creates and shows the windows).
