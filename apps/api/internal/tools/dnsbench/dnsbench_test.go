package dnsbench

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestBenchmark(t *testing.T) {
	fast := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		rr, _ := dns.NewRR(r.Question[0].Name + " 60 IN A 192.0.2.1")
		m.Answer = []dns.RR{rr}
		_ = w.WriteMsg(m)
	})
	slow := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		time.Sleep(15 * time.Millisecond)
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	})
	broken := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
	})
	list := []dnsx.Server{
		{Label: "slow", Addr: slow, Transport: dnsx.UDP},
		{Label: "broken", Addr: broken, Transport: dnsx.UDP},
		{Label: "fast", Addr: fast, Transport: dnsx.UDP},
	}
	env := tooltest.New(t, &Tool{List: list}, tooltest.Options{AllowPrivate: true})
	body := env.Get("/api/v1/dns/benchmark/stream?names=a.test,b.test&rounds=2").Raw.Body.String()
	if strings.Count(body, "event: resolver") != 3 || !strings.Contains(body, "event: done") {
		t.Fatalf("body = %s", body)
	}
	done := body[strings.Index(body, "event: done\ndata: ")+len("event: done\ndata: "):]
	var d struct {
		Summary Summary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(done)), &d); err != nil {
		t.Fatal(err)
	}
	r := d.Summary.Ranking
	if r[0].Name != "fast" || r[1].Name != "slow" || r[2].Name != "broken" || r[0].Queries != 4 || r[0].Answered != 4 || r[2].Failed != 4 || r[2].Errors[0] != "SERVFAIL for a.test" || r[1].MedianMs < 10 {
		t.Errorf("ranking = %+v", r)
	}
}

func TestBenchmarkValidationAndPolicy(t *testing.T) {
	env := tooltest.New(t, &Tool{List: []dnsx.Server{}}, tooltest.Options{})
	for q, want := range map[string]string{
		"names=bad%20name":                       "INVALID_INPUT",
		"names=" + strings.Repeat("a.test,", 11): "INVALID_INPUT",
		"rounds=9":                               "INVALID_INPUT",
		"custom=10.0.0.53":                       "TARGET_PRIVATE_DENIED",
		"custom=169.254.169.254":                 "TARGET_BLOCKED",
		"custom=1.1.1.1,8.8.8.8,9.9.9.9,8.8.4.4": "INVALID_INPUT",
	} {
		body := env.Get("/api/v1/dns/benchmark/stream?" + q).Raw.Body.String()
		if !strings.Contains(body, "event: error") || !strings.Contains(body, want) {
			t.Errorf("%s: %s", q, body)
		}
	}
}
