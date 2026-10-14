package ripestat

import (
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Limits on what we return (RIPEstat answers for large networks can be huge).
const (
	MaxPrefixes   = 1000
	MaxNeighbours = 200
	MaxLGPeers    = 300
)

func mount(r chi.Router, d tools.Deps, path string, h http.HandlerFunc) {
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(25*time.Second)).Get(path, h)
}

// ---------- BGP-02 ASN info ----------

// ASNTool is BGP-02.
type ASNTool struct {
	deps tools.Deps
	RS   Client
}

func (*ASNTool) ID() string                                { return "bgp.asn" }
func (*ASNTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *ASNTool) Mount(r chi.Router, d tools.Deps)        { t.deps = d; mount(r, d, "/asn/{asn}", t.handle) }

// Neighbour is an adjacent AS; Type is left (upstream), right (downstream) or uncertain.
type Neighbour struct {
	ASN   uint32 `json:"asn"`
	Type  string `json:"type"`
	Power int    `json:"power"`
}

// ASNInfo is the BGP-02 payload.
type ASNInfo struct {
	ASN             uint32      `json:"asn"`
	Holder          string      `json:"holder"`
	Announced       bool        `json:"announced"`
	Block           string      `json:"block"`
	PrefixesV4      []string    `json:"prefixesV4"`
	PrefixesV6      []string    `json:"prefixesV6"`
	PrefixesTotal   int         `json:"prefixesTotal"`
	Upstreams       int         `json:"upstreams"`
	Downstreams     int         `json:"downstreams"`
	Neighbours      []Neighbour `json:"neighbours"`
	NeighboursTotal int         `json:"neighboursTotal"`
}

func (t *ASNTool) handle(w http.ResponseWriter, r *http.Request) {
	asn, err := ParseASN(chi.URLParam(r, "asn"), "asn")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := url.Values{"resource": {"AS" + strconv.FormatUint(uint64(asn), 10)}}
	var (
		ov struct {
			Holder    string `json:"holder"`
			Announced bool   `json:"announced"`
			Block     struct {
				Desc string `json:"desc"`
				Name string `json:"name"`
			} `json:"block"`
		}
		pf struct {
			Prefixes []struct {
				Prefix string `json:"prefix"`
			} `json:"prefixes"`
		}
		nb struct {
			Counts struct {
				Left  int `json:"left"`
				Right int `json:"right"`
			} `json:"neighbour_counts"`
			Neighbours []struct {
				ASN   FlexASN `json:"asn"`
				Type  string  `json:"type"`
				Power int     `json:"power"`
			} `json:"neighbours"`
		}
	)
	ctx := r.Context()
	errs := make([]error, 3)
	var wg sync.WaitGroup
	wg.Go(func() { errs[0] = t.RS.Call(ctx, t.deps, "as-overview", res, &ov) })
	wg.Go(func() { errs[1] = t.RS.Call(ctx, t.deps, "announced-prefixes", res, &pf) })
	wg.Go(func() { errs[2] = t.RS.Call(ctx, t.deps, "asn-neighbours", res, &nb) })
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			httpapi.Error(w, r, e)
			return
		}
	}
	out := ASNInfo{ASN: asn, Holder: ov.Holder, Announced: ov.Announced, Block: ov.Block.Desc, PrefixesV4: []string{}, PrefixesV6: []string{}, Neighbours: []Neighbour{}, Upstreams: nb.Counts.Left, Downstreams: nb.Counts.Right}
	out.PrefixesTotal = len(pf.Prefixes)
	for i, p := range pf.Prefixes {
		if i >= MaxPrefixes {
			break
		}
		if pp, err := netip.ParsePrefix(p.Prefix); err == nil && pp.Addr().Is6() {
			out.PrefixesV6 = append(out.PrefixesV6, p.Prefix)
		} else {
			out.PrefixesV4 = append(out.PrefixesV4, p.Prefix)
		}
	}
	sort.SliceStable(nb.Neighbours, func(i, j int) bool { return nb.Neighbours[i].Power > nb.Neighbours[j].Power })
	out.NeighboursTotal = len(nb.Neighbours)
	for i, n := range nb.Neighbours {
		if i >= MaxNeighbours {
			break
		}
		out.Neighbours = append(out.Neighbours, Neighbour{ASN: uint32(n.ASN), Type: n.Type, Power: n.Power})
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}

// ---------- BGP-03 prefix routing status ----------

// PrefixTool is BGP-03.
type PrefixTool struct {
	deps tools.Deps
	RS   Client
}

func (*PrefixTool) ID() string                                { return "bgp.prefix" }
func (*PrefixTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *PrefixTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	mount(r, d, "/prefix/status", t.handle)
}

// Seen is a first/last-seen event.
type Seen struct {
	Prefix string  `json:"prefix"`
	Origin FlexASN `json:"origin"`
	Time   string  `json:"time"`
}

// Origin is an AS originating the prefix, with its route objects.
type Origin struct {
	ASN          uint32   `json:"asn"`
	RouteObjects []string `json:"routeObjects"`
}

// Related is a less- or more-specific prefix.
type Related struct {
	Prefix string `json:"prefix"`
	Origin uint32 `json:"origin"`
}

// PrefixStatus is the BGP-03 payload.
type PrefixStatus struct {
	Prefix        string    `json:"prefix"`
	Announced     bool      `json:"announced"`
	SeenBy        int       `json:"seenBy"`
	TotalPeers    int       `json:"totalPeers"`
	FirstSeen     *Seen     `json:"firstSeen"`
	LastSeen      *Seen     `json:"lastSeen"`
	Origins       []Origin  `json:"origins"`
	MOAS          bool      `json:"moas"`
	LessSpecifics []Related `json:"lessSpecifics"`
	MoreSpecifics []Related `json:"moreSpecifics"`
}

type visibility struct {
	Seeing int `json:"ris_peers_seeing"`
	Total  int `json:"total_ris_peers"`
}

func relatedList(in []struct {
	Prefix string  `json:"prefix"`
	Origin FlexASN `json:"origin"`
}) []Related {
	out := []Related{}
	for i, x := range in {
		if i >= MaxPrefixes {
			break
		}
		out = append(out, Related{Prefix: x.Prefix, Origin: uint32(x.Origin)})
	}
	return out
}

func (t *PrefixTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("prefix", "", true, 64)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	p, err := ParsePrefix(raw, "prefix")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	var rs struct {
		FirstSeen  *Seen `json:"first_seen"`
		LastSeen   *Seen `json:"last_seen"`
		Visibility struct {
			V4 visibility `json:"v4"`
			V6 visibility `json:"v6"`
		} `json:"visibility"`
		Origins []struct {
			Origin       FlexASN  `json:"origin"`
			RouteObjects []string `json:"route_objects"`
		} `json:"origins"`
		Less []struct {
			Prefix string  `json:"prefix"`
			Origin FlexASN `json:"origin"`
		} `json:"less_specifics"`
		More []struct {
			Prefix string  `json:"prefix"`
			Origin FlexASN `json:"origin"`
		} `json:"more_specifics"`
	}
	if err := t.RS.Call(r.Context(), t.deps, "routing-status", url.Values{"resource": {p.String()}}, &rs); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	vis := rs.Visibility.V4
	if p.Addr().Is6() {
		vis = rs.Visibility.V6
	}
	out := PrefixStatus{Prefix: p.String(), SeenBy: vis.Seeing, TotalPeers: vis.Total, FirstSeen: rs.FirstSeen, LastSeen: rs.LastSeen, Origins: []Origin{}, LessSpecifics: relatedList(rs.Less), MoreSpecifics: relatedList(rs.More)}
	for _, o := range rs.Origins {
		ro := o.RouteObjects
		if ro == nil {
			ro = []string{}
		}
		out.Origins = append(out.Origins, Origin{ASN: uint32(o.Origin), RouteObjects: ro})
	}
	out.Announced = len(out.Origins) > 0 && vis.Seeing > 0
	out.MOAS = len(out.Origins) > 1
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}

// ---------- BGP-04 RPKI ----------

// RPKITool is BGP-04.
type RPKITool struct {
	deps tools.Deps
	RS   Client
}

func (*RPKITool) ID() string                                { return "bgp.rpki" }
func (*RPKITool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *RPKITool) Mount(r chi.Router, d tools.Deps)        { t.deps = d; mount(r, d, "/rpki", t.handle) }

// ROA is a validating route origin authorization.
type ROA struct {
	Origin    uint32 `json:"origin"`
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	Validity  string `json:"validity"`
}

// RPKIResult is the BGP-04 payload. Status: valid, invalid_asn, invalid_length or unknown.
type RPKIResult struct {
	Prefix    string `json:"prefix"`
	ASN       uint32 `json:"asn"`
	Status    string `json:"status"`
	Validator string `json:"validator"`
	ROAs      []ROA  `json:"roas"`
}

func (t *RPKITool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	rawPrefix := q.String("prefix", "", true, 64)
	rawASN := q.String("asn", "", true, 16)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	p, err := ParsePrefix(rawPrefix, "prefix")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	asn, err := ParseASN(rawASN, "asn")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	var rv struct {
		Status    string `json:"status"`
		Validator string `json:"validator"`
		ROAs      []struct {
			Origin    FlexASN `json:"origin"`
			Prefix    string  `json:"prefix"`
			MaxLength int     `json:"max_length"`
			Validity  string  `json:"validity"`
		} `json:"validating_roas"`
	}
	if err := t.RS.Call(r.Context(), t.deps, "rpki-validation", url.Values{"resource": {"AS" + strconv.FormatUint(uint64(asn), 10)}, "prefix": {p.String()}}, &rv); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := RPKIResult{Prefix: p.String(), ASN: asn, Status: rv.Status, Validator: rv.Validator, ROAs: []ROA{}}
	if out.Status == "" {
		out.Status = "unknown"
	}
	for _, x := range rv.ROAs {
		out.ROAs = append(out.ROAs, ROA{Origin: uint32(x.Origin), Prefix: x.Prefix, MaxLength: x.MaxLength, Validity: x.Validity})
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}

// ---------- BGP-05 looking glass ----------

// LookingGlassTool is BGP-05.
type LookingGlassTool struct {
	deps tools.Deps
	RS   Client
}

func (*LookingGlassTool) ID() string                                { return "bgp.lg" }
func (*LookingGlassTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *LookingGlassTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	mount(r, d, "/lookingglass", t.handle)
}

// Route is one route seen by a RIS collector peer.
type Route struct {
	Collector string `json:"collector"`
	Location  string `json:"location"`
	Peer      string `json:"peer"`
	Prefix    string `json:"prefix"`
	ASPath    string `json:"asPath"`
	Origin    uint32 `json:"origin"`
	Community string `json:"community"`
	NextHop   string `json:"nextHop"`
	Updated   string `json:"updated"`
}

// LookingGlass is the BGP-05 payload.
type LookingGlass struct {
	Prefix     string   `json:"prefix"`
	Collectors int      `json:"collectors"`
	Total      int      `json:"total"`
	Routes     []Route  `json:"routes"`
	Origins    []uint32 `json:"origins"`
}

func (t *LookingGlassTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("prefix", "", true, 64)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	p, err := ParsePrefix(raw, "prefix")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	var lg struct {
		RRCs []struct {
			RRC      string `json:"rrc"`
			Location string `json:"location"`
			Peers    []struct {
				Origin    FlexASN `json:"asn_origin"`
				ASPath    string  `json:"as_path"`
				Community string  `json:"community"`
				Updated   string  `json:"last_updated"`
				Prefix    string  `json:"prefix"`
				Peer      string  `json:"peer"`
				NextHop   string  `json:"next_hop"`
			} `json:"peers"`
		} `json:"rrcs"`
	}
	if err := t.RS.Call(r.Context(), t.deps, "looking-glass", url.Values{"resource": {p.String()}}, &lg); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := LookingGlass{Prefix: p.String(), Collectors: len(lg.RRCs), Routes: []Route{}, Origins: []uint32{}}
	origins := map[uint32]bool{}
	for _, c := range lg.RRCs {
		for _, peer := range c.Peers {
			out.Total++
			if !origins[uint32(peer.Origin)] && peer.Origin != 0 {
				origins[uint32(peer.Origin)] = true
				out.Origins = append(out.Origins, uint32(peer.Origin))
			}
			if len(out.Routes) < MaxLGPeers {
				out.Routes = append(out.Routes, Route{Collector: c.RRC, Location: c.Location, Peer: peer.Peer, Prefix: peer.Prefix, ASPath: peer.ASPath, Origin: uint32(peer.Origin), Community: peer.Community, NextHop: peer.NextHop, Updated: peer.Updated})
			}
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "ripestat"})
}
