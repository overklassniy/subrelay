// Package ui (logs_win.go) implements the log viewer window. It displays
// the ring buffer contents, optionally filtered by level, and offers
// actions to refresh, auto-refresh, clear, open the log file, and dump
// the last built sing-box configuration to disk.
package ui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"subrelay/internal/i18n"
	"subrelay/internal/logging"
	"subrelay/internal/paths"
	"subrelay/internal/state"
)

// autoRefreshInterval is how often the log view refreshes itself while
// auto-refresh is enabled and the window is visible.
const autoRefreshInterval = 2 * time.Second

// LogsWindow manages the log viewer window.
type LogsWindow struct {
	window  fyne.Window
	logger  *logging.Logger
	state   *state.Manager
	content *widget.TextGrid

	levelFilter *widget.Select
	autoRefresh *widget.Check
	stopAuto    chan struct{}
}

// NewLogsWindow creates a logs window bound to the given app, logger,
// and state manager (used for the config dump action).
//
// Args:
//   - a: the Fyne application.
//   - log: the shared logger whose ring buffer is displayed.
//   - sm: the state manager holding the last built configuration.
//
// Returns:
//   - A pointer to the new LogsWindow.
func NewLogsWindow(a fyne.App, log *logging.Logger, sm *state.Manager) *LogsWindow {
	w := a.NewWindow(i18n.T("logs.title"))
	content := widget.NewTextGrid()

	l := &LogsWindow{window: w, logger: log, state: sm, content: content}

	levelFilter := widget.NewSelect(
		[]string{i18n.T("logs.level.all"), "INFO", "WARN", "ERROR"},
		func(string) { l.Refresh() },
	)
	levelFilter.SetSelectedIndex(0)
	l.levelFilter = levelFilter

	refreshBtn := widget.NewButtonWithIcon(i18n.T("logs.refresh"), theme.ViewRefreshIcon(), l.Refresh)
	clearBtn := widget.NewButtonWithIcon(i18n.T("logs.clear"), theme.ContentClearIcon(), l.clear)
	openBtn := widget.NewButtonWithIcon(i18n.T("logs.open.file"), theme.FileIcon(), l.openFile)
	dumpBtn := widget.NewButtonWithIcon(i18n.T("logs.dump.config"), theme.DocumentSaveIcon(), l.dumpConfig)

	autoRefresh := widget.NewCheck(i18n.T("logs.autorefresh"), func(on bool) {
		if on {
			l.startAutoRefresh()
		} else {
			l.stopAutoRefresh()
		}
	})
	l.autoRefresh = autoRefresh

	toolbar := container.NewHBox(levelFilter, autoRefresh, refreshBtn, clearBtn, openBtn, dumpBtn)
	w.SetContent(container.NewBorder(toolbar, nil, nil, nil, container.NewScroll(content)))
	w.Resize(fyne.NewSize(800, 500))
	w.SetCloseIntercept(func() {
		l.stopAutoRefresh()
		w.Hide()
	})
	return l
}

// Show opens (or refreshes) the logs window with the current ring buffer.
func (l *LogsWindow) Show() {
	l.Refresh()
	l.window.Show()
}

// Refresh re-reads the ring buffer, applies the active level filter, and
// updates the displayed text.
func (l *LogsWindow) Refresh() {
	lines := l.logger.Lines()
	level := ""
	if l.levelFilter != nil && l.levelFilter.SelectedIndex() > 0 {
		level = l.levelFilter.Selected
	}
	l.content.SetText(formatLogs(filterLines(lines, level)))
}

// clear empties the in-memory log ring buffer and refreshes the view.
func (l *LogsWindow) clear() {
	l.logger.Clear()
	l.logger.Info("%s", i18n.T("logs.cleared"))
	l.Refresh()
}

// openFile opens the on-disk log file with the system's default handler.
func (l *LogsWindow) openFile() {
	path, err := paths.LogFile()
	if err != nil {
		dialog.ShowError(fmt.Errorf(i18n.T("logs.open.error"), err), l.window)
		return
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	if err := fyne.CurrentApp().OpenURL(u); err != nil {
		dialog.ShowError(fmt.Errorf(i18n.T("logs.open.error"), err), l.window)
	}
}

// dumpConfig writes the last built sing-box configuration (as pretty
// JSON) next to the log file and shows a confirmation dialog.
func (l *LogsWindow) dumpConfig() {
	snap := l.state.Snapshot()
	if snap.ConfigJSON == "" {
		dialog.ShowInformation(i18n.T("logs.dump.config"), i18n.T("logs.dump.empty"), l.window)
		return
	}

	dir, err := paths.ConfigDir()
	if err != nil {
		dialog.ShowError(fmt.Errorf(i18n.T("logs.dump.error"), err), l.window)
		return
	}
	path := filepath.Join(dir, "config-dump.json")
	if err := os.WriteFile(path, []byte(snap.ConfigJSON), 0o644); err != nil {
		dialog.ShowError(fmt.Errorf(i18n.T("logs.dump.error"), err), l.window)
		return
	}
	dialog.ShowInformation(i18n.T("logs.dump.config"), fmt.Sprintf(i18n.T("logs.dump.saved"), path), l.window)
}

// startAutoRefresh launches a ticker goroutine that refreshes the view
// periodically until stopAutoRefresh is called.
func (l *LogsWindow) startAutoRefresh() {
	if l.stopAuto != nil {
		return
	}
	l.stopAuto = make(chan struct{})
	stop := l.stopAuto
	go func() {
		ticker := time.NewTicker(autoRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fyne.Do(l.Refresh)
			}
		}
	}()
}

// stopAutoRefresh stops the auto-refresh goroutine started by
// startAutoRefresh, if any.
func (l *LogsWindow) stopAutoRefresh() {
	if l.stopAuto == nil {
		return
	}
	close(l.stopAuto)
	l.stopAuto = nil
}

// filterLines returns only the lines matching level, or all lines when
// level is empty.
func filterLines(lines []logging.Line, level string) []logging.Line {
	if level == "" {
		return lines
	}
	out := make([]logging.Line, 0, len(lines))
	for _, line := range lines {
		if line.Level == level {
			out = append(out, line)
		}
	}
	return out
}

// formatLogs converts the log lines into a plain monospace string for
// display in a TextGrid.
func formatLogs(lines []logging.Line) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line.Time.Format("2006-01-02 15:04:05"))
		b.WriteString(" [")
		b.WriteString(line.Level)
		b.WriteString("] ")
		b.WriteString(line.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
