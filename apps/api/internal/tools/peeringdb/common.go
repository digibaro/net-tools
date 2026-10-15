package peeringdb

import (
	"cmp"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/ripestat"
)

// CommonTool is BGP-16: where two networks can peer — shared exchanges and facilities.
type CommonTool struct {
	deps tools.Deps
	// Base overrides https://www.peeringdb.com/api (tests).
	Base string
}

func (*CommonTool) ID() string                                { return "bgp.common" }
func (*CommonTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *CommonTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://www.peeringdb.com/api"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(30*time.Second)).Get("/peeringdb/common", t.handle)
}

// Port is one network's connection to an exchange.
type Port struct {
	SpeedMb int    `json:"speedMbps"`
	IPv4    string `json:"ipv4"`
	IPv6    string `json:"ipv6"`
	RSPeer  bool   `json:"rsPeer"`
}

// CommonIX is an exchange both networks are on.
type CommonIX struct {
	IXID int    `json:"ixId"`
	Name string `json:"name"`
	A    []Port `json:"a"`
	B    []Port `json:"b"`
	// BothRS: both peer with the route servers, so they likely exchange routes already.
	BothRS bool `json:"bothRs"`
}

// Side summarizes one network.
type Side struct {
	ASN        uint32 `json:"asn"`
	Name       string `json:"name"`
	Found      bool   `json:"found"`
	Policy     string `json:"policy"`
	IXCount    int    `json:"ixCount"`
	FacCount   int    `json:"facilityCount"`
	PeeringURL string `json:"peeringdbUrl"`
}

// Common is the BGP-16 payload.
type Common struct {
	A          Side       `json:"a"`
	B          Side       `json:"b"`
	IXs        []CommonIX `json:"ixs"`
	Facilities []Facility `json:"facilities"`
}

func side(asn uint32, n apiNet, found bool) Side {
	return Side{ASN: asn, Name: n.Name, Found: found, Policy: n.PolicyGeneral, IXCount: len(n.IXs), FacCount: len(n.Facs), PeeringURL: "https://www.peeringdb.com/asn/" + strconv.FormatUint(uint64(asn), 10)}
}

// Intersect finds shared exchanges (by ix_id) and facilities (by fac_id).
func Intersect(a, b apiNet) ([]CommonIX, []Facility) {
	ports := func(n apiNet) map[int][]Port {
		m := map[int][]Port{}
		for _, x := range n.IXs {
			m[x.IXID] = append(m[x.IXID], Port{SpeedMb: x.Speed, IPv4: str(x.IPv4), IPv6: str(x.IPv6), RSPeer: x.RSPeer})
		}
		return m
	}
	pa, pb := ports(a), ports(b)
	ixs := []CommonIX{}
	for _, x := range a.IXs {
		if _, ok := pb[x.IXID]; !ok || slices.ContainsFunc(ixs, func(c CommonIX) bool { return c.IXID == x.IXID }) {
			continue
		}
		c := CommonIX{IXID: x.IXID, Name: x.Name, A: pa[x.IXID], B: pb[x.IXID]}
		c.BothRS = slices.ContainsFunc(c.A, func(p Port) bool { return p.RSPeer }) && slices.ContainsFunc(c.B, func(p Port) bool { return p.RSPeer })
		ixs = append(ixs, c)
	}
	slices.SortFunc(ixs, func(x, y CommonIX) int { return cmp.Compare(x.Name, y.Name) })
	facs := []Facility{}
	inB := map[int]bool{}
	for _, f := range b.Facs {
		inB[f.FacID] = true
	}
	for _, f := range a.Facs {
		if inB[f.FacID] && !slices.ContainsFunc(facs, func(x Facility) bool { return x.FacID == f.FacID }) {
			facs = append(facs, Facility{Name: f.Name, City: f.City, Country: f.Country, FacID: f.FacID})
		}
	}
	slices.SortFunc(facs, func(x, y Facility) int {
		return cmp.Or(cmp.Compare(x.Country, y.Country), cmp.Compare(x.City, y.City), cmp.Compare(x.Name, y.Name))
	})
	return ixs, facs
}

func (t *CommonTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	ra := q.String("a", "", true, 16)
	rb := q.String("b", "", true, 16)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	a, err := ripestat.ParseASN(ra, "a")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	b, err := ripestat.ParseASN(rb, "b")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	var (
		na, nb apiNet
		fa, fb bool
		ea, eb error
		wg     sync.WaitGroup
		ctx    = r.Context()
	)
	wg.Go(func() { na, fa, ea = fetchNet(ctx, t.deps, t.Base, a) })
	wg.Go(func() { nb, fb, eb = fetchNet(ctx, t.deps, t.Base, b) })
	wg.Wait()
	if err := cmp.Or(ea, eb); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := Common{A: side(a, na, fa), B: side(b, nb, fb), IXs: []CommonIX{}, Facilities: []Facility{}}
	if fa && fb {
		out.IXs, out.Facilities = Intersect(na, nb)
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "peeringdb"})
}
