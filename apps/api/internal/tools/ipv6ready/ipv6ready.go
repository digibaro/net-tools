// Package ipv6ready implements CON-14: is a domain reachable over IPv6 — web, DNS and mail?
package ipv6ready

import (
	"context"
	"net"
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

// Limits.
const (
	MaxHosts    = 8
	DialTimeout = 4 * time.Second
)

// Probe checks whether this server can reach addr over TCP (injectable for tests).
type Probe func(ctx context.Context, network string, addr netip.AddrPort) bool

func tcpProbe(ctx context.Context, network string, addr netip.AddrPort) bool {
	ctx, cancel := context.WithTimeout(ctx, DialTimeout)
	defer cancel()
	d := net.Dialer{}
	c, err := d.DialContext(ctx, network, addr.String())
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// Tool is net.ipv6ready.
type Tool struct {
	deps  tools.Deps
	Probe Probe
	// Canary is a well-known IPv6 address used to detect whether this server has IPv6 at all.
	Canary netip.AddrPort
}

func (*Tool) ID() string                                { return "net.ipv6ready" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Probe == nil {
		t.Probe = tcpProbe
	}
	if !t.Canary.IsValid() {
		t.Canary = netip.MustParseAddrPort("[2606:4700:4700::1111]:53")
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/ipv6/readiness", t.handle)
}

// Host is one name (web, NS or MX) with its IPv6 status.
type Host struct {
	Role      string   `json:"role"` // web, www, ns, mx
	Name      string   `json:"name"`
	AAAA      []string `json:"aaaa"`
	HasA      bool     `json:"hasA"`
	Reachable *bool    `json:"reachable"` // nil = not tested
	Port      int      `json:"port"`
	Error     string   `json:"error"`
}

// Result is the CON-14 payload.
type Result struct {
	Domain     string `json:"domain"`
	ServerIPv6 bool   `json:"serverIpv6"`
	Hosts      []Host `json:"hosts"`
	Score      int    `json:"score"` // 0..100
	Verdict    string `json:"verdict"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("domain", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	domain, err := target.NormalizeHostname(raw)
	if err != nil || !strings.Contains(domain, ".") {
		httpapi.Error(w, r, httpapi.InvalidInput("domain", "enter a domain name such as example.com"))
		return
	}
	sys, err := dnsx.SystemServer(dnsx.UDP)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	res := Result{Domain: domain, Hosts: []Host{}}
	res.ServerIPv6 = t.Probe(ctx, "tcp6", t.Canary)

	hosts := []Host{{Role: "web", Name: domain, Port: 443}}
	if !strings.HasPrefix(domain, "www.") {
		hosts = append(hosts, Host{Role: "www", Name: "www." + domain, Port: 443})
	}
	if m, _, err := dnsx.Query(ctx, sys, domain, dns.TypeNS); err == nil {
		for _, rr := range m.Answer {
			if ns, ok := rr.(*dns.NS); ok && len(hosts) < MaxHosts {
				hosts = append(hosts, Host{Role: "ns", Name: strings.TrimSuffix(ns.Ns, "."), Port: 53})
			}
		}
	} else {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	if m, _, err := dnsx.Query(ctx, sys, domain, dns.TypeMX); err == nil {
		for _, rr := range m.Answer {
			if mx, ok := rr.(*dns.MX); ok && mx.Mx != "." && len(hosts) < MaxHosts+4 {
				hosts = append(hosts, Host{Role: "mx", Name: strings.TrimSuffix(mx.Mx, "."), Port: 25})
			}
		}
	}

	var wg sync.WaitGroup
	for i := range hosts {
		h := &hosts[i]
		h.AAAA = []string{}
		wg.Go(func() { t.check(ctx, sys, res.ServerIPv6, h) })
	}
	wg.Wait()
	res.Hosts = hosts
	res.Score, res.Verdict = score(hosts)
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}

func (t *Tool) check(ctx context.Context, sys dnsx.Server, serverV6 bool, h *Host) {
	addrs, err := dnsx.Addresses(ctx, sys, h.Name)
	if err != nil {
		h.Error = "lookup failed"
		return
	}
	var first netip.Addr
	for _, a := range addrs {
		if a.Is4() || a.Is4In6() {
			h.HasA = true
			continue
		}
		h.AAAA = append(h.AAAA, a.String())
		if !first.IsValid() {
			first = a
		}
	}
	if !first.IsValid() || !serverV6 {
		return
	}
	// AAAA addresses come from DNS data: the target policy applies before connecting.
	if err := t.deps.Target.Policy.Check(first); err != nil {
		h.Error = "not tested: " + httpapi.ToAPIError(err).Message
		return
	}
	ok := t.Probe(ctx, "tcp6", netip.AddrPortFrom(first, uint16(h.Port))) //nolint:gosec // 25/53/443
	h.Reachable = &ok
}

// score weights web 50, DNS 25, mail 25 (mail counts as ready when there is no MX).
func score(hosts []Host) (int, string) {
	part := func(roles ...string) (has, total int) {
		for _, h := range hosts {
			if slices.Contains(roles, h.Role) {
				total++
				if len(h.AAAA) > 0 && (h.Reachable == nil || *h.Reachable) {
					has++
				}
			}
		}
		return has, total
	}
	web, webN := part("web", "www")
	ns, nsN := part("ns")
	mx, mxN := part("mx")
	s := 0
	if webN > 0 {
		s += 50 * web / webN
	}
	if nsN > 0 {
		s += 25 * ns / nsN
	}
	if mxN > 0 {
		s += 25 * mx / mxN
	} else {
		s += 25
	}
	switch {
	case s >= 100:
		return s, "ready"
	case web > 0:
		return s, "partial"
	default:
		return s, "not-ready"
	}
}
