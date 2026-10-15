package dnswildcard

import (
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestWildcard(t *testing.T) {
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		q := r.Question[0]
		m := new(dns.Msg)
		m.SetReply(r)
		switch {
		case strings.HasSuffix(q.Name, ".wild.test.") && q.Qtype == dns.TypeA:
			rr, _ := dns.NewRR(q.Name + " 300 IN A 192.0.2.80")
			m.Answer = []dns.RR{rr}
		case strings.HasSuffix(q.Name, ".lb.test.") && q.Qtype == dns.TypeA:
			rr, _ := dns.NewRR(q.Name + " 300 IN A 192.0.2." + strconv.Itoa(len(q.Name)))
			m.Answer = []dns.RR{rr}
		case strings.HasSuffix(q.Name, ".wild.test."), strings.HasSuffix(q.Name, ".lb.test."):
		default:
			m.Rcode = dns.RcodeNameError
		}
		_ = w.WriteMsg(m)
	})
	dnsx.SystemOverride = &addr
	defer func() { dnsx.SystemOverride = nil }()
	var n atomic.Int32
	label := func() string { return "rnd" + strings.Repeat("x", int(n.Add(1))) }
	env := tooltest.New(t, &Tool{Label: label}, tooltest.Options{})

	var res Result
	env.Get("/api/v1/dns/wildcard?domain=wild.test").Must(t).Decode(t, &res)
	if !res.Wildcard || !res.Consistent || res.Types[0] != "A" || len(res.Probes) != Probes || res.Probes[0].Rcode != "NOERROR" {
		t.Errorf("wild = %+v", res)
	}
	env.Get("/api/v1/dns/wildcard?domain=lb.test").Must(t).Decode(t, &res)
	if !res.Wildcard || res.Consistent {
		t.Errorf("varying = %+v", res)
	}
	env.Get("/api/v1/dns/wildcard?domain=*.plain.test").Must(t).Decode(t, &res)
	if res.Wildcard || res.Domain != "plain.test" || res.Probes[0].Rcode != "NXDOMAIN" {
		t.Errorf("plain = %+v", res)
	}
	env.Get("/api/v1/dns/wildcard?domain=localhost").ExpectError(t, "INVALID_INPUT")
}
