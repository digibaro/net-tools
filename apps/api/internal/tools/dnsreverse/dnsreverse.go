// Package dnsreverse implements DNS-02: PTR lookup with forward confirmation (FCrDNS).
package dnsreverse

import (
	"net/http"
	"net/netip"
	"strings"
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

// MaxNames bounds forward confirmation work.
const MaxNames = 10

// Tool is reverse DNS.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.reverse" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(15*time.Second)).Get("/dns/reverse", t.handle)
}

// Name is one PTR target with its forward-confirmation status.
type Name struct {
	Name             string   `json:"name"`
	ForwardAddresses []string `json:"forwardAddresses"`
	ForwardConfirmed bool     `json:"forwardConfirmed"`
}

// Result is the response payload.
type Result struct {
	IP       string  `json:"ip"`
	PTRName  string  `json:"ptrName"`
	Resolver string  `json:"resolver"`
	Rcode    string  `json:"rcode"`
	Names    []Name  `json:"names"`
	RTTMs    float64 `json:"rttMs"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("ip", "", true, 64)
	resolver := q.String("resolver", "system", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ip, err := netip.ParseAddr(strings.Trim(raw, "[]"))
	if err != nil || ip.Zone() != "" {
		httpapi.Error(w, r, httpapi.InvalidInput("ip", "enter an IPv4 or IPv6 address"))
		return
	}
	// Reverse names of private addresses can reveal internal hostnames through the server's
	// resolver, so the queried address must pass the same policy as a dial target.
	if err := t.deps.Target.Policy.Check(ip); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ptr, _ := dns.ReverseAddr(ip.String())
	start := time.Now()
	msg, rtt, err := dnsx.Query(r.Context(), srv, ptr, dns.TypePTR)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	res := Result{IP: ip.String(), PTRName: strings.TrimSuffix(ptr, "."), Resolver: srv.Label, Rcode: dnsx.RcodeName(msg.Rcode), Names: []Name{}, RTTMs: dnslookup.Ms(rtt)}
	for _, rr := range msg.Answer {
		p, ok := rr.(*dns.PTR)
		if !ok || len(res.Names) >= MaxNames {
			continue
		}
		n := Name{Name: p.Ptr, ForwardAddresses: []string{}}
		addrs, err := dnsx.Addresses(r.Context(), srv, p.Ptr)
		if err == nil {
			for _, a := range addrs {
				n.ForwardAddresses = append(n.ForwardAddresses, a.String())
				if a.Unmap() == ip.Unmap() {
					n.ForwardConfirmed = true
				}
			}
		}
		res.Names = append(res.Names, n)
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", DurationMs: dnslookup.Ms(time.Since(start)), Target: &httpapi.TargetMeta{Input: raw, Used: srv.Addr.String()}})
}
