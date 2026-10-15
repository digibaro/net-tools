package dnsconsistency

import (
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func u32(v uint32) *uint32 { return &v }

func TestSummarize(t *testing.T) {
	a := Server{Serial: u32(5), NSSet: []string{"a.", "b."}, Authoritative: true}
	b := Server{Serial: u32(5), NSSet: []string{"a.", "b."}, Authoritative: true}
	c := Server{Serial: u32(6), NSSet: []string{"a."}, Authoritative: false}
	broken := Server{Error: "no answer"}
	s, n, au := summarize([]Server{a, b, broken})
	if !s || !n || !au {
		t.Errorf("consistent set: %v %v %v", s, n, au)
	}
	s, n, au = summarize([]Server{a, c})
	if s || n || au {
		t.Errorf("inconsistent set: %v %v %v", s, n, au)
	}
	if s, _, _ := summarize([]Server{broken}); s {
		t.Error("no answers must not be reported consistent")
	}
}

// One fake server plays both the recursive system resolver and the authoritative servers.
func handler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	q := r.Question[0]
	add := func(s string) {
		rr, _ := dns.NewRR(s)
		m.Answer = append(m.Answer, rr)
	}
	switch {
	case q.Qtype == dns.TypeNS && q.Name == "example.test.":
		m.Authoritative = !r.RecursionDesired
		add("example.test. 300 IN NS ns1.example.test.")
		add("example.test. 300 IN NS ns2.example.test.")
	case q.Qtype == dns.TypeA && (q.Name == "ns1.example.test." || q.Name == "ns2.example.test."):
		add(q.Name + " 300 IN A 127.0.0.1")
	case q.Qtype == dns.TypeSOA:
		m.Authoritative = true
		add("example.test. 300 IN SOA ns1.example.test. hostmaster.example.test. 2026101301 7200 3600 1209600 300")
	}
	_ = w.WriteMsg(m)
}

func TestConsistency(t *testing.T) {
	addr := dnstest.Start(t, handler)
	dnsx.SystemOverride = &addr
	defer func() { dnsx.SystemOverride = nil }()

	env := tooltest.New(t, &Tool{Port: addr.Port()}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/consistency?zone=Example.Test").Must(t).Decode(t, &res)
	if len(res.Nameservers) != 2 || len(res.Servers) != 2 || !res.SerialsConsistent || !res.NSSetsConsistent || !res.AllAuthoritative {
		t.Fatalf("res = %+v", res)
	}
	if *res.Servers[0].Serial != 2026101301 || res.Servers[0].NSSet[1] != "ns2.example.test." {
		t.Errorf("server = %+v", res.Servers[0])
	}

	// Public profile: the authoritative addresses (127.0.0.1) are refused, not queried.
	strict := tooltest.New(t, &Tool{Port: addr.Port()}, tooltest.Options{})
	strict.Get("/api/v1/dns/consistency?zone=example.test").Must(t).Decode(t, &res)
	if res.Servers[0].Serial != nil || res.Servers[0].Error == "" || res.SerialsConsistent {
		t.Errorf("policy not applied: %+v", res.Servers[0])
	}
}

func TestValidation(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{})
	env.Get("/api/v1/dns/consistency").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/consistency?zone=bad%20zone").ExpectError(t, "INVALID_INPUT")
}
