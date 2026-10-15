// Package dns64 implements IP-22: NAT64 / DNS64 detection by RFC 7050 prefix discovery.
package dns64

import (
	"net/http"
	"net/netip"
	"slices"
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

// DiscoveryName only has A records (192.0.0.170 and .171); an AAAA answer must be synthesized.
const DiscoveryName = "ipv4only.arpa"

var wellKnownV4 = []netip.Addr{netip.MustParseAddr("192.0.0.170"), netip.MustParseAddr("192.0.0.171")}

// WellKnownPrefix is the NAT64 well-known prefix (RFC 6052).
var WellKnownPrefix = netip.MustParsePrefix("64:ff9b::/96")

// Tool is the DNS64 check.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.dns64" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(15*time.Second)).Get("/dns/dns64", t.handle)
}

// Prefix is a discovered NAT64 prefix.
type Prefix struct {
	Prefix    string `json:"prefix"`
	Length    int    `json:"length"`
	WellKnown bool   `json:"wellKnown"`
}

// Result is the response payload.
type Result struct {
	Resolver    string   `json:"resolver"`
	DNS64       bool     `json:"dns64"`
	Rcode       string   `json:"rcode"`
	Synthesized []string `json:"synthesized"`
	Prefixes    []Prefix `json:"prefixes"`
	// Native lists A records of ipv4only.arpa as seen by this resolver (sanity check).
	IPv4 []string `json:"ipv4"`
}

// RFC 6052 §2.2: where the 32 IPv4 bits sit for each prefix length (bits 64..71 are skipped).
var layouts = map[int][]int{
	32: {4, 5, 6, 7},
	40: {5, 6, 7, 9},
	48: {6, 7, 9, 10},
	56: {7, 9, 10, 11},
	64: {9, 10, 11, 12},
	96: {12, 13, 14, 15},
}

// Lengths are tried longest first: a /96 also matches shorter layouts by accident less often.
var lengths = []int{96, 64, 56, 48, 40, 32}

// Extract returns the IPv4 address embedded in a for a NAT64 prefix of the given length.
func Extract(a netip.Addr, length int) netip.Addr {
	b := a.As16()
	idx := layouts[length]
	return netip.AddrFrom4([4]byte{b[idx[0]], b[idx[1]], b[idx[2]], b[idx[3]]})
}

// PrefixOf finds the NAT64 prefix of a synthesized ipv4only.arpa address (RFC 7050 §3).
func PrefixOf(a netip.Addr) (netip.Prefix, bool) {
	if !a.Is6() || a.Is4In6() {
		return netip.Prefix{}, false
	}
	for _, l := range lengths {
		if slices.Contains(wellKnownV4, Extract(a, l)) {
			p, _ := a.Prefix(l)
			return p, true
		}
	}
	return netip.Prefix{}, false
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	resolver := q.String("resolver", "system", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	start := time.Now()
	msg, _, err := dnsx.Query(r.Context(), srv, DiscoveryName, dns.TypeAAAA)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	res := Result{Resolver: srv.Label, Rcode: dnsx.RcodeName(msg.Rcode), Synthesized: []string{}, Prefixes: []Prefix{}, IPv4: []string{}}
	seen := map[netip.Prefix]bool{}
	for _, rr := range msg.Answer {
		aaaa, ok := rr.(*dns.AAAA)
		if !ok {
			continue
		}
		a, ok := netip.AddrFromSlice(aaaa.AAAA)
		if !ok {
			continue
		}
		res.Synthesized = append(res.Synthesized, a.String())
		if p, ok := PrefixOf(a); ok && !seen[p] {
			seen[p] = true
			res.Prefixes = append(res.Prefixes, Prefix{Prefix: p.String(), Length: p.Bits(), WellKnown: p == WellKnownPrefix})
		}
	}
	res.DNS64 = len(res.Prefixes) > 0
	if m4, _, err := dnsx.Query(r.Context(), srv, DiscoveryName, dns.TypeA); err == nil {
		for _, rr := range m4.Answer {
			if a, ok := rr.(*dns.A); ok {
				res.IPv4 = append(res.IPv4, a.A.String())
			}
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", DurationMs: dnslookup.Ms(time.Since(start)), Target: &httpapi.TargetMeta{Input: resolver, Used: srv.Addr.String()}})
}
