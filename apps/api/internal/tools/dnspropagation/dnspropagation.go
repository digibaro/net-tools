// Package dnspropagation implements DNS-03: the same query against many public resolvers.
package dnspropagation

import (
	"context"
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
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// PublicResolver is a well-known open resolver.
type PublicResolver struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Location string `json:"location"`
	Addr     netip.Addr
}

func pr(provider, location, ip string) PublicResolver {
	return PublicResolver{Name: provider + " (" + ip + ")", Provider: provider, Location: location, Addr: netip.MustParseAddr(ip)}
}

// Resolvers are fixed public addresses (never user input).
var Resolvers = []PublicResolver{
	pr("Cloudflare", "Anycast", "1.1.1.1"),
	pr("Cloudflare", "Anycast", "1.0.0.1"),
	pr("Google", "Anycast", "8.8.8.8"),
	pr("Google", "Anycast", "8.8.4.4"),
	pr("Quad9", "Anycast (CH)", "9.9.9.9"),
	pr("Quad9", "Anycast (CH)", "149.112.112.112"),
	pr("OpenDNS", "Anycast (US)", "208.67.222.222"),
	pr("OpenDNS", "Anycast (US)", "208.67.220.220"),
	pr("AdGuard", "Anycast (CY)", "94.140.14.14"),
	pr("Control D", "Anycast (CA)", "76.76.2.0"),
	pr("CleanBrowsing", "Anycast (US)", "185.228.168.9"),
	pr("Hurricane Electric", "Anycast (US)", "74.82.42.42"),
	pr("Level 3 / Lumen", "US", "4.2.2.1"),
	pr("DNS.WATCH", "DE", "84.200.69.80"),
	pr("Mullvad", "Anycast (SE)", "194.242.2.2"),
	pr("Yandex", "RU", "77.88.8.8"),
	pr("AliDNS", "CN", "223.5.5.5"),
	pr("Quad101 (TWNIC)", "TW", "101.101.101.101"),
}

// Tool is the propagation checker.
type Tool struct {
	deps tools.Deps
	// List overrides Resolvers (tests).
	List []PublicResolver
	// Port overrides 53 (tests).
	Port uint16
}

func (*Tool) ID() string                                { return "dns.propagation" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/dns/propagation", t.handle)
}

// Row is one resolver's answer.
type Row struct {
	Resolver string   `json:"resolver"`
	Provider string   `json:"provider"`
	Location string   `json:"location"`
	Address  string   `json:"address"`
	Rcode    string   `json:"rcode"`
	Answers  []string `json:"answers"`
	TTL      *uint32  `json:"ttl"`
	RTTMs    float64  `json:"rttMs"`
	Error    string   `json:"error"`
	// Group indexes Result.Groups (-1 on error).
	Group int `json:"group"`
}

// Group is a distinct answer set and how many resolvers returned it.
type Group struct {
	Answers []string `json:"answers"`
	Rcode   string   `json:"rcode"`
	Count   int      `json:"count"`
}

// Result is the response payload.
type Result struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Rows      []Row   `json:"rows"`
	Groups    []Group `json:"groups"`
	Consensus bool    `json:"consensus"`
}

// PropagationTypes are the supported record types.
var PropagationTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA", "CAA", "SRV", "PTR"}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("name", "", true, 253)
	typ := q.Enum("type", "A", PropagationTypes...)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	qtype, _ := dnsx.ParseType(typ)
	name, err := dnslookup.ParseName(raw, qtype)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	list := t.List
	if list == nil {
		list = Resolvers
	}
	port := t.Port
	if port == 0 {
		port = 53
	}
	ctx := r.Context()
	rows := make([]Row, len(list))
	var wg sync.WaitGroup
	for i, res := range list {
		wg.Go(func() {
			rows[i] = query(ctx, res, port, name, qtype)
		})
	}
	wg.Wait()
	httpapi.JSON(w, r, summarize(name, typ, rows), &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}

func query(ctx context.Context, res PublicResolver, port uint16, name string, qtype uint16) Row {
	row := Row{Resolver: res.Name, Provider: res.Provider, Location: res.Location, Address: res.Addr.String(), Answers: []string{}, Group: -1}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	srv := dnsx.Server{Label: res.Name, Addr: netip.AddrPortFrom(res.Addr, port), Transport: dnsx.UDP}
	msg, rtt, err := dnsx.Query(ctx, srv, name, qtype)
	if err != nil {
		row.Error = "no answer"
		return row
	}
	row.RTTMs = dnslookup.Ms(rtt)
	row.Rcode = dnsx.RcodeName(msg.Rcode)
	for _, rec := range dnsx.Records(msg.Answer) {
		if rec.Type == dns.TypeToString[qtype] {
			row.Answers = append(row.Answers, rec.Data)
			ttl := rec.TTL
			if row.TTL == nil || ttl < *row.TTL {
				row.TTL = &ttl
			}
		}
	}
	slices.Sort(row.Answers)
	return row
}

func summarize(name, typ string, rows []Row) Result {
	res := Result{Name: name, Type: typ, Rows: rows, Groups: []Group{}}
	keyOf := func(r Row) string { return r.Rcode + "|" + strings.Join(r.Answers, "\n") }
	index := map[string]int{}
	for i := range res.Rows {
		row := &res.Rows[i]
		if row.Error != "" {
			row.Group = -1
			continue
		}
		k := keyOf(*row)
		g, ok := index[k]
		if !ok {
			g = len(res.Groups)
			index[k] = g
			res.Groups = append(res.Groups, Group{Answers: row.Answers, Rcode: row.Rcode})
		}
		res.Groups[g].Count++
		row.Group = g
	}
	// Largest group first; renumber rows accordingly.
	order := make([]int, len(res.Groups))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return res.Groups[b].Count - res.Groups[a].Count })
	remap := make([]int, len(order))
	sorted := make([]Group, len(order))
	for newIdx, old := range order {
		remap[old] = newIdx
		sorted[newIdx] = res.Groups[old]
	}
	res.Groups = sorted
	for i := range res.Rows {
		if res.Rows[i].Group >= 0 {
			res.Rows[i].Group = remap[res.Rows[i].Group]
		}
	}
	res.Consensus = len(res.Groups) == 1
	return res
}
