// Package main is the Subrelay application entry point. It initializes
// the settings, logger, engine, state, tray, UI windows, and the
// auto-update timer, then runs the Fyne event loop until the user exits
// from the tray.
//
// A single-instance lock prevents two concurrent instances from binding
// the same local ports. On Windows the lock uses a named mutex; on Linux
// it uses a file lock in the data directory.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sagernet/sing-box"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"subrelay/internal/config"
	"subrelay/internal/core"
	"subrelay/internal/i18n"
	"subrelay/internal/logging"
	"subrelay/internal/ports"
	"subrelay/internal/state"
	"subrelay/internal/sub"
	"subrelay/internal/tray"
	"subrelay/internal/ui"
	"subrelay/internal/update"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "subrelay: %v\n", err)
		os.Exit(1)
	}
}

// run initializes all subsystems and blocks on the Fyne event loop.
//
// Errors:
//   - Returns an error wrapping any fatal startup failure.
func run() error {
	// Single-instance lock.
	lock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("another instance is already running: %w", err)
	}
	defer releaseLock(lock)

	// Load settings.
	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	i18n.SetLanguage(settings.Language)

	// Logger.
	log := logging.Global()

	// Core engine.
	engine := core.NewEngine(log)

	// State manager.
	sm := state.NewManager()

	// Port planner.
	planner := ports.NewPlanner(settings)

	// Fyne application.
	a := app.New()
	a.SetIcon(tray.AppIcon())
	// Hidden window used only for clipboard access and as a dialog
	// parent. It is never shown on startup; the app lives in the tray.
	w := a.NewWindow("Subrelay")
	w.SetIcon(tray.AppIcon())
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// UI windows.
	settingsWin := ui.NewSettingsWindow(a, settings, ui.SettingsCallbacks{
		OnApply: func() {
			refreshNow(engine, sm, planner, settings, log)
		},
		OnCheckUpdates: func(onResult func(text string, releaseURL string)) {
			checkUpdatesNow(log, onResult)
		},
	})
	nodesWin := ui.NewNodesWindow(a, sm, settings, ui.NodesCallbacks{
		OnRUOverrideChanged: func(tag string, isRU bool) {
			refreshNow(engine, sm, planner, settings, log)
		},
		OnRefresh: func() { refreshNow(engine, sm, planner, settings, log) },
	})
	logsWin := ui.NewLogsWindow(a, log, sm)

	// Tray.
	t := tray.New(a, w, sm, settings, tray.Callbacks{
		OnNodes:    nodesWin.Show,
		OnSettings: settingsWin.Show,
		OnLogs:     logsWin.Show,
		OnExit:     func() { shutdown(engine, log); a.Quit() },
	})

	// Auto-update timer.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timer := update.NewTimer(settings, engine, sm, planner, log)
	timer.SetOnRefreshed(func() {
		if globalTray != nil {
			globalTray.Rebuild()
		}
	})
	timer.Start(ctx)

	// GitHub release checker. Runs on startup and then once per day.
	// On a newer release it sends a system notification; the manual
	// check from the settings window shows a dialog with a link to the
	// release.
	updChecker := update.NewChecker(update.CurrentVersion, update.DefaultCheckInterval, log)
	updChecker.SetOnAvailable(func(rel update.Release) {
		notify := fyne.NewNotification(
			i18n.T("update.available.title"),
			fmt.Sprintf(i18n.T("update.available.notify"), rel.TagName),
		)
		fyne.Do(func() { a.SendNotification(notify) })
	})
	updChecker.Start(ctx)

	// First-run wizard when no subscription URL is configured.
	// The wizard needs a visible parent window, so show w temporarily
	// and hide it after the dialog is dismissed.
	settings.Lock()
	url := settings.SubscriptionURL
	settings.Unlock()
	if url == "" {
		w.Show()
		settingsWin.ShowWizard(w)
	}

	// Expose the tray reference so language changes can rebuild it.
	globalTray = t

	// Run the event loop without showing the main window. The app
	// lives entirely in the system tray; windows are shown on demand.
	a.Run()

	timer.Stop()
	updChecker.Stop()
	return nil
}

// globalTray holds the tray instance so language-change callbacks can
// rebuild the menu.
var globalTray *tray.Tray

// refreshNow performs a synchronous fetch-parse-plan-build-reload cycle
// and updates the state manager and tray menu.
func refreshNow(engine *core.Engine, sm *state.Manager, planner *ports.Planner, settings *config.Settings, log *logging.Logger) {
	settings.Lock()
	url := settings.SubscriptionURL
	settings.Unlock()
	if url == "" {
		log.Warn("refresh: no subscription URL configured")
		return
	}

	fetcher := sub.NewFetcher(settings)
	nodes, err := fetcher.Fetch(context.Background())
	if err != nil {
		log.Error("refresh: fetch failed: %v", err)
		sm.SetError(err.Error())
		if globalTray != nil {
			globalTray.Rebuild()
		}
		return
	}

	assignments, err := planner.Plan(nodes)
	if err != nil {
		log.Error("refresh: port planning failed: %v", err)
		sm.SetError(err.Error())
		if globalTray != nil {
			globalTray.Rebuild()
		}
		return
	}

	result, err := core.Build(core.BuildInput{
		Settings:    settings,
		Nodes:       nodes,
		Assignments: assignments,
	})
	if err != nil {
		log.Error("refresh: build failed: %v", err)
		sm.SetError(err.Error())
		if globalTray != nil {
			globalTray.Rebuild()
		}
		return
	}

	if err := engine.Reload(box.Options{Options: result.Options}); err != nil {
		log.Error("refresh: reload failed: %v", err)
		sm.SetError(err.Error())
		if globalTray != nil {
			globalTray.Rebuild()
		}
		return
	}

	sm.Update(nodes, assignments, result, engine.State())
	sm.ClearError()
	if globalTray != nil {
		globalTray.Rebuild()
	}
	log.Info("refresh: %d nodes loaded", len(nodes))
}

// shutdown stops the engine and closes the logger.
func shutdown(engine *core.Engine, log *logging.Logger) {
	_ = engine.Stop()
	_ = log.Close()
}

// checkUpdatesNow runs a GitHub release check on a background goroutine
// and reports the result via the onResult callback instead of opening a
// dialog. When a newer release is found, the verdict text and the release
// page URL are passed to onResult so the caller can display them inline.
//
// Args:
//   - log: the shared logger.
//   - onResult: the callback invoked exactly once with the verdict text
//     and, when non-empty, the release page URL. It is called from the
//     checker goroutine, so any UI work must be dispatched to the main
//     thread by the caller.
func checkUpdatesNow(log *logging.Logger, onResult func(text string, releaseURL string)) {
	checker := update.NewChecker(update.CurrentVersion, update.DefaultCheckInterval, log)
	checker.SetOnAvailable(func(rel update.Release) {
		text := fmt.Sprintf(i18n.T("update.available.verdict"), rel.TagName, update.CurrentVersion)
		onResult(text, rel.HTMLURL)
	})
	checker.SetOnUpToDate(func() {
		text := fmt.Sprintf(i18n.T("update.uptodate.verdict"), update.CurrentVersion)
		onResult(text, "")
	})
	checker.SetOnNoReleases(func() {
		onResult(i18n.T("update.no_releases.verdict"), "")
	})
	checker.SetOnError(func(err error) {
		text := fmt.Sprintf(i18n.T("update.error.verdict"), err)
		onResult(text, "")
	})

	go checker.CheckNow(context.Background())
}
