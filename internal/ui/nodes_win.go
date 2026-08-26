// Package ui (nodes_win.go) implements the nodes table window showing
// each node's name, transport type, RU/non-RU checkboxes, and SOCKS/HTTP
// ports with copy buttons. The RU and non-RU checkboxes are mutually
// exclusive and toggle the override, triggering a group rebuild.
package ui

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"subrelay/internal/config"
	"subrelay/internal/i18n"
	"subrelay/internal/state"
	"subrelay/internal/sub"
)

// NodesCallbacks holds actions triggered from the nodes window.
type NodesCallbacks struct {
	OnRUOverrideChanged func(tag string, isRU bool)
	OnRefresh           func()
}

// nodesTableColumns is the fixed set of columns rendered by the table:
// name, transport, RU, non-RU, SOCKS, HTTP.
const nodesTableColumns = 6

// NodesWindow manages the nodes table window.
type NodesWindow struct {
	window    fyne.Window
	state     *state.Manager
	settings  *config.Settings
	callbacks NodesCallbacks

	search *widget.Entry
	table  *widget.Table
	nodes  []sub.Node
	ports  map[string]struct{ socks, http int }
}

// NewNodesWindow creates a nodes window bound to the given app.
//
// Args:
//   - a: the Fyne application.
//   - sm: the state manager.
//   - settings: the application settings (for RU overrides).
//   - cb: callbacks for RU override changes.
//
// Returns:
//   - A pointer to the new NodesWindow.
func NewNodesWindow(a fyne.App, sm *state.Manager, settings *config.Settings, cb NodesCallbacks) *NodesWindow {
	w := a.NewWindow(i18n.T("nodes.title"))
	nw := &NodesWindow{window: w, state: sm, settings: settings, callbacks: cb}
	w.SetCloseIntercept(func() {
		w.Hide()
	})
	return nw
}

// Show opens the nodes window, (re)building its content from the latest
// state snapshot.
func (n *NodesWindow) Show() {
	n.window.SetContent(n.buildContent())
	n.window.Resize(fyne.NewSize(760, 500))
	n.window.Show()
}

// buildContent assembles the toolbar (search + refresh) and the node
// table. It is only called once per Show(); subsequent updates refresh
// the table in place instead of rebuilding the widget tree.
func (n *NodesWindow) buildContent() fyne.CanvasObject {
	n.search = widget.NewEntry()
	n.search.SetPlaceHolder(i18n.T("nodes.search"))
	n.search.OnChanged = func(string) { n.reload() }

	refreshBtn := widget.NewButtonWithIcon(i18n.T("nodes.refresh"), theme.ViewRefreshIcon(), func() {
		if n.callbacks.OnRefresh != nil {
			n.callbacks.OnRefresh()
		}
		n.reload()
	})
	toolbar := container.NewBorder(nil, nil, nil, refreshBtn, n.search)

	n.table = n.buildTable()
	n.reload()

	return container.NewBorder(toolbar, nil, nil, nil, n.table)
}

// reload re-reads the state snapshot, applies the current search filter,
// and refreshes the table in place (no widget tree rebuild).
func (n *NodesWindow) reload() {
	snap := n.state.Snapshot()

	n.ports = map[string]struct{ socks, http int }{}
	for _, a := range snap.Assignments {
		n.ports[a.Tag] = struct{ socks, http int }{a.SOCKS, a.HTTP}
	}

	query := ""
	if n.search != nil {
		query = n.search.Text
	}
	n.nodes = filterNodes(snap.Nodes, query)
	n.table.Refresh()
}

// filterNodes returns nodes whose tag contains the query
// (case-insensitive), excluding auto nodes, sorted alphabetically by tag
// for a stable, predictable order.
func filterNodes(nodes []sub.Node, q string) []sub.Node {
	ql := strings.ToLower(q)
	out := make([]sub.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.IsAuto() {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(node.Tag), ql) {
			continue
		}
		out = append(out, node)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

// buildTable builds the (initially empty) node table with columns: name,
// transport, RU, non-RU, SOCKS, HTTP, and a header row. Row content is
// populated by reload() via the table's data callbacks reading
// n.nodes/n.ports.
//
// The cell template pre-creates all possible widgets (label, two checks,
// button) and UpdateCell reuses them by toggling visibility instead of
// allocating new widgets on every scroll-driven cell recycle, which
// would cause severe scroll lag.
//
// The RU and non-RU checkboxes are mutually exclusive: checking one
// unchecks the other by updating the override and refreshing the table.
func (n *NodesWindow) buildTable() *widget.Table {
	table := widget.NewTable(
		func() (int, int) {
			return len(n.nodes), nodesTableColumns
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			ruCheck := widget.NewCheck("", nil)
			nonruCheck := widget.NewCheck("", nil)
			btn := widget.NewButtonWithIcon(i18n.T("nodes.copy"), theme.ContentCopyIcon(), nil)
			ruCheck.Hide()
			nonruCheck.Hide()
			btn.Hide()
			return container.NewHBox(lbl, ruCheck, nonruCheck, btn)
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			if id.Row >= len(n.nodes) {
				return
			}
			node := n.nodes[id.Row]
			hbox := o.(*fyne.Container)
			lbl := hbox.Objects[0].(*widget.Label)
			ruCheck := hbox.Objects[1].(*widget.Check)
			nonruCheck := hbox.Objects[2].(*widget.Check)
			btn := hbox.Objects[3].(*widget.Button)

			lbl.Hide()
			ruCheck.Hide()
			nonruCheck.Hide()
			btn.Hide()

			switch id.Col {
			case 0:
				lbl.SetText(node.Tag)
				lbl.Show()
			case 1:
				lbl.SetText(node.TransportType())
				lbl.Show()
			case 2:
				isRU := node.IsRU()
				n.settings.Lock()
				if ov, ok := n.settings.RUOverrides[node.Tag]; ok {
					isRU = ov
				}
				n.settings.Unlock()
				ruCheck.OnChanged = nil
				ruCheck.SetChecked(isRU)
				ruCheck.OnChanged = func(checked bool) {
					n.settings.Lock()
					n.settings.RUOverrides[node.Tag] = checked
					n.settings.Unlock()
					_ = n.settings.Save()
					n.callbacks.OnRUOverrideChanged(node.Tag, checked)
					n.table.Refresh()
				}
				ruCheck.Show()
			case 3:
				isRU := node.IsRU()
				n.settings.Lock()
				if ov, ok := n.settings.RUOverrides[node.Tag]; ok {
					isRU = ov
				}
				n.settings.Unlock()
				nonruCheck.OnChanged = nil
				nonruCheck.SetChecked(!isRU)
				nonruCheck.OnChanged = func(checked bool) {
					n.settings.Lock()
					n.settings.RUOverrides[node.Tag] = !checked
					n.settings.Unlock()
					_ = n.settings.Save()
					n.callbacks.OnRUOverrideChanged(node.Tag, !checked)
					n.table.Refresh()
				}
				nonruCheck.Show()
			case 4:
				ports := n.ports[node.Tag]
				addr := fmt.Sprintf("socks5://127.0.0.1:%d", ports.socks)
				btn.OnTapped = func() { n.window.Clipboard().SetContent(addr) }
				btn.Show()
			case 5:
				ports := n.ports[node.Tag]
				addr := fmt.Sprintf("http://127.0.0.1:%d", ports.http)
				btn.OnTapped = func() { n.window.Clipboard().SetContent(addr) }
				btn.Show()
			}
		},
	)

	table.ShowHeaderRow = true
	table.CreateHeader = func() fyne.CanvasObject {
		hdr := widget.NewLabel("Column")
		hdr.TextStyle = fyne.TextStyle{Bold: true}
		return hdr
	}
	table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		label := o.(*widget.Label)
		label.TextStyle = fyne.TextStyle{Bold: true}
		switch id.Col {
		case 0:
			label.SetText(i18n.T("nodes.name"))
		case 1:
			label.SetText(i18n.T("nodes.transport"))
		case 2:
			label.SetText(i18n.T("nodes.ru"))
		case 3:
			label.SetText(i18n.T("nodes.nonru"))
		case 4:
			label.SetText(i18n.T("nodes.socks"))
		case 5:
			label.SetText(i18n.T("nodes.http"))
		}
	}

	// Column widths.
	table.SetColumnWidth(0, 250)
	table.SetColumnWidth(1, 80)
	table.SetColumnWidth(2, 50)
	table.SetColumnWidth(3, 60)
	table.SetColumnWidth(4, 150)
	table.SetColumnWidth(5, 150)
	return table
}
