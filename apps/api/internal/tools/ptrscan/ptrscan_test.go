package ptrscan

import (
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// .1 has a confirmed PTR, .2 a PTR whose name points elsewhere, the rest nothing.
func zone(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	add := func(s string) {
		rr, _ := dns.NewRR(s)
		m.Answer = append(m.Answer, rr)
	}
	switch r.Question[0].Name {
	case "1.2.0.192.in-addr.arpa.":
		add("1.2.0.192.in-addr.arpa. 300 IN PTR gw.example.test.")
	case "2.2.0.192.in-addr.arpa.":
		add("2.2.0.192.in-addr.arpa. 300 IN PTR stale.example.test.")
	case "gw.example.test.":
		if r.Question[0].Qtype == dns.TypeA {
			add("gw.example.test. 300 IN A 192.0.2.1")
		}
	case "stale.example.test.":
		if r.Question[0].Qtype == dns.TypeA {
			add("stale.example.test. 300 IN A 198.51.100.7")
		}
	default:
		m.Rcode = dns.RcodeNameError
	}
	_ = w.WriteMsg(m)
}

func TestPTRScan(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	q := "&resolver=" + url.QueryEscape(dnstest.Start(t, zone).String())
	var res Result
	env.Get("/api/v1/dns/ptr-scan?prefix=192.0.2.0/30"+q).Must(t).Decode(t, &res)
	if res.Total != 4 || res.WithPTR != 2 || res.Confirmed != 1 || !res.Rows[1].Confirmed || res.Rows[2].Confirmed || res.Rows[0].Rcode != "NXDOMAIN" {
		t.Errorf("res = %+v", res)
	}
	env.Get("/api/v1/dns/ptr-scan?prefix=192.0.2.9"+q).Must(t).Decode(t, &res)
	if res.Prefix != "192.0.2.0/24" || res.Total != 256 {
		t.Errorf("bare address = %s %d", res.Prefix, res.Total)
	}
	env.Get("/api/v1/dns/ptr-scan?prefix=192.0.0.0/23"+q).ExpectError(t, "INVALID_INPUT")
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/dns/ptr-scan?prefix=10.0.0.0/24").ExpectError(t, "TARGET_PRIVATE_DENIED")
}
