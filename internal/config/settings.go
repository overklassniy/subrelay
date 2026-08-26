// Package config defines the user-facing application settings, their
// persistence to settings.json, and the default values applied on first
// launch.
//
// Settings are created and edited only through the UI; no manual config
// files are expected. The HWID (hardware identifier sent in subscription
// request headers) is generated once and persisted.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"subrelay/internal/paths"
)

// DefaultUpdateIntervalMin is the default subscription refresh interval
// in minutes.
const DefaultUpdateIntervalMin = 30

// DefaultSOCKSPortStart is the first port used for per-node SOCKS5
// inbounds.
const DefaultSOCKSPortStart = 17253

// DefaultHTTPPortStart is the first port used for per-node HTTP
// inbounds.
const DefaultHTTPPortStart = 52116

// DefaultRU SOCKS port for the RU balancer inbound.
const DefaultRUSocksPort = 17053

// DefaultRUHTTPPort for the RU balancer HTTP inbound.
const DefaultRUHTTPPort = 52016

// DefaultNonRUSocksPort for the non-RU balancer inbound.
const DefaultNonRUSocksPort = 17054

// DefaultNonRUHTTPPort for the non-RU balancer HTTP inbound.
const DefaultNonRUHTTPPort = 52017

// DefaultURLTestIntervalSec is the default urltest probe interval.
const DefaultURLTestIntervalSec = 180

// DefaultURLTestToleranceMs is the default urltest latency tolerance.
const DefaultURLTestToleranceMs = 50

// DefaultURLTestURL is the default urltest probe target.
// Uses Ubuntu connectivity check instead of gstatic.com because Google
// infrastructure is periodically blocked or throttled by Roskomnadzor,
// making urltest unreliable for Russian proxy nodes. The Ubuntu endpoint
// is hosted on Canonical infrastructure, which is not targeted by RKN.
// HTTP is used instead of HTTPS to avoid TLS handshake overhead in
// latency measurements and to prevent SNI-based DPI blocking.
const DefaultURLTestURL = "http://connectivity-check.ubuntu.com/generate_204"

// DefaultUserAgent is the default User-Agent header for subscription
// requests.
const DefaultUserAgent = "subrelay/1.0"

// DefaultDeviceOS is the default X-Device-OS header value.
const DefaultDeviceOS = "Linux"

// DefaultVerOS is the default X-Ver-OS header value.
const DefaultVerOS = "1.0"

// DefaultDeviceModel is the default X-Device-Model header value.
const DefaultDeviceModel = "Desktop"

// DefaultAppVersion is the default X-App-Version header value.
const DefaultAppVersion = "1.0.0"

// hwidLen is the length in bytes of a generated HWID; the hex encoding
// is twice this length.
const hwidLen = 16

// BalancerPorts holds the four inbound ports used by the RU and non-RU
// balancer groups.
type BalancerPorts struct {
	RUSocks    int `json:"ru_socks"`
	RUHTTP     int `json:"ru_http"`
	NonRUSocks int `json:"nonru_socks"`
	NonRUHTTP  int `json:"nonru_http"`
}

// URLTestSettings holds the urltest group parameters.
type URLTestSettings struct {
	IntervalSec int    `json:"interval_sec"`
	ToleranceMs int    `json:"tolerance_ms"`
	URL         string `json:"url"`
}

// HeadersSettings holds the Happ subscription request headers.
type HeadersSettings struct {
	UserAgent   string `json:"user_agent"`
	XHWID       string `json:"x_hwid"`
	XDeviceOS   string `json:"x_device_os"`
	XVerOS      string `json:"x_ver_os"`
	XDeviceModel string `json:"x_device_model"`
	XAppVersion string `json:"x_app_version"`
}

// Settings is the full application configuration. Fields are populated
// once during startup (from settings.json or defaults) and mutated only
// through the UI.
type Settings struct {
	Language         string            `json:"language"`
	SubscriptionURL  string            `json:"subscription_url"`
	UpdateIntervalMin int              `json:"update_interval_min"`
	Autostart        bool              `json:"autostart"`
	SOCKSPortStart   int               `json:"socks_port_start"`
	HTTPPortStart    int               `json:"http_port_start"`
	BalancerPorts    BalancerPorts     `json:"balancer_ports"`
	URLTest          URLTestSettings   `json:"urltest"`
	Headers          HeadersSettings   `json:"headers"`
	RUOverrides      map[string]bool   `json:"ru_overrides"`
	PortAssignments   map[string]string `json:"port_assignments"`

	mu sync.Mutex
}

// Lock acquires the settings mutex.
func (s *Settings) Lock() { s.mu.Lock() }

// Unlock releases the settings mutex.
func (s *Settings) Unlock() { s.mu.Unlock() }

// Snapshot returns a copy of the settings values without the internal
// mutex, safe to read by callers (e.g. the settings UI) without holding
// a lock afterward. The caller must still hold the lock while calling
// this method.
func (s *Settings) Snapshot() Settings {
	return Settings{
		Language:          s.Language,
		SubscriptionURL:   s.SubscriptionURL,
		UpdateIntervalMin: s.UpdateIntervalMin,
		Autostart:         s.Autostart,
		SOCKSPortStart:    s.SOCKSPortStart,
		HTTPPortStart:     s.HTTPPortStart,
		BalancerPorts:     s.BalancerPorts,
		URLTest:           s.URLTest,
		Headers:           s.Headers,
		RUOverrides:       s.RUOverrides,
		PortAssignments:   s.PortAssignments,
	}
}

// Defaults returns a Settings populated with default values and a newly
// generated HWID. PortAssignments and RUOverrides are initialized to
// empty maps.
//
// Returns:
//   - A pointer to a new Settings with default values.
func Defaults() *Settings {
	return &Settings{
		Language:          "ru",
		SubscriptionURL:   "",
		UpdateIntervalMin: DefaultUpdateIntervalMin,
		Autostart:         false,
		SOCKSPortStart:    DefaultSOCKSPortStart,
		HTTPPortStart:     DefaultHTTPPortStart,
		BalancerPorts: BalancerPorts{
			RUSocks:    DefaultRUSocksPort,
			RUHTTP:     DefaultRUHTTPPort,
			NonRUSocks: DefaultNonRUSocksPort,
			NonRUHTTP:  DefaultNonRUHTTPPort,
		},
		URLTest: URLTestSettings{
			IntervalSec: DefaultURLTestIntervalSec,
			ToleranceMs: DefaultURLTestToleranceMs,
			URL:         DefaultURLTestURL,
		},
		Headers: HeadersSettings{
			UserAgent:    DefaultUserAgent,
			XHWID:        GenerateHWID(),
			XDeviceOS:    DefaultDeviceOS,
			XVerOS:       DefaultVerOS,
			XDeviceModel: DefaultDeviceModel,
			XAppVersion:  DefaultAppVersion,
		},
		RUOverrides:     make(map[string]bool),
		PortAssignments: make(map[string]string),
	}
}

// GenerateHWID returns a freshly generated 32-character lowercase hex
// HWID. The value is random and intended to be persisted on first launch.
//
// Returns:
//   - A 32-character hex string.
//
// Errors:
//   - Panics only when the system CSPRNG is unavailable, which is a
//     fatal startup condition.
func GenerateHWID() string {
	b := make([]byte, hwidLen)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("config: crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b)
}

// Load reads settings.json from the config directory. When the file does
// not exist, Defaults is returned. When the file exists but is missing
// fields, the loaded values are merged over Defaults so new fields keep
// their defaults.
//
// Returns:
//   - A pointer to the loaded Settings.
//
// Errors:
//   - Returns an error wrapping os.ReadFile when the file cannot be read
//     for reasons other than not existing.
//   - Returns an error wrapping json.Unmarshal when the file is invalid.
func Load() (*Settings, error) {
	path, err := paths.SettingsFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return nil, fmt.Errorf("config: read settings: %w", err)
	}

	s := Defaults()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("config: parse settings: %w", err)
	}

	if s.RUOverrides == nil {
		s.RUOverrides = make(map[string]bool)
	}
	if s.PortAssignments == nil {
		s.PortAssignments = make(map[string]string)
	}
	if s.Headers.XHWID == "" {
		s.Headers.XHWID = GenerateHWID()
	}
	return s, nil
}

// Save writes the settings to settings.json in the config directory. The
// RUOverrides and PortAssignments maps are serialized in sorted key order
// for stable diffs.
//
// Errors:
//   - Returns an error wrapping json.Marshal or os.WriteFile on failure.
func (s *Settings) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := paths.SettingsFile()
	if err != nil {
		return err
	}

	overrides := sortedMap(s.RUOverrides)
	assignments := sortedMap(s.PortAssignments)

	tmp := struct {
		Language          string            `json:"language"`
		SubscriptionURL   string            `json:"subscription_url"`
		UpdateIntervalMin int               `json:"update_interval_min"`
		Autostart         bool              `json:"autostart"`
		SOCKSPortStart    int               `json:"socks_port_start"`
		HTTPPortStart     int               `json:"http_port_start"`
		BalancerPorts     BalancerPorts     `json:"balancer_ports"`
		URLTest           URLTestSettings   `json:"urltest"`
		Headers           HeadersSettings   `json:"headers"`
		RUOverrides       map[string]bool   `json:"ru_overrides"`
		PortAssignments   map[string]string `json:"port_assignments"`
	}{
		Language:          s.Language,
		SubscriptionURL:   s.SubscriptionURL,
		UpdateIntervalMin: s.UpdateIntervalMin,
		Autostart:         s.Autostart,
		SOCKSPortStart:    s.SOCKSPortStart,
		HTTPPortStart:     s.HTTPPortStart,
		BalancerPorts:     s.BalancerPorts,
		URLTest:           s.URLTest,
		Headers:           s.Headers,
		RUOverrides:       overrides,
		PortAssignments:   assignments,
	}

	data, err := json.MarshalIndent(tmp, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: write settings: %w", err)
	}
	return nil
}

// sortedMap returns a copy of m with keys serialized deterministically.
// json.Marshal sorts map keys alphabetically already, so this is a thin
// pass-through that ensures a non-nil map.
func sortedMap[K string, V any](m map[K]V) map[K]V {
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// SortedOverrideTags returns the RU override tags in sorted order. This
// is a convenience for UI rendering and tests.
func (s *Settings) SortedOverrideTags() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	tags := make([]string, 0, len(s.RUOverrides))
	for tag := range s.RUOverrides {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}
