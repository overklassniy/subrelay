# internal/tray

## Purpose

Builds and updates the system tray icon, menu, and notifications using
Fyne's `desktop.App` integration. The menu is rebuilt whenever the state
snapshot or active language changes.

## Contents

- `tray.go` - Manages the tray icon and menu. Builds status line,
  balancer submenu (with copy-to-clipboard), nodes submenu (with
  copy-to-clipboard), and action items (nodes list, settings, logs,
  exit).
- `icon.go` - Embeds the tray icon PNG as a Fyne static resource.
- `icon.png` - The tray icon image (32x32 PNG).
- `autotag.go` - Local auto-node classifier used to skip auto-selector
  entries in the tray menu.

## Dependencies and interactions

- Imports `fyne.io/fyne/v2` and `fyne.io/fyne/v2/driver/desktop` for
  tray integration.
- Imports `subrelay/internal/config`, `subrelay/internal/i18n`,
  `subrelay/internal/state` for menu content.
- Consumed by `cmd/subrelay` (main creates and manages the tray).
