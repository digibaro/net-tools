package ipv6ready

import (
	"context"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestIPv6Readiness(t *testing.T) {
	zone := map[string]string{
		"full.test. A":        "full.test. 300 IN A 93.0.0.1",
		"full.test. AAAA":     "full.test. 300 IN AAAA 2001:db8:1::1",
		"www.full.test. AAAA": "www.full.test. 300 IN AAAA 2001:db8:1::2",
		"full.test. NS":       "full.test. 300 IN NS ns1.full.test.",
		"ns1.full.test. AAAA": "ns1.full.test. 300 IN AAAA 2001:db8:1::53",
		"full.test. MX":       "full.test. 300 IN MX 10 mx.full.test.",
		"mx.full.test. AAAA":  "mx.full.test. 300 IN AAAA 2001:db8:1::25",
		"old.test. A":         "old.test. 300 IN A 93.0.0.2",
		"old.test. NS":        "old.test. 300 IN NS ns1.old.test.",
		"ns1.old.test. A":     "ns1.old.test. 300 IN A 93.0.0.53",
	}
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		q := r.Question[0]
		m := new(dns.Msg)
		m.SetReply(r)
		if s, ok := zone[q.Name+" "+dns.TypeToString[q.Qtype]]; ok {
			rr, _ := dns.NewRR(s)
			m.Answer = []dns.RR{rr}
		}
		_ = w.WriteMsg(m)
	})
	dnsx.SystemOverride = &addr
	defer func() { dnsx.SystemOverride = nil }()

	probe := func(_ context.Context, _ string, a netip.AddrPort) bool {
		return a.Addr() != netip.MustParseAddr("2001:db8:1::25") // the MX does not answer on v6
	}
	// 2001:db8::/32 is documentation space; allow it via AllowPrivate for the test.
	env := tooltest.New(t, &Tool{Probe: probe}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/ipv6/readiness?domain=full.test").Must(t).Decode(t, &res)
	if !res.ServerIPv6 || len(res.Hosts) != 4 || res.Verdict != "partial" || res.Score != 75 {
		t.Errorf("full = %+v", res)
	}
	for _, h := range res.Hosts {
		if h.Role == "mx" && (h.Reachable == nil || *h.Reachable) {
			t.Errorf("mx = %+v", h)
		}
	}
	env.Get("/api/v1/ipv6/readiness?domain=old.test").Must(t).Decode(t, &res)
	if res.Verdict != "not-ready" || res.Score != 25 {
		t.Errorf("old = %+v", res)
	}

	// Without IPv6 on the server, AAAA records still count but nothing is dialed.
	noV6 := tooltest.New(t, &Tool{Probe: func(context.Context, string, netip.AddrPort) bool { return false }}, tooltest.Options{AllowPrivate: true})
	noV6.Get("/api/v1/ipv6/readiness?domain=full.test").Must(t).Decode(t, &res)
	if res.ServerIPv6 || res.Verdict != "ready" || res.Hosts[0].Reachable != nil {
		t.Errorf("no server v6 = %+v", res)
	}
	env.Get("/api/v1/ipv6/readiness?domain=localhost").ExpectError(t, "INVALID_INPUT")
}

// AAAA records pointing at private space are not dialed in the public profile.
func TestIPv6ReadinessPolicy(t *testing.T) {
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		q := r.Question[0]
		m := new(dns.Msg)
		m.SetReply(r)
		if q.Name == "inside.test." && q.Qtype == dns.TypeAAAA {
			rr, _ := dns.NewRR("inside.test. 300 IN AAAA fd00::1")
			m.Answer = []dns.RR{rr}
		}
		_ = w.WriteMsg(m)
	})
	dnsx.SystemOverride = &addr
	defer func() { dnsx.SystemOverride = nil }()
	var dialed atomic.Bool
	probe := func(_ context.Context, _ string, a netip.AddrPort) bool {
		if a.Addr() == netip.MustParseAddr("fd00::1") {
			dialed.Store(true)
		}
		return true
	}
	var res Result
	tooltest.New(t, &Tool{Probe: probe}, tooltest.Options{}).Get("/api/v1/ipv6/readiness?domain=inside.test").Must(t).Decode(t, &res)
	if dialed.Load() || !strings.Contains(res.Hosts[0].Error, "private") {
		t.Errorf("dialed=%v hosts=%+v", dialed.Load(), res.Hosts)
	}
}
