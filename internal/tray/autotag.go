// Package tray (autotag.go) provides a local auto-node classifier used by
// the tray menu to skip auto-selector entries. It mirrors sub.IsAutoTag.
package tray

import "strings"

// isAutoTag reports whether tag marks an auto-selector node, matching
// the Latin "auto" and Cyrillic "авто" substrings case-insensitively.
func isAutoTag(tag string) bool {
	lower := strings.ToLower(tag)
	return strings.Contains(lower, "auto") ||
		strings.Contains(lower, "авто")
}
