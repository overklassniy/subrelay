// Package ports assigns stable local port pairs (SOCKS + HTTP) to each
// node tag and to the balancer inbounds. Assignments are persisted in
// settings so node ports do not shift between subscription updates.
//
// The planner allocates ports from configurable ranges, detects
// conflicts, and reuses existing assignments for known tags.
package ports

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"subrelay/internal/config"
	"subrelay/internal/sub"
)

// maxPort is the highest TCP port number allowed for an assignment.
const maxPort = 65535

// Assignment is a single tag-to-ports mapping.
type Assignment struct {
	Tag   string
	SOCKS int
	HTTP  int
}

// Planner assigns and remembers port pairs for node tags.
type Planner struct {
	settings *config.Settings
	mu       sync.Mutex
}

// NewPlanner creates a Planner bound to the given settings. Port
// assignments are read and written through settings.PortAssignments.
//
// Args:
//   - settings: the application settings holding the port ranges and the
//     persisted tag-to-ports mapping.
//
// Returns:
//   - A pointer to the new Planner.
func NewPlanner(settings *config.Settings) *Planner {
	return &Planner{settings: settings}
}

// Plan computes port assignments for the given nodes. Existing
// assignments in settings.PortAssignments are reused; new tags receive
// the next free port in each range. The result excludes auto nodes.
//
// Args:
//   - nodes: the parsed subscription nodes.
//
// Returns:
//   - A slice of Assignment values in the same order as the input nodes
//     (auto nodes excluded).
//
// Errors:
//   - Returns an error when the port range is exhausted.
func (p *Planner) Plan(nodes []sub.Node) ([]Assignment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.settings.Lock()
	defer p.settings.Unlock()

	socksStart := p.settings.SOCKSPortStart
	httpStart := p.settings.HTTPPortStart
	assignments := p.settings.PortAssignments

	// Collect the set of ports already in use by persisted assignments and
	// by the balancer ports.
	usedSOCKS := make(map[int]struct{})
	usedHTTP := make(map[int]struct{})
	reserveBalancer(usedSOCKS, usedHTTP, p.settings.BalancerPorts)

	for _, raw := range assignments {
		s, h, ok := parseAssignment(raw)
		if !ok {
			continue
		}
		usedSOCKS[s] = struct{}{}
		usedHTTP[h] = struct{}{}
	}

	nextSOCKS := socksStart
	nextHTTP := httpStart

	out := make([]Assignment, 0, len(nodes))
	for i := range nodes {
		node := &nodes[i]
		if node.IsAuto() {
			continue
		}
		if raw, ok := assignments[node.Tag]; ok {
			if s, h, ok := parseAssignment(raw); ok {
				out = append(out, Assignment{Tag: node.Tag, SOCKS: s, HTTP: h})
				continue
			}
		}
		s, err := nextFree(usedSOCKS, &nextSOCKS, socksStart)
		if err != nil {
			return nil, err
		}
		h, err := nextFree(usedHTTP, &nextHTTP, httpStart)
		if err != nil {
			return nil, err
		}
		usedSOCKS[s] = struct{}{}
		usedHTTP[h] = struct{}{}
		assignments[node.Tag] = formatAssignment(s, h)
		out = append(out, Assignment{Tag: node.Tag, SOCKS: s, HTTP: h})
	}
	return out, nil
}

// nextFree returns the next port at or after *cursor that is not in used,
// advancing *cursor past it. When *cursor is below the range start it is
// reset to start.
//
// Args:
//   - used: the set of ports already taken.
//   - cursor: the allocation cursor, advanced in place.
//   - start: the minimum port of the range.
//
// Returns:
//   - The next free port.
//
// Errors:
//   - Returns an error when the range is exhausted.
func nextFree(used map[int]struct{}, cursor *int, start int) (int, error) {
	if *cursor < start {
		*cursor = start
	}
	for *cursor <= maxPort {
		port := *cursor
		*cursor++
		if _, taken := used[port]; !taken {
			return port, nil
		}
	}
	return 0, fmt.Errorf("ports: range exhausted at %d", maxPort)
}

// reserveBalancer marks the four balancer ports as in use so per-node
// allocation never collides with them.
func reserveBalancer(usedSOCKS, usedHTTP map[int]struct{}, bp config.BalancerPorts) {
	usedSOCKS[bp.RUSocks] = struct{}{}
	usedSOCKS[bp.NonRUSocks] = struct{}{}
	usedHTTP[bp.RUHTTP] = struct{}{}
	usedHTTP[bp.NonRUHTTP] = struct{}{}
}

// formatAssignment serializes a port pair into the persisted string form
// "socks:http".
func formatAssignment(s, h int) string {
	return strconv.Itoa(s) + ":" + strconv.Itoa(h)
}

// parseAssignment parses a persisted "socks:http" string.
func parseAssignment(raw string) (int, int, bool) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	s, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || s <= 0 || h <= 0 {
		return 0, 0, false
	}
	return s, h, true
}

// DetectConflicts returns the tags whose assigned ports collide with the
// balancer ports or with each other. This is used by the settings window
// to warn the user before applying changes.
//
// Args:
//   - assignments: the planned assignments.
//   - bp: the configured balancer ports.
//
// Returns:
//   - A slice of human-readable conflict descriptions, empty when there
//     are no conflicts.
func DetectConflicts(assignments []Assignment, bp config.BalancerPorts) []string {
	seenSOCKS := make(map[int]string)
	seenHTTP := make(map[int]string)
	balancerPorts := map[int]string{
		bp.RUSocks:    "balancer ru_socks",
		bp.NonRUSocks: "balancer nonru_socks",
		bp.RUHTTP:     "balancer ru_http",
		bp.NonRUHTTP:  "balancer nonru_http",
	}
	var conflicts []string

	add := func(seen map[int]string, balancer map[int]string, port int, tag, proto string) {
		if owner, ok := balancer[port]; ok {
			conflicts = append(conflicts, fmt.Sprintf(
				"%s port %d (%s) collides with %s", proto, port, tag, owner))
			return
		}
		if prev, ok := seen[port]; ok {
			conflicts = append(conflicts, fmt.Sprintf(
				"%s port %d used by %q and %q", proto, port, prev, tag))
			return
		}
		seen[port] = tag
	}

	for _, a := range assignments {
		add(seenSOCKS, balancerPorts, a.SOCKS, a.Tag, "socks")
		add(seenHTTP, balancerPorts, a.HTTP, a.Tag, "http")
	}
	sort.Strings(conflicts)
	return conflicts
}
