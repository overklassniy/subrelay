# internal/update

## Purpose

Runs the periodic subscription refresh goroutine and the GitHub release
checker. On each refresh tick the timer fetches the subscription, parses
it, plans ports, builds new options, and reloads the engine. The checker
queries the project GitHub repository for newer releases and notifies the
application when an update is available. Errors do not stop either loop
or crash the process; the previous engine instance keeps running when a
reload fails, and a failed release check is logged and retried on the
next tick.

## Contents

- `timer.go` - Defines `Timer` with `Start`, `Stop`, and the internal
  `loop`/`refreshOnce` methods. The first refresh runs immediately;
  subsequent refreshes run at the configured interval.
- `checker.go` - Defines `Release`, `LatestRelease`, `IsNewer`, and the
  `Checker` periodic release-check goroutine. `CurrentVersion` holds the
  running application version (overridable via build ldflags) and
  `GitHubRepo` identifies the source repository.
- `checker_test.go` - Unit tests for `IsNewer` and `parseSemver`.

## Dependencies and interactions

- Imports `subrelay/internal/config` for the refresh interval, the
  subscription settings, and the default application version used as the
  release-check baseline.
- Imports `subrelay/internal/core` for the engine reload.
- Imports `subrelay/internal/ports` for port planning.
- Imports `subrelay/internal/state` for snapshot updates.
- Imports `subrelay/internal/sub` for the subscription fetcher.
- Imports `subrelay/internal/logging` for log output.
- Consumed by `cmd/subrelay` (main starts and stops both the timer and
  the release checker, and wires the manual check to the tray menu).
