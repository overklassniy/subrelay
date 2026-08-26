// Package state holds the runtime snapshot of the application: the
// parsed nodes, port assignments, engine state, last update time, and
// last error. The tray and UI windows read this snapshot to render
// status and node lists without touching the engine directly.
package state

import (
	"sync"
	"time"

	"subrelay/internal/core"
	"subrelay/internal/ports"
	"subrelay/internal/sub"
)

// Snapshot is an immutable point-in-time view of the application state.
// Callers receive a copy so they can render without holding the manager
// lock.
type Snapshot struct {
	Nodes       []sub.Node
	Assignments []ports.Assignment
	RUNodes     []string
	NonRUNodes  []string
	EngineState core.State
	LastUpdate  time.Time
	LastError   string
	NodeCount   int
	ConfigJSON  string
}

// Manager stores the runtime snapshot under a mutex.
type Manager struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

// NewManager creates an empty Manager.
func NewManager() *Manager {
	return &Manager{}
}

// Update replaces the snapshot with the given values and clears the last
// error.
//
// Args:
//   - nodes: the parsed subscription nodes.
//   - assignments: the planned port assignments.
//   - result: the build result containing RU/non-RU node lists and the
//     configuration used to populate ConfigJSON.
//   - engineState: the current engine lifecycle state.
func (m *Manager) Update(nodes []sub.Node, assignments []ports.Assignment, result *core.BuildResult, engineState core.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot = Snapshot{
		Nodes:       nodes,
		Assignments: assignments,
		RUNodes:     result.RUNodes,
		NonRUNodes:  result.NonRUNodes,
		EngineState: engineState,
		LastUpdate:  time.Now(),
		NodeCount:   len(nodes),
		ConfigJSON:  result.JSON(),
	}
}

// SetEngineState updates only the engine state field.
func (m *Manager) SetEngineState(s core.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.EngineState = s
}

// SetError records an error message and preserves the last update time.
func (m *Manager) SetError(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.LastError = err
}

// ClearError removes the last error message.
func (m *Manager) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.LastError = ""
}

// Snapshot returns a copy of the current snapshot. The returned slice
// fields are safe to use without further synchronization.
func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := m.snapshot
	// Copy slices so the caller cannot mutate the internal state.
	if m.snapshot.Nodes != nil {
		snap.Nodes = append([]sub.Node(nil), m.snapshot.Nodes...)
	}
	if m.snapshot.Assignments != nil {
		snap.Assignments = append([]ports.Assignment(nil), m.snapshot.Assignments...)
	}
	if m.snapshot.RUNodes != nil {
		snap.RUNodes = append([]string(nil), m.snapshot.RUNodes...)
	}
	if m.snapshot.NonRUNodes != nil {
		snap.NonRUNodes = append([]string(nil), m.snapshot.NonRUNodes...)
	}
	return snap
}
