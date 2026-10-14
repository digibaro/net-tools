package dnslookup

import (
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func handler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.RecursionAvailable = true
	q := r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		rr, _ := dns.NewRR(q.Name + " 300 IN A 192.0.2.10")
		m.Answer = append(m.Answer, rr)
	case dns.TypePTR:
		rr, _ := dns.NewRR(q.Name + " 300 IN PTR host.example.")
		m.Answer = append(m.Answer, rr)
	default:
		m.Rcode = dns.RcodeNameError
		soa, _ := dns.NewRR("example. 300 IN SOA ns.example. host.example. 1 7200 3600 1209600 300")
		m.Ns = append(m.Ns, soa)
	}
	_ = w.WriteMsg(m)
}

func TestLookup(t *testing.T) {
	addr := dnstest.Start(t, handler)
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/lookup?name=WWW.Example.&type=a&resolver="+url.QueryEscape(addr.String())).Must(t).Decode(t, &res)
	if res.Name != "www.example" || res.Type != "A" || res.Rcode != "NOERROR" || !res.Flags.RecursionAvailable || len(res.Answer) != 1 || res.Answer[0].Data != "192.0.2.10" {
		t.Errorf("res = %+v", res)
	}

	env.Get("/api/v1/dns/lookup?name=missing.example&type=MX&transport=tcp&resolver="+url.QueryEscape(addr.String())).Must(t).Decode(t, &res)
	if res.Rcode != "NXDOMAIN" || len(res.Authority) != 1 || res.Authority[0].Type != "SOA" {
		t.Errorf("nxdomain = %+v", res)
	}

	env.Get("/api/v1/dns/lookup?name=192.0.2.10&type=PTR&resolver="+url.QueryEscape(addr.String())).Must(t).Decode(t, &res)
	if res.Name != "10.2.0.192.in-addr.arpa" || res.Answer[0].Data != "host.example." {
		t.Errorf("ptr = %+v", res)
	}
}

func TestValidation(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{})
	for path, code := range map[string]string{
		"/api/v1/dns/lookup":                                                "INVALID_INPUT",
		"/api/v1/dns/lookup?name=exa%20mple.com":                            "INVALID_INPUT",
		"/api/v1/dns/lookup?name=example.com&type=BOGUS":                    "INVALID_INPUT",
		"/api/v1/dns/lookup?name=example.com&transport=quic":                "INVALID_INPUT",
		"/api/v1/dns/lookup?name=example.com&family=5":                      "INVALID_INPUT",
		"/api/v1/dns/lookup?name=example.com&resolver=-x":                   "INVALID_INPUT",
		"/api/v1/dns/lookup?name=example.com&transport=dot&resolver=system": "INVALID_INPUT",
	} {
		env.Get(path).ExpectError(t, code)
	}
}

// SSRF: resolvers are targets and go through the policy.
func TestResolverSSRF(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Lookup: tooltest.StaticLookup("10.0.0.53")})
	cases := map[string]string{
		"127.0.0.1":            "TARGET_PRIVATE_DENIED",
		"10.0.0.53":            "TARGET_PRIVATE_DENIED",
		"[::1]:53":             "TARGET_PRIVATE_DENIED",
		"[fe80::1]":            "TARGET_PRIVATE_DENIED",
		"[::ffff:127.0.0.1]":   "TARGET_PRIVATE_DENIED",
		"169.254.169.254":      "TARGET_BLOCKED",
		"0.0.0.0":              "TARGET_BLOCKED",
		"2130706433":           "INVALID_INPUT",
		"0x7f.1":               "INVALID_INPUT",
		"internal-dns.example": "TARGET_PRIVATE_DENIED",
	}
	for resolver, code := range cases {
		env.Get("/api/v1/dns/lookup?name=example.com&resolver="+url.QueryEscape(resolver)).ExpectError(t, code)
	}
	env.Get("/api/v1/dns/lookup?name=example.com&transport=doh&resolver="+url.QueryEscape("https://internal-dns.example/dns-query")).ExpectError(t, "TARGET_PRIVATE_DENIED")
}

func TestUnreachableResolver(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	r := env.Get("/api/v1/dns/lookup?name=example.com&transport=tcp&resolver=127.0.0.1:1")
	if r.Error.Code != "PROVIDER_ERROR" && r.Error.Code != "TIMEOUT" {
		t.Fatalf("got %d %s", r.Status, r.Error.Code)
	}
}
