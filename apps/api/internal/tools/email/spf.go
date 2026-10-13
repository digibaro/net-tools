package email

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// RFC 7208 limits.
const (
	MaxLookups     = 10
	MaxVoidLookups = 2
	maxDepth       = 12
)

// SPFTool is EM-02.
type SPFTool struct {
	deps tools.Deps
	res  resolver
}

func (*SPFTool) ID() string                                { return "email.spf" }
func (*SPFTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *SPFTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.res == nil {
		t.res = sysResolver{}
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(30*time.Second)).Get("/email/spf", t.handle)
}

// SPFTerm is one mechanism or modifier.
type SPFTerm struct {
	Raw       string     `json:"raw"`
	Qualifier string     `json:"qualifier"`
	Kind      string     `json:"kind"`
	Value     string     `json:"value"`
	Lookup    bool       `json:"lookup"`
	Include   *SPFRecord `json:"include,omitempty"`
}

// SPFRecord is a parsed (and expanded) SPF record.
type SPFRecord struct {
	Domain string    `json:"domain"`
	Record string    `json:"record"`
	Terms  []SPFTerm `json:"terms"`
	Error  string    `json:"error"`
}

// SPFResult is the response payload.
type SPFResult struct {
	Domain      string     `json:"domain"`
	Record      *SPFRecord `json:"record"`
	Lookups     int        `json:"lookups"`
	VoidLookups int        `json:"voidLookups"`
	All         string     `json:"all"`
	Problems    []string   `json:"problems"`
	Warnings    []string   `json:"warnings"`
}

type spfWalker struct {
	ctx      context.Context
	res      resolver
	lookups  int
	voids    int
	problems []string
	warnings []string
	seen     map[string]bool
}

func (w *spfWalker) problem(code string) {
	for _, p := range w.problems {
		if p == code {
			return
		}
	}
	w.problems = append(w.problems, code)
}

func (w *spfWalker) warn(code string) {
	for _, p := range w.warnings {
		if p == code {
			return
		}
	}
	w.warnings = append(w.warnings, code)
}

// fetch returns the single v=spf1 record of domain.
func (w *spfWalker) fetch(domain string) (string, string) {
	txts, rcode, err := w.res.TXT(w.ctx, domain)
	if err != nil {
		return "", "dns-error"
	}
	var spf []string
	for _, t := range txts {
		l := strings.ToLower(t)
		if l == "v=spf1" || strings.HasPrefix(l, "v=spf1 ") {
			spf = append(spf, t)
		}
	}
	switch {
	case len(spf) == 0 && rcode == "NXDOMAIN":
		return "", "nxdomain"
	case len(spf) == 0:
		return "", "no-record"
	case len(spf) > 1:
		return "", "multiple-records"
	}
	return spf[0], ""
}

func (w *spfWalker) walk(domain string, depth int) *SPFRecord {
	rec := &SPFRecord{Domain: domain, Terms: []SPFTerm{}}
	if depth > maxDepth || w.seen[domain] {
		rec.Error = "loop"
		w.problem("include-loop")
		return rec
	}
	w.seen[domain] = true
	defer delete(w.seen, domain)
	txt, ferr := w.fetch(domain)
	if ferr != "" {
		rec.Error = ferr
		return rec
	}
	rec.Record = txt
	for _, raw := range strings.Fields(txt)[1:] {
		term := parseTerm(raw)
		switch term.Kind {
		case "include", "redirect":
			term.Lookup = true
			w.lookups++
			if term.Value == "" {
				w.problem("syntax")
				break
			}
			term.Include = w.walk(strings.ToLower(term.Value), depth+1)
			if term.Include.Error == "no-record" || term.Include.Error == "nxdomain" {
				w.voids++
				w.problem("include-missing")
			}
		case "a", "mx":
			term.Lookup = true
			w.lookups++
			target := term.Value
			if target == "" {
				target = domain
			}
			target = strings.SplitN(target, "/", 2)[0]
			qt := dns.TypeA
			if term.Kind == "mx" {
				qt = dns.TypeMX
			}
			if n, _, err := w.res.Exists(w.ctx, target, qt); err == nil && n == 0 {
				if term.Kind == "a" {
					if n6, _, _ := w.res.Exists(w.ctx, target, dns.TypeAAAA); n6 > 0 {
						break
					}
				}
				w.voids++
			}
		case "ptr":
			term.Lookup = true
			w.lookups++
			w.warn("ptr-deprecated")
		case "exists":
			term.Lookup = true
			w.lookups++
		case "ip4", "ip6", "all", "exp":
		default:
			if !strings.Contains(raw, "=") {
				w.problem("unknown-mechanism")
			}
		}
		rec.Terms = append(rec.Terms, term)
	}
	return rec
}

func parseTerm(raw string) SPFTerm {
	t := SPFTerm{Raw: raw, Qualifier: "+"}
	s := raw
	if strings.ContainsAny(s[:1], "+-~?") {
		t.Qualifier, s = s[:1], s[1:]
	}
	if k, v, ok := strings.Cut(s, "="); ok && !strings.Contains(k, ":") {
		t.Kind, t.Value = strings.ToLower(k), v
		return t
	}
	k, v, _ := strings.Cut(s, ":")
	if kk, vv, ok := strings.Cut(k, "/"); ok && v == "" {
		k, v = kk, "/"+vv
	}
	t.Kind, t.Value = strings.ToLower(k), v
	return t
}

// CheckSPF evaluates domain's SPF policy.
func CheckSPF(ctx context.Context, res resolver, domain string) SPFResult {
	w := &spfWalker{ctx: ctx, res: res, seen: map[string]bool{}}
	rec := w.walk(domain, 0)
	out := SPFResult{Domain: domain, Record: rec, Problems: []string{}, Warnings: []string{}}
	if rec.Error != "" {
		w.problem(rec.Error)
	}
	for _, term := range rec.Terms {
		if term.Kind == "all" {
			out.All = term.Qualifier + "all"
		}
	}
	switch out.All {
	case "+all":
		w.problem("plus-all")
	case "?all":
		w.warn("neutral-all")
	case "":
		hasRedirect := false
		for _, term := range rec.Terms {
			hasRedirect = hasRedirect || term.Kind == "redirect"
		}
		if rec.Error == "" && !hasRedirect {
			w.warn("no-all")
		}
	}
	if w.lookups > MaxLookups {
		w.problem("too-many-lookups")
	}
	if w.voids > MaxVoidLookups {
		w.problem("too-many-void-lookups")
	}
	out.Lookups, out.VoidLookups = w.lookups, w.voids
	out.Problems, out.Warnings = append(out.Problems, w.problems...), append(out.Warnings, w.warnings...)
	return out
}

func (t *SPFTool) handle(w http.ResponseWriter, r *http.Request) {
	domain, ok := domainParam(w, r, "domain")
	if !ok {
		return
	}
	res := CheckSPF(r.Context(), t.res, domain)
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: domain}})
}
