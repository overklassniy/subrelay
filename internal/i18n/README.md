# internal/i18n

## Purpose

Provides minimal internationalization for the tray and dialog UI with
Russian and English dictionaries.

## Contents

- `i18n.go` - Defines the translation dictionary, `SetLanguage`,
  `Language`, and `T` (translation lookup). The active language is
  selected at startup from settings and can be switched at runtime,
  which triggers a tray menu rebuild.

## Dependencies and interactions

- No internal dependencies.
- Consumed by `subrelay/internal/tray` (menu labels),
  `subrelay/internal/ui` (window and dialog labels), and
  `cmd/subrelay` (initial language selection).
