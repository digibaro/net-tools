package ripestat

import (
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"go4.org/netipx"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// ---------- IP-23 country IP blocks ----------

// CountryTool is IP-23: every IPv4/IPv6 block registered to a country (RIR delegation stats via
// RIPEstat country-resource-list), merged into the smallest list of prefixes.
type CountryTool struct {
	deps tools.Deps
	RS   Client
}

func (*CountryTool) ID() string                                { return "ip.country" }
func (*CountryTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }
func (t *CountryTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	mount(r, d, "/ip/country-blocks", t.handle)
}

// CountryBlocks is the IP-23 payload.
type CountryBlocks struct {
	Country string   `json:"country"`
	IPv4    []string `json:"ipv4"`
	IPv6    []string `json:"ipv6"`
	// Registered entries before merging (RIR ranges are not always CIDR-aligned).
	IPv4Entries int `json:"ipv4Entries"`
	IPv6Entries int `json:"ipv6Entries"`
	// IPv4Addresses is the number of IPv4 addresses covered.
	IPv4Addresses uint64 `json:"ipv4Addresses"`
	// IPv6Slash48s is the IPv6 space in /48 units (saturates at 2^63).
	IPv6Slash48s uint64 `json:"ipv6Slash48s"`
	ASNs         int    `json:"asns"`
	QueryTime    string `json:"queryTime"`
}

var countryRe = regexp.MustCompile(`^[A-Z]{2}$`)

// MergeBlocks parses "a.b.c.d/len" and "first-last" entries and returns the minimal prefix list.
func MergeBlocks(entries []string) ([]netip.Prefix, int) {
	var b netipx.IPSetBuilder
	n := 0
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if first, last, ok := strings.Cut(e, "-"); ok {
			f, err1 := netip.ParseAddr(strings.TrimSpace(first))
			l, err2 := netip.ParseAddr(strings.TrimSpace(last))
			if r := netipx.IPRangeFrom(f, l); err1 == nil && err2 == nil && r.IsValid() {
				b.AddRange(r)
				n++
			}
			continue
		}
		if p, err := netip.ParsePrefix(e); err == nil {
			b.AddPrefix(p.Masked())
			n++
		} else if a, err := netip.ParseAddr(e); err == nil {
			b.Add(a)
			n++
		}
	}
	set, _ := b.IPSet()
	if set == nil {
		return nil, n
	}
	return set.Prefixes(), n
}

func (t *CountryTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	cc := strings.ToUpper(strings.TrimSpace(q.String("country", "", true, 2)))
	family := q.String("family", "all", false, 3)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if !countryRe.MatchString(cc) {
		httpapi.Error(w, r, httpapi.InvalidInput("country", "use a two-letter ISO 3166 country code, e.g. NL"))
		return
	}
	if family != "all" && family != "4" && family != "6" {
		httpapi.Error(w, r, httpapi.InvalidInput("family", "family must be 4, 6 or all"))
		return
	}
	var data struct {
		Resources struct {
			ASN  []any    `json:"asn"`
			IPv4 []string `json:"ipv4"`
			IPv6 []string `json:"ipv6"`
		} `json:"resources"`
		QueryTime string `json:"query_time"`
	}
	if err := t.RS.Call(r.Context(), t.deps, "country-resource-list", url.Values{"resource": {cc}, "v4_format": {"prefix"}}, &data); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := CountryBlocks{Country: cc, IPv4: []string{}, IPv6: []string{}, ASNs: len(data.Resources.ASN), QueryTime: data.QueryTime}
	if family != "6" {
		v4, n := MergeBlocks(data.Resources.IPv4)
		out.IPv4Entries = n
		for _, p := range v4 {
			out.IPv4 = append(out.IPv4, p.String())
			out.IPv4Addresses += 1 << (32 - p.Bits())
		}
	}
	if family != "4" {
		v6, n := MergeBlocks(data.Resources.IPv6)
		out.IPv6Entries = n
		for _, p := range v6 {
			out.IPv6 = append(out.IPv6, p.String())
			if p.Bits() <= 48 {
				if add := uint64(1) << min(63, 48-p.Bits()); out.IPv6Slash48s+add >= out.IPv6Slash48s {
					out.IPv6Slash48s += add
				}
			}
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
