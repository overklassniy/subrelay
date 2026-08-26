# internal/core

## Purpose

Generates sing-box `option.Options` from parsed subscription nodes, port
assignments, and user settings, and manages the sing-box instance
lifecycle (start, stop, reload).

## Contents

- `builder.go` - Converts `sub.Node`, `ports.Assignment`, and
  `config.Settings` into `option.Options` with inbounds, VLESS outbounds,
  RU/non-RU urltest groups, route rules, and DNS. `BuildResult.JSON()`
  pretty-prints the built options for the log viewer's config dump
  action.
- `engine.go` - Wraps `box.Box` lifecycle: creates the context with
  `include.Context`, attaches the urltest history storage, starts and
  stops the instance, and reloads by recreating the instance.
- `builder_test.go` - Tests config generation including balancer groups,
  RU overrides, auto-node exclusion, and xhttp transport with xmux.

## Dependencies and interactions

- Imports `subrelay/internal/config`, `subrelay/internal/ports`,
  `subrelay/internal/sub` for input data.
- Imports `github.com/sagernet/sing-box` (replaced by
  `github.com/shtorm-7/sing-box-extended`) for `box.Box`, `option.*`,
  `include.Context`, and `urltest.HistoryStorage`.
- Consumed by `subrelay/internal/update` (timer triggers reload) and
  `cmd/subrelay` (main starts/stops the engine).
