# internal/sub

## Purpose

Fetches and parses VLESS subscription links into `Node` structs
consumed by the port planner and configuration builder.

## Contents

- `fetch.go` - Downloads the subscription body over HTTP with the Happ
  client headers (User-Agent, X-HWID, X-Device-OS, etc.). Defines
  `Fetcher` with a `Fetch` method.
- `parse.go` - Parses VLESS links from plain text or Base64-encoded
  subscription bodies. Supports TCP, gRPC, WebSocket, and xhttp
  transports; TLS with uTLS; and Reality. Defines `Node`, `Transport`,
  `TLS`, `UTLS`, `Reality`, and helper functions `SafeTag`, `IsAutoTag`,
  `IsRU`.
- `parse_test.go` - Tests plain text, Base64, Reality, TLS, gRPC,
  xhttp, WebSocket, malformed links, Unicode tags, and duplicate/empty
  tags.

## Dependencies and interactions

- Imports `subrelay/internal/config` for subscription request headers.
- Consumed by `subrelay/internal/ports` (node tags for port planning),
  `subrelay/internal/core` (builder converts nodes to outbounds),
  `subrelay/internal/update` (timer fetches and parses), and
  `subrelay/internal/ui` (nodes table).
