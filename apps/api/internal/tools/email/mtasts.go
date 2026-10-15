package email

import (
	"bufio"
	"context"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
)

// Fetched is a file retrieved over HTTPS.
type Fetched struct {
	Status      int
	ContentType string
	Body        string
	CertError   string // TLS verification problem ("" = valid)
}

// Fetcher retrieves an https URL through the target policy.
type Fetcher func(ctx context.Context, url string) (Fetched, error)

// MTASTSTool is EM-05: MTA-STS, TLS-RPT and BIMI.
type MTASTSTool struct {
	deps  tools.Deps
	res   resolver
	fetch Fetcher
}

func (*MTASTSTool) ID() string                                { return "email.mtasts" }
func (*MTASTSTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *MTASTSTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.res == nil {
		t.res = sysResolver{}
	}
	if t.fetch == nil {
		t.fetch = func(ctx context.Context, u string) (Fetched, error) {
			hops, err := (&httpcore.Client{Resolver: d.Target}).Do(ctx, httpcore.Request{URL: u, Method: "GET", UserAgent: d.UserAgent, MaxBody: 64 << 10, PreviewBytes: 64 << 10, Timeout: 10 * time.Second})
			if err != nil {
				return Fetched{}, err //nolint:wrapcheck // client-facing
			}
			h := hops[len(hops)-1]
			f := Fetched{Status: h.Status, ContentType: h.ContentType, Body: h.BodyPreview}
			if h.TLS != nil {
				f.CertError = h.TLS.VerifyError
			}
			return f, nil
		}
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/email/mtasts", t.handle)
}

// MTASTS is the MTA-STS part.
type MTASTS struct {
	Record    string   `json:"record"`
	ID        string   `json:"id"`
	PolicyURL string   `json:"policyUrl"`
	Mode      string   `json:"mode"`
	MaxAge    int      `json:"maxAge"`
	MX        []string `json:"mx"`
	Unmatched []string `json:"unmatched"`
	Problems  []string `json:"problems"`
	Warnings  []string `json:"warnings"`
}

// TLSRPT is the TLS reporting record.
type TLSRPT struct {
	Record   string   `json:"record"`
	RUA      []string `json:"rua"`
	Problems []string `json:"problems"`
}

// BIMI is the brand indicator record.
type BIMI struct {
	Record      string   `json:"record"`
	Logo        string   `json:"logo"`
	Certificate string   `json:"certificate"`
	Problems    []string `json:"problems"`
	Warnings    []string `json:"warnings"`
}

// MTASTSResult is the EM-05 payload.
type MTASTSResult struct {
	Domain string `json:"domain"`
	MTASTS MTASTS `json:"mtaSts"`
	TLSRPT TLSRPT `json:"tlsRpt"`
	BIMI   BIMI   `json:"bimi"`
}

func findRecord(txts []string, prefix string) (string, int) {
	found, n := "", 0
	for _, t := range txts {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), strings.ToLower(prefix)) {
			found, n = t, n+1
		}
	}
	return found, n
}

// ParseSTSPolicy parses an MTA-STS policy file (RFC 8461 §3.2).
func ParseSTSPolicy(body string) (mode string, maxAge int, mx []string, version string) {
	mx = []string{}
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "version":
			version = v
		case "mode":
			mode = v
		case "max_age":
			maxAge, _ = strconv.Atoi(v)
		case "mx":
			mx = append(mx, strings.ToLower(v))
		}
	}
	return mode, maxAge, mx, version
}

// MXMatches reports whether host matches an MTA-STS mx pattern ("*.example.com" covers one label).
func MXMatches(pattern, host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if rest, ok := strings.CutPrefix(pattern, "*."); ok {
		_, after, found := strings.Cut(host, ".")
		return found && after == rest
	}
	return host == pattern
}

func (t *MTASTSTool) handle(w http.ResponseWriter, r *http.Request) {
	domain, ok := domainParam(w, r, "domain")
	if !ok {
		return
	}
	ctx := r.Context()
	out := MTASTSResult{Domain: domain}
	out.MTASTS = t.checkSTS(ctx, domain)
	out.TLSRPT = t.checkTLSRPT(ctx, domain)
	out.BIMI = t.checkBIMI(ctx, domain)
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: domain}})
}

func (t *MTASTSTool) checkSTS(ctx context.Context, domain string) MTASTS {
	s := MTASTS{MX: []string{}, Unmatched: []string{}, Problems: []string{}, Warnings: []string{}, PolicyURL: "https://mta-sts." + domain + "/.well-known/mta-sts.txt"}
	txts, _, err := t.res.TXT(ctx, "_mta-sts."+domain)
	if err != nil {
		s.Problems = append(s.Problems, "dns-error")
		return s
	}
	rec, n := findRecord(txts, "v=STSv1")
	switch {
	case n == 0:
		s.Problems = append(s.Problems, "no-record")
		return s
	case n > 1:
		s.Problems = append(s.Problems, "multiple-records")
	}
	s.Record = rec
	tags := tagList(rec)
	s.ID = tags["id"]
	if s.ID == "" {
		s.Problems = append(s.Problems, "no-id")
	}
	f, err := t.fetch(ctx, s.PolicyURL)
	switch {
	case err != nil:
		s.Problems = append(s.Problems, "policy-unreachable")
		return s
	case f.Status != http.StatusOK:
		s.Problems = append(s.Problems, "policy-not-found")
		return s
	case f.CertError != "":
		s.Problems = append(s.Problems, "policy-certificate")
	}
	if ct := strings.ToLower(f.ContentType); !strings.HasPrefix(ct, "text/plain") {
		s.Warnings = append(s.Warnings, "policy-content-type")
	}
	var version string
	s.Mode, s.MaxAge, s.MX, version = ParseSTSPolicy(f.Body)
	if version != "STSv1" {
		s.Problems = append(s.Problems, "policy-version")
	}
	switch s.Mode {
	case "enforce":
	case "testing":
		s.Warnings = append(s.Warnings, "mode-testing")
	case "none":
		s.Warnings = append(s.Warnings, "mode-none")
	default:
		s.Problems = append(s.Problems, "policy-mode")
	}
	switch {
	case s.MaxAge <= 0 || s.MaxAge > 31557600:
		s.Problems = append(s.Problems, "policy-max-age")
	case s.MaxAge < 86400:
		s.Warnings = append(s.Warnings, "short-max-age")
	}
	if len(s.MX) == 0 && s.Mode != "none" {
		s.Problems = append(s.Problems, "policy-no-mx")
	}
	// Every MX of the domain must match a policy pattern, or senders in enforce mode refuse it.
	if mxs, _, err := t.res.MX(ctx, domain); err == nil {
		for _, m := range mxs {
			host := strings.TrimSuffix(strings.ToLower(m.Mx), ".")
			matched := false
			for _, p := range s.MX {
				matched = matched || MXMatches(p, host)
			}
			if !matched && host != "" {
				s.Unmatched = append(s.Unmatched, host)
			}
		}
		if len(s.Unmatched) > 0 {
			s.Problems = append(s.Problems, "mx-not-covered")
		}
	}
	return s
}

func (t *MTASTSTool) checkTLSRPT(ctx context.Context, domain string) TLSRPT {
	r := TLSRPT{RUA: []string{}, Problems: []string{}}
	txts, _, err := t.res.TXT(ctx, "_smtp._tls."+domain)
	if err != nil {
		r.Problems = append(r.Problems, "dns-error")
		return r
	}
	rec, n := findRecord(txts, "v=TLSRPTv1")
	if n == 0 {
		r.Problems = append(r.Problems, "no-record")
		return r
	}
	r.Record = rec
	for _, u := range strings.Split(tagList(rec)["rua"], ",") {
		if u = strings.TrimSpace(u); u != "" {
			r.RUA = append(r.RUA, u)
			if !strings.HasPrefix(u, "mailto:") && !strings.HasPrefix(u, "https://") {
				r.Problems = append(r.Problems, "bad-rua")
			}
		}
	}
	if len(r.RUA) == 0 {
		r.Problems = append(r.Problems, "no-rua")
	}
	return r
}

func (t *MTASTSTool) checkBIMI(ctx context.Context, domain string) BIMI {
	b := BIMI{Problems: []string{}, Warnings: []string{}}
	txts, _, err := t.res.TXT(ctx, "default._bimi."+domain)
	if err != nil {
		b.Problems = append(b.Problems, "dns-error")
		return b
	}
	rec, n := findRecord(txts, "v=BIMI1")
	if n == 0 {
		b.Problems = append(b.Problems, "no-record")
		return b
	}
	b.Record = rec
	tags := tagList(rec)
	b.Logo, b.Certificate = tags["l"], tags["a"]
	if b.Logo == "" {
		b.Warnings = append(b.Warnings, "no-logo")
	} else if !strings.HasPrefix(b.Logo, "https://") || path.Ext(strings.SplitN(b.Logo, "?", 2)[0]) != ".svg" {
		b.Problems = append(b.Problems, "logo-not-https-svg")
	} else if f, err := t.fetch(ctx, b.Logo); err != nil || f.Status != http.StatusOK {
		b.Problems = append(b.Problems, "logo-unreachable")
	} else {
		if !strings.Contains(f.Body, `baseProfile="tiny-ps"`) && !strings.Contains(f.Body, `baseProfile='tiny-ps'`) {
			b.Problems = append(b.Problems, "logo-not-tiny-ps")
		}
		if len(f.Body) > 32<<10 {
			b.Warnings = append(b.Warnings, "logo-too-large")
		}
	}
	if b.Certificate == "" {
		b.Warnings = append(b.Warnings, "no-vmc") // Gmail and Apple Mail require a VMC/CMC
	}
	// BIMI requires an enforced DMARC policy.
	d := CheckDMARC(ctx, t.res, domain)
	if d.Policy != "quarantine" && d.Policy != "reject" || d.Pct < 100 {
		b.Problems = append(b.Problems, "dmarc-not-enforced")
	}
	return b
}
