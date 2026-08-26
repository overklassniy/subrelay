// Package core (builder_test.go) verifies config generation from parsed
// nodes, port assignments, and settings.
package core

import (
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"

	"subrelay/internal/config"
	"subrelay/internal/ports"
	"subrelay/internal/sub"
)

func testBuildSettings() *config.Settings {
	s := config.Defaults()
	s.Headers.XHWID = "testhwid"
	return s
}

func sampleNodes() []sub.Node {
	return []sub.Node{
		{
			Tag:        "RU " + sub.RUFlag + " LTE",
			Server:     "1.2.3.4",
			ServerPort: 443,
			UUID:       "uuid-1",
			Flow:       "xtls-rprx-vision",
			TLS: &sub.TLS{
				Enabled:    true,
				ServerName: "sni.example",
				UTLS:       &sub.UTLS{Enabled: true, Fingerprint: "qq"},
				Reality: &sub.Reality{
					Enabled:   true,
					PublicKey: "pk1",
					ShortID:   "sid1",
				},
			},
		},
		{
			Tag:        "FR VPN",
			Server:     "5.6.7.8",
			ServerPort: 443,
			UUID:       "uuid-2",
			Transport:  &sub.Transport{Type: "grpc", ServiceName: "grpc.ServiceName"},
			TLS:        &sub.TLS{Enabled: true, ServerName: "sni.example"},
		},
		{
			Tag:        "Auto selector",
			Server:     "9.9.9.9",
			ServerPort: 443,
			UUID:       "uuid-auto",
		},
	}
}

func TestBuildProducesBalancersAndPerNodeInbounds(t *testing.T) {
	s := testBuildSettings()
	p := ports.NewPlanner(s)
	nodes := sampleNodes()
	assignments, err := p.Plan(nodes)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	result, err := Build(BuildInput{Settings: s, Nodes: nodes, Assignments: assignments})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	// Two non-auto nodes -> 2 VLESS outbounds + 2 urltest groups + direct.
	countByType := map[string]int{}
	for _, ob := range result.Options.Outbounds {
		countByType[ob.Type]++
	}
	if countByType[C.TypeVLESS] != 2 {
		t.Errorf("vless outbounds = %d, want 2", countByType[C.TypeVLESS])
	}
	if countByType[C.TypeURLTest] != 2 {
		t.Errorf("urltest outbounds = %d, want 2", countByType[C.TypeURLTest])
	}
	if countByType[C.TypeDirect] != 1 {
		t.Errorf("direct outbounds = %d, want 1", countByType[C.TypeDirect])
	}

	// Inbounds: 4 balancer + 2 nodes * 2 (socks+http) = 8.
	if len(result.Options.Inbounds) != 8 {
		t.Errorf("inbounds = %d, want 8", len(result.Options.Inbounds))
	}

	// Rules: 2 per-node + 2 balancer = 4.
	if len(result.Options.Route.Rules) != 4 {
		t.Errorf("rules = %d, want 4", len(result.Options.Route.Rules))
	}

	// RU group has the RU node, non-RU group has the FR node.
	if len(result.RUNodes) != 1 || result.RUNodes[0] != "RU "+sub.RUFlag+" LTE" {
		t.Errorf("RU nodes = %v", result.RUNodes)
	}
	if len(result.NonRUNodes) != 1 || result.NonRUNodes[0] != "FR VPN" {
		t.Errorf("non-RU nodes = %v", result.NonRUNodes)
	}
}

func TestBuildAppliesRUOverride(t *testing.T) {
	s := testBuildSettings()
	// Force the FR node to be RU via override.
	s.RUOverrides["FR VPN"] = true
	p := ports.NewPlanner(s)
	nodes := sampleNodes()
	assignments, _ := p.Plan(nodes)

	result, err := Build(BuildInput{Settings: s, Nodes: nodes, Assignments: assignments})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if len(result.RUNodes) != 2 {
		t.Errorf("RU nodes = %v, want 2", result.RUNodes)
	}
	if len(result.NonRUNodes) != 0 {
		t.Errorf("non-RU nodes = %v, want 0", result.NonRUNodes)
	}
}

func TestBuildExcludesAutoNodes(t *testing.T) {
	s := testBuildSettings()
	nodes := []sub.Node{{Tag: "Auto node", Server: "1.1.1.1", ServerPort: 443, UUID: "u"}}
	assignments, _ := ports.NewPlanner(s).Plan(nodes)
	_, err := Build(BuildInput{Settings: s, Nodes: nodes, Assignments: assignments})
	if err == nil {
		t.Errorf("expected error when only auto nodes are present")
	}
}

func TestBuildXHTTPTransport(t *testing.T) {
	s := testBuildSettings()
	nodes := []sub.Node{
		{
			Tag:        "RU xhttp",
			Server:     "1.2.3.4",
			ServerPort: 443,
			UUID:       "u",
			Transport: &sub.Transport{
				Type:          "xhttp",
				Path:          "/api/v1/",
				Host:          "edge.example.com",
				Mode:          "packet-up",
				XPaddingBytes: &sub.PaddingRange{From: 64, To: 256},
				Xmux: map[string]any{
					"cMaxReuseTimes":   "64-128",
					"maxConcurrency":   "24-64",
					"maxConnections":   float64(0),
					"hKeepAlivePeriod": float64(30),
					"hMaxRequestTimes": "800-1200",
					"hMaxReusableSecs": "1800-3600",
				},
			},
			TLS: &sub.TLS{Enabled: true, ServerName: "edge.example.com"},
		},
	}
	assignments, _ := ports.NewPlanner(s).Plan(nodes)
	result, err := Build(BuildInput{Settings: s, Nodes: nodes, Assignments: assignments})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	var vlessOpts *option.VLESSOutboundOptions
	for i := range result.Options.Outbounds {
		ob := &result.Options.Outbounds[i]
		if ob.Type == C.TypeVLESS {
			vlessOpts = ob.Options.(*option.VLESSOutboundOptions)
			break
		}
	}
	if vlessOpts == nil {
		t.Fatalf("no vless outbound found")
	}
	tr := vlessOpts.Transport
	if tr == nil || tr.Type != C.V2RayTransportTypeXHTTP {
		t.Fatalf("transport = %+v", tr)
	}
	if tr.XHTTPOptions.Mode != "packet-up" {
		t.Errorf("xhttp mode = %q", tr.XHTTPOptions.Mode)
	}
	if tr.XHTTPOptions.Path != "/api/v1/" {
		t.Errorf("xhttp path = %q", tr.XHTTPOptions.Path)
	}
	if tr.XHTTPOptions.XPaddingBytes.From != 64 || tr.XHTTPOptions.XPaddingBytes.To != 256 {
		t.Errorf("xpadding = %+v", tr.XHTTPOptions.XPaddingBytes)
	}
	if tr.XHTTPOptions.Xmux == nil {
		t.Fatalf("expected xmux")
	}
	if tr.XHTTPOptions.Xmux.CMaxReuseTimes.From != 64 || tr.XHTTPOptions.Xmux.CMaxReuseTimes.To != 128 {
		t.Errorf("xmux cMaxReuseTimes = %+v", tr.XHTTPOptions.Xmux.CMaxReuseTimes)
	}
	if tr.XHTTPOptions.Xmux.HKeepAlivePeriod != 30 {
		t.Errorf("xmux hKeepAlivePeriod = %d", tr.XHTTPOptions.Xmux.HKeepAlivePeriod)
	}
}
