# assets/readme

## Purpose

Holds the SVG visual assets used by the project README files
(`README.md` and `README.ru.md`). All assets are pure, hand-authored
SVG with system fonts, so they render reliably on GitHub light and
dark themes without external dependencies.

## Contents

- `hero.svg` - One-board hero combining the Subrelay wordmark with a
  compact system-map diagram (subscription to tray). Embedded at the
  top of both README files.

## Visual system

Frozen palette and grammar shared by every asset in this directory,
derived from the application tray icon (`internal/tray/icon.png`):

- Background `#0E1116`, foreground `#E6EDF3`, primary `#3C5AA0`
  (icon blue, marks the non-RU lane), secondary `#6B8AC8` (lighter
  blue, marks the RU lane), muted `#8B949E`.
- System sans-serif stack for text, monospace for ports, tags, and
  commands.
- Radius 10, stroke 1.5, 16-unit spacing grid.
- Recurring motif: the tray-icon silhouette plus a two-lane RU/non-RU
  port split, derived from Subrelay's defining balancer mechanic.

## Dependencies and interactions

- Consumed only by `README.md` and `README.ru.md` at the repository
  root via relative `./assets/readme/hero.svg` references.
- No runtime or build dependency. Editing an SVG does not require
  rebuilding the application.
