// Package dnsconsistency implements DNS-09: compare SOA serials and NS sets across every
// authoritative server of a zone.
package dnsconsistency

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Limits on fan-out.
const (
	MaxNameservers = 13
	MaxServers     = 26
	Parallel       = 8
)

// Tool is SOA/NS consistency.
type Tool struct {
	deps tools.Deps
	// Port overrides 53 (tests).
	Port uint16
}

func (*Tool) ID() string                                { return "dns.consistency" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(25*time.Second)).Get("/dns/consistency", t.handle)
}

// Server is one authoritative server address.
type Server struct {
	Name          string   `json:"name"`
	Address       string   `json:"address"`
	Serial        *uint32  `json:"serial"`
	Mname         string   `json:"mname"`
	Rname         string   `json:"rname"`
	Authoritative bool     `json:"authoritative"`
	NSSet         []string `json:"nsSet"`
	RTTMs         float64  `json:"rttMs"`
	Error         string   `json:"error"`
}

// Result is the response payload.
type Result struct {
	Zone              string   `json:"zone"`
	Nameservers       []string `json:"nameservers"`
	Servers           []Server `json:"servers"`
	SerialsConsistent bool     `json:"serialsConsistent"`
	NSSetsConsistent  bool     `json:"nsSetsConsistent"`
	AllAuthoritative  bool     `json:"allAuthoritative"`
}

func nonRecursive(name string, qtype uint16) *dns.Msg {
	m := dnsx.NewQuery(name, qtype, false)
	m.RecursionDesired = false
	return m
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
	names, servers, err := dnsx.ZoneServers(ctx, sys, zone, MaxNameservers, MaxServers)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	res := Result{Zone: zone, Nameservers: names, Servers: []Server{}}
	if len(res.Nameservers) == 0 {
		httpapi.Error(w, r, httpapi.InvalidInput("zone", "no NS records found; is this the zone apex?"))
		return
	}
	for _, zs := range servers {
		s := Server{Name: zs.Name, NSSet: []string{}, Error: zs.Error}
		if zs.Addr.IsValid() {
			s.Address = zs.Addr.String()
		}
		res.Servers = append(res.Servers, s)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, Parallel)
	for i := range res.Servers {
		s := &res.Servers[i]
		if s.Address == "" {
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			t.probe(r, zone, s)
		})
	}
	wg.Wait()

	res.SerialsConsistent, res.NSSetsConsistent, res.AllAuthoritative = summarize(res.Servers)
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}

func (t *Tool) probe(r *http.Request, zone string, s *Server) {
	addr := netip.MustParseAddr(s.Address)
	// Authoritative servers are dial targets: the policy applies to each address.
	if err := t.deps.Target.Policy.Check(addr); err != nil {
		s.Error = "not queried: " + err.Error()
		return
	}
	port := t.Port
	if port == 0 {
		port = 53
	}
	srv := dnsx.Server{Label: s.Name, Addr: netip.AddrPortFrom(addr, port), Transport: dnsx.UDP}
	soa, rtt, err := dnsx.Exchange(r.Context(), srv, nonRecursive(zone, dns.TypeSOA))
	if err != nil {
		s.Error = "no answer"
		return
	}
	s.RTTMs = dnslookup.Ms(rtt)
	s.Authoritative = soa.Authoritative
	for _, rr := range soa.Answer {
		if v, ok := rr.(*dns.SOA); ok {
			serial := v.Serial
			s.Serial, s.Mname, s.Rname = &serial, v.Ns, v.Mbox
		}
	}
	if s.Serial == nil {
		s.Error = "no SOA in answer (" + dnsx.RcodeName(soa.Rcode) + ")"
	}
	if nsMsg, _, err := dnsx.Exchange(r.Context(), srv, nonRecursive(zone, dns.TypeNS)); err == nil {
		for _, rr := range nsMsg.Answer {
			if ns, ok := rr.(*dns.NS); ok {
				s.NSSet = append(s.NSSet, strings.ToLower(ns.Ns))
			}
		}
		slices.Sort(s.NSSet)
	}
}

// summarize reports whether answering servers agree. Servers with errors are ignored.
func summarize(servers []Server) (serials, nsSets, authoritative bool) {
	serials, nsSets, authoritative = true, true, true
	var firstSerial *uint32
	var firstSet []string
	answered := 0
	for _, s := range servers {
		if s.Serial == nil {
			continue
		}
		answered++
		if !s.Authoritative {
			authoritative = false
		}
		if firstSerial == nil {
			firstSerial, firstSet = s.Serial, s.NSSet
			continue
		}
		if *s.Serial != *firstSerial {
			serials = false
		}
		if !slices.Equal(s.NSSet, firstSet) {
			nsSets = false
		}
	}
	if answered == 0 {
		return false, false, false
	}
	return serials, nsSets, authoritative
}
