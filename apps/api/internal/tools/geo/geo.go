// Package geo implements REP-01: IP geolocation from local mmdb files in the data directory.
package geo

import (
	"context"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/asn"
	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/geoip"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/providers"
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
	// IPInfoBase overrides https://ipinfo.io (tests).
	IPInfoBase string
	deps       tools.Deps
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

// Enabled requires a city database or an ipinfo.io token; the ASN database is optional.
func (t *Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	t.open(d.Config)
	if t.city == nil && !d.Config.HasSecret("IPINFO_TOKEN") {
		return false, tools.ReasonDataMissing
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.IPInfoBase == "" {
		t.IPInfoBase = "https://ipinfo.io"
	}
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
	def := "local"
	if t.city == nil {
		def = "ipinfo" // enabled without a database means a token is configured
	}
	source := q.Enum("source", def, "local", "ipinfo")
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ip, err := netip.ParseAddr(strings.Trim(raw, "[]"))
	if err != nil || ip.Zone() != "" {
		httpapi.Error(w, r, httpapi.InvalidInput("ip", "enter an IPv4 or IPv6 address"))
		return
	}
	ip = ip.Unmap()
	switch {
	case source == "ipinfo":
		res, err := t.ipinfo(r.Context(), ip)
		if err != nil {
			httpapi.Error(w, r, err)
			return
		}
		httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "ipinfo"})
	case t.city == nil:
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeToolDisabled, Message: "no local GeoIP database is installed on this server", Details: map[string]any{"field": "source"}})
	default:
		res := Lookup(r.Context(), t.city, t.asn, t.TXT, ip)
		httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
	}
}

// ipinfo geolocates through ipinfo.io (IPINFO_TOKEN, sent as a header so it never appears in
// URLs, cache keys or logs).
func (t *Tool) ipinfo(ctx context.Context, ip netip.Addr) (Result, error) {
	res := Result{IP: ip.String(), Public: asn.IsPublic(ip), Source: "ipinfo", Database: "ipinfo.io"}
	tok := t.deps.Config.Features.ProviderSecrets["IPINFO_TOKEN"]
	if tok == "" {
		return res, &httpapi.APIError{Code: httpapi.CodeProviderKeyMissing, Message: "ipinfo.io needs IPINFO_TOKEN, which is not configured on this server"}
	}
	if !res.Public {
		return res, nil
	}
	var d struct {
		City     string `json:"city"`
		Region   string `json:"region"`
		Country  string `json:"country"`
		Loc      string `json:"loc"`
		Org      string `json:"org"`
		Postal   string `json:"postal"`
		Timezone string `json:"timezone"`
		Bogon    bool   `json:"bogon"`
	}
	err := t.deps.Providers.GetJSON(ctx, providers.Request{Provider: "ipinfo.io", URL: t.IPInfoBase + "/" + ip.String() + "/json", Header: http.Header{"Authorization": {"Bearer " + tok}}, TTL: 24 * time.Hour}, &d)
	if err != nil {
		return res, err //nolint:wrapcheck // client-safe provider error
	}
	res.Found = d.Country != ""
	res.City, res.Region, res.Postal, res.TimeZone, res.CountryCode = d.City, d.Region, d.Postal, d.Timezone, d.Country
	res.Country = d.Country
	if lat, lon, ok := strings.Cut(d.Loc, ","); ok {
		if la, err := strconv.ParseFloat(lat, 64); err == nil {
			res.Latitude = &la
		}
		if lo, err := strconv.ParseFloat(lon, 64); err == nil {
			res.Longitude = &lo
		}
	}
	if num, name, ok := strings.Cut(d.Org, " "); ok && strings.HasPrefix(num, "AS") {
		if n, err := strconv.ParseUint(num[2:], 10, 32); err == nil {
			v := uint32(n)
			res.ASN, res.ASOrg, res.ASNSource = &v, name, "ipinfo"
		}
	}
	return res, nil
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
