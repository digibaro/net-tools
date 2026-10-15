package ednscomp

import (
	"context"
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// compliant answers like a correct RFC 6891 server; broken ignores EDNS versions and echoes options.
func server(compliant bool) func(dns.ResponseWriter, *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		if o := r.IsEdns0(); o != nil {
			out := &dns.OPT{Hdr: dns.RR_Header{Name: ".", Rrtype: dns.TypeOPT}}
			out.SetUDPSize(1232)
			out.SetDo(o.Do())
			if compliant {
				if o.Version() != 0 {
					m.Rcode = dns.RcodeBadVers
					m.Extra = append(m.Extra, out)
					_ = w.WriteMsg(m)
					return
				}
				out.Option = append(out.Option, &dns.EDNS0_NSID{Code: dns.EDNS0NSID, Nsid: "6e73312e74657374"})
			} else {
				out.Hdr.Ttl = o.Hdr.Ttl // echoes flags (and the version)
				out.Option = o.Option
			}
			m.Extra = append(m.Extra, out)
		}
		rr, _ := dns.NewRR("example.test. 300 IN SOA ns1.example.test. hostmaster.example.test. 1 7200 3600 1209600 300")
		m.Answer = append(m.Answer, rr)
		_ = w.WriteMsg(m)
	}
}

func TestRunServer(t *testing.T) {
	addr := dnstest.Start(t, server(true))
	s := RunServer(context.Background(), "ns1", addr.String(), "example.test")
	for _, c := range s.Checks {
		if !c.Pass && c.ID != "tcp" { // dnstest listens on UDP only
			t.Errorf("compliant %s: %s", c.ID, c.Detail)
		}
	}
	if s.NSID != "ns1.test" {
		t.Errorf("nsid = %q", s.NSID)
	}
	b := RunServer(context.Background(), "ns2", dnstest.Start(t, server(false)).String(), "example.test")
	failed := map[string]bool{}
	for _, c := range b.Checks {
		failed[c.ID] = !c.Pass
	}
	for _, id := range []string{"edns1", "ednsopt", "edns1opt", "ednsflags"} {
		if !failed[id] {
			t.Errorf("broken server passed %s", id)
		}
	}
	if failed["dns"] || failed["edns"] || failed["do"] {
		t.Errorf("broken server failed a basic test: %+v", b.Checks)
	}
}

func TestHandler(t *testing.T) {
	addr := dnstest.Start(t, server(true))
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/edns-compliance?zone=example.test&server="+url.QueryEscape(addr.String())).Must(t).Decode(t, &res)
	if res.Zone != "example.test" || len(res.Servers) != 1 || res.Servers[0].Passed < 8 {
		t.Errorf("res = %+v", res)
	}
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/dns/edns-compliance?zone=example.test&server=10.1.1.1").Must(t).Decode(t, &res)
	if res.Servers[0].Error == "" {
		t.Errorf("private server not refused: %+v", res.Servers[0])
	}
	env.Get("/api/v1/dns/edns-compliance?zone=bad%20zone").ExpectError(t, "INVALID_INPUT")
}
