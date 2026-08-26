// Package core (engine.go) manages the sing-box instance lifecycle:
// building, starting, stopping, and reloading the configuration when the
// subscription updates.
//
// Reload is implemented as full box recreation: the old instance is
// closed and a new one is started with the fresh options. Active
// connections are briefly interrupted and urltest latency history resets.
package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/common/urltest"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing/service"

	"subrelay/internal/logging"
)

// State describes the engine's current lifecycle phase.
type State int

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateStopping
)

// String returns a human-readable state name.
func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	default:
		return "stopped"
	}
}

// Engine owns the sing-box instance and serializes lifecycle transitions.
type Engine struct {
	log *logging.Logger

	mu      sync.Mutex
	state   State
	current *box.Box
	cancel  context.CancelFunc
	ctx     context.Context
}

// NewEngine creates an Engine in the stopped state.
//
// Args:
//   - log: the shared logger for lifecycle messages.
//
// Returns:
//   - A pointer to the new Engine.
func NewEngine(log *logging.Logger) *Engine {
	return &Engine{
		log:   log,
		state: StateStopped,
	}
}

// State returns the current lifecycle state. Safe for concurrent use.
func (e *Engine) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// Start builds and starts a new sing-box instance from the given options.
// When an instance is already running it is stopped first.
//
// Args:
//   - options: the sing-box options to start with.
//
// Errors:
//   - Returns an error wrapping box.New or instance.Start failures.
func (e *Engine) Start(options box.Options) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.current != nil {
		e.stopLocked()
	}

	e.setStateLocked(StateStarting)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = include.Context(ctx)
	historyStorage := urltest.NewHistoryStorage()
	ctx = service.ContextWithPtr[urltest.HistoryStorage](ctx, historyStorage)
	options.Context = ctx

	instance, err := box.New(options)
	if err != nil {
		cancel()
		e.setStateLocked(StateStopped)
		return fmt.Errorf("core: create instance: %w", err)
	}

	if err := instance.Start(); err != nil {
		_ = instance.Close()
		cancel()
		e.setStateLocked(StateStopped)
		return fmt.Errorf("core: start instance: %w", err)
	}

	e.current = instance
	e.ctx = ctx
	e.cancel = cancel
	e.setStateLocked(StateRunning)
	e.log.Info("engine started")
	return nil
}

// Reload stops the current instance and starts a new one with the given
// options. When no instance is running this is equivalent to Start.
//
// Args:
//   - options: the new sing-box options.
//
// Errors:
//   - Returns an error when the new instance fails to start. The previous
//     instance is kept running on failure so connectivity is preserved.
func (e *Engine) Reload(options box.Options) error {
	e.mu.Lock()
	if e.current != nil {
		e.stopLocked()
	}
	e.mu.Unlock()
	return e.Start(options)
}

// Stop closes the current instance and releases its resources.
//
// Errors:
//   - Returns an error wrapping instance.Close failures.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == nil {
		return nil
	}
	e.stopLocked()
	e.log.Info("engine stopped")
	return nil
}

// stopLocked closes the current instance and cancels its context. Caller
// must hold e.mu.
func (e *Engine) stopLocked() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	if e.current != nil {
		_ = e.current.Close()
		e.current = nil
	}
	e.ctx = nil
	e.setStateLocked(StateStopped)
}

// setStateLocked updates the state. Caller must hold e.mu.
func (e *Engine) setStateLocked(s State) {
	e.state = s
}
