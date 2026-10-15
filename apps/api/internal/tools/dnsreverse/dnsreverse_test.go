package dnsreverse

import (
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func handler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	q := r.Question[0]
	add := func(s string) {
		rr, _ := dns.NewRR(s)
		m.Answer = append(m.Answer, rr)
	}
	switch {
	case q.Qtype == dns.TypePTR && q.Name == "10.2.0.192.in-addr.arpa.":
		add(q.Name + " 300 IN PTR good.example.")
		add(q.Name + " 300 IN PTR liar.example.")
	case q.Qtype == dns.TypeA && q.Name == "good.example.":
		add("good.example. 300 IN A 192.0.2.10")
	case q.Qtype == dns.TypeA && q.Name == "liar.example.":
		add("liar.example. 300 IN A 198.51.100.99")
	default:
		m.Rcode = dns.RcodeNameError
	}
	_ = w.WriteMsg(m)
}

func TestReverse(t *testing.T) {
	addr := dnstest.Start(t, handler)
	// Private resolver allowed for the test server; the queried IP is public.
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/reverse?ip=192.0.2.10&resolver="+url.QueryEscape(addr.String())).Must(t).Decode(t, &res)
	if res.PTRName != "10.2.0.192.in-addr.arpa" || len(res.Names) != 2 {
		t.Fatalf("res = %+v", res)
	}
	if !res.Names[0].ForwardConfirmed || res.Names[1].ForwardConfirmed || res.Names[1].ForwardAddresses[0] != "198.51.100.99" {
		t.Errorf("names = %+v", res.Names)
	}
}

func TestReverseValidationAndPolicy(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{})
	for ip, code := range map[string]string{
		"":                "INVALID_INPUT",
		"not-an-ip":       "INVALID_INPUT",
		"fe80::1%25eth0":  "INVALID_INPUT",
		"10.0.0.1":        "TARGET_PRIVATE_DENIED",
		"192.168.1.1":     "TARGET_PRIVATE_DENIED",
		"::1":             "TARGET_PRIVATE_DENIED",
		"169.254.169.254": "TARGET_BLOCKED",
	} {
		env.Get("/api/v1/dns/reverse?ip="+url.QueryEscape(ip)).ExpectError(t, code)
	}
	env.Get("/api/v1/dns/reverse?ip=192.0.2.1&resolver=127.0.0.1").ExpectError(t, "TARGET_PRIVATE_DENIED")
}
