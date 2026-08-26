// Package tray builds and updates the system tray icon, menu, and
// notifications using Fyne's desktop.App integration. The menu is rebuilt
// whenever the state snapshot or active language changes.
package tray

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"subrelay/internal/config"
	"subrelay/internal/i18n"
	"subrelay/internal/state"
)

// Callbacks holds the actions triggered by tray menu items.
type Callbacks struct {
	OnNodes    func()
	OnSettings func()
	OnLogs     func()
	OnExit     func()
}

// Tray manages the system tray icon and menu.
type Tray struct {
	app        desktop.App
	window     fyne.Window
	state      *state.Manager
	settings   *config.Settings
	callbacks  Callbacks
	balancerPorts config.BalancerPorts
}

// New creates a Tray bound to the given Fyne app and window.
//
// Args:
//   - a: the Fyne application (must implement desktop.App).
//   - w: the hidden window used for clipboard access.
//   - sm: the state manager.
//   - settings: the application settings.
//   - cb: the menu action callbacks.
//
// Returns:
//   - A pointer to the new Tray, or nil when the app does not support
//     desktop tray integration.
func New(a fyne.App, w fyne.Window, sm *state.Manager, settings *config.Settings, cb Callbacks) *Tray {
	desk, ok := a.(desktop.App)
	if !ok {
		return nil
	}
	t := &Tray{
		app:       desk,
		window:    w,
		state:     sm,
		settings:  settings,
		callbacks: cb,
	}
	settings.Lock()
	t.balancerPorts = settings.BalancerPorts
	settings.Unlock()
	t.SetIcon()
	t.Rebuild()
	return t
}

// SetIcon sets the tray icon from the embedded asset. Used only for the
// initial icon before the first state update.
func (t *Tray) SetIcon() {
	t.app.SetSystemTrayIcon(iconResource())
}

// updateIcon generates and sets a dynamic tray icon reflecting the
// current engine state and error status.
func (t *Tray) updateIcon(snap state.Snapshot) {
	t.app.SetSystemTrayIcon(iconForSnapshot(snap))
}

// Rebuild reconstructs the tray menu and icon from the current state
// snapshot and active language. Call this after a subscription update,
// language change, or engine state transition.
func (t *Tray) Rebuild() {
	snap := t.state.Snapshot()
	t.updateIcon(snap)
	t.app.SetSystemTrayMenu(t.buildMenu(snap))
}

// buildMenu assembles the full tray menu.
func (t *Tray) buildMenu(snap state.Snapshot) *fyne.Menu {
	items := []*fyne.MenuItem{}

	// Status line.
	items = append(items, fyne.NewMenuItem(t.statusText(snap), nil))
	items[len(items)-1].Disabled = true

	// Balancers submenu.
	balItem := fyne.NewMenuItem(i18n.T("tray.balancers"), nil)
	balItem.ChildMenu = t.balancerMenu()
	items = append(items, balItem)

	// Nodes submenu.
	nodeItem := fyne.NewMenuItem(i18n.T("tray.nodes"), nil)
	nodeItem.ChildMenu = t.nodesMenu(snap)
	items = append(items, nodeItem)

	// Separator between submenus and actions.
	items = append(items, fyne.NewMenuItemSeparator())

	// Actions.
	items = append(items,
		fyne.NewMenuItem(i18n.T("tray.nodes_window"), t.callbacks.OnNodes),
		fyne.NewMenuItem(i18n.T("tray.settings"), t.callbacks.OnSettings),
		fyne.NewMenuItem(i18n.T("tray.logs"), t.callbacks.OnLogs),
		fyne.NewMenuItem(i18n.T("tray.exit"), t.callbacks.OnExit),
	)

	return fyne.NewMenu(i18n.T("app.name"), items...)
}

// statusText renders the status line from the snapshot.
func (t *Tray) statusText(snap state.Snapshot) string {
	if snap.LastError != "" {
		return fmt.Sprintf(i18n.T("tray.status.error"), snap.LastError)
	}
	if snap.EngineState.String() == "running" && !snap.LastUpdate.IsZero() {
		return fmt.Sprintf(i18n.T("tray.status.running"),
			snap.NodeCount, snap.LastUpdate.Format(time.RFC822))
	}
	return i18n.T("tray.status.idle")
}

// balancerMenu builds the four balancer address entries with copy actions.
func (t *Tray) balancerMenu() *fyne.Menu {
	bp := t.balancerPorts
	entries := []struct {
		label string
		port  int
	}{
		{i18n.T("tray.balancer.ru.socks"), bp.RUSocks},
		{i18n.T("tray.balancer.ru.http"), bp.RUHTTP},
		{i18n.T("tray.balancer.nonru.socks"), bp.NonRUSocks},
		{i18n.T("tray.balancer.nonru.http"), bp.NonRUHTTP},
	}
	items := []*fyne.MenuItem{}
	for _, e := range entries {
		addr := fmt.Sprintf("127.0.0.1:%d", e.port)
		items = append(items, fyne.NewMenuItem(
			fmt.Sprintf("%s - %s", e.label, addr),
			t.copyAction(addr),
		))
	}
	return fyne.NewMenu(i18n.T("tray.balancers"), items...)
}

// nodesMenu builds the per-node entries with copy actions.
func (t *Tray) nodesMenu(snap state.Snapshot) *fyne.Menu {
	assignmentByTag := map[string]int{}
	for _, a := range snap.Assignments {
		assignmentByTag[a.Tag] = a.SOCKS
	}
	items := []*fyne.MenuItem{}
	for _, node := range snap.Nodes {
		if isAutoNode(node.Tag) {
			continue
		}
		port, ok := assignmentByTag[node.Tag]
		if !ok {
			continue
		}
		addr := fmt.Sprintf("socks5://127.0.0.1:%d", port)
		items = append(items, fyne.NewMenuItem(node.Tag, t.copyAction(addr)))
	}
	if len(items) == 0 {
		placeholder := fyne.NewMenuItem(i18n.T("nodes.empty"), nil)
		placeholder.Disabled = true
		items = append(items, placeholder)
	}
	return fyne.NewMenu(i18n.T("tray.nodes"), items...)
}

// copyAction returns a callback that copies the address to the clipboard.
func (t *Tray) copyAction(addr string) func() {
	return func() {
		t.window.Clipboard().SetContent(addr)
	}
}

// isAutoNode reports whether a tag marks an auto-selector node. Mirrors
// sub.IsAutoTag without importing the sub package (to avoid a cycle in
// future refactors).
func isAutoNode(tag string) bool {
	return isAutoTag(tag)
}
