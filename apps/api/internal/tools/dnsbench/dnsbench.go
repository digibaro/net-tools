// Package dnsbench implements DNS-07: latency and success rate of several resolvers over the
// same set of queries, streamed as each resolver finishes.
package dnsbench

import (
	"context"
	"net/http"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnspropagation"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// Limits.
const (
	MaxNames    = 10
	MaxRounds   = 3
	MaxCustom   = 3
	Parallel    = 6
	PerQuery    = 2 * time.Second
	MaxDuration = 90 * time.Second
)

// DefaultNames are popular domains, so warm-cache behaviour is realistic.
var DefaultNames = []string{"google.com", "youtube.com", "facebook.com", "wikipedia.org", "amazon.com", "microsoft.com", "apple.com", "netflix.com", "cloudflare.com", "github.com"}

// Tool is dns.benchmark.
type Tool struct {
	deps tools.Deps
	// List overrides the built-in resolver sets (tests).
	List []dnsx.Server
}

func (*Tool) ID() string                                { return "dns.benchmark" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(MaxDuration+10*time.Second)).Get("/dns/benchmark/stream", t.handle)
}

// ResolverResult is the "resolver" event: one resolver's measurements.
type ResolverResult struct {
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Queries  int      `json:"queries"`
	Answered int      `json:"answered"`
	Failed   int      `json:"failed"`
	MinMs    float64  `json:"minMs"`
	AvgMs    float64  `json:"avgMs"`
	MedianMs float64  `json:"medianMs"`
	P95Ms    float64  `json:"p95Ms"`
	MaxMs    float64  `json:"maxMs"`
	FirstMs  float64  `json:"firstMs"`
	Errors   []string `json:"errors"`
}

// Summary is the done payload: resolvers ranked by median latency (failures last).
type Summary struct {
	Ranking []ResolverResult `json:"ranking"`
	Names   []string         `json:"names"`
	Rounds  int              `json:"rounds"`
}

func (t *Tool) servers(set string) ([]dnsx.Server, error) {
	if t.List != nil {
		return t.List, nil
	}
	var out []dnsx.Server
	if sys, err := dnsx.SystemServer(dnsx.UDP); err == nil {
		out = append(out, sys)
	}
	if set == "quick" {
		for _, p := range dnsx.Presets {
			out = append(out, dnsx.Server{Label: p.Label + " (" + p.V4.String() + ")", Addr: netipPort(p.V4.String()), Transport: dnsx.UDP})
		}
		return out, nil
	}
	for _, r := range dnspropagation.Resolvers {
		out = append(out, dnsx.Server{Label: r.Name, Addr: netipPort(r.Addr.String()), Transport: dnsx.UDP})
	}
	return out, nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		set := q.Enum("set", "quick", "quick", "full")
		customRaw := q.String("custom", "", false, 600)
		namesRaw := q.String("names", "", false, 2000)
		rounds := q.Int("rounds", 2, 1, MaxRounds)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		names, err := parseNames(namesRaw)
		if err != nil {
			return nil, err
		}
		servers, err := t.servers(set)
		if err != nil {
			return nil, err
		}
		for i, spec := range splitList(customRaw) {
			if i >= MaxCustom {
				return nil, httpapi.InvalidInput("custom", "at most 3 custom resolvers")
			}
			// Custom resolvers are user targets: ResolveServer applies the target policy.
			s, err := dnsx.ResolveServer(ctx, t.deps.Target, spec, dnsx.UDP, 0)
			if err != nil {
				return nil, err //nolint:wrapcheck // client-facing target error
			}
			s.Label = spec
			servers = append(servers, s)
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "resolvers": len(servers), "names": names, "rounds": rounds}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return t.run(ctx, servers, names, rounds, emit), nil
		}}, nil
	})
}

func (t *Tool) run(ctx context.Context, servers []dnsx.Server, names []string, rounds int, emit stream.Emit) Summary {
	ctx, cancel := context.WithTimeout(ctx, MaxDuration)
	defer cancel()
	results := make([]ResolverResult, len(servers))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, Parallel)
	for i, s := range servers {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			res := measure(ctx, s, names, rounds)
			results[i] = res
			mu.Lock()
			emit("resolver", res)
			mu.Unlock()
		})
	}
	wg.Wait()
	ranking := slices.Clone(results)
	sort.SliceStable(ranking, func(i, j int) bool {
		a, b := ranking[i], ranking[j]
		if (a.Answered == 0) != (b.Answered == 0) {
			return b.Answered == 0
		}
		if a.Failed != b.Failed && (a.Failed*10 > a.Queries || b.Failed*10 > b.Queries) {
			return a.Failed < b.Failed // more than 10 % failures outweighs latency
		}
		return a.MedianMs < b.MedianMs
	})
	return Summary{Ranking: ranking, Names: names, Rounds: rounds}
}

func measure(ctx context.Context, s dnsx.Server, names []string, rounds int) ResolverResult {
	res := ResolverResult{Name: s.Label, Address: s.Addr.String(), Errors: []string{}}
	var rtts []time.Duration
	seenErr := map[string]bool{}
	for round := range rounds {
		for _, name := range names {
			if ctx.Err() != nil {
				break
			}
			res.Queries++
			qctx, cancel := context.WithTimeout(ctx, PerQuery)
			m, rtt, err := dnsx.Exchange(qctx, s, dnsx.NewQuery(name, dns.TypeA, false))
			cancel()
			msg := ""
			switch {
			case err != nil:
				msg = "timeout or no answer"
			case m.Rcode != dns.RcodeSuccess:
				msg = dnsx.RcodeName(m.Rcode) + " for " + name
			}
			if msg != "" {
				res.Failed++
				if !seenErr[msg] && len(res.Errors) < 5 {
					seenErr[msg] = true
					res.Errors = append(res.Errors, msg)
				}
				continue
			}
			res.Answered++
			rtts = append(rtts, rtt)
			if round == 0 && len(rtts) == 1 {
				res.FirstMs = rttstats.Ms(rtt)
			}
		}
	}
	if len(rtts) > 0 {
		sum := rttstats.Summarize(len(rtts), rtts)
		res.MinMs, res.AvgMs, res.MaxMs = sum.MinMs, sum.AvgMs, sum.MaxMs
		sorted := slices.Clone(rtts)
		slices.Sort(sorted)
		res.MedianMs = rttstats.Ms(sorted[len(sorted)/2])
		res.P95Ms = rttstats.Ms(sorted[min(len(sorted)-1, len(sorted)*95/100)])
	}
	return res
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseNames(raw string) ([]string, error) {
	list := splitList(raw)
	if len(list) == 0 {
		return slices.Clone(DefaultNames), nil
	}
	if len(list) > MaxNames {
		return nil, httpapi.InvalidInput("names", "at most 10 names")
	}
	out := make([]string, 0, len(list))
	for _, n := range list {
		h, err := target.NormalizeHostname(n)
		if err != nil || !strings.Contains(h, ".") {
			return nil, httpapi.InvalidInput("names", "not a valid domain name: "+n)
		}
		out = append(out, h)
	}
	return out, nil
}

func netipPort(ip string) netip.AddrPort { return netip.AddrPortFrom(netip.MustParseAddr(ip), 53) }
