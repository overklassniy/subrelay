// Package update (timer.go) runs the periodic subscription refresh
// goroutine. On each tick it fetches the subscription, parses it, plans
// ports, builds new options, and reloads the engine. Errors do not stop
// the timer or crash the process; the previous engine instance keeps
// running when a reload fails.
package update

import (
	"context"
	"time"

	"github.com/sagernet/sing-box"

	"subrelay/internal/config"
	"subrelay/internal/core"
	"subrelay/internal/logging"
	"subrelay/internal/ports"
	"subrelay/internal/state"
	"subrelay/internal/sub"
)

// Timer owns the periodic refresh goroutine.
type Timer struct {
	settings    *config.Settings
	engine      *core.Engine
	state       *state.Manager
	planner     *ports.Planner
	fetcher     *sub.Fetcher
	log         *logging.Logger
	onRefreshed func() // called after each refresh cycle (success or error)

	cancel context.CancelFunc
	done   chan struct{}
}

// NewTimer creates a Timer bound to the given dependencies.
//
// Args:
//   - settings: the application settings (read for the interval and
//     subscription URL).
//   - engine: the sing-box engine to reload.
//   - sm: the state manager to update after each refresh.
//   - planner: the port planner used for stable port allocation.
//   - log: the shared logger.
//
// Returns:
//   - A pointer to the new Timer.
func NewTimer(settings *config.Settings, engine *core.Engine, sm *state.Manager, planner *ports.Planner, log *logging.Logger) *Timer {
	return &Timer{
		settings: settings,
		engine:   engine,
		state:    sm,
		planner:  planner,
		fetcher:  sub.NewFetcher(settings),
		log:      log,
	}
}

// SetOnRefreshed sets a callback invoked after each refresh cycle. The
// callback is used to rebuild the tray icon and menu so they reflect
// the latest state.
//
// Args:
//   - fn: the callback to invoke after each refresh.
func (t *Timer) SetOnRefreshed(fn func()) {
	t.onRefreshed = fn
}

// Start launches the refresh goroutine. The first refresh runs
// immediately; subsequent refreshes run at the configured interval.
//
// Args:
//   - ctx: the parent context. When it is cancelled the goroutine exits.
func (t *Timer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.done = make(chan struct{})

	go t.loop(ctx)
}

// Stop cancels the refresh goroutine and waits for it to exit.
func (t *Timer) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
	if t.done != nil {
		<-t.done
	}
}

// loop runs the refresh cycle until the context is cancelled.
func (t *Timer) loop(ctx context.Context) {
	defer close(t.done)

	t.refreshOnce(ctx)

	t.settings.Lock()
	interval := time.Duration(t.settings.UpdateIntervalMin) * time.Minute
	t.settings.Unlock()

	if interval <= 0 {
		interval = time.Duration(config.DefaultUpdateIntervalMin) * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.refreshOnce(ctx)
			// Re-read the interval in case the user changed it.
			t.settings.Lock()
			newInterval := time.Duration(t.settings.UpdateIntervalMin) * time.Minute
			t.settings.Unlock()
			if newInterval > 0 && newInterval != interval {
				interval = newInterval
				ticker.Reset(interval)
			}
		}
	}
}

// refreshOnce performs a single fetch-parse-plan-build-reload cycle.
// Errors are logged and recorded in the state manager but do not stop
// the timer or shut down the engine.
func (t *Timer) refreshOnce(ctx context.Context) {
	nodes, err := t.fetcher.Fetch(ctx)
	if err != nil {
		t.log.Error("update: fetch/parse failed: %v", err)
		t.state.SetError(err.Error())
		t.notifyRefreshed()
		return
	}

	assignments, err := t.planner.Plan(nodes)
	if err != nil {
		t.log.Error("update: port planning failed: %v", err)
		t.state.SetError(err.Error())
		t.notifyRefreshed()
		return
	}

	result, err := core.Build(core.BuildInput{
		Settings:    t.settings,
		Nodes:       nodes,
		Assignments: assignments,
	})
	if err != nil {
		t.log.Error("update: build failed: %v", err)
		t.state.SetError(err.Error())
		t.notifyRefreshed()
		return
	}

	// Reload the engine. On failure the engine keeps the previous
	// instance running (see core.Engine.Reload).
	if err := t.engine.Reload(box.Options{Options: result.Options}); err != nil {
		t.log.Error("update: reload failed: %v", err)
		t.state.SetError(err.Error())
		t.notifyRefreshed()
		return
	}

	t.state.Update(nodes, assignments, result, t.engine.State())
	t.state.ClearError()
	t.notifyRefreshed()
	t.log.Info("update: refreshed %d nodes", len(nodes))
}

// notifyRefreshed calls the onRefreshed callback when set. It is invoked
// after every refresh cycle so the tray icon and menu can be rebuilt.
func (t *Timer) notifyRefreshed() {
	if t.onRefreshed != nil {
		t.onRefreshed()
	}
}
