// Package peeringdb implements BGP-09: a network's PeeringDB record (peering policy, IXPs,
// facilities, contacts).
package peeringdb

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/providers"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/ripestat"
)

// MaxRows bounds IXP and facility lists.
const MaxRows = 500

// Tool is bgp.peeringdb.
type Tool struct {
	deps tools.Deps
	// Base overrides https://www.peeringdb.com/api (tests).
	Base string
}

func (*Tool) ID() string                                { return "bgp.peeringdb" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) } // key optional

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://www.peeringdb.com/api"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(20*time.Second)).Get("/peeringdb/{asn}", t.handle)
}

type apiNet struct {
	Name            string `json:"name"`
	AKA             string `json:"aka"`
	ASN             uint32 `json:"asn"`
	Website         string `json:"website"`
	IRRASSet        string `json:"irr_as_set"`
	InfoType        string `json:"info_type"`
	InfoScope       string `json:"info_scope"`
	InfoTraffic     string `json:"info_traffic"`
	InfoRatio       string `json:"info_ratio"`
	InfoPrefixes4   int    `json:"info_prefixes4"`
	InfoPrefixes6   int    `json:"info_prefixes6"`
	PolicyGeneral   string `json:"policy_general"`
	PolicyURL       string `json:"policy_url"`
	PolicyLocations string `json:"policy_locations"`
	PolicyContracts string `json:"policy_contracts"`
	LookingGlass    string `json:"looking_glass"`
	RouteServer     string `json:"route_server"`
	Updated         string `json:"updated"`
	IXs             []struct {
		Name    string  `json:"name"`
		IXID    int     `json:"ix_id"`
		Speed   int     `json:"speed"`
		IPv4    *string `json:"ipaddr4"`
		IPv6    *string `json:"ipaddr6"`
		RSPeer  bool    `json:"is_rs_peer"`
		Operate bool    `json:"operational"`
	} `json:"netixlan_set"`
	Facs []struct {
		Name    string `json:"name"`
		City    string `json:"city"`
		Country string `json:"country"`
		FacID   int    `json:"fac_id"`
	} `json:"netfac_set"`
	POCs []struct {
		Role    string `json:"role"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		URL     string `json:"url"`
		Visible string `json:"visible"`
	} `json:"poc_set"`
}

// IX is one exchange connection.
type IX struct {
	Name     string `json:"name"`
	IXID     int    `json:"ixId"`
	SpeedMb  int    `json:"speedMbps"`
	IPv4     string `json:"ipv4"`
	IPv6     string `json:"ipv6"`
	RSPeer   bool   `json:"rsPeer"`
	Operates bool   `json:"operational"`
}

// Facility is a colocation presence.
type Facility struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
	FacID   int    `json:"facId"`
}

// Contact is a public point of contact.
type Contact struct {
	Role  string `json:"role"`
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// Network is the BGP-09 payload.
type Network struct {
	Found           bool       `json:"found"`
	ASN             uint32     `json:"asn"`
	Name            string     `json:"name"`
	AKA             string     `json:"aka"`
	Website         string     `json:"website"`
	IRRASSet        string     `json:"irrAsSet"`
	Type            string     `json:"type"`
	Scope           string     `json:"scope"`
	Traffic         string     `json:"traffic"`
	Ratio           string     `json:"ratio"`
	Prefixes4       int        `json:"prefixes4"`
	Prefixes6       int        `json:"prefixes6"`
	Policy          string     `json:"policy"`
	PolicyURL       string     `json:"policyUrl"`
	PolicyLocations string     `json:"policyLocations"`
	PolicyContracts string     `json:"policyContracts"`
	LookingGlass    string     `json:"lookingGlass"`
	RouteServer     string     `json:"routeServer"`
	Updated         string     `json:"updated"`
	IXs             []IX       `json:"ixs"`
	IXCount         int        `json:"ixCount"`
	Facilities      []Facility `json:"facilities"`
	FacilityCount   int        `json:"facilityCount"`
	Contacts        []Contact  `json:"contacts"`
	PeeringDBURL    string     `json:"peeringdbUrl"`
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	asn, err := ripestat.ParseASN(chi.URLParam(r, "asn"), "asn")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	req := providers.Request{Provider: "PeeringDB", URL: t.Base + "/net?depth=2&asn=" + strconv.FormatUint(uint64(asn), 10), Header: http.Header{}}
	if key := t.deps.Config.Features.ProviderSecrets["PEERINGDB_KEY"]; key != "" {
		req.Header.Set("Authorization", "Api-Key "+key) // higher rate limits; cache key stays the URL
	}
	var resp struct {
		Data []apiNet `json:"data"`
	}
	if err := t.deps.Providers.GetJSON(r.Context(), req, &resp); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := Network{ASN: asn, IXs: []IX{}, Facilities: []Facility{}, Contacts: []Contact{}}
	if len(resp.Data) == 0 {
		httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "peeringdb"})
		return
	}
	n := resp.Data[0]
	out = Network{
		Found: true, ASN: n.ASN, Name: n.Name, AKA: n.AKA, Website: n.Website, IRRASSet: n.IRRASSet, Type: n.InfoType, Scope: n.InfoScope,
		Traffic: n.InfoTraffic, Ratio: n.InfoRatio, Prefixes4: n.InfoPrefixes4, Prefixes6: n.InfoPrefixes6, Policy: n.PolicyGeneral,
		PolicyURL: n.PolicyURL, PolicyLocations: n.PolicyLocations, PolicyContracts: n.PolicyContracts, LookingGlass: n.LookingGlass,
		RouteServer: n.RouteServer, Updated: n.Updated, IXs: []IX{}, Facilities: []Facility{}, Contacts: []Contact{},
		IXCount: len(n.IXs), FacilityCount: len(n.Facs), PeeringDBURL: "https://www.peeringdb.com/asn/" + strconv.FormatUint(uint64(asn), 10),
	}
	for i, x := range n.IXs {
		if i >= MaxRows {
			break
		}
		out.IXs = append(out.IXs, IX{Name: x.Name, IXID: x.IXID, SpeedMb: x.Speed, IPv4: str(x.IPv4), IPv6: str(x.IPv6), RSPeer: x.RSPeer, Operates: x.Operate})
	}
	sort.SliceStable(out.IXs, func(i, j int) bool { return out.IXs[i].SpeedMb > out.IXs[j].SpeedMb })
	for i, f := range n.Facs {
		if i >= MaxRows {
			break
		}
		out.Facilities = append(out.Facilities, Facility{Name: f.Name, City: f.City, Country: f.Country, FacID: f.FacID})
	}
	for _, p := range n.POCs {
		if p.Visible == "Public" {
			out.Contacts = append(out.Contacts, Contact{Role: p.Role, Name: p.Name, Email: p.Email, URL: p.URL})
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "peeringdb"})
}
