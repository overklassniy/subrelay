// Package tray (icon.go) embeds the tray icon PNG and exposes it as a
// Fyne resource.
package tray

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconBytes []byte

// iconResource returns the embedded tray icon as a Fyne static resource.
//
// Returns:
//   - A fyne.Resource wrapping the embedded PNG bytes.
func iconResource() fyne.Resource {
	return fyne.NewStaticResource("icon.png", iconBytes)
}

// AppIcon returns the application icon as a Fyne resource. This is the
// same embedded PNG used for the tray icon, exposed for use as the
// application icon and window icons.
//
// Returns:
//   - A fyne.Resource wrapping the embedded PNG bytes.
func AppIcon() fyne.Resource {
	return iconResource()
}
