package dnsbl

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestReverseName(t *testing.T) {
	if got := ReverseName(netip.MustParseAddr("192.0.2.99"), "zen.test"); got != "99.2.0.192.zen.test." {
		t.Error(got)
	}
	if got := ReverseName(netip.MustParseAddr("2001:db8::1"), "zen.test"); !strings.HasPrefix(got, "1.0.0.0.") || !strings.HasSuffix(got, ".8.b.d.0.1.0.0.2.zen.test.") {
		t.Error(got)
	}
}

func TestDNSBL(t *testing.T) {
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		q := r.Question[0]
		m := new(dns.Msg)
		m.SetReply(r)
		add := func(s string) {
			rr, _ := dns.NewRR(s)
			m.Answer = append(m.Answer, rr)
		}
		switch {
		case q.Name == "4.3.2.93.listed.test." && q.Qtype == dns.TypeA:
			add(q.Name + " 300 IN A 127.0.0.2")
		case q.Name == "4.3.2.93.listed.test." && q.Qtype == dns.TypeTXT:
			add(q.Name + ` 300 IN TXT "Listed for spam, see https://example.test/lookup"`)
		case q.Name == "4.3.2.93.refusing.test.":
			add(q.Name + " 300 IN A 127.255.255.254")
		case q.Name == "bad.example.dbl.test." && q.Qtype == dns.TypeA:
			add(q.Name + " 300 IN A 127.0.1.2")
		case q.Name == "bad.example." && q.Qtype == dns.TypeA:
			add(q.Name + " 300 IN A 93.2.3.4")
		default:
			m.Rcode = dns.RcodeNameError
		}
		_ = w.WriteMsg(m)
	})
	dnsx.SystemOverride = &addr
	defer func() { dnsx.SystemOverride = nil }()
	lists := []List{{Zone: "listed.test", Name: "Listed", Kind: "ip"}, {Zone: "clean.test", Name: "Clean", Kind: "ip"}, {Zone: "refusing.test", Name: "Refusing", Kind: "ip"}, {Zone: "dbl.test", Name: "DBL", Kind: "domain"}}
	env := tooltest.New(t, &Tool{Lists: lists}, tooltest.Options{})

	summary := func(body string) Summary {
		i := strings.Index(body, "event: done\ndata: ")
		var d struct {
			Summary Summary `json:"summary"`
		}
		_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
		return d.Summary
	}
	body := env.Get("/api/v1/email/dnsbl/stream?target=93.2.3.4").Raw.Body.String()
	s := summary(body)
	if s.Checked != 3 || len(s.Listed) != 1 || s.Listed[0] != "Listed" || s.Errors != 1 || !strings.Contains(body, "Listed for spam") || !strings.Contains(body, "127.0.0.2") {
		t.Errorf("ip: %+v\n%s", s, body)
	}
	body = env.Get("/api/v1/email/dnsbl/stream?target=bad.example").Raw.Body.String()
	s = summary(body)
	if s.Checked != 4 || len(s.Listed) != 2 { // DBL + the domain's address on "Listed"
		t.Errorf("domain: %+v\n%s", s, body)
	}
	for q, want := range map[string]string{"target=10.0.0.1": "INVALID_INPUT", "target=bad%20name": "INVALID_INPUT", "": "INVALID_INPUT"} {
		if b := env.Get("/api/v1/email/dnsbl/stream?" + q).Raw.Body.String(); !strings.Contains(b, "event: error") || !strings.Contains(b, want) {
			t.Errorf("%s: %s", q, b)
		}
	}
}
