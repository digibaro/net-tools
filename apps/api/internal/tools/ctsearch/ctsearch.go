// Package ctsearch implements TLS-08: certificates logged in Certificate Transparency for a
// domain (crt.sh), useful for auditing issuance and discovering subdomains.
package ctsearch

import (
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/providers"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// Limits.
const (
	MaxCerts = 500
	MaxNames = 2000
)

// Tool is tls.ct.
type Tool struct {
	deps tools.Deps
	// Base overrides https://crt.sh (tests).
	Base string
	Now  func() time.Time
}

func (*Tool) ID() string                                { return "tls.ct" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://crt.sh"
	}
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/ct", t.handle)
}

type entry struct {
	ID         int64  `json:"id"`
	IssuerName string `json:"issuer_name"`
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"`
	NotBefore  string `json:"not_before"`
	NotAfter   string `json:"not_after"`
	Serial     string `json:"serial_number"`
}

// Cert is one logged certificate (precertificate and final certificate merged by serial).
type Cert struct {
	ID        int64    `json:"id"`
	Issuer    string   `json:"issuer"`
	Names     []string `json:"names"`
	NotBefore string   `json:"notBefore"`
	NotAfter  string   `json:"notAfter"`
	Expired   bool     `json:"expired"`
	URL       string   `json:"url"`
}

// Result is the TLS-08 payload.
type Result struct {
	Domain    string   `json:"domain"`
	Total     int      `json:"total"`
	Certs     []Cert   `json:"certs"`
	Names     []string `json:"names"`
	Issuers   []Issuer `json:"issuers"`
	Truncated bool     `json:"truncated"`
}

// Issuer counts certificates per issuing CA.
type Issuer struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

var cnRe = regexp.MustCompile(`(?:^|,\s*)CN=([^,]+)`)

func issuerCN(dn string) string {
	if m := cnRe.FindStringSubmatch(dn); m != nil {
		return strings.TrimSpace(m[1])
	}
	return dn
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("domain", "", true, 253)
	expired := q.Bool("includeExpired", false)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	domain, err := target.NormalizeHostname(strings.TrimPrefix(strings.TrimSpace(raw), "*."))
	if err != nil || !strings.Contains(domain, ".") {
		httpapi.Error(w, r, httpapi.InvalidInput("domain", "enter a domain name such as example.com"))
		return
	}
	v := url.Values{"q": {"%." + domain}, "output": {"json"}, "deduplicate": {"Y"}}
	if !expired {
		v.Set("exclude", "expired")
	}
	var entries []entry
	err = t.deps.Providers.GetJSON(r.Context(), providers.Request{Provider: "crt.sh", URL: t.Base + "/?" + v.Encode(), Timeout: providers.MaxTimeout, TTL: 30 * time.Minute}, &entries)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	httpapi.JSON(w, r, t.summarize(domain, entries), &httpapi.Meta{Tool: t.ID(), Vantage: "crt.sh"})
}

func (t *Tool) summarize(domain string, entries []entry) Result {
	out := Result{Domain: domain, Certs: []Cert{}, Names: []string{}, Issuers: []Issuer{}}
	bySerial := map[string]int{}
	names := map[string]bool{}
	issuers := map[string]int{}
	now := t.Now()
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].NotBefore > entries[j].NotBefore })
	for _, e := range entries {
		var certNames []string
		for _, n := range strings.Split(e.NameValue, "\n") {
			n = strings.ToLower(strings.TrimSpace(n))
			if n == "" {
				continue
			}
			certNames = append(certNames, n)
			if n == domain || strings.HasSuffix(n, "."+domain) {
				names[n] = true
			}
		}
		key := e.Serial + "|" + e.IssuerName
		if _, dup := bySerial[key]; dup || e.Serial == "" && e.ID == 0 {
			continue
		}
		bySerial[key] = len(out.Certs)
		out.Total++
		iss := issuerCN(e.IssuerName)
		issuers[iss]++
		if len(out.Certs) >= MaxCerts {
			out.Truncated = true
			continue
		}
		na, _ := time.Parse("2006-01-02T15:04:05", e.NotAfter)
		out.Certs = append(out.Certs, Cert{ID: e.ID, Issuer: iss, Names: certNames, NotBefore: e.NotBefore, NotAfter: e.NotAfter, Expired: !na.IsZero() && na.Before(now), URL: "https://crt.sh/?id=" + strconv.FormatInt(e.ID, 10)})
	}
	for n := range names {
		out.Names = append(out.Names, n)
	}
	sort.Strings(out.Names)
	if len(out.Names) > MaxNames {
		out.Names, out.Truncated = out.Names[:MaxNames], true
	}
	for n, c := range issuers {
		out.Issuers = append(out.Issuers, Issuer{Name: n, Count: c})
	}
	sort.Slice(out.Issuers, func(i, j int) bool {
		return out.Issuers[i].Count > out.Issuers[j].Count || out.Issuers[i].Count == out.Issuers[j].Count && out.Issuers[i].Name < out.Issuers[j].Name
	})
	return out
}
