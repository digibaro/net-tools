package ripestat

import (
	"cmp"
	"context"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// names looks up AS names in one RIPEstat call (missing names stay empty).
func (c Client) names(ctx context.Context, d tools.Deps, asns []uint32) map[uint32]string {
	out := map[uint32]string{}
	if len(asns) == 0 {
		return out
	}
	parts := make([]string, 0, len(asns))
	for _, a := range asns {
		parts = append(parts, "AS"+strconv.FormatUint(uint64(a), 10))
	}
	var data struct {
		Names map[string]string `json:"names"`
	}
	if c.Call(ctx, d, "as-names", url.Values{"resource": {strings.Join(parts, ",")}}, &data) == nil {
		for k, v := range data.Names {
			if n, err := strconv.ParseUint(k, 10, 32); err == nil {
				out[uint32(n)] = v
			}
		}
	}
	return out
}

// ---------- BGP-13 route visibility & history ----------

// HistoryTool is BGP-13: which origin ASNs announced a prefix over time and how visible it is now.
type HistoryTool struct {
	deps tools.Deps
	RS   Client
}

func (*HistoryTool) ID() string                                { return "bgp.history" }
func (*HistoryTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *HistoryTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	mount(r, d, "/bgp/route-history", t.handle)
}

// HistoryPrefix is one prefix announced by an origin.
type HistoryPrefix struct {
	Prefix    string  `json:"prefix"`
	FirstSeen string  `json:"firstSeen"`
	LastSeen  string  `json:"lastSeen"`
	Current   bool    `json:"current"`
	MaxPeers  float64 `json:"maxPeers"`
	Periods   int     `json:"periods"`
}

// HistoryOrigin groups the prefixes of one origin AS.
type HistoryOrigin struct {
	ASN      uint32          `json:"asn"`
	Name     string          `json:"name"`
	Current  bool            `json:"current"`
	Prefixes []HistoryPrefix `json:"prefixes"`
}

// Collector is the visibility at one RIS route collector.
type Collector struct {
	Name       string `json:"name"`
	City       string `json:"city"`
	Country    string `json:"country"`
	Peers      int    `json:"peers"`
	NotSeeing  int    `json:"notSeeing"`
	FullTables int    `json:"fullTables"`
}

// History is the BGP-13 payload.
type History struct {
	Resource   string          `json:"resource"`
	Family     int             `json:"family"`
	Origins    []HistoryOrigin `json:"origins"`
	Collectors []Collector     `json:"collectors"`
	// Seeing / FullTablePeers: RIS full-table peers that currently see the prefix.
	Seeing         int    `json:"seeing"`
	FullTablePeers int    `json:"fullTablePeers"`
	QueryStart     string `json:"queryStart"`
	QueryEnd       string `json:"queryEnd"`
	Related        int    `json:"related"`
}

func (t *HistoryTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := strings.TrimSpace(q.String("resource", "", true, 64))
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	family := 4
	if p, err := netip.ParsePrefix(raw); err == nil {
		raw = p.Masked().String()
		if p.Addr().Is6() {
			family = 6
		}
	} else if a, err := netip.ParseAddr(raw); err == nil {
		if a.Is6() {
			family = 6
		}
	} else {
		httpapi.Error(w, r, httpapi.InvalidInput("resource", "enter a prefix such as 193.0.0.0/21 or an IP address"))
		return
	}
	ctx := r.Context()
	var (
		hist struct {
			ByOrigin []struct {
				Origin   FlexASN `json:"origin"`
				Prefixes []struct {
					Prefix    string `json:"prefix"`
					Timelines []struct {
						Start string  `json:"starttime"`
						End   string  `json:"endtime"`
						Peers float64 `json:"full_peers_seeing"`
					} `json:"timelines"`
				} `json:"prefixes"`
			} `json:"by_origin"`
			Start string `json:"query_starttime"`
			End   string `json:"query_endtime"`
		}
		vis struct {
			Visibilities []struct {
				Probe struct {
					Name    string `json:"name"`
					City    string `json:"city"`
					Country string `json:"country"`
					V4Peers int    `json:"ipv4_peer_count"`
					V6Peers int    `json:"ipv6_peer_count"`
				} `json:"probe"`
				V4Full      int   `json:"ipv4_full_table_peer_count"`
				V6Full      int   `json:"ipv6_full_table_peer_count"`
				V4NotSeeing []any `json:"ipv4_full_table_peers_not_seeing"`
				V6NotSeeing []any `json:"ipv6_full_table_peers_not_seeing"`
			} `json:"visibilities"`
			Related []string `json:"related_prefixes"`
		}
	)
	res := url.Values{"resource": {raw}}
	errs := make([]error, 2)
	var wg sync.WaitGroup
	wg.Go(func() { errs[0] = t.RS.Call(ctx, t.deps, "routing-history", res, &hist) })
	wg.Go(func() { errs[1] = t.RS.Call(ctx, t.deps, "visibility", res, &vis) })
	wg.Wait()
	if errs[0] != nil {
		httpapi.Error(w, r, errs[0])
		return
	}
	out := History{Resource: raw, Family: family, Origins: []HistoryOrigin{}, Collectors: []Collector{}, QueryStart: hist.Start, QueryEnd: hist.End, Related: len(vis.Related)}
	end := hist.End
	var asns []uint32
	for _, o := range hist.ByOrigin {
		ho := HistoryOrigin{ASN: uint32(o.Origin), Prefixes: []HistoryPrefix{}}
		for _, p := range o.Prefixes {
			if len(p.Timelines) == 0 {
				continue
			}
			hp := HistoryPrefix{Prefix: p.Prefix, FirstSeen: p.Timelines[0].Start, LastSeen: p.Timelines[len(p.Timelines)-1].End, Periods: len(p.Timelines)}
			for _, tl := range p.Timelines {
				hp.MaxPeers = max(hp.MaxPeers, tl.Peers)
			}
			hp.Current = currentAt(hp.LastSeen, end)
			ho.Current = ho.Current || hp.Current
			ho.Prefixes = append(ho.Prefixes, hp)
		}
		slices.SortFunc(ho.Prefixes, func(a, b HistoryPrefix) int { return cmp.Compare(b.LastSeen, a.LastSeen) })
		out.Origins = append(out.Origins, ho)
		asns = append(asns, ho.ASN)
	}
	slices.SortStableFunc(out.Origins, func(a, b HistoryOrigin) int {
		if a.Current != b.Current {
			if a.Current {
				return -1
			}
			return 1
		}
		return cmp.Compare(b.Prefixes[0].LastSeen, a.Prefixes[0].LastSeen)
	})
	names := t.RS.names(ctx, t.deps, asns)
	for i := range out.Origins {
		out.Origins[i].Name = names[out.Origins[i].ASN]
	}
	if errs[1] == nil {
		for _, v := range vis.Visibilities {
			c := Collector{Name: v.Probe.Name, City: v.Probe.City, Country: v.Probe.Country}
			if family == 6 {
				c.Peers, c.FullTables, c.NotSeeing = v.Probe.V6Peers, v.V6Full, len(v.V6NotSeeing)
			} else {
				c.Peers, c.FullTables, c.NotSeeing = v.Probe.V4Peers, v.V4Full, len(v.V4NotSeeing)
			}
			out.FullTablePeers += c.FullTables
			out.Seeing += c.FullTables - c.NotSeeing
			out.Collectors = append(out.Collectors, c)
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}

// currentAt: the last period ends within a day of the end of the query window.
func currentAt(last, end string) bool {
	l, err1 := time.Parse("2006-01-02T15:04:05", last)
	e, err2 := time.Parse("2006-01-02T15:04:05", end)
	return err1 == nil && err2 == nil && e.Sub(l) < 36*time.Hour
}

// ---------- BGP-14 neighbour graph ----------

// GraphTool is BGP-14: upstreams and downstreams of an ASN with names, for a graph.
type GraphTool struct {
	deps tools.Deps
	RS   Client
}

func (*GraphTool) ID() string                                { return "bgp.graph" }
func (*GraphTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *GraphTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	mount(r, d, "/bgp/neighbours/{asn}", t.handle)
}

// MaxGraphNeighbours bounds each side of the graph.
const MaxGraphNeighbours = 30

// GraphNode is one neighbour.
type GraphNode struct {
	ASN   uint32 `json:"asn"`
	Name  string `json:"name"`
	Power int    `json:"power"`
}

// Graph is the BGP-14 payload.
type Graph struct {
	ASN             uint32      `json:"asn"`
	Name            string      `json:"name"`
	Upstreams       []GraphNode `json:"upstreams"`
	Downstreams     []GraphNode `json:"downstreams"`
	Uncertain       []GraphNode `json:"uncertain"`
	UpstreamTotal   int         `json:"upstreamTotal"`
	DownstreamTotal int         `json:"downstreamTotal"`
	UncertainTotal  int         `json:"uncertainTotal"`
}

func (t *GraphTool) handle(w http.ResponseWriter, r *http.Request) {
	asn, err := ParseASN(chi.URLParam(r, "asn"), "asn")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	var nb struct {
		Neighbours []struct {
			ASN   FlexASN `json:"asn"`
			Type  string  `json:"type"`
			Power int     `json:"power"`
		} `json:"neighbours"`
	}
	ctx := r.Context()
	if err := t.RS.Call(ctx, t.deps, "asn-neighbours", url.Values{"resource": {"AS" + strconv.FormatUint(uint64(asn), 10)}}, &nb); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	slices.SortFunc(nb.Neighbours, func(a, b struct {
		ASN   FlexASN `json:"asn"`
		Type  string  `json:"type"`
		Power int     `json:"power"`
	}) int {
		return cmp.Compare(b.Power, a.Power)
	})
	out := Graph{ASN: asn, Upstreams: []GraphNode{}, Downstreams: []GraphNode{}, Uncertain: []GraphNode{}}
	want := []uint32{asn}
	for _, n := range nb.Neighbours {
		node := GraphNode{ASN: uint32(n.ASN), Power: n.Power}
		var list *[]GraphNode
		switch n.Type {
		case "left":
			out.UpstreamTotal++
			list = &out.Upstreams
		case "right":
			out.DownstreamTotal++
			list = &out.Downstreams
		default:
			out.UncertainTotal++
			list = &out.Uncertain
		}
		if len(*list) < MaxGraphNeighbours {
			*list = append(*list, node)
			want = append(want, node.ASN)
		}
	}
	names := t.RS.names(ctx, t.deps, want)
	out.Name = names[asn]
	for _, l := range []*[]GraphNode{&out.Upstreams, &out.Downstreams, &out.Uncertain} {
		for i := range *l {
			(*l)[i].Name = names[(*l)[i].ASN]
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}

// ---------- BGP-15 ROA coverage ----------

// ROATool is BGP-15: RPKI status of every prefix an ASN originates, its ROAs and suggestions.
type ROATool struct {
	deps tools.Deps
	RS   Client
}

func (*ROATool) ID() string                                { return "bgp.roacoverage" }
func (*ROATool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *ROATool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	mount(r, d, "/bgp/roa-coverage/{asn}", t.handle)
}

// MaxROAChecks bounds the per-prefix RPKI lookups for prefixes this ASN's ROAs do not cover.
const MaxROAChecks = 40

// ROAEntry is one ROA of the ASN.
type ROAEntry struct {
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	TA        string `json:"ta"`
	Used      bool   `json:"used"`
	// Loose: maxLength is longer than any announced prefix needs (RFC 9319 advises minimal ROAs).
	Loose bool `json:"loose"`
}

// ROAPrefix is one originated prefix with its status.
type ROAPrefix struct {
	Prefix string `json:"prefix"`
	// Status: valid, invalid, not-found, unchecked (beyond MaxROAChecks).
	Status string `json:"status"`
	// Reason for invalid: asn (covered by a ROA for another AS) or length (longer than maxLength).
	Reason string `json:"reason"`
	ROA    string `json:"roa"`
}

// ROACoverage is the BGP-15 payload.
type ROACoverage struct {
	ASN       uint32      `json:"asn"`
	Name      string      `json:"name"`
	Prefixes  []ROAPrefix `json:"prefixes"`
	ROAs      []ROAEntry  `json:"roas"`
	Valid     int         `json:"valid"`
	Invalid   int         `json:"invalid"`
	NotFound  int         `json:"notFound"`
	Unchecked int         `json:"unchecked"`
	Validator string      `json:"validator"`
	Generated string      `json:"generated"`
}

// Validate applies RFC 6811 for prefix p originated by asn against ROAs (all of the same AS).
// Returns valid, or "" when no ROA of this AS covers p (another AS may still make it invalid).
func Validate(p netip.Prefix, roas []ROAEntry) (status, reason, roa string) {
	covered := false
	for _, r := range roas {
		rp, err := netip.ParsePrefix(r.Prefix)
		if err != nil || rp.Addr().Is4() != p.Addr().Is4() || rp.Bits() > p.Bits() || !rp.Contains(p.Addr()) {
			continue
		}
		covered = true
		if p.Bits() <= r.MaxLength {
			return "valid", "", r.Prefix + "-" + strconv.Itoa(r.MaxLength)
		}
		roa = r.Prefix + "-" + strconv.Itoa(r.MaxLength)
	}
	if covered {
		return "invalid", "length", roa
	}
	return "", "", ""
}

func (t *ROATool) handle(w http.ResponseWriter, r *http.Request) {
	asn, err := ParseASN(chi.URLParam(r, "asn"), "asn")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := url.Values{"resource": {"AS" + strconv.FormatUint(uint64(asn), 10)}}
	var (
		pfx struct {
			Prefixes struct {
				V4 struct {
					Originating []string `json:"originating"`
				} `json:"v4"`
				V6 struct {
					Originating []string `json:"originating"`
				} `json:"v6"`
			} `json:"prefixes"`
		}
		roas struct {
			ROAs []struct {
				Prefix    string  `json:"prefix"`
				MaxLength int     `json:"maxLength"`
				TA        string  `json:"ta"`
				ASN       FlexASN `json:"asn"`
			} `json:"roas"`
			Metadata struct {
				Generated string `json:"generatedTime"`
			} `json:"metadata"`
			Validator string `json:"validator"`
		}
	)
	ctx := r.Context()
	errs := make([]error, 2)
	var wg sync.WaitGroup
	wg.Go(func() {
		q := url.Values{"resource": res["resource"], "list_prefixes": {"true"}, "types": {"o"}}
		errs[0] = t.RS.Call(ctx, t.deps, "ris-prefixes", q, &pfx)
	})
	wg.Go(func() { errs[1] = t.RS.Call(ctx, t.deps, "rpki-roas", res, &roas) })
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			httpapi.Error(w, r, e)
			return
		}
	}
	out := ROACoverage{ASN: asn, Prefixes: []ROAPrefix{}, ROAs: []ROAEntry{}, Validator: roas.Validator, Generated: roas.Metadata.Generated}
	for _, ro := range roas.ROAs {
		if uint32(ro.ASN) == asn {
			out.ROAs = append(out.ROAs, ROAEntry{Prefix: ro.Prefix, MaxLength: ro.MaxLength, TA: ro.TA})
		}
	}
	all := append(slices.Clone(pfx.Prefixes.V4.Originating), pfx.Prefixes.V6.Originating...)
	var pending []int
	for _, s := range all {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			continue
		}
		st, reason, roa := Validate(p, out.ROAs)
		out.Prefixes = append(out.Prefixes, ROAPrefix{Prefix: p.String(), Status: st, Reason: reason, ROA: roa})
		if st == "" {
			pending = append(pending, len(out.Prefixes)-1)
		}
	}
	// Prefixes without a ROA of this AS: ask RIPEstat (a ROA of another AS makes them invalid).
	sem := make(chan struct{}, 6)
	for n, i := range pending {
		if n >= MaxROAChecks {
			out.Prefixes[i].Status = "unchecked"
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			var v struct {
				Status string `json:"status"`
			}
			q := url.Values{"resource": res["resource"], "prefix": {out.Prefixes[i].Prefix}}
			if t.RS.Call(ctx, t.deps, "rpki-validation", q, &v) != nil {
				out.Prefixes[i].Status = "unchecked"
				return
			}
			switch strings.ToLower(v.Status) {
			case "valid":
				out.Prefixes[i].Status = "valid"
			case "invalid", "invalid_asn", "invalid_length":
				out.Prefixes[i].Status, out.Prefixes[i].Reason = "invalid", "asn"
			default:
				out.Prefixes[i].Status = "not-found"
			}
		})
	}
	wg.Wait()
	// ROA usage and loose maxLength.
	for i := range out.ROAs {
		rp, err := netip.ParsePrefix(out.ROAs[i].Prefix)
		if err != nil {
			continue
		}
		longest := -1
		for _, p := range out.Prefixes {
			if pp, err := netip.ParsePrefix(p.Prefix); err == nil && rp.Bits() <= pp.Bits() && rp.Contains(pp.Addr()) && pp.Addr().Is4() == rp.Addr().Is4() {
				out.ROAs[i].Used = true
				longest = max(longest, pp.Bits())
			}
		}
		out.ROAs[i].Loose = out.ROAs[i].Used && out.ROAs[i].MaxLength > longest
	}
	for _, p := range out.Prefixes {
		switch p.Status {
		case "valid":
			out.Valid++
		case "invalid":
			out.Invalid++
		case "not-found":
			out.NotFound++
		default:
			out.Unchecked++
		}
	}
	out.Name = t.RS.names(ctx, t.deps, []uint32{asn})[asn]
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}
