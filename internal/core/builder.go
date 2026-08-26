// Package core (builder.go) generates sing-box option.Options from parsed
// subscription nodes, port assignments, and user settings. The builder
// produces in-memory structs (no JSON files on disk); a JSON dump is
// available for debugging via the log window.
package core

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"

	"subrelay/internal/config"
	"subrelay/internal/ports"
	"subrelay/internal/sub"
)

// localAddr is the 127.0.0.1 listen address shared by all inbounds.
var localAddr = badoption.Addr(netip.MustParseAddr("127.0.0.1"))

// BuildInput bundles everything the builder needs to produce options.
type BuildInput struct {
	Settings    *config.Settings
	Nodes       []sub.Node
	Assignments []ports.Assignment
}

// BuildResult holds the generated options plus the classified node lists
// so the caller (state, tray) can report them without recomputing.
type BuildResult struct {
	Options    option.Options
	RUNodes    []string
	NonRUNodes []string
}

// JSON returns the pretty-printed sing-box configuration, used by the
// log viewer's config dump action. It returns an empty string when the
// options cannot be marshaled, which should not happen in practice.
func (r *BuildResult) JSON() string {
	data, err := json.MarshalIndent(r.Options, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// Build generates option.Options for the sing-box instance.
//
// Args:
//   - input: the settings, parsed nodes, and port assignments.
//
// Returns:
//   - A BuildResult containing the options and the RU/non-RU node tag
//     lists.
//
// Errors:
//   - Returns an error when no non-auto nodes are available.
func Build(input BuildInput) (*BuildResult, error) {
	s := input.Settings
	s.Lock()
	defer s.Unlock()

	assignments := indexByTag(input.Assignments)

	var outbounds []option.Outbound
	var ruNodes, nonRUNodes []string

	for i := range input.Nodes {
		node := &input.Nodes[i]
		if node.IsAuto() {
			continue
		}
		ob, err := buildVLESSOutbound(node)
		if err != nil {
			// Skip a single malformed node without aborting the whole build.
			continue
		}
		outbounds = append(outbounds, *ob)

		if isRU(node, s.RUOverrides) {
			ruNodes = append(ruNodes, node.Tag)
		} else {
			nonRUNodes = append(nonRUNodes, node.Tag)
		}
	}

	if len(outbounds) == 0 {
		return nil, fmt.Errorf("core: no usable outbounds after filtering auto nodes")
	}

	// urltest balancer groups.
	if len(ruNodes) > 0 {
		outbounds = append(outbounds, option.Outbound{
			Type: C.TypeURLTest,
			Tag:  ruGroupTag,
			Options: &option.URLTestOutboundOptions{
				GroupCommonOption: option.GroupCommonOption{
					Outbounds: ruNodes,
				},
				URL:      s.URLTest.URL,
				Interval: badoption.Duration(time.Duration(s.URLTest.IntervalSec) * time.Second),
				Tolerance: uint16(s.URLTest.ToleranceMs),
			},
		})
	}
	if len(nonRUNodes) > 0 {
		outbounds = append(outbounds, option.Outbound{
			Type: C.TypeURLTest,
			Tag:  nonruGroupTag,
			Options: &option.URLTestOutboundOptions{
				GroupCommonOption: option.GroupCommonOption{
					Outbounds: nonRUNodes,
				},
				URL:      s.URLTest.URL,
				Interval: badoption.Duration(time.Duration(s.URLTest.IntervalSec) * time.Second),
				Tolerance: uint16(s.URLTest.ToleranceMs),
			},
		})
	}

	// direct fallback outbound.
	outbounds = append(outbounds, option.Outbound{
		Type:    C.TypeDirect,
		Tag:     directTag,
		Options: &option.DirectOutboundOptions{},
	})

	inbounds := buildInbounds(s, input.Nodes, assignments)
	rules := buildRules(input.Nodes, assignments, len(ruNodes) > 0, len(nonRUNodes) > 0)

	opts := option.Options{
		Log: &option.LogOptions{
			Disabled:  false,
			Level:     "info",
			Timestamp: true,
		},
		DNS: buildDNS(),
		Inbounds:  inbounds,
		Outbounds: outbounds,
		Route: &option.RouteOptions{
			Rules: rules,
			Final: directTag,
		},
	}

	return &BuildResult{
		Options:    opts,
		RUNodes:    ruNodes,
		NonRUNodes: nonRUNodes,
	}, nil
}

// buildInbounds creates the balancer inbounds and one SOCKS+HTTP pair per
// non-auto node.
func buildInbounds(s *config.Settings, nodes []sub.Node, assignments map[string]ports.Assignment) []option.Inbound {
	bp := s.BalancerPorts
	var inbounds []option.Inbound

	// RU balancer inbounds.
	inbounds = append(inbounds,
		socksInbound(ruSOCKSTag, bp.RUSocks),
		httpInbound(ruHTTPTag, bp.RUHTTP),
	)
	// non-RU balancer inbounds.
	inbounds = append(inbounds,
		socksInbound(nonruSOCKSTag, bp.NonRUSocks),
		httpInbound(nonruHTTPTag, bp.NonRUHTTP),
	)

	// Per-node inbounds.
	for i := range nodes {
		node := &nodes[i]
		if node.IsAuto() {
			continue
		}
		a, ok := assignments[node.Tag]
		if !ok {
			continue
		}
		st := sub.SafeTag(node.Tag)
		inbounds = append(inbounds,
			socksInbound("socks-"+st, a.SOCKS),
			httpInbound("http-"+st, a.HTTP),
		)
	}
	return inbounds
}

// buildRules creates one route rule per node (inbound pair -> outbound)
// and two balancer rules, with direct as the final fallback.
func buildRules(nodes []sub.Node, assignments map[string]ports.Assignment, hasRU, hasNonRU bool) []option.Rule {
	var rules []option.Rule

	for i := range nodes {
		node := &nodes[i]
		if node.IsAuto() {
			continue
		}
		if _, ok := assignments[node.Tag]; !ok {
			continue
		}
		st := sub.SafeTag(node.Tag)
		rules = append(rules, routeRule(
			[]string{"socks-" + st, "http-" + st},
			node.Tag,
		))
	}

	if hasRU {
		rules = append(rules, routeRule(
			[]string{ruSOCKSTag, ruHTTPTag},
			ruGroupTag,
		))
	}
	if hasNonRU {
		rules = append(rules, routeRule(
			[]string{nonruSOCKSTag, nonruHTTPTag},
			nonruGroupTag,
		))
	}
	return rules
}

// routeRule builds a default rule that routes the given inbounds to the
// given outbound via a route action.
func routeRule(inbounds []string, outbound string) option.Rule {
	return option.Rule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultRule{
			RawDefaultRule: option.RawDefaultRule{
				Inbound: inbounds,
			},
			RuleAction: option.RuleAction{
				Action: C.RuleActionTypeRoute,
				RouteOptions: option.RouteActionOptions{
					Outbound: outbound,
				},
			},
		},
	}
}

// socksInbound creates a no-auth SOCKS5 inbound bound to 127.0.0.1.
func socksInbound(tag string, port int) option.Inbound {
	return option.Inbound{
		Type: C.TypeSOCKS,
		Tag:  tag,
		Options: &option.SocksInboundOptions{
			ListenOptions: option.ListenOptions{
				Listen:     &localAddr,
				ListenPort: uint16(port),
			},
		},
	}
}

// httpInbound creates a no-auth HTTP inbound bound to 127.0.0.1.
func httpInbound(tag string, port int) option.Inbound {
	return option.Inbound{
		Type: C.TypeHTTP,
		Tag:  tag,
		Options: &option.HTTPMixedInboundOptions{
			ListenOptions: option.ListenOptions{
				Listen:     &localAddr,
				ListenPort: uint16(port),
			},
		},
	}
}

// buildVLESSOutbound converts a parsed Node into a sing-box VLESS outbound.
func buildVLESSOutbound(node *sub.Node) (*option.Outbound, error) {
	ob := &option.Outbound{
		Type: C.TypeVLESS,
		Tag:  node.Tag,
		Options: &option.VLESSOutboundOptions{
			ServerOptions: option.ServerOptions{
				Server:     node.Server,
				ServerPort: uint16(node.ServerPort),
			},
			UUID: node.UUID,
			Flow: node.Flow,
		},
	}

	if node.Transport != nil {
		transport, err := buildTransport(node.Transport)
		if err != nil {
			return nil, err
		}
		ob.Options.(*option.VLESSOutboundOptions).Transport = transport
	}

	if node.TLS != nil {
		ob.Options.(*option.VLESSOutboundOptions).TLS = buildTLS(node.TLS)
	}

	return ob, nil
}

// buildTransport converts a parsed Transport into a V2RayTransportOptions.
func buildTransport(t *sub.Transport) (*option.V2RayTransportOptions, error) {
	switch t.Type {
	case "grpc":
		return &option.V2RayTransportOptions{
			Type:        C.V2RayTransportTypeGRPC,
			GRPCOptions: option.V2RayGRPCOptions{ServiceName: t.ServiceName},
		}, nil
	case "ws":
		ws := option.V2RayWebsocketOptions{Path: t.Path}
		if t.Host != "" {
			ws.Headers = badoption.HTTPHeader{"Host": {t.Host}}
		}
		return &option.V2RayTransportOptions{
			Type:             C.V2RayTransportTypeWebsocket,
			WebsocketOptions: ws,
		}, nil
	case "xhttp":
		xhttp := option.V2RayXHTTPOptions{
			Mode: t.Mode,
			V2RayXHTTPBaseOptions: option.V2RayXHTTPBaseOptions{
				Host: t.Host,
				Path: t.Path,
			},
		}
		if t.XPaddingBytes != nil {
			xhttp.XPaddingBytes = badoption.Range[int]{
				From: t.XPaddingBytes.From,
				To:   t.XPaddingBytes.To,
			}
		}
		if len(t.Headers) > 0 {
			xhttp.Headers = t.Headers
		}
		if len(t.Xmux) > 0 {
			xhttp.Xmux = buildXmux(t.Xmux)
		}
		return &option.V2RayTransportOptions{
			Type:        C.V2RayTransportTypeXHTTP,
			XHTTPOptions: xhttp,
		}, nil
	default:
		return nil, fmt.Errorf("core: unsupported transport %q", t.Type)
	}
}

// buildXmux converts the parser's camelCase xmux map into the sing-box
// V2RayXHTTPXmuxOptions struct. Values may be strings ("64-128"), ints,
// or JSON floats.
func buildXmux(m map[string]any) *option.V2RayXHTTPXmuxOptions {
	x := &option.V2RayXHTTPXmuxOptions{}
	x.MaxConcurrency = rangeFromAny(m["maxConcurrency"])
	x.MaxConnections = rangeFromAny(m["maxConnections"])
	x.CMaxReuseTimes = rangeFromAny(m["cMaxReuseTimes"])
	x.HMaxRequestTimes = rangeFromAny(m["hMaxRequestTimes"])
	x.HMaxReusableSecs = rangeFromAny(m["hMaxReusableSecs"])
	x.HKeepAlivePeriod = int64FromAny(m["hKeepAlivePeriod"])
	return x
}

// rangeFromAny parses a badoption.Range[int] from a value that may be a
// string ("64-128"), an int, or a JSON float64.
func rangeFromAny(v any) badoption.Range[int] {
	switch val := v.(type) {
	case string:
		from, to := parseRangeString(val)
		return badoption.Range[int]{From: from, To: to}
	case float64:
		return badoption.Range[int]{From: int(val), To: int(val)}
	case int:
		return badoption.Range[int]{From: val, To: val}
	case json.Number:
		n, err := val.Int64()
		if err == nil {
			return badoption.Range[int]{From: int(n), To: int(n)}
		}
	}
	return badoption.Range[int]{}
}

// parseRangeString parses a "from-to" or single-value string into a pair.
func parseRangeString(s string) (int, int) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "-"); idx > 0 {
		from, err1 := strconv.Atoi(s[:idx])
		to, err2 := strconv.Atoi(s[idx+1:])
		if err1 == nil && err2 == nil {
			return from, to
		}
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, n
	}
	return 0, 0
}

// int64FromAny extracts an int64 from a value that may be int, float64,
// string, or json.Number.
func int64FromAny(v any) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	case string:
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	case json.Number:
		if n, err := val.Int64(); err == nil {
			return n
		}
	}
	return 0
}

// buildTLS converts a parsed TLS descriptor into a sing-box
// OutboundTLSOptions.
func buildTLS(t *sub.TLS) *option.OutboundTLSOptions {
	tls := &option.OutboundTLSOptions{
		Enabled:    t.Enabled,
		ServerName: t.ServerName,
	}
	if t.UTLS != nil {
		tls.UTLS = &option.OutboundUTLSOptions{
			Enabled:     t.UTLS.Enabled,
			Fingerprint: t.UTLS.Fingerprint,
		}
	}
	if t.Reality != nil {
		tls.Reality = &option.OutboundRealityOptions{
			Enabled:   t.Reality.Enabled,
			PublicKey: t.Reality.PublicKey,
			ShortID:   t.Reality.ShortID,
		}
	}
	return tls
}

// buildDNS creates a minimal DNS configuration using the system resolver
// (local) as the default. This is sufficient for a passthrough proxy that
// does not perform domain-based routing.
func buildDNS() *option.DNSOptions {
	return &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Servers: []option.DNSServerOptions{
				{
					Type: C.DNSTypeLocal,
					Tag:  "local",
					Options: &option.LocalDNSServerOptions{},
				},
			},
			Final: "local",
		},
	}
}

// isRU reports whether a node is classified as Russian, applying the
// user override when present and falling back to the flag-emoji default.
func isRU(node *sub.Node, overrides map[string]bool) bool {
	if override, ok := overrides[node.Tag]; ok {
		return override
	}
	return node.IsRU()
}

// indexByTag converts a slice of assignments into a tag-keyed map.
func indexByTag(assignments []ports.Assignment) map[string]ports.Assignment {
	m := make(map[string]ports.Assignment, len(assignments))
	for _, a := range assignments {
		m[a.Tag] = a
	}
	return m
}

// Outbound tag constants for balancer groups and the direct fallback.
const (
	ruGroupTag    = "ru-group"
	nonruGroupTag = "nonru-group"
	directTag     = "direct"
	ruSOCKSTag    = "socks-ru"
	ruHTTPTag     = "http-ru"
	nonruSOCKSTag = "socks-nonru"
	nonruHTTPTag  = "http-nonru"
)
