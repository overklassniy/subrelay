// Package sub parses VLESS subscription links into node descriptors
// consumed by the config builder. The parser is a direct port of the
// Python reference implementation in old/proxy-stack/relay/parsers.py.
//
// Supported transports: tcp (none), grpc, ws, xhttp (with xmux/extra).
// Supported TLS modes: reality (with uTLS fingerprint), tls, none.
package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// Node is a parsed outbound descriptor independent of sing-box option
// types. The builder converts Node values into sing-box option structs.
type Node struct {
	Tag        string
	Server     string
	ServerPort int
	UUID       string
	Flow       string
	Transport  *Transport
	TLS        *TLS
}

// Transport describes the outbound network transport.
type Transport struct {
	Type          string // grpc, ws, xhttp
	ServiceName   string // grpc
	Path          string // ws, xhttp
	Host          string // ws, xhttp
	Mode          string // xhttp
	XPaddingBytes *PaddingRange
	Xmux          map[string]any
	Headers       map[string]string
}

// PaddingRange is the x_padding_bytes range used by xhttp.
type PaddingRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// TLS describes the outbound TLS configuration.
type TLS struct {
	Enabled     bool
	ServerName  string
	UTLS        *UTLS
	Reality     *Reality
}

// UTLS holds the uTLS fingerprint settings.
type UTLS struct {
	Enabled     bool
	Fingerprint string
}

// Reality holds the reality public key and short id.
type Reality struct {
	Enabled   bool
	PublicKey string
	ShortID   string
}

// TransportType returns the transport type string, or "tcp" when no
// transport is configured.
func (n *Node) TransportType() string {
	if n.Transport == nil {
		return "tcp"
	}
	return n.Transport.Type
}

// IsAuto reports whether the node tag marks it as an auto-selector node
// that should be excluded from per-node inbounds and balancer groups.
// The check is case-insensitive and matches "auto" or "avto" (the
// Cyrillic transliteration) anywhere in the tag.
func (n *Node) IsAuto() bool {
	return IsAutoTag(n.Tag)
}

// IsAutoTag reports whether tag marks an auto-selector node. The check
// is case-insensitive and matches the Latin "auto" or the Cyrillic
// "авто" anywhere in the tag (the subscription uses both spellings).
func IsAutoTag(tag string) bool {
	lower := strings.ToLower(tag)
	return strings.Contains(lower, "auto") ||
		strings.Contains(lower, "авто")
}

// RUFlag is the Russian flag emoji used as the default RU classifier.
const RUFlag = "\U0001F1F7\U0001F1FA"

// IsRU reports whether the node is classified as Russian by default
// (tag contains the Russian flag emoji). Caller-supplied overrides are
// not applied here; see the state package for override handling.
func (n *Node) IsRU() bool {
	return strings.Contains(n.Tag, RUFlag)
}

// MaybeBase64Decode decodes a base64 subscription body when the decoded
// content looks like a link list (contains "vless://", "vmess://", or
// "trojan://"). Otherwise the original text is returned unchanged.
//
// Args:
//   - text: the raw subscription response body.
//
// Returns:
//   - The decoded text when base64 was detected, otherwise the original.
func MaybeBase64Decode(text string) string {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return raw
	}
	padded := raw + strings.Repeat("=", (-len(raw))%4)
	decoded, err := base64.StdEncoding.DecodeString(padded)
	if err != nil {
		// Try URL-safe encoding as a fallback.
		decoded, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return raw
		}
	}
	s := string(decoded)
	if strings.Contains(s, "vless://") ||
		strings.Contains(s, "vmess://") ||
		strings.Contains(s, "trojan://") {
		return s
	}
	return raw
}

// ParseSubscriptionText decodes (if needed) and parses every vless://
// line in the subscription body. Lines that are not vless or that fail
// to parse are silently skipped, matching the reference implementation.
//
// Args:
//   - text: the raw subscription response body.
//
// Returns:
//   - A slice of parsed Node values.
func ParseSubscriptionText(text string) []Node {
	decoded := MaybeBase64Decode(text)
	nodes := make([]Node, 0)
	for _, line := range strings.Split(decoded, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "vless://") {
			continue
		}
		node, err := ParseVLESS(line, len(nodes)+1)
		if err != nil {
			continue
		}
		nodes = append(nodes, *node)
	}
	return nodes
}

// ParseVLESS parses a single vless:// URL into a Node.
//
// Args:
//   - raw: the vless:// URL.
//   - index: 1-based index used to synthesize a tag when the fragment is
//     empty.
//
// Returns:
//   - A pointer to the parsed Node.
//
// Errors:
//   - Returns an error when the URL is not vless, has no UUID, or cannot
//     be parsed by net/url.
func ParseVLESS(raw string, index int) (*Node, error) {
	p, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("sub: parse url: %w", err)
	}
	if p.Scheme != "vless" {
		return nil, fmt.Errorf("sub: not vless scheme %q", p.Scheme)
	}

	uuid := p.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("sub: missing uuid")
	}
	server := p.Hostname()
	portStr := p.Port()
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		port = 443
	}

	q := p.Query()
	tag := p.Fragment
	if tag == "" {
		tag = fmt.Sprintf("node-%d", index)
	}
	// url.Fragment is already percent-decoded by net/url.

	node := &Node{
		Tag:        tag,
		Server:     server,
		ServerPort: port,
		UUID:       uuid,
	}

	if flow := q.Get("flow"); flow != "" {
		node.Flow = flow
	}

	security := q.Get("security")
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}

	if network != "tcp" {
		transport, err := buildTransport(network, q)
		if err != nil {
			return nil, err
		}
		node.Transport = transport
	}

	switch security {
	case "reality":
		node.TLS = buildRealityTLS(q, server)
	case "tls":
		node.TLS = &TLS{
			Enabled:    true,
			ServerName: q.Get("sni"),
		}
		if node.TLS.ServerName == "" {
			node.TLS.ServerName = server
		}
	}

	return node, nil
}

// buildTransport constructs the Transport descriptor for the given
// network type from the URL query parameters.
//
// Args:
//   - network: the transport type (grpc, ws, xhttp).
//   - q: the parsed URL query.
//
// Returns:
//   - A pointer to the Transport descriptor.
//
// Errors:
//   - Returns an error for an unsupported network type.
func buildTransport(network string, q url.Values) (*Transport, error) {
	t := &Transport{Type: network}
	switch network {
	case "grpc":
		if sn := q.Get("serviceName"); sn != "" {
			t.ServiceName = sn
		}
	case "ws":
		if path := q.Get("path"); path != "" {
			t.Path = path
		}
		if host := q.Get("host"); host != "" {
			t.Host = host
		}
	case "xhttp":
		if path := q.Get("path"); path != "" {
			t.Path = path
		}
		if host := q.Get("host"); host != "" {
			t.Host = host
		}
		if mode := q.Get("mode"); mode != "" {
			t.Mode = mode
		}
		t.XPaddingBytes = &PaddingRange{From: 64, To: 256}
		if extraRaw := q.Get("extra"); extraRaw != "" {
			applyXHTTPExtra(t, extraRaw)
		}
	default:
		return nil, fmt.Errorf("sub: unsupported transport %q", network)
	}
	return t, nil
}

// applyXHTTPExtra parses the xhttp "extra" JSON query parameter and
// applies its xmux and headers fields to the transport descriptor.
// Errors are swallowed so a malformed extra does not drop the node.
func applyXHTTPExtra(t *Transport, extraRaw string) {
	var extra map[string]json.RawMessage
	if err := json.Unmarshal([]byte(extraRaw), &extra); err != nil {
		return
	}
	if xmuxRaw, ok := extra["xmux"]; ok {
		var xmux map[string]any
		if err := json.Unmarshal(xmuxRaw, &xmux); err == nil && len(xmux) > 0 {
			t.Xmux = xmux
		}
	}
	if headersRaw, ok := extra["headers"]; ok {
		var headers map[string]string
		if err := json.Unmarshal(headersRaw, &headers); err == nil && len(headers) > 0 {
			t.Headers = headers
		}
	}
}

// buildRealityTLS constructs the TLS descriptor for a reality outbound.
func buildRealityTLS(q url.Values, server string) *TLS {
	serverName := q.Get("sni")
	if serverName == "" {
		serverName = server
	}
	fingerprint := q.Get("fp")
	if fingerprint == "" {
		fingerprint = "chrome"
	}
	return &TLS{
		Enabled:    true,
		ServerName: serverName,
		UTLS: &UTLS{
			Enabled:     true,
			Fingerprint: fingerprint,
		},
		Reality: &Reality{
			Enabled:   true,
			PublicKey: q.Get("pbk"),
			ShortID:   q.Get("sid"),
		},
	}
}

// SafeTag converts a node tag into a safe identifier suitable for use in
// inbound names. Spaces become underscores, the micro sign is replaced
// with "u", pipes are removed, and only Unicode letters, digits, "_",
// and "-" are retained. This mirrors the Python reference
// str.isalnum() behavior so Cyrillic and other script letters are kept.
func SafeTag(tag string) string {
	t := strings.ReplaceAll(tag, " ", "_")
	t = strings.ReplaceAll(t, "|", "")
	t = strings.ReplaceAll(t, "\u00b5", "u")
	var b strings.Builder
	b.Grow(len(t))
	for _, c := range t {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}
