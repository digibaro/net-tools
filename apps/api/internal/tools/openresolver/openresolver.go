// Package openresolver implements DNS-15: does a server answer recursive queries for anyone?
package openresolver

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// Tool is the open resolver test.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.openresolver" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/dns/open-resolver", t.handle)
}

// Amplification measures how much larger a DNSSEC answer is than the question.
type Amplification struct {
	Query         string  `json:"query"`
	RequestBytes  int     `json:"requestBytes"`
	ResponseBytes int     `json:"responseBytes"`
	Factor        float64 `json:"factor"`
	Truncated     bool    `json:"truncated"`
}

// Result is the response payload.
type Result struct {
	Server             string         `json:"server"`
	Responds           bool           `json:"responds"`
	Rcode              string         `json:"rcode"`
	RecursionAvailable bool           `json:"recursionAvailable"`
	Answers            int            `json:"answers"`
	Open               bool           `json:"open"`
	RTTMs              float64        `json:"rttMs"`
	Amplification      *Amplification `json:"amplification"`
}

// TestName is resolved recursively; an open resolver returns its A records.
const TestName = "example.com"

// rawUDP exchanges over UDP without falling back to TCP, so sizes are what an attacker gets.
func rawUDP(ctx context.Context, s dnsx.Server, m *dns.Msg) (*dns.Msg, time.Duration, error) {
	c := &dns.Client{Net: "udp", Timeout: 3 * time.Second, UDPSize: 4096}
	r, rtt, err := c.ExchangeContext(ctx, m, s.Addr.String())
	return r, rtt, err //nolint:wrapcheck // timeouts mean "does not respond"
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	server := q.String("server", "", true, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, server, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := Result{Server: srv.Label}
	m := dnsx.NewQuery(TestName, dns.TypeA, false)
	resp, rtt, err := rawUDP(r.Context(), srv, m)
	if err == nil {
		res.Responds = true
		res.RTTMs = dnslookup.Ms(rtt)
		res.Rcode = dnsx.RcodeName(resp.Rcode)
		res.RecursionAvailable = resp.RecursionAvailable
		res.Answers = len(resp.Answer)
		res.Open = resp.Rcode == dns.RcodeSuccess && resp.RecursionAvailable && len(resp.Answer) > 0
	}
	if res.Open {
		am := dnsx.NewQuery(".", dns.TypeDNSKEY, true)
		if a, _, err := rawUDP(r.Context(), srv, am); err == nil {
			req, resp := am.Len(), a.Len()
			res.Amplification = &Amplification{Query: ". DNSKEY (DNSSEC)", RequestBytes: req, ResponseBytes: resp, Factor: math.Round(float64(resp)/float64(req)*10) / 10, Truncated: a.Truncated}
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: server, Used: srv.Addr.String()}})
}
