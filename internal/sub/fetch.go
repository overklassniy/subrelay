// Package sub (fetch.go) fetches the subscription body over HTTP with
// the Happ client headers (User-Agent, X-HWID, X-Device-OS, etc.).
package sub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"subrelay/internal/config"
)

// fetchTimeout is the maximum time allowed for a subscription request.
const fetchTimeout = 60 * time.Second

// maxBodyBytes limits the subscription response size to avoid unbounded
// memory use on a malformed response.
const maxBodyBytes = 8 << 20 // 8 MiB

// Fetcher downloads and parses the subscription.
type Fetcher struct {
	settings *config.Settings
	client   *http.Client
}

// NewFetcher creates a Fetcher bound to the given settings. The settings
// pointer is read under its mutex on every fetch so header changes take
// effect immediately.
//
// Args:
//   - settings: the application settings providing the URL and headers.
//
// Returns:
//   - A pointer to the new Fetcher.
func NewFetcher(settings *config.Settings) *Fetcher {
	return &Fetcher{
		settings: settings,
		client: &http.Client{
			Timeout: fetchTimeout,
		},
	}
}

// Fetch downloads the subscription, validates the response, and returns
// the parsed nodes.
//
// Args:
//   - ctx: context for cancellation.
//
// Returns:
//   - A slice of parsed Node values.
//
// Errors:
//   - Returns an error when the subscription URL is empty.
//   - Returns an error wrapping http request/response failures.
//   - Returns an error when the response status is not 2xx.
//   - Returns an error when the body exceeds maxBodyBytes.
func (f *Fetcher) Fetch(ctx context.Context) ([]Node, error) {
	f.settings.Lock()
	subURL := f.settings.SubscriptionURL
	headers := f.settings.Headers
	f.settings.Unlock()

	if subURL == "" {
		return nil, fmt.Errorf("sub: subscription URL is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("sub: build request: %w", err)
	}
	applyHappHeaders(req, headers)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sub: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sub: subscription returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("sub: read body: %w", err)
	}

	nodes := ParseSubscriptionText(string(body))
	if len(nodes) == 0 {
		return nil, fmt.Errorf("sub: subscription produced no vless nodes")
	}
	return nodes, nil
}

// applyHappHeaders sets the Happ client headers on the request from the
// settings.
func applyHappHeaders(req *http.Request, h config.HeadersSettings) {
	req.Header.Set("User-Agent", h.UserAgent)
	req.Header.Set("X-HWID", h.XHWID)
	req.Header.Set("X-Device-OS", h.XDeviceOS)
	req.Header.Set("X-Ver-OS", h.XVerOS)
	req.Header.Set("X-Device-Model", h.XDeviceModel)
	req.Header.Set("X-App-Version", h.XAppVersion)
}
