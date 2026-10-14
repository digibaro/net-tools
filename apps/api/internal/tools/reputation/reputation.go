// Package reputation implements REP-02 (AbuseIPDB) and REP-03 (Shodan InternetDB).
package reputation

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/providers"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// PublicIP parses ip and rejects addresses no public service knows anything about.
func PublicIP(raw string) (netip.Addr, error) {
	ip, err := netip.ParseAddr(strings.Trim(strings.TrimSpace(raw), "[]"))
	if err != nil || ip.Zone() != "" {
		return ip, httpapi.InvalidInput("ip", "enter an IPv4 or IPv6 address")
	}
	ip = ip.Unmap()
	if (target.Policy{}).Check(ip) != nil {
		return ip, httpapi.InvalidInput("ip", "private and reserved addresses have no public reputation")
	}
	return ip, nil
}

func ipParam(w http.ResponseWriter, r *http.Request) (netip.Addr, bool) {
	q := params.New(r.URL.Query())
	raw := q.String("ip", "", true, 64)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return netip.Addr{}, false
	}
	ip, err := PublicIP(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return ip, false
	}
	return ip, true
}

// ---------- REP-03 InternetDB ----------

// InternetDBTool is rep.internetdb (keyless).
type InternetDBTool struct {
	deps tools.Deps
	// Base overrides https://internetdb.shodan.io (tests).
	Base string
}

func (*InternetDBTool) ID() string                                { return "rep.internetdb" }
func (*InternetDBTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *InternetDBTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://internetdb.shodan.io"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/reputation/internetdb", t.handle)
}

// Exposure is the REP-03 payload.
type Exposure struct {
	IP        string   `json:"ip"`
	Found     bool     `json:"found"`
	Ports     []int    `json:"ports"`
	Hostnames []string `json:"hostnames"`
	CPEs      []string `json:"cpes"`
	Tags      []string `json:"tags"`
	Vulns     []string `json:"vulns"`
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (t *InternetDBTool) handle(w http.ResponseWriter, r *http.Request) {
	ip, ok := ipParam(w, r)
	if !ok {
		return
	}
	var e Exposure
	err := t.deps.Providers.GetJSON(r.Context(), providers.Request{Provider: "Shodan InternetDB", URL: t.Base + "/" + url.PathEscape(ip.String()), TTL: time.Hour}, &e)
	switch {
	case providers.IsNotFound(err):
		e = Exposure{} // InternetDB answers 404 for addresses it has never seen
	case err != nil:
		httpapi.Error(w, r, err)
		return
	default:
		e.Found = true
	}
	e.IP = ip.String()
	e.Ports, e.Hostnames, e.CPEs, e.Tags, e.Vulns = nonNil(e.Ports), nonNil(e.Hostnames), nonNil(e.CPEs), nonNil(e.Tags), nonNil(e.Vulns)
	httpapi.JSON(w, r, e, &httpapi.Meta{Tool: t.ID(), Vantage: "internetdb"})
}

// ---------- REP-02 AbuseIPDB ----------

// AbuseIPDBTool is rep.abuseipdb; it needs ABUSEIPDB_KEY.
type AbuseIPDBTool struct {
	deps tools.Deps
	// Base overrides https://api.abuseipdb.com/api/v2 (tests).
	Base string
}

func (*AbuseIPDBTool) ID() string { return "rep.abuseipdb" }
func (*AbuseIPDBTool) Enabled(d tools.Deps) (bool, tools.Reason) {
	return tools.RequireSecret(d, "ABUSEIPDB_KEY")
}

func (t *AbuseIPDBTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://api.abuseipdb.com/api/v2"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/reputation/abuseipdb", t.handle)
}

// Report is one recent abuse report (comment trimmed).
type Report struct {
	ReportedAt string `json:"reportedAt"`
	Comment    string `json:"comment"`
	Categories []int  `json:"categories"`
	Country    string `json:"country"`
}

// Abuse is the REP-02 payload.
type Abuse struct {
	IP            string   `json:"ip"`
	Score         int      `json:"score"`
	TotalReports  int      `json:"totalReports"`
	DistinctUsers int      `json:"distinctUsers"`
	LastReported  string   `json:"lastReported"`
	Whitelisted   bool     `json:"whitelisted"`
	Tor           bool     `json:"tor"`
	Country       string   `json:"country"`
	UsageType     string   `json:"usageType"`
	ISP           string   `json:"isp"`
	Domain        string   `json:"domain"`
	Hostnames     []string `json:"hostnames"`
	Reports       []Report `json:"reports"`
	URL           string   `json:"url"`
}

// MaxReports bounds the recent reports returned.
const MaxReports = 20

func (t *AbuseIPDBTool) handle(w http.ResponseWriter, r *http.Request) {
	ip, ok := ipParam(w, r)
	if !ok {
		return
	}
	var resp struct {
		Data struct {
			Score         int      `json:"abuseConfidenceScore"`
			Whitelisted   *bool    `json:"isWhitelisted"`
			Tor           bool     `json:"isTor"`
			Country       string   `json:"countryCode"`
			UsageType     string   `json:"usageType"`
			ISP           string   `json:"isp"`
			Domain        string   `json:"domain"`
			Hostnames     []string `json:"hostnames"`
			TotalReports  int      `json:"totalReports"`
			DistinctUsers int      `json:"numDistinctUsers"`
			LastReported  *string  `json:"lastReportedAt"`
			Reports       []struct {
				ReportedAt string `json:"reportedAt"`
				Comment    string `json:"comment"`
				Categories []int  `json:"categories"`
				Country    string `json:"reporterCountryCode"`
			} `json:"reports"`
		} `json:"data"`
	}
	q := url.Values{"ipAddress": {ip.String()}, "maxAgeInDays": {"90"}, "verbose": {""}}
	h := http.Header{"Key": {t.deps.Config.Features.ProviderSecrets["ABUSEIPDB_KEY"]}}
	if err := t.deps.Providers.GetJSON(r.Context(), providers.Request{Provider: "AbuseIPDB", URL: t.Base + "/check?" + q.Encode(), Header: h, TTL: time.Hour}, &resp); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	d := resp.Data
	out := Abuse{IP: ip.String(), Score: d.Score, TotalReports: d.TotalReports, DistinctUsers: d.DistinctUsers, Tor: d.Tor, Country: d.Country, UsageType: d.UsageType, ISP: d.ISP, Domain: d.Domain, Hostnames: nonNil(d.Hostnames), Reports: []Report{}, URL: "https://www.abuseipdb.com/check/" + ip.String()}
	if d.Whitelisted != nil {
		out.Whitelisted = *d.Whitelisted
	}
	if d.LastReported != nil {
		out.LastReported = *d.LastReported
	}
	for i, rep := range d.Reports {
		if i >= MaxReports {
			break
		}
		c := rep.Comment
		if len(c) > 300 {
			c = c[:300] + "…"
		}
		out.Reports = append(out.Reports, Report{ReportedAt: rep.ReportedAt, Comment: c, Categories: nonNil(rep.Categories), Country: rep.Country})
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "abuseipdb"})
}
