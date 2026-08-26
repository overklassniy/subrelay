// Package ports (planner_test.go) tests stable port assignment and
// conflict detection.
package ports

import (
	"subrelay/internal/config"
	"subrelay/internal/sub"
	"testing"
)

func testSettings() *config.Settings {
	s := config.Defaults()
	// Use a fixed HWID to avoid randomness in defaults.
	s.Headers.XHWID = "test"
	return s
}

func TestPlanAssignsSequentialPorts(t *testing.T) {
	s := testSettings()
	p := NewPlanner(s)
	nodes := []sub.Node{
		{Tag: "node-a"},
		{Tag: "node-b"},
		{Tag: "node-c"},
	}
	assignments, err := p.Plan(nodes)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(assignments))
	}
	// First node gets the range start (balancer ports are reserved but
	// they are 17053/17054 which are below socks start 17253).
	if assignments[0].SOCKS != 17253 || assignments[0].HTTP != 52116 {
		t.Errorf("first assignment = %+v", assignments[0])
	}
	if assignments[1].SOCKS != 17254 || assignments[1].HTTP != 52117 {
		t.Errorf("second assignment = %+v", assignments[1])
	}
	if assignments[2].SOCKS != 17255 || assignments[2].HTTP != 52118 {
		t.Errorf("third assignment = %+v", assignments[2])
	}
}

func TestPlanExcludesAutoNodes(t *testing.T) {
	s := testSettings()
	p := NewPlanner(s)
	nodes := []sub.Node{
		{Tag: "node-a"},
		{Tag: "Auto selector"},
		{Tag: "node-b"},
	}
	assignments, err := p.Plan(nodes)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments (auto excluded), got %d", len(assignments))
	}
	for _, a := range assignments {
		if a.Tag == "Auto selector" {
			t.Errorf("auto node was not excluded")
		}
	}
}

func TestPlanReusesPersistedAssignments(t *testing.T) {
	s := testSettings()
	s.PortAssignments["node-a"] = "18000:53000"
	p := NewPlanner(s)
	nodes := []sub.Node{
		{Tag: "node-a"},
		{Tag: "node-b"},
	}
	assignments, err := p.Plan(nodes)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if assignments[0].Tag != "node-a" ||
		assignments[0].SOCKS != 18000 || assignments[0].HTTP != 53000 {
		t.Errorf("persisted assignment not reused: %+v", assignments[0])
	}
	// node-b should get the default range start since 18000/53000 are
	// reserved but the cursor starts at the range start.
	if assignments[1].SOCKS != 17253 || assignments[1].HTTP != 52116 {
		t.Errorf("new node assignment = %+v", assignments[1])
	}
	// The new assignment must be persisted.
	if s.PortAssignments["node-b"] != "17253:52116" {
		t.Errorf("node-b not persisted: %q", s.PortAssignments["node-b"])
	}
}

func TestPlanAvoidsBalancerPorts(t *testing.T) {
	s := testSettings()
	// Move the socks range start to overlap with a balancer port.
	s.SOCKSPortStart = 17053
	p := NewPlanner(s)
	nodes := []sub.Node{{Tag: "node-a"}}
	assignments, err := p.Plan(nodes)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	// 17053 is the RU socks balancer port; node-a must skip it.
	if assignments[0].SOCKS == 17053 {
		t.Errorf("node-a socks collided with balancer port 17053")
	}
}

func TestPlanStableAcrossUpdates(t *testing.T) {
	s := testSettings()
	p := NewPlanner(s)
	first := []sub.Node{{Tag: "node-a"}, {Tag: "node-b"}}
	a1, err := p.Plan(first)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	// Second update adds a node in the middle; existing ports must stay.
	second := []sub.Node{{Tag: "node-a"}, {Tag: "node-c"}, {Tag: "node-b"}}
	a2, err := p.Plan(second)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	byTag := map[string]Assignment{}
	for _, a := range a2 {
		byTag[a.Tag] = a
	}
	if byTag["node-a"].SOCKS != a1[0].SOCKS || byTag["node-a"].HTTP != a1[0].HTTP {
		t.Errorf("node-a port drifted: was %+v, now %+v", a1[0], byTag["node-a"])
	}
	if byTag["node-b"].SOCKS != a1[1].SOCKS || byTag["node-b"].HTTP != a1[1].HTTP {
		t.Errorf("node-b port drifted: was %+v, now %+v", a1[1], byTag["node-b"])
	}
}

func TestDetectConflictsBalancerCollision(t *testing.T) {
	s := testSettings()
	assignments := []Assignment{
		{Tag: "node-a", SOCKS: s.BalancerPorts.RUSocks, HTTP: 52116},
	}
	conflicts := DetectConflicts(assignments, s.BalancerPorts)
	if len(conflicts) == 0 {
		t.Errorf("expected a balancer collision conflict")
	}
}

func TestDetectConflictsDuplicatePort(t *testing.T) {
	s := testSettings()
	assignments := []Assignment{
		{Tag: "node-a", SOCKS: 20000, HTTP: 20000},
		{Tag: "node-b", SOCKS: 20000, HTTP: 20001},
	}
	conflicts := DetectConflicts(assignments, s.BalancerPorts)
	if len(conflicts) == 0 {
		t.Errorf("expected a duplicate port conflict")
	}
}

func TestDetectConflictsNone(t *testing.T) {
	s := testSettings()
	assignments := []Assignment{
		{Tag: "node-a", SOCKS: 17253, HTTP: 52116},
		{Tag: "node-b", SOCKS: 17254, HTTP: 52117},
	}
	conflicts := DetectConflicts(assignments, s.BalancerPorts)
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
}
