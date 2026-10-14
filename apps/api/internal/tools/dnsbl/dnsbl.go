// Package dnsbl implements EM-06: is an IP or domain listed on common DNS blocklists?
package dnsbl

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
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
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// List is one blocklist zone.
type List struct {
	Zone   string
	Name   string
	Kind   string // ip or domain
	IPv6   bool
	Delist string
}

// IPLists are widely used IP blocklists.
var IPLists = []List{
	{Zone: "zen.spamhaus.org", Name: "Spamhaus ZEN", Kind: "ip", IPv6: true, Delist: "https://check.spamhaus.org/"},
	{Zone: "bl.spamcop.net", Name: "SpamCop", Kind: "ip", Delist: "https://www.spamcop.net/bl.shtml"},
	{Zone: "b.barracudacentral.org", Name: "Barracuda", Kind: "ip", Delist: "https://www.barracudacentral.org/rbl/removal-request"},
	{Zone: "dnsbl.sorbs.net", Name: "SORBS", Kind: "ip"},
	{Zone: "psbl.surriel.com", Name: "PSBL", Kind: "ip", Delist: "https://psbl.org/"},
	{Zone: "bl.mailspike.net", Name: "Mailspike", Kind: "ip"},
	{Zone: "dnsbl-1.uceprotect.net", Name: "UCEPROTECT 1", Kind: "ip"},
	{Zone: "dnsbl-2.uceprotect.net", Name: "UCEPROTECT 2", Kind: "ip"},
	{Zone: "dnsbl-3.uceprotect.net", Name: "UCEPROTECT 3", Kind: "ip"},
	{Zone: "ix.dnsbl.manitu.net", Name: "NiX Spam", Kind: "ip"},
	{Zone: "truncate.gbudb.net", Name: "GBUdb Truncate", Kind: "ip"},
	{Zone: "dnsbl.dronebl.org", Name: "DroneBL", Kind: "ip"},
	{Zone: "db.wpbl.info", Name: "WPBL", Kind: "ip"},
	{Zone: "all.s5h.net", Name: "S5H", Kind: "ip"},
	{Zone: "spam.dnsbl.anonmails.de", Name: "Anonmails", Kind: "ip"},
	{Zone: "ubl.unsubscore.com", Name: "LashBack UBL", Kind: "ip"},
	{Zone: "dnsbl.spfbl.net", Name: "SPFBL", Kind: "ip"},
	{Zone: "rbl.interserver.net", Name: "InterServer", Kind: "ip"},
	{Zone: "bl.blocklist.de", Name: "blocklist.de", Kind: "ip"},
	{Zone: "dyna.spamrats.com", Name: "SpamRATS Dyna", Kind: "ip"},
	{Zone: "noptr.spamrats.com", Name: "SpamRATS NoPtr", Kind: "ip"},
	{Zone: "spam.spamrats.com", Name: "SpamRATS Spam", Kind: "ip"},
}

// DomainLists are domain (URI) blocklists.
var DomainLists = []List{
	{Zone: "dbl.spamhaus.org", Name: "Spamhaus DBL", Kind: "domain", Delist: "https://check.spamhaus.org/"},
	{Zone: "multi.surbl.org", Name: "SURBL", Kind: "domain"},
	{Zone: "multi.uribl.com", Name: "URIBL", Kind: "domain"},
}

// Parallel bounds concurrent queries.
const Parallel = 10

// Tool is email.dnsbl.
type Tool struct {
	deps tools.Deps
	// Lists overrides the built-in lists (tests).
	Lists []List
}

func (*Tool) ID() string                                { return "email.dnsbl" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Get("/email/dnsbl/stream", t.handle)
}

// Check is the "list" event.
type Check struct {
	Zone    string   `json:"zone"`
	Name    string   `json:"name"`
	Queried string   `json:"queried"`
	Status  string   `json:"status"` // clean, listed, error
	Codes   []string `json:"codes"`
	Reason  string   `json:"reason"`
	Delist  string   `json:"delist"`
	Error   string   `json:"error"`
}

// Summary is the done payload.
type Summary struct {
	Checked int      `json:"checked"`
	Listed  []string `json:"listed"`
	Errors  int      `json:"errors"`
}

// ReverseName is the DNSBL query name for an IP (nibbles for IPv6).
func ReverseName(ip netip.Addr, zone string) string {
	if ip.Is4() {
		b := ip.As4()
		return fmt.Sprintf("%d.%d.%d.%d.%s.", b[3], b[2], b[1], b[0], zone)
	}
	arpa, _ := dns.ReverseAddr(ip.String()) // "...ip6.arpa."
	return strings.TrimSuffix(arpa, "ip6.arpa.") + zone + "."
}

type job struct {
	list List
	name string
	what string
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("target", "", true, 253)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		sys, err := dnsx.SystemServer(dnsx.UDP)
		if err != nil {
			return nil, err //nolint:wrapcheck // surfaced as INTERNAL
		}
		ipLists, domLists := IPLists, DomainLists
		if t.Lists != nil {
			ipLists, domLists = nil, nil
			for _, l := range t.Lists {
				if l.Kind == "ip" {
					ipLists = append(ipLists, l)
				} else {
					domLists = append(domLists, l)
				}
			}
		}
		var jobs []job
		addIP := func(ip netip.Addr) error {
			ip = ip.Unmap()
			if (target.Policy{}).Check(ip) != nil {
				return httpapi.InvalidInput("target", "private and reserved addresses are not on public blocklists")
			}
			for _, l := range ipLists {
				if ip.Is4() || l.IPv6 {
					jobs = append(jobs, job{l, ReverseName(ip, l.Zone), ip.String()})
				}
			}
			return nil
		}
		s := strings.TrimSpace(raw)
		if ip, err := netip.ParseAddr(strings.Trim(s, "[]")); err == nil {
			if err := addIP(ip); err != nil {
				return nil, err
			}
		} else {
			domain, err := target.NormalizeHostname(s)
			if err != nil || !strings.Contains(domain, ".") {
				return nil, httpapi.InvalidInput("target", "enter an IP address or a domain name")
			}
			for _, l := range domLists {
				jobs = append(jobs, job{l, domain + "." + l.Zone + ".", domain})
			}
			// Also check the domain's first IPv4 address against the IP lists.
			if addrs, err := dnsx.Addresses(ctx, sys, domain); err == nil {
				for _, a := range addrs {
					if a.Is4() && (target.Policy{}).Check(a) == nil {
						_ = addIP(a)
						break
					}
				}
			}
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "target": s, "lists": len(jobs)}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return run(ctx, sys, jobs, emit), nil
		}}, nil
	})
}

func run(ctx context.Context, sys dnsx.Server, jobs []job, emit stream.Emit) Summary {
	sum := Summary{Listed: []string{}}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, Parallel)
	for _, j := range jobs {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			c := query(ctx, sys, j)
			mu.Lock()
			defer mu.Unlock()
			sum.Checked++
			switch c.Status {
			case "listed":
				sum.Listed = append(sum.Listed, c.Name)
			case "error":
				sum.Errors++
			}
			emit("list", c)
		})
	}
	wg.Wait()
	return sum
}

func query(ctx context.Context, sys dnsx.Server, j job) Check {
	c := Check{Zone: j.list.Zone, Name: j.list.Name, Queried: j.what, Status: "clean", Codes: []string{}, Delist: j.list.Delist}
	qctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	m, _, err := dnsx.Query(qctx, sys, j.name, dns.TypeA)
	if err != nil {
		c.Status, c.Error = "error", "no answer"
		return c
	}
	if m.Rcode == dns.RcodeNameError {
		return c
	}
	for _, rr := range m.Answer {
		a, ok := rr.(*dns.A)
		if !ok {
			continue
		}
		ip := a.A.String()
		// 127.255.255.x is Spamhaus refusing the query (public/open resolver or over quota).
		if strings.HasPrefix(ip, "127.255.255.") {
			c.Status, c.Error = "error", "query refused by the list (resolver blocked or rate limited)"
			return c
		}
		if strings.HasPrefix(ip, "127.") {
			c.Status = "listed"
			c.Codes = append(c.Codes, ip)
		}
	}
	if c.Status == "listed" {
		if t, _, err := dnsx.Query(qctx, sys, j.name, dns.TypeTXT); err == nil {
			for _, rr := range t.Answer {
				if txt, ok := rr.(*dns.TXT); ok {
					c.Reason = strings.Join(txt.Txt, "")
					break
				}
			}
		}
	}
	return c
}
