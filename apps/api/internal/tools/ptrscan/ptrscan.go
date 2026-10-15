// Package ptrscan implements DNS-16: PTR records of every address in a small prefix, with
// forward-confirmed reverse DNS (FCrDNS).
package ptrscan

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// MaxAddresses bounds a scan: an IPv4 /24 or an IPv6 /120.
const MaxAddresses = 256

// Workers is the number of parallel PTR lookups.
const Workers = 16

// Tool is the reverse DNS coverage scan.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.ptrscan" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(45*time.Second)).Get("/dns/ptr-scan", t.handle)
}

// Row is one address.
type Row struct {
	IP        string   `json:"ip"`
	Rcode     string   `json:"rcode"`
	Names     []string `json:"names"`
	Confirmed bool     `json:"confirmed"`
	Error     string   `json:"error"`
}

// Result is the response payload.
type Result struct {
	Prefix    string `json:"prefix"`
	Resolver  string `json:"resolver"`
	Total     int    `json:"total"`
	WithPTR   int    `json:"withPtr"`
	Confirmed int    `json:"confirmed"`
	Rows      []Row  `json:"rows"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("prefix", "", true, 64)
	resolver := q.String("resolver", "system", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	p, err := netip.ParsePrefix(strings.TrimSpace(raw))
	if err != nil {
		if a, aerr := netip.ParseAddr(strings.TrimSpace(raw)); aerr == nil {
			p, err = a.Prefix(a.BitLen() - 8) // a bare address scans its /24 or /120
		}
	}
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("prefix", "enter a prefix such as 192.0.2.0/24 or 2001:db8::/120"))
		return
	}
	p = p.Masked()
	if p.Addr().BitLen()-p.Bits() > 8 {
		httpapi.Error(w, r, httpapi.InvalidInput("prefix", "at most 256 addresses: /24 or longer for IPv4, /120 or longer for IPv6"))
		return
	}
	// Reverse names of private ranges can reveal internal hostnames (same rule as reverse DNS).
	if err := t.deps.Target.Policy.Check(p.Addr()); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	var addrs []netip.Addr
	for a := p.Addr(); p.Contains(a) && len(addrs) < MaxAddresses; a = a.Next() {
		addrs = append(addrs, a)
	}
	rows := make([]Row, len(addrs))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range Workers {
		wg.Go(func() {
			for i := range jobs {
				rows[i] = t.lookup(r, srv, addrs[i])
			}
		})
	}
	for i := range addrs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	res := Result{Prefix: p.String(), Resolver: srv.Label, Total: len(rows), Rows: rows}
	for _, row := range rows {
		if len(row.Names) > 0 {
			res.WithPTR++
		}
		if row.Confirmed {
			res.Confirmed++
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

func (t *Tool) lookup(r *http.Request, srv dnsx.Server, a netip.Addr) Row {
	row := Row{IP: a.String(), Names: []string{}}
	arpa, _ := dns.ReverseAddr(a.String())
	msg, _, err := dnsx.Query(r.Context(), srv, arpa, dns.TypePTR)
	if err != nil {
		row.Error = "no answer from the resolver"
		return row
	}
	row.Rcode = dnsx.RcodeName(msg.Rcode)
	for _, rr := range msg.Answer {
		if ptr, ok := rr.(*dns.PTR); ok {
			row.Names = append(row.Names, strings.TrimSuffix(ptr.Ptr, "."))
		}
	}
	for _, n := range row.Names {
		if fwd, err := dnsx.Addresses(r.Context(), srv, n); err == nil && slices.Contains(fwd, a) {
			row.Confirmed = true
			break
		}
	}
	return row
}
