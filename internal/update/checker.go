// Package update (checker.go) queries the GitHub repository for newer
// releases and notifies the application when an update is available.
//
// The check is performed on startup and then periodically. A manual check
// can be triggered from the tray menu. Version comparison is semantic
// (major.minor.patch); a leading "v" prefix on the release tag is
// stripped before comparison.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"subrelay/internal/config"
	"subrelay/internal/logging"
)

// GitHubRepo is the GitHub repository used for update checks.
const GitHubRepo = "overklassniy/subrelay"

// ErrNoReleases is returned by LatestRelease when the GitHub repository
// has no published releases (the API responds with HTTP 404). Callers
// should treat this as a non-error condition and inform the user that no
// releases have been published yet.
var ErrNoReleases = errors.New("no releases published")

// githubAPIBase is the GitHub REST API root. Stored as a constant so it
// can be overridden in tests if needed.
const githubAPIBase = "https://api.github.com"

// CurrentVersion is the running application version. It defaults to
// config.DefaultAppVersion and can be overridden at build time via
//
//	go build -ldflags "-X subrelay/internal/update.CurrentVersion=1.2.3"
//
// so release builds report the published version rather than the default.
var CurrentVersion = config.DefaultAppVersion

// DefaultCheckInterval is the default period between automatic update
// checks.
const DefaultCheckInterval = 24 * time.Hour

// Release describes the subset of a GitHub release used by the update
// check.
type Release struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Prerelease bool   `json:"prerelease"`
}

// LatestRelease fetches the latest published release from GitHub.
//
// Args:
//   - ctx: the request context, used for cancellation and timeout.
//
// Returns:
//   - The latest release.
//
// Errors:
//   - Returns an error wrapping the request construction failure.
//   - Returns an error wrapping the HTTP transport failure.
//   - Returns an error when the GitHub API responds with a non-200 status
//     (e.g. 404 when no releases have been published yet).
//   - Returns an error wrapping the response body read or parse failure.
func LatestRelease(ctx context.Context) (Release, error) {
	var r Release
	url := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, GitHubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return r, fmt.Errorf("update: build request: %w", err)
	}
	// The GitHub JSON media type is the documented default response
	// format; set it explicitly for forward compatibility.
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "subrelay/"+CurrentVersion)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return r, fmt.Errorf("update: fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return r, ErrNoReleases
	}
	if resp.StatusCode != http.StatusOK {
		return r, fmt.Errorf("update: github api status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return r, fmt.Errorf("update: read body: %w", err)
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return r, fmt.Errorf("update: parse body: %w", err)
	}
	return r, nil
}

// IsNewer reports whether the latest release version is strictly newer
// than the current version. A leading "v" prefix on either value is
// stripped, and only the major.minor.patch triple is compared; any
// pre-release or build metadata suffix is ignored.
//
// Args:
//   - current: the running application version (e.g. "1.0.0").
//   - latest: the latest release tag (e.g. "v1.2.0").
//
// Returns:
//   - true when latest is strictly newer than current.
func IsNewer(current, latest string) bool {
	c := parseSemver(current)
	l := parseSemver(latest)
	if l[0] != c[0] {
		return l[0] > c[0]
	}
	if l[1] != c[1] {
		return l[1] > c[1]
	}
	return l[2] > c[2]
}

// parseSemver extracts the major.minor.patch triple from a version
// string. A leading "v" is stripped and any pre-release or build
// metadata suffix (introduced by "-" or "+") is discarded. Components
// that fail to parse are treated as 0.
//
// Returns:
//   - A three-element array with major, minor, and patch numbers.
func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var parts [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		n, _ := strconv.Atoi(s)
		parts[i] = n
	}
	return parts
}

// Checker periodically queries GitHub for a newer release and invokes
// callbacks when a new version is available, when the running version is
// up to date, when no releases have been published, or when a check
// fails. The first check runs immediately on Start; subsequent checks
// run at the configured interval.
type Checker struct {
	current      string
	interval     time.Duration
	log          *logging.Logger
	onAvailable  func(Release)
	onUpToDate   func()
	onNoReleases func()
	onError      func(error)

	cancel context.CancelFunc
	done   chan struct{}
}

// NewChecker creates a Checker bound to the given dependencies.
//
// Args:
//   - current: the running application version.
//   - interval: the period between automatic checks. When non-positive,
//     DefaultCheckInterval is used.
//   - log: the shared logger.
//
// Returns:
//   - A pointer to the new Checker.
func NewChecker(current string, interval time.Duration, log *logging.Logger) *Checker {
	if interval <= 0 {
		interval = DefaultCheckInterval
	}
	return &Checker{
		current:  current,
		interval: interval,
		log:      log,
	}
}

// SetOnAvailable sets the callback invoked when a newer release is found.
// The callback receives the latest release. It is called from the checker
// goroutine, so any UI work must be dispatched to the main thread by the
// callback itself.
//
// Args:
//   - fn: the callback to invoke when a newer release is available.
func (c *Checker) SetOnAvailable(fn func(Release)) { c.onAvailable = fn }

// SetOnUpToDate sets the callback invoked when the running version
// matches or exceeds the latest release. It is intended for manual checks
// where the user expects feedback regardless of the outcome.
//
// Args:
//   - fn: the callback to invoke when the application is up to date.
func (c *Checker) SetOnUpToDate(fn func()) { c.onUpToDate = fn }

// SetOnNoReleases sets the callback invoked when the GitHub repository
// has no published releases (HTTP 404). It is intended for manual checks
// where the user expects feedback regardless of the outcome.
//
// Args:
//   - fn: the callback to invoke when no releases have been published.
func (c *Checker) SetOnNoReleases(fn func()) { c.onNoReleases = fn }

// SetOnError sets the callback invoked when a check fails (network error,
// non-200 response, or parse failure). It is called from the checker
// goroutine.
//
// Args:
//   - fn: the callback to invoke on check failure.
func (c *Checker) SetOnError(fn func(error)) { c.onError = fn }

// Start launches the background check goroutine. The first check runs
// immediately; subsequent checks run at the configured interval.
//
// Args:
//   - ctx: the parent context. When it is cancelled the goroutine exits.
func (c *Checker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.done = make(chan struct{})
	go c.loop(ctx)
}

// Stop cancels the check goroutine and waits for it to exit.
func (c *Checker) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.done != nil {
		<-c.done
	}
}

// CheckNow performs a single synchronous check and invokes the matching
// callback. It is safe to call from any goroutine, including the UI
// thread; callers that need UI feedback should dispatch the callback work
// to the main thread themselves.
//
// Args:
//   - ctx: the request context.
func (c *Checker) CheckNow(ctx context.Context) {
	c.checkOnce(ctx)
}

// loop runs the periodic check until the context is cancelled.
func (c *Checker) loop(ctx context.Context) {
	defer close(c.done)

	c.checkOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkOnce(ctx)
		}
	}
}

// checkOnce performs a single GitHub release query and dispatches the
// appropriate callback. Errors are logged and forwarded to onError,
// except for ErrNoReleases which is forwarded to onNoReleases.
func (c *Checker) checkOnce(ctx context.Context) {
	rel, err := LatestRelease(ctx)
	if err != nil {
		if errors.Is(err, ErrNoReleases) {
			c.log.Info("update check: no releases published yet")
			if c.onNoReleases != nil {
				c.onNoReleases()
			}
			return
		}
		c.log.Warn("update check: %v", err)
		if c.onError != nil {
			c.onError(err)
		}
		return
	}
	if !IsNewer(c.current, rel.TagName) {
		c.log.Info("update check: running latest version %s", c.current)
		if c.onUpToDate != nil {
			c.onUpToDate()
		}
		return
	}
	c.log.Info("update check: newer version available %s (current %s)", rel.TagName, c.current)
	if c.onAvailable != nil {
		c.onAvailable(rel)
	}
}
