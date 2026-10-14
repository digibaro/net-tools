// Package geo implements REP-01: IP geolocation from local mmdb files in the data directory.
package geo

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/asn"
	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/geoip"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Tool is IP geolocation.
type Tool struct {
	once sync.Once
	city *geoip.DB
	asn  *geoip.DB
	// TXT resolves ASNs via Team Cymru when no ASN database is present.
	TXT asn.TXTFunc
}

func (*Tool) ID() string { return "rep.geo" }

func (t *Tool) open(cfg config.Config) {
	t.once.Do(func() {
		if p, _, err := geoip.Find(cfg.DataDir, cfg.Features.GeoIPCityDB, geoip.CityFiles); err == nil {
			t.city, _ = geoip.Open(p)
		}
		if p, _, err := geoip.Find(cfg.DataDir, cfg.Features.GeoIPASNDB, geoip.ASNFiles); err == nil {
			t.asn, _ = geoip.Open(p)
		}
	})
}

// Enabled requires a city database; the ASN database is optional.
func (t *Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	t.open(d.Config)
	if t.city == nil {
		return false, tools.ReasonDataMissing
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	if t.TXT == nil {
		t.TXT = asn.SystemTXT
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(10*time.Second)).Get("/geo", t.handle)
}

// Result is the response payload.
type Result struct {
	IP          string   `json:"ip"`
	Public      bool     `json:"public"`
	Found       bool     `json:"found"`
	Prefix      string   `json:"prefix"`
	City        string   `json:"city"`
	Region      string   `json:"region"`
	Postal      string   `json:"postal"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
	Continent   string   `json:"continent"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	AccuracyKm  int      `json:"accuracyKm"`
	TimeZone    string   `json:"timeZone"`
	ASN         *uint32  `json:"asn"`
	ASOrg       string   `json:"asOrg"`
	ASNSource   string   `json:"asnSource"`
	Source      string   `json:"source"`
	Database    string   `json:"database"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("ip", "", true, 64)
	source := q.Enum("source", "local", "local")
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	_ = source // only the local database is implemented; providers come later (catalog REP-01)
	ip, err := netip.ParseAddr(strings.Trim(raw, "[]"))
	if err != nil || ip.Zone() != "" {
		httpapi.Error(w, r, httpapi.InvalidInput("ip", "enter an IPv4 or IPv6 address"))
		return
	}
	ip = ip.Unmap()
	res := Lookup(r.Context(), t.city, t.asn, t.TXT, ip)
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

// Lookup geolocates ip. Private and reserved addresses are reported without a lookup.
func Lookup(ctx context.Context, city, asnDB *geoip.DB, txt asn.TXTFunc, ip netip.Addr) Result {
	res := Result{IP: ip.String(), Public: asn.IsPublic(ip), Source: "local"}
	if city != nil {
		typ, built := city.Info()
		res.Database = typ + " " + built.Format("2006-01-02")
	}
	if !res.Public {
		return res
	}
	if city != nil {
		var c geoip.City
		if prefix, found, err := city.Lookup(ip, &c); err == nil && found {
			res.Found, res.Prefix = true, prefix.String()
			res.City, res.Country, res.CountryCode = c.City.Names.Name(), c.Country.Names.Name(), c.Country.ISOCode
			res.Continent, res.Postal, res.TimeZone = c.Continent.Names.Name(), c.Postal.Code, c.Location.TimeZone
			res.Latitude, res.Longitude, res.AccuracyKm = c.Location.Latitude, c.Location.Longitude, int(c.Location.AccuracyRadius)
			if len(c.Subdivisions) > 0 {
				res.Region = c.Subdivisions[0].Names.Name()
			}
		}
	}
	if asnDB != nil {
		var a geoip.ASN
		if _, found, err := asnDB.Lookup(ip, &a); err == nil && found && a.Number != 0 {
			n := a.Number
			res.ASN, res.ASOrg, res.ASNSource = &n, a.Org, "mmdb"
		}
	}
	if res.ASN == nil && txt != nil {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if origins, err := asn.Lookup(ctx, txt, ip); err == nil && len(origins) > 0 {
			n := origins[0].ASN
			res.ASN, res.ASOrg, res.ASNSource = &n, origins[0].ASName, "cymru"
		}
	}
	return res
}
