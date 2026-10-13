package email

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/net/publicsuffix"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// DMARCTool is EM-03.
type DMARCTool struct {
	deps tools.Deps
	res  resolver
}

func (*DMARCTool) ID() string                                { return "email.dmarc" }
func (*DMARCTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *DMARCTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.res == nil {
		t.res = sysResolver{}
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(20*time.Second)).Get("/email/dmarc", t.handle)
}

// ReportURI is one rua/ruf destination.
type ReportURI struct {
	URI string `json:"uri"`
	// External destinations need <domain>._report._dmarc.<dest> to authorize them.
	External   bool  `json:"external"`
	Authorized *bool `json:"authorized"`
}

// DMARCResult is the response payload.
type DMARCResult struct {
	Domain    string            `json:"domain"`
	FoundAt   string            `json:"foundAt"`
	OrgDomain string            `json:"orgDomain"`
	Record    string            `json:"record"`
	Tags      map[string]string `json:"tags"`
	Policy    string            `json:"policy"`
	SubPolicy string            `json:"subPolicy"`
	Pct       int               `json:"pct"`
	ADKIM     string            `json:"adkim"`
	ASPF      string            `json:"aspf"`
	RUA       []ReportURI       `json:"rua"`
	RUF       []ReportURI       `json:"ruf"`
	Problems  []string          `json:"problems"`
	Warnings  []string          `json:"warnings"`
}

func lookupDMARC(ctx context.Context, res resolver, domain string) (string, string) {
	txts, _, err := res.TXT(ctx, "_dmarc."+domain)
	if err != nil {
		return "", "dns-error"
	}
	var found []string
	for _, t := range txts {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "v=dmarc1") {
			found = append(found, t)
		}
	}
	switch len(found) {
	case 0:
		return "", "no-record"
	case 1:
		return found[0], ""
	}
	return "", "multiple-records"
}

// CheckDMARC evaluates domain's DMARC policy (falling back to the organizational domain).
func CheckDMARC(ctx context.Context, res resolver, domain string) DMARCResult {
	out := DMARCResult{Domain: domain, Tags: map[string]string{}, RUA: []ReportURI{}, RUF: []ReportURI{}, Problems: []string{}, Warnings: []string{}}
	out.OrgDomain, _ = publicsuffix.EffectiveTLDPlusOne(domain)
	rec, perr := lookupDMARC(ctx, res, domain)
	out.FoundAt = domain
	if perr == "no-record" && out.OrgDomain != "" && out.OrgDomain != domain {
		rec, perr = lookupDMARC(ctx, res, out.OrgDomain)
		out.FoundAt = out.OrgDomain
	}
	if perr != "" {
		out.Problems = append(out.Problems, perr)
		out.FoundAt = ""
		return out
	}
	out.Record = rec
	out.Tags = tagList(rec)
	out.Policy = strings.ToLower(out.Tags["p"])
	out.SubPolicy = strings.ToLower(out.Tags["sp"])
	if out.SubPolicy == "" {
		out.SubPolicy = out.Policy
	}
	out.Pct = 100
	if v, ok := out.Tags["pct"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 100 {
			out.Problems = append(out.Problems, "invalid-pct")
		} else {
			out.Pct = n
		}
	}
	out.ADKIM, out.ASPF = defaultTo(strings.ToLower(out.Tags["adkim"]), "r"), defaultTo(strings.ToLower(out.Tags["aspf"]), "r")
	switch out.Policy {
	case "none":
		out.Warnings = append(out.Warnings, "policy-none")
	case "quarantine", "reject":
	case "":
		out.Problems = append(out.Problems, "missing-policy")
	default:
		out.Problems = append(out.Problems, "invalid-policy")
	}
	if out.Pct < 100 {
		out.Warnings = append(out.Warnings, "partial-pct")
	}
	if out.Tags["rua"] == "" {
		out.Warnings = append(out.Warnings, "no-rua")
	}
	out.RUA = reportURIs(ctx, res, out.FoundAt, out.Tags["rua"])
	out.RUF = reportURIs(ctx, res, out.FoundAt, out.Tags["ruf"])
	for _, u := range append(append([]ReportURI{}, out.RUA...), out.RUF...) {
		if u.External && u.Authorized != nil && !*u.Authorized {
			out.Problems = append(out.Problems, "unauthorized-external-report")
			break
		}
	}
	return out
}

func defaultTo(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

// MaxReportURIs bounds rua/ruf entries examined per list.
const MaxReportURIs = 10

// reportURIs parses mailto: URIs and checks external-destination authorization (RFC 7489 §7.1).
func reportURIs(ctx context.Context, res resolver, domain, list string) []ReportURI {
	out := []ReportURI{}
	orgFrom, _ := publicsuffix.EffectiveTLDPlusOne(domain)
	for _, raw := range strings.Split(list, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if len(out) == MaxReportURIs {
			break // bounds the authorization lookups a hostile record can trigger
		}
		u := ReportURI{URI: raw}
		if p, err := url.Parse(strings.SplitN(raw, "!", 2)[0]); err == nil && p.Scheme == "mailto" {
			if at := strings.LastIndex(p.Opaque, "@"); at >= 0 {
				dest := strings.ToLower(p.Opaque[at+1:])
				orgTo, _ := publicsuffix.EffectiveTLDPlusOne(dest)
				if orgTo != orgFrom {
					u.External = true
					txts, _, err := res.TXT(ctx, domain+"._report._dmarc."+dest)
					ok := false
					for _, t := range txts {
						ok = ok || strings.HasPrefix(strings.ToLower(t), "v=dmarc1")
					}
					if err == nil {
						u.Authorized = &ok
					}
				}
			}
		}
		out = append(out, u)
	}
	return out
}

func (t *DMARCTool) handle(w http.ResponseWriter, r *http.Request) {
	domain, ok := domainParam(w, r, "domain")
	if !ok {
		return
	}
	httpapi.JSON(w, r, CheckDMARC(r.Context(), t.res, domain), &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: domain}})
}
