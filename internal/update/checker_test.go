// Package update (checker_test.go) verifies the version comparison and
// semver parsing helpers used by the GitHub release checker.
package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"patch newer", "1.0.0", "v1.0.1", true},
		{"minor newer", "1.0.0", "v1.1.0", true},
		{"major newer", "1.9.9", "v2.0.0", true},
		{"equal", "1.0.0", "v1.0.0", false},
		{"older release", "1.2.0", "v1.1.0", false},
		{"no v prefix", "1.0.0", "1.0.5", true},
		{"prerelease ignored", "1.0.0", "v1.1.0-rc1", true},
		{"build metadata ignored", "1.0.0", "v1.0.0+build.42", false},
		{"empty latest", "1.0.0", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsNewer(c.current, c.latest); got != c.want {
				t.Fatalf("IsNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
			}
		})
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in   string
		want [3]int
	}{
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3", [3]int{1, 2, 3}},
		{"v2.0.0-rc1", [3]int{2, 0, 0}},
		{"1.0.0+build.42", [3]int{1, 0, 0}},
		{"  v3.1.4  ", [3]int{3, 1, 4}},
		{"1.2", [3]int{1, 2, 0}},
		{"", [3]int{0, 0, 0}},
		{"vX.Y.Z", [3]int{0, 0, 0}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := parseSemver(c.in); got != c.want {
				t.Fatalf("parseSemver(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
