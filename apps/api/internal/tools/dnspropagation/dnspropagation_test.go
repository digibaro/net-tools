package dnspropagation

import (
	"net/netip"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestResolverListIsPublic(t *testing.T) {
	seen := map[netip.Addr]bool{}
	for _, r := range Resolvers {
		if !r.Addr.IsGlobalUnicast() || r.Addr.IsPrivate() || seen[r.Addr] {
			t.Errorf("%s is not a unique public address", r.Addr)
		}
		seen[r.Addr] = true
	}
}

func TestSummarize(t *testing.T) {
	rows := []Row{
		{Rcode: "NOERROR", Answers: []string{"192.0.2.2"}},
		{Rcode: "NOERROR", Answers: []string{"192.0.2.1"}},
		{Rcode: "NOERROR", Answers: []string{"192.0.2.1"}},
		{Error: "no answer"},
	}
	res := summarize("x", "A", rows)
	if len(res.Groups) != 2 || res.Groups[0].Count != 2 || res.Groups[0].Answers[0] != "192.0.2.1" || res.Consensus {
		t.Fatalf("groups = %+v", res.Groups)
	}
	if res.Rows[0].Group != 1 || res.Rows[1].Group != 0 || res.Rows[3].Group != -1 {
		t.Errorf("row groups = %d %d %d", res.Rows[0].Group, res.Rows[1].Group, res.Rows[3].Group)
	}
}

func TestPropagation(t *testing.T) {
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		rr, _ := dns.NewRR(r.Question[0].Name + " 120 IN A 192.0.2.10")
		m.Answer = append(m.Answer, rr)
		_ = w.WriteMsg(m)
	})
	tool := &Tool{List: []PublicResolver{pr("Test", "here", "127.0.0.1"), pr("Test2", "here", "127.0.0.1")}, Port: addr.Port()}
	env := tooltest.New(t, tool, tooltest.Options{})
	var res Result
	env.Get("/api/v1/dns/propagation?name=example.com&type=A").Must(t).Decode(t, &res)
	if !res.Consensus || len(res.Rows) != 2 || res.Rows[0].Answers[0] != "192.0.2.10" || *res.Rows[0].TTL != 120 {
		t.Errorf("res = %+v", res)
	}
	env.Get("/api/v1/dns/propagation?name=example.com&type=DNSKEY").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/propagation?name=bad%20name").ExpectError(t, "INVALID_INPUT")
}
