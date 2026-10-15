package dns64

import (
	"net/netip"
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestPrefixOf(t *testing.T) {
	for in, want := range map[string]string{
		"64:ff9b::c000:aa":             "64:ff9b::/96",
		"2001:db8:122:344::c000:ab":    "2001:db8:122:344::/96",
		"2001:db8:c000:aa::":           "2001:db8::/32",
		"2001:db8:1c0:0:aa::":          "2001:db8:100::/40",
		"2001:db8:122:c000:0:aa00::":   "2001:db8:122::/48",
		"2001:db8:122:3c0:0:aa:0:0":    "2001:db8:122:300::/56",
		"2001:db8:122:344:c0:0:aa00:0": "2001:db8:122:344::/64",
	} {
		p, ok := PrefixOf(netip.MustParseAddr(in))
		if !ok || p.String() != want {
			t.Errorf("%s: %v %v, want %s", in, p, ok, want)
		}
	}
	if _, ok := PrefixOf(netip.MustParseAddr("2001:db8::1")); ok {
		t.Error("plain address matched")
	}
}

func TestDNS64(t *testing.T) {
	synth := func(prefix string) func(dns.ResponseWriter, *dns.Msg) {
		return func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			add := func(s string) {
				rr, _ := dns.NewRR(s)
				m.Answer = append(m.Answer, rr)
			}
			if r.Question[0].Qtype == dns.TypeA {
				add("ipv4only.arpa. 300 IN A 192.0.0.170")
				add("ipv4only.arpa. 300 IN A 192.0.0.171")
			} else if prefix != "" {
				add("ipv4only.arpa. 300 IN AAAA " + prefix + "c000:aa")
				add("ipv4only.arpa. 300 IN AAAA " + prefix + "c000:ab")
			}
			_ = w.WriteMsg(m)
		}
	}
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/dns64?resolver="+url.QueryEscape(dnstest.Start(t, synth("64:ff9b::")).String())).Must(t).Decode(t, &res)
	if !res.DNS64 || len(res.Synthesized) != 2 || len(res.Prefixes) != 1 || !res.Prefixes[0].WellKnown || len(res.IPv4) != 2 {
		t.Errorf("dns64 = %+v", res)
	}
	env.Get("/api/v1/dns/dns64?resolver="+url.QueryEscape(dnstest.Start(t, synth("")).String())).Must(t).Decode(t, &res)
	if res.DNS64 || len(res.Prefixes) != 0 {
		t.Errorf("plain = %+v", res)
	}
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/dns/dns64?resolver=10.0.0.1").ExpectError(t, "TARGET_PRIVATE_DENIED")
}
