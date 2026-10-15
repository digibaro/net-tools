package dnstrace

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/miekg/dns"

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

// Three loopback "servers" on the same port: root (127.0.0.1) → test. (127.0.0.2) → example.test. (127.0.0.3).
func startHierarchy(t *testing.T) uint16 {
	t.Helper()
	root := dnstest.StartOn(t, netip.MustParseAddrPort("127.0.0.1:0"), func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Ns = []dns.RR{rr("test. 172800 IN NS ns.nic.test.")}
		m.Extra = []dns.RR{rr("ns.nic.test. 172800 IN A 127.0.0.2")}
		_ = w.WriteMsg(m)
	})
	port := root.Port()
	dnstest.StartOn(t, netip.AddrPortFrom(netip.MustParseAddr("127.0.0.2"), port), func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Ns = []dns.RR{rr("example.test. 3600 IN NS ns1.example.test.")}
		m.Extra = []dns.RR{rr("ns1.example.test. 3600 IN A 127.0.0.3")}
		_ = w.WriteMsg(m)
	})
	dnstest.StartOn(t, netip.AddrPortFrom(netip.MustParseAddr("127.0.0.3"), port), func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		if r.Question[0].Name == "www.example.test." {
			m.Answer = []dns.RR{rr("www.example.test. 300 IN A 192.0.2.80")}
		} else {
			m.Rcode = dns.RcodeNameError
		}
		_ = w.WriteMsg(m)
	})
	return port
}

func TestTrace(t *testing.T) {
	port := startHierarchy(t)
	env := tooltest.New(t, &Tool{Roots: []string{"127.0.0.1"}, Port: port}, tooltest.Options{AllowPrivate: true})
	body := env.Get("/api/v1/dns/trace/stream?name=www.example.test&type=A").Raw.Body.String()
	if strings.Count(body, "event: step") != 3 {
		t.Fatalf("expected 3 steps: %s", body)
	}
	for _, want := range []string{`"zone":"."`, `"zone":"test."`, `"zone":"example.test."`, `"data":"192.0.2.80"`, `event: done`, `"answers":1`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
	body = env.Get("/api/v1/dns/trace/stream?name=missing.example.test").Raw.Body.String()
	if !strings.Contains(body, `"rcode":"NXDOMAIN"`) || !strings.Contains(body, "event: done") {
		t.Errorf("nxdomain: %s", body)
	}
}

// Glue pointing at private addresses must not be followed in the public profile.
func TestTracePolicy(t *testing.T) {
	port := startHierarchy(t)
	env := tooltest.New(t, &Tool{Roots: []string{"127.0.0.1"}, Port: port}, tooltest.Options{})
	body := env.Get("/api/v1/dns/trace/stream?name=www.example.test").Raw.Body.String()
	if !strings.Contains(body, "event: error") || !strings.Contains(body, "TARGET_PRIVATE_DENIED") {
		t.Errorf("policy not enforced: %s", body)
	}
}

func TestTraceValidationIsAnSSEError(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{})
	r := env.Get("/api/v1/dns/trace/stream?name=bad%20name")
	if r.Status != 200 || !strings.Contains(r.Raw.Body.String(), "event: error\ndata: {\"code\":\"INVALID_INPUT\"") {
		t.Errorf("status %d body %s", r.Status, r.Raw.Body.String())
	}
}
