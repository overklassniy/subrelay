# cmd

## Purpose

Contains the application entry point packages for Subrelay.

## Contents

- `subrelay/` - The main application entry point. Wires together all
  internal packages, manages the Fyne event loop, and implements the
  single-instance lock.

## Dependencies and interactions

- `cmd/subrelay` imports all `subrelay/internal/*` packages and
  `fyne.io/fyne/v2/app`.
- Requires CGO for the Fyne GLFW backend.
