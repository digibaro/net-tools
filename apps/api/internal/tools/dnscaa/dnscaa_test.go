package dnscaa

import (
	"net/url"
	"slices"
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
	switch q.Name {
	case "example.com.":
		add(`example.com. 300 IN CAA 0 issue "letsencrypt.org"`)
		add(`example.com. 300 IN CAA 0 issuewild ";"`)
		add(`example.com. 300 IN CAA 0 iodef "mailto:sec@example.com"`)
	case "critical.test.":
		add(`critical.test. 300 IN CAA 128 tbs "x"`)
	}
	_ = w.WriteMsg(m)
}

func TestClimb(t *testing.T) {
	if got := Climb("a.b.example.com"); !slices.Equal(got, []string{"a.b.example.com", "b.example.com", "example.com", "com"}) {
		t.Errorf("got %v", got)
	}
}

func TestCAA(t *testing.T) {
	addr := dnstest.Start(t, handler)
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	q := "&resolver=" + url.QueryEscape(addr.String())

	var res Result
	env.Get("/api/v1/dns/caa?name=*.www.Example.com"+q).Must(t).Decode(t, &res)
	if res.Name != "www.example.com" || res.RelevantName != "example.com" || len(res.Steps) != 2 {
		t.Fatalf("res = %+v", res)
	}
	if !slices.Equal(res.Issue, []string{"letsencrypt.org"}) || !slices.Equal(res.IssueWild, []string{";"}) || res.Iodef[0] != "mailto:sec@example.com" {
		t.Errorf("parsed = %+v", res)
	}

	env.Get("/api/v1/dns/caa?name=nothing.example.net"+q).Must(t).Decode(t, &res)
	if res.RelevantName != "" || len(res.Steps) != 3 {
		t.Errorf("no-CAA = %+v", res)
	}

	env.Get("/api/v1/dns/caa?name=critical.test"+q).Must(t).Decode(t, &res)
	if !slices.Equal(res.UnknownCritical, []string{"tbs"}) {
		t.Errorf("critical = %+v", res)
	}
}

func TestCAAValidation(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{})
	env.Get("/api/v1/dns/caa").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/caa?name=bad%20name").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/caa?name=example.com&resolver=192.168.0.1").ExpectError(t, "TARGET_PRIVATE_DENIED")
	env.Get("/api/v1/dns/caa?name=example.com&resolver=169.254.169.254").ExpectError(t, "TARGET_BLOCKED")
}
