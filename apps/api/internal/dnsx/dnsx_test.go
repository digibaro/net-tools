package dnsx

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools/apps/api/internal/target"
)

func answer(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	q := r.Question[0]
	switch {
	case q.Name == "big.test." && w.LocalAddr().Network() == "udp":
		m.Truncated = true
	case q.Name == "big.test.":
		for i := range 40 {
			rr, _ := dns.NewRR("big.test. 60 IN TXT \"" + strings.Repeat("x", 200) + string(rune('a'+i%26)) + "\"")
			m.Answer = append(m.Answer, rr)
		}
	case q.Qtype == dns.TypeA:
		rr, _ := dns.NewRR(q.Name + " 300 IN A 192.0.2.10")
		m.Answer = append(m.Answer, rr)
	case q.Qtype == dns.TypeMX:
		rr, _ := dns.NewRR(q.Name + " 300 IN MX 10 mail." + q.Name)
		m.Answer = append(m.Answer, rr)
	default:
		m.Rcode = dns.RcodeNameError
	}
	_ = w.WriteMsg(m)
}

func TestExchangeUDPTCPAndTruncation(t *testing.T) {
	addr := dnstest.Start(t, answer)
	for _, tr := range []Transport{UDP, TCP} {
		r, _, err := Exchange(context.Background(), Server{Label: "test", Addr: addr, Transport: tr}, NewQuery("example.test", dns.TypeMX, false))
		if err != nil {
			t.Fatalf("%s: %v", tr, err)
		}
		recs := Records(r.Answer)
		if len(recs) != 1 || recs[0].Type != "MX" || recs[0].Data != "10 mail.example.test." || recs[0].TTL != 300 {
			t.Errorf("%s: %+v", tr, recs)
		}
	}
	r, _, err := Exchange(context.Background(), Server{Addr: addr, Transport: UDP}, NewQuery("big.test", dns.TypeTXT, false))
	if err != nil || len(r.Answer) != 40 || r.Truncated {
		t.Fatalf("truncated UDP answer was not retried over TCP: %v %v", err, r)
	}
	r, _, _ = Exchange(context.Background(), Server{Addr: addr, Transport: UDP}, NewQuery("nope.test", dns.TypeTXT, false))
	if RcodeName(r.Rcode) != "NXDOMAIN" {
		t.Errorf("rcode = %s", RcodeName(r.Rcode))
	}
}

func TestResolveServer(t *testing.T) {
	strict := &target.Resolver{Policy: target.Policy{}, Lookup: func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("9.9.9.9")}, nil
	}}
	ctx := context.Background()

	s, err := ResolveServer(ctx, strict, "cloudflare", DoT, 6)
	if err != nil || s.Addr.String() != "[2606:4700:4700::1111]:853" || s.TLSName != "one.one.one.one" {
		t.Errorf("preset: %+v %v", s, err)
	}
	s, err = ResolveServer(ctx, strict, "google", DoH, 0)
	if err != nil || s.DoHURL != "https://dns.google/dns-query" {
		t.Errorf("doh preset: %+v %v", s, err)
	}
	s, err = ResolveServer(ctx, strict, "8.8.4.4:5353", UDP, 0)
	if err != nil || s.Addr.String() != "8.8.4.4:5353" {
		t.Errorf("ip:port: %+v %v", s, err)
	}
	s, err = ResolveServer(ctx, strict, "dns.example", DoT, 0)
	if err != nil || s.Addr.String() != "9.9.9.9:853" || s.TLSName != "dns.example" {
		t.Errorf("hostname: %+v %v", s, err)
	}
	s, err = ResolveServer(ctx, strict, "https://doh.example", DoH, 0)
	if err != nil || s.DoHURL != "https://doh.example/dns-query" || s.Addr.String() != "9.9.9.9:443" {
		t.Errorf("doh url: %+v %v", s, err)
	}
	for _, bad := range []string{"10.0.0.53", "127.0.0.1", "169.254.169.254", "-x"} {
		if _, err := ResolveServer(ctx, strict, bad, UDP, 0); err == nil {
			t.Errorf("%q accepted as resolver", bad)
		}
	}
	if _, err := ResolveServer(ctx, strict, "http://doh.example", DoH, 0); err == nil {
		t.Error("plain http DoH accepted")
	}
}

func TestSystemServer(t *testing.T) {
	p := filepath.Join(t.TempDir(), "resolv.conf")
	if err := os.WriteFile(p, []byte("nameserver 127.0.0.11\noptions ndots:0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := ResolvConf
	ResolvConf = p
	defer func() { ResolvConf = old }()
	s, err := ResolveServer(context.Background(), &target.Resolver{}, "system", UDP, 0)
	if err != nil || s.Addr.String() != "127.0.0.11:53" {
		t.Errorf("system: %+v %v", s, err)
	}
	if _, err := SystemServer(DoT); err == nil {
		t.Error("system resolver must reject DoT")
	}
}

func TestParseType(t *testing.T) {
	if v, ok := ParseType("mx"); !ok || v != dns.TypeMX {
		t.Error("mx")
	}
	if _, ok := ParseType("BOGUS"); ok {
		t.Error("bogus accepted")
	}
}
