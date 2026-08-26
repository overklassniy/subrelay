// Package sub (parse_test.go) contains table-driven tests for the VLESS
// parser using synthetic links that exercise the same field combinations
// as a real subscription without exposing any real credentials.
package sub

import (
	"encoding/base64"
	"net/url"
	"reflect"
	"testing"
)

// buildVLESS constructs a vless:// URL string from the given parts,
// handling query encoding and fragment embedding so the test does not
// depend on hand-rolled percent-encoding.
func buildVLESS(uuid, host string, port int, fragment string, query url.Values) string {
	u := &url.URL{
		Scheme:   "vless",
		User:     url.User(uuid),
		Host:     host,
		RawQuery: query.Encode(),
		Fragment: fragment,
	}
	if port > 0 {
		u.Host = u.Host + ":" + itoa(port)
	}
	return u.String()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// Synthetic test constants. These are NOT real credentials; they are
// randomly generated placeholders that exercise the same parser paths.
const (
	testUUID    = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	testSNI     = "example.com"
	testSNI2    = "edge.example.com"
	testSNI3    = "ads.example.com"
	testPBK     = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	testPBK2    = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	testSID     = "aaaaaaaaaaaaaaaa"
	testSID2    = "bbbbbbbbbbbbbbbb"
	testGRPCSvc = "grpc.ServiceName"
	testPath    = "/api/v1/"
	testHost    = "edge.example.com"
)

func TestParseVLESSReality(t *testing.T) {
	q := url.Values{
		"flow":     {"xtls-rprx-vision"},
		"security": {"reality"},
		"sni":      {testSNI},
		"fp":       {"qq"},
		"pbk":      {testPBK},
		"sid":      {testSID},
		"type":     {"tcp"},
	}
	raw := buildVLESS(testUUID, "10.0.0.1", 6001,
		"FR VPN | France", q)

	node, err := ParseVLESS(raw, 1)
	if err != nil {
		t.Fatalf("ParseVLESS error: %v", err)
	}
	if node.UUID != testUUID {
		t.Errorf("uuid = %q", node.UUID)
	}
	if node.Server != "10.0.0.1" {
		t.Errorf("server = %q", node.Server)
	}
	if node.ServerPort != 6001 {
		t.Errorf("port = %d", node.ServerPort)
	}
	if node.Flow != "xtls-rprx-vision" {
		t.Errorf("flow = %q", node.Flow)
	}
	if node.Transport != nil {
		t.Errorf("expected no transport for tcp, got %+v", node.Transport)
	}
	if node.TLS == nil || !node.TLS.Enabled {
		t.Fatalf("expected enabled TLS")
	}
	if node.TLS.ServerName != testSNI {
		t.Errorf("sni = %q", node.TLS.ServerName)
	}
	if node.TLS.UTLS == nil || node.TLS.UTLS.Fingerprint != "qq" {
		t.Errorf("utls fingerprint = %+v", node.TLS.UTLS)
	}
	if node.TLS.Reality == nil ||
		node.TLS.Reality.PublicKey != testPBK ||
		node.TLS.Reality.ShortID != testSID {
		t.Errorf("reality = %+v", node.TLS.Reality)
	}
}

func TestParseVLESSGRPC(t *testing.T) {
	q := url.Values{
		"security":    {"tls"},
		"sni":         {testSNI},
		"type":        {"grpc"},
		"serviceName": {testGRPCSvc},
	}
	raw := buildVLESS(testUUID, "10.0.0.2", 443,
		"KZ VPN | Kazakhstan", q)

	node, err := ParseVLESS(raw, 1)
	if err != nil {
		t.Fatalf("ParseVLESS error: %v", err)
	}
	if node.Transport == nil || node.Transport.Type != "grpc" {
		t.Fatalf("transport = %+v", node.Transport)
	}
	if node.Transport.ServiceName != testGRPCSvc {
		t.Errorf("serviceName = %q", node.Transport.ServiceName)
	}
	if node.TLS == nil || node.TLS.ServerName != testSNI {
		t.Errorf("tls = %+v", node.TLS)
	}
	if node.TLS.UTLS != nil || node.TLS.Reality != nil {
		t.Errorf("plain tls must not set utls/reality, got %+v", node.TLS)
	}
}

func TestParseVLESSWS(t *testing.T) {
	q := url.Values{
		"security": {"tls"},
		"sni":      {testSNI},
		"type":     {"ws"},
	}
	raw := buildVLESS(testUUID, "10.0.0.3", 443,
		"RU LTE", q)

	node, err := ParseVLESS(raw, 1)
	if err != nil {
		t.Fatalf("ParseVLESS error: %v", err)
	}
	if node.Transport == nil || node.Transport.Type != "ws" {
		t.Fatalf("transport = %+v", node.Transport)
	}
}

func TestParseVLESSXHTTPWithExtra(t *testing.T) {
	extra := `{"xmux":{"cMaxReuseTimes":"64-128","maxConcurrency":"24-64","maxConnections":0,"hKeepAlivePeriod":30,"hMaxRequestTimes":"800-1200","hMaxReusableSecs":"1800-3600"}}`
	q := url.Values{
		"security": {"tls"},
		"sni":      {testSNI2},
		"type":     {"xhttp"},
		"path":     {testPath},
		"host":     {testHost},
		"mode":     {"packet-up"},
		"extra":    {extra},
	}
	raw := buildVLESS(testUUID, "10.0.0.4", 443,
		"RU LTE 4", q)

	node, err := ParseVLESS(raw, 1)
	if err != nil {
		t.Fatalf("ParseVLESS error: %v", err)
	}
	if node.Transport == nil || node.Transport.Type != "xhttp" {
		t.Fatalf("transport = %+v", node.Transport)
	}
	if node.Transport.Path != testPath {
		t.Errorf("path = %q", node.Transport.Path)
	}
	if node.Transport.Host != testHost {
		t.Errorf("host = %q", node.Transport.Host)
	}
	if node.Transport.Mode != "packet-up" {
		t.Errorf("mode = %q", node.Transport.Mode)
	}
	if node.Transport.XPaddingBytes == nil ||
		node.Transport.XPaddingBytes.From != 64 ||
		node.Transport.XPaddingBytes.To != 256 {
		t.Errorf("xPaddingBytes = %+v", node.Transport.XPaddingBytes)
	}
	if node.Transport.Xmux == nil {
		t.Errorf("expected xmux from extra")
	} else {
		if node.Transport.Xmux["maxConnections"] != float64(0) {
			t.Errorf("xmux maxConnections = %v", node.Transport.Xmux["maxConnections"])
		}
		if node.Transport.Xmux["hKeepAlivePeriod"] != float64(30) {
			t.Errorf("xmux hKeepAlivePeriod = %v", node.Transport.Xmux["hKeepAlivePeriod"])
		}
		if node.Transport.Xmux["cMaxReuseTimes"] != "64-128" {
			t.Errorf("xmux cMaxReuseTimes = %v", node.Transport.Xmux["cMaxReuseTimes"])
		}
	}
}

func TestParseVLESSDefaultsPortAndSNI(t *testing.T) {
	// No port, no sni: port defaults to 443, sni falls back to server.
	q := url.Values{"security": {"tls"}}
	raw := buildVLESS("uuid-value", "1.2.3.4", 0, "node", q)

	node, err := ParseVLESS(raw, 1)
	if err != nil {
		t.Fatalf("ParseVLESS error: %v", err)
	}
	if node.ServerPort != 443 {
		t.Errorf("default port = %d, want 443", node.ServerPort)
	}
	if node.TLS == nil || node.TLS.ServerName != "1.2.3.4" {
		t.Errorf("sni fallback = %+v", node.TLS)
	}
}

func TestParseVLESSRejectsNonVLESS(t *testing.T) {
	if _, err := ParseVLESS("https://example.com", 1); err == nil {
		t.Errorf("expected error for non-vless scheme")
	}
}

func TestParseSubscriptionTextBase64(t *testing.T) {
	plain := "vless://uuid@1.2.3.4:443?security=tls&type=tcp#node1\n" +
		"vless://uuid@5.6.7.8:443?security=tls&type=tcp#node2\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))

	nodes := ParseSubscriptionText(encoded)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Tag != "node1" || nodes[1].Tag != "node2" {
		t.Errorf("tags = %q, %q", nodes[0].Tag, nodes[1].Tag)
	}
}

func TestParseSubscriptionTextPlain(t *testing.T) {
	plain := "vless://uuid@1.2.3.4:443?security=tls&type=tcp#node1\n" +
		"not a link\n" +
		"vless://uuid@5.6.7.8:443?security=tls&type=tcp#node2\n"
	nodes := ParseSubscriptionText(plain)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes from plain text, got %d", len(nodes))
	}
}

func TestSafeTag(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Pipe is removed, leaving the adjacent underscores from the
		// surrounding spaces (matches the Python reference behavior).
		{"FR VPN | France", "FR_VPN__France"},
		{"KZ VPN | Kazakhstan \u00b5Torrent", "KZ_VPN__Kazakhstan_uTorrent"},
		// Cyrillic letters are retained because str.isalnum() is
		// Unicode-aware in the Python reference.
		{"RU LTE | Все операторы-4", "RU_LTE__Все_операторы-4"},
		{"  spaces  ", "__spaces__"},
	}
	for _, c := range cases {
		got := SafeTag(c.in)
		if got != c.want {
			t.Errorf("SafeTag(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsAutoTag(t *testing.T) {
	cases := []struct {
		tag  string
		want bool
	}{
		{"Auto selector", true},
		{"Авто узел", true},
		{"auto-node", true},
		// Latin transliteration "avto" is NOT matched by the reference,
		// which checks Cyrillic "авто" and Latin "auto".
		{"avto-node", false},
		{"France VPN", false},
		{"manual", false},
	}
	for _, c := range cases {
		if got := IsAutoTag(c.tag); got != c.want {
			t.Errorf("IsAutoTag(%q) = %v, want %v", c.tag, got, c.want)
		}
	}
}

func TestNodeIsRU(t *testing.T) {
	node := &Node{Tag: "RU " + RUFlag + " LTE"}
	if !node.IsRU() {
		t.Errorf("expected IsRU true for tag with flag")
	}
	node2 := &Node{Tag: "FR VPN"}
	if node2.IsRU() {
		t.Errorf("expected IsRU false for non-RU tag")
	}
}

func TestNodeTransportType(t *testing.T) {
	n := &Node{}
	if n.TransportType() != "tcp" {
		t.Errorf("empty transport type = %q, want tcp", n.TransportType())
	}
	n2 := &Node{Transport: &Transport{Type: "grpc"}}
	if n2.TransportType() != "grpc" {
		t.Errorf("transport type = %q, want grpc", n2.TransportType())
	}
}

func TestParseVLESSRoundTripFields(t *testing.T) {
	// Verify the full set of fields against a synthetic Reality entry.
	q := url.Values{
		"flow":     {"xtls-rprx-vision"},
		"security": {"reality"},
		"sni":      {testSNI3},
		"fp":       {"qq"},
		"pbk":      {testPBK2},
		"sid":      {testSID2},
		"type":     {"tcp"},
	}
	raw := buildVLESS(testUUID, "10.0.0.5", 9443,
		"RU LTE 2", q)
	node, err := ParseVLESS(raw, 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := &Node{
		Tag:        "RU LTE 2",
		Server:     "10.0.0.5",
		ServerPort: 9443,
		UUID:       testUUID,
		Flow:       "xtls-rprx-vision",
		TLS: &TLS{
			Enabled:    true,
			ServerName: testSNI3,
			UTLS:       &UTLS{Enabled: true, Fingerprint: "qq"},
			Reality: &Reality{
				Enabled:   true,
				PublicKey: testPBK2,
				ShortID:   testSID2,
			},
		},
	}
	// Compare the simple fields explicitly; reflect.DeepEqual would also
	// work but this makes failures more readable.
	if !reflect.DeepEqual(node, want) {
		t.Errorf("node mismatch:\n got  %+v\n want %+v", node, want)
	}
}
