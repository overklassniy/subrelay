# internal/ports

## Purpose

Plans stable port assignments for per-node SOCKS5 and HTTP inbounds.
Ports are persisted across application restarts so proxy URLs remain
valid.

## Contents

- `planner.go` - Defines `Planner` and `Assignment`. The planner
  allocates SOCKS and HTTP ports from configurable start offsets,
  detects conflicts, preserves existing assignments, and persists the
  tag-to-port mapping in settings.
- `planner_test.go` - Tests initial allocation, stable remapping,
  added/removed nodes, conflicts, invalid ranges, and exhaustion.

## Dependencies and interactions

- Imports `subrelay/internal/config` for port range settings and
  persistence.
- Imports `subrelay/internal/sub` for node tags.
- Consumed by `subrelay/internal/core` (builder uses assignments for
  inbound ports) and `subrelay/internal/update` (timer plans ports on
  each refresh).
