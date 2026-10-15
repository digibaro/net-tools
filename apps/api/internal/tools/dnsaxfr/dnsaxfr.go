// Package dnsaxfr implements DNS-08: does any of a zone's name servers allow a full zone
// transfer (AXFR) to anyone? An open AXFR leaks every record of the zone.
package dnsaxfr

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// Limits.
const (
	MaxNameservers = 13
	MaxServers     = 26
	MaxRecords     = 10000 // stop reading a transfer after this many records
	SampleRecords  = 100   // records returned per server
	PerServer      = 10 * time.Second
)

// Tool is dns.axfr.
type Tool struct {
	deps tools.Deps
	// Port overrides 53 (tests).
	Port uint16
}

func (*Tool) ID() string { return "dns.axfr" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	return tools.RequireFlag(d.Config.Features.AXFR)
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/dns/axfr", t.handle)
}

// Server is the transfer attempt against one name server address.
type Server struct {
	Name      string         `json:"name"`
	Address   string         `json:"address"`
	Allowed   bool           `json:"allowed"`
	Records   int            `json:"records"`
	Truncated bool           `json:"truncated"`
	Types     map[string]int `json:"types"`
	Sample    []dnsx.Record  `json:"sample"`
	Error     string         `json:"error"`
}

// Result is the DNS-08 payload.
type Result struct {
	Zone        string   `json:"zone"`
	Nameservers []string `json:"nameservers"`
	Servers     []Server `json:"servers"`
	AnyAllowed  bool     `json:"anyAllowed"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("zone", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	zone, err := target.NormalizeHostname(raw)
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("zone", "not a valid domain name"))
		return
	}
	sys, err := dnsx.SystemServer(dnsx.UDP)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	names, zs, err := dnsx.ZoneServers(ctx, sys, zone, MaxNameservers, MaxServers)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	if len(names) == 0 {
		httpapi.Error(w, r, httpapi.InvalidInput("zone", "no NS records found; is this the zone apex?"))
		return
	}
	res := Result{Zone: zone, Nameservers: names, Servers: make([]Server, len(zs))}
	port := t.Port
	if port == 0 {
		port = 53
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, s := range zs {
		out := &res.Servers[i]
		*out = Server{Name: s.Name, Types: map[string]int{}, Sample: []dnsx.Record{}, Error: s.Error}
		if !s.Addr.IsValid() {
			continue
		}
		out.Address = s.Addr.String()
		// Name server addresses come from DNS data: the target policy applies to each.
		if err := t.deps.Target.Policy.Check(s.Addr); err != nil {
			out.Error = "not contacted: " + httpapi.ToAPIError(err).Message
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			transfer(ctx, zone, netip.AddrPortFrom(s.Addr, port), out)
		})
	}
	wg.Wait()
	for _, s := range res.Servers {
		res.AnyAllowed = res.AnyAllowed || s.Allowed
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}

func transfer(ctx context.Context, zone string, addr netip.AddrPort, out *Server) {
	ctx, cancel := context.WithTimeout(ctx, PerServer)
	defer cancel()
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		out.Error = "TCP port 53 not reachable"
		return
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() }) // deadline or cap: abort the stream
	defer stop()
	_ = c.SetDeadline(time.Now().Add(PerServer))
	m := new(dns.Msg)
	m.SetAxfr(dns.Fqdn(zone))
	tr := &dns.Transfer{Conn: &dns.Conn{Conn: c}}
	ch, err := tr.In(m, addr.String())
	if err != nil {
		out.Error = "transfer request failed"
		return
	}
	for env := range ch {
		if env.Error != nil {
			if out.Records == 0 {
				out.Error = refusal(env.Error)
			}
			break
		}
		for _, rr := range env.RR {
			out.Records++
			out.Types[dns.TypeToString[rr.Header().Rrtype]]++
			if len(out.Sample) < SampleRecords {
				out.Sample = append(out.Sample, dnsx.Records([]dns.RR{rr})...)
			}
		}
		if out.Records >= MaxRecords {
			out.Truncated = true
			cancel()
			break
		}
	}
	for range ch { //nolint:revive // drain so the transfer goroutine exits
	}
	out.Allowed = out.Records > 0
}

func refusal(err error) string {
	if s := err.Error(); s != "" && len(s) < 120 {
		return "refused: " + s
	}
	return "refused"
}
