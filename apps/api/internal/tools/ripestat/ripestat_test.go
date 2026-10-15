package ripestat

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeRIPEstat serves testdata/<call>.json; special resources select alternate fixtures.
func fakeRIPEstat(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sourceapp") != "nettools" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		call := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), "/data.json")
		res := r.URL.Query().Get("resource")
		switch {
		case call == "routing-status" && res == "203.0.113.0/24":
			call = "routing-status-moas"
		case call == "rpki-validation" && res == "AS64500":
			call = "rpki-invalid"
		case call == "rpki-validation" && r.URL.Query().Get("prefix") == "192.0.2.0/24":
			call = "rpki-unknown"
		case call == "rpki-validation" && r.URL.Query().Get("prefix") == "198.51.100.0/24":
			call = "rpki-invalid"
		case res == "1.2.3.0/24":
			w.WriteHeader(http.StatusBadRequest)
			b, _ := os.ReadFile("testdata/error.json")
			_, _ = w.Write(b)
			return
		}
		b, err := os.ReadFile("testdata/" + call + ".json")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func env(t *testing.T, tool tools.Tool) *tooltest.Env {
	return tooltest.New(t, tool, tooltest.Options{AllowPrivate: true})
}

func TestASNInfo(t *testing.T) {
	rs := Client{Base: fakeRIPEstat(t)}
	var out ASNInfo
	env(t, &ASNTool{RS: rs}).Get("/api/v1/asn/AS13335").Must(t).Decode(t, &out)
	if out.ASN != 13335 || out.Holder != "CLOUDFLARENET - Cloudflare, Inc." || !out.Announced || len(out.PrefixesV4) != 6 || out.PrefixesV6[0] != "2400:cb00:a301::/48" || out.Upstreams != 318 || out.Downstreams != 1032 || out.Neighbours[0].ASN != 1299 || out.Neighbours[0].Power < out.Neighbours[1].Power {
		t.Errorf("out = %+v", out)
	}
	env(t, &ASNTool{RS: rs}).Get("/api/v1/asn/not-an-asn").ExpectError(t, "INVALID_INPUT")
}

func TestPrefixStatus(t *testing.T) {
	e := env(t, &PrefixTool{RS: Client{Base: fakeRIPEstat(t)}})
	var out PrefixStatus
	e.Get("/api/v1/prefix/status?prefix=1.1.1.1").Must(t).Decode(t, &out)
	if out.Prefix != "1.1.1.1/32" {
		t.Errorf("host route = %q", out.Prefix)
	}
	e.Get("/api/v1/prefix/status?prefix=1.1.1.0/24").Must(t).Decode(t, &out)
	if !out.Announced || out.SeenBy != 109 || out.TotalPeers != 111 || out.MOAS || out.Origins[0].ASN != 13335 || out.FirstSeen.Origin != 19855 || out.FirstSeen.Time != "2001-11-14T00:00:00" || len(out.LessSpecifics) != 0 {
		t.Errorf("out = %+v", out)
	}
	e.Get("/api/v1/prefix/status?prefix=203.0.113.0/24").Must(t).Decode(t, &out)
	if !out.MOAS || out.Origins[1].ASN != 64511 {
		t.Errorf("moas = %+v", out)
	}
	for _, bad := range []string{"10.0.0.0/8", "192.168.1.1", "100.64.0.0/10", "fd00::/8", "nope", "1.1.1.0/33"} {
		e.Get("/api/v1/prefix/status?prefix="+bad).ExpectError(t, "INVALID_INPUT")
	}
	if r := e.Get("/api/v1/prefix/status?prefix=1.2.3.0/24"); r.OK || r.Error.Code != "PROVIDER_ERROR" || !strings.Contains(r.Error.Message, "Invalid resource") {
		t.Errorf("upstream error = %+v", r.Error)
	}
}

func TestRPKI(t *testing.T) {
	e := env(t, &RPKITool{RS: Client{Base: fakeRIPEstat(t)}})
	var out RPKIResult
	e.Get("/api/v1/rpki?prefix=1.1.1.0/24&asn=13335").Must(t).Decode(t, &out)
	if out.Status != "valid" || out.ROAs[0].MaxLength != 24 || out.ROAs[0].Origin != 13335 {
		t.Errorf("out = %+v", out)
	}
	e.Get("/api/v1/rpki?prefix=1.1.1.0/24&asn=AS64500").Must(t).Decode(t, &out)
	if out.Status != "invalid_asn" {
		t.Errorf("invalid = %+v", out)
	}
	e.Get("/api/v1/rpki?prefix=1.1.1.0/24").ExpectError(t, "INVALID_INPUT")
}

func TestLookingGlass(t *testing.T) {
	var out LookingGlass
	env(t, &LookingGlassTool{RS: Client{Base: fakeRIPEstat(t)}}).Get("/api/v1/lookingglass?prefix=1.1.1.0/24").Must(t).Decode(t, &out)
	if out.Collectors != 2 || out.Total != 4 || len(out.Origins) != 1 || out.Origins[0] != 13335 || out.Routes[1].ASPath != "8218 13335" || out.Routes[2].Location != "Amsterdam, NL" {
		t.Errorf("out = %+v", out)
	}
}

func TestCountryBlocks(t *testing.T) {
	rs := Client{Base: fakeRIPEstat(t)}
	var out CountryBlocks
	env(t, &CountryTool{RS: rs}).Get("/api/v1/ip/country-blocks?country=nl").Must(t).Decode(t, &out)
	if out.Country != "NL" || out.ASNs != 3 || out.IPv4Entries != 4 || out.IPv6Entries != 3 {
		t.Fatalf("out = %+v", out)
	}
	want4 := []string{"192.0.2.0/24", "198.51.100.0/24", "203.0.113.10/31", "203.0.113.12/30", "203.0.113.16/30", "203.0.113.20/32"}
	if strings.Join(out.IPv4, " ") != strings.Join(want4, " ") || out.IPv4Addresses != 256+256+11 {
		t.Errorf("ipv4 = %v (%d addresses)", out.IPv4, out.IPv4Addresses)
	}
	if strings.Join(out.IPv6, " ") != "2001:db8::/32 2001:db9::/48" || out.IPv6Slash48s != 65536+1 {
		t.Errorf("ipv6 = %v (%d /48s)", out.IPv6, out.IPv6Slash48s)
	}
	env(t, &CountryTool{RS: rs}).Get("/api/v1/ip/country-blocks?country=NL&family=6").Must(t).Decode(t, &out)
	if len(out.IPv4) != 0 || len(out.IPv6) != 2 {
		t.Errorf("family=6: %+v", out)
	}
	env(t, &CountryTool{RS: rs}).Get("/api/v1/ip/country-blocks?country=N1").ExpectError(t, "INVALID_INPUT")
	env(t, &CountryTool{RS: rs}).Get("/api/v1/ip/country-blocks?country=NL&family=5").ExpectError(t, "INVALID_INPUT")
}

func TestRouteHistory(t *testing.T) {
	var out History
	env(t, &HistoryTool{RS: Client{Base: fakeRIPEstat(t)}}).Get("/api/v1/bgp/route-history?resource=193.0.0.0/21").Must(t).Decode(t, &out)
	if len(out.Origins) != 2 || out.Origins[0].ASN != 3333 || !out.Origins[0].Current || out.Origins[1].Current || out.Origins[0].Name != "RIPE-NCC-AS" || out.Origins[0].Prefixes[0].MaxPeers != 330.9 || out.Origins[0].Prefixes[0].Periods != 2 {
		t.Errorf("origins = %+v", out.Origins)
	}
	if out.FullTablePeers != 40 || out.Seeing != 39 || len(out.Collectors) != 2 || out.Related != 1 {
		t.Errorf("visibility = %+v", out)
	}
	env(t, &HistoryTool{RS: Client{Base: fakeRIPEstat(t)}}).Get("/api/v1/bgp/route-history?resource=nope").ExpectError(t, "INVALID_INPUT")
}

func TestNeighbourGraph(t *testing.T) {
	var out Graph
	env(t, &GraphTool{RS: Client{Base: fakeRIPEstat(t)}}).Get("/api/v1/bgp/neighbours/AS13335").Must(t).Decode(t, &out)
	if out.Name != "CLOUDFLARENET" || out.UpstreamTotal == 0 || out.Upstreams[0].ASN != 1299 || out.Upstreams[0].Name != "TWELVE99" || out.UpstreamTotal+out.DownstreamTotal+out.UncertainTotal != 8 {
		t.Errorf("graph = %+v", out)
	}
}

func TestROACoverage(t *testing.T) {
	var out ROACoverage
	env(t, &ROATool{RS: Client{Base: fakeRIPEstat(t)}}).Get("/api/v1/bgp/roa-coverage/AS13335").Must(t).Decode(t, &out)
	status := map[string]string{}
	for _, p := range out.Prefixes {
		status[p.Prefix] = p.Status + "/" + p.Reason
	}
	want := map[string]string{"1.1.1.0/24": "valid/", "104.16.0.0/13": "valid/", "1.0.0.0/25": "invalid/length", "192.0.2.0/24": "not-found/", "198.51.100.0/24": "invalid/asn", "2606:4700::/32": "valid/"}
	for k, v := range want {
		if status[k] != v {
			t.Errorf("%s = %s, want %s", k, status[k], v)
		}
	}
	if out.Valid != 3 || out.Invalid != 2 || out.NotFound != 1 || out.Validator != "routinator" {
		t.Errorf("counts = %+v", out)
	}
	roas := map[string]ROAEntry{}
	for _, r := range out.ROAs {
		roas[r.Prefix] = r
	}
	if roas["203.0.113.0/24"].Used || !roas["2606:4700::/32"].Loose || roas["1.1.1.0/24"].Loose || !roas["104.16.0.0/12"].Loose {
		t.Errorf("roas = %+v", out.ROAs)
	}
}
