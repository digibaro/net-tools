package dnsaxfr

import (
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func rr(s string) dns.RR {
	r, err := dns.NewRR(s)
	if err != nil {
		panic(err)
	}
	return r
}

// One server answers NS/A queries and allows AXFR for open.test but refuses it for closed.test.
func handler(w dns.ResponseWriter, r *dns.Msg) {
	q := r.Question[0]
	m := new(dns.Msg)
	m.SetReply(r)
	switch {
	case q.Qtype == dns.TypeAXFR && q.Name == "open.test.":
		soa := rr("open.test. 3600 IN SOA ns1.open.test. host.open.test. 1 7200 3600 1209600 300")
		tr := new(dns.Transfer)
		ch := make(chan *dns.Envelope)
		go func() {
			ch <- &dns.Envelope{RR: []dns.RR{soa, rr("open.test. 300 IN NS ns1.open.test."), rr("www.open.test. 300 IN A 192.0.2.1"), rr("secret.open.test. 300 IN A 192.0.2.99")}}
			ch <- &dns.Envelope{RR: []dns.RR{soa}}
			close(ch)
		}()
		_ = tr.Out(w, r, ch)
		return
	case q.Qtype == dns.TypeAXFR:
		m.Rcode = dns.RcodeRefused
	case q.Qtype == dns.TypeNS:
		m.Answer = []dns.RR{rr(q.Name + " 300 IN NS ns1." + q.Name)}
	case q.Qtype == dns.TypeA:
		m.Answer = []dns.RR{rr(q.Name + " 300 IN A 127.0.0.1")}
	}
	_ = w.WriteMsg(m)
}

func TestAXFR(t *testing.T) {
	addr := dnstest.Start(t, handler)
	dnsx.SystemOverride = &addr
	defer func() { dnsx.SystemOverride = nil }()
	env := tooltest.New(t, &Tool{Port: addr.Port()}, tooltest.Options{AllowPrivate: true})

	var res Result
	env.Get("/api/v1/dns/axfr?zone=open.test").Must(t).Decode(t, &res)
	if !res.AnyAllowed || len(res.Servers) != 1 || res.Servers[0].Records != 5 || res.Servers[0].Types["A"] != 2 || res.Servers[0].Sample[2].Name != "www.open.test." {
		t.Fatalf("open = %+v", res)
	}
	env.Get("/api/v1/dns/axfr?zone=closed.test").Must(t).Decode(t, &res)
	if res.AnyAllowed || res.Servers[0].Allowed || res.Servers[0].Error == "" {
		t.Errorf("closed = %+v", res.Servers)
	}

	// Public profile: the name server (127.0.0.1) is not contacted.
	strict := tooltest.New(t, &Tool{Port: addr.Port()}, tooltest.Options{})
	strict.Get("/api/v1/dns/axfr?zone=open.test").Must(t).Decode(t, &res)
	if res.AnyAllowed || res.Servers[0].Records != 0 || res.Servers[0].Error == "" {
		t.Errorf("policy = %+v", res.Servers)
	}
	env.Get("/api/v1/dns/axfr?zone=bad%20zone").ExpectError(t, "INVALID_INPUT")
}
