package web

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// SecHeadersTool is WEB-04: grade a site's security headers.
type SecHeadersTool struct{ deps tools.Deps }

func (*SecHeadersTool) ID() string                                { return "web.secheaders" }
func (*SecHeadersTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *SecHeadersTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(45*time.Second)).Get("/http/security-headers", t.handle)
}

// Check is one rule outcome. Detail is a machine-readable code the UI translates.
type Check struct {
	ID     string `json:"id"`
	Header string `json:"header"`
	Status string `json:"status"` // pass | warn | fail | info
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

// SecResult is the response payload.
type SecResult struct {
	URL    string       `json:"url"`
	Status int          `json:"status"`
	Grade  string       `json:"grade"`
	Score  int          `json:"score"`
	Checks []Check      `json:"checks"`
	Hop    httpcore.Hop `json:"hop"`
}

var maxAgeRe = regexp.MustCompile(`(?i)max-age\s*=\s*"?(\d+)`)

// Analyze grades a final response. https reports whether the page was served over TLS.
func Analyze(h httpcore.Hop, https bool) ([]Check, int) {
	var cs []Check
	score := 100
	add := func(c Check, penalty int) {
		cs = append(cs, c)
		score -= penalty
	}

	// HSTS
	if v, ok := header(h, "Strict-Transport-Security"); !https {
		add(Check{ID: "hsts", Header: "Strict-Transport-Security", Status: "fail", Detail: "not-https"}, 20)
	} else if !ok {
		add(Check{ID: "hsts", Header: "Strict-Transport-Security", Status: "fail", Detail: "missing"}, 20)
	} else {
		age := 0
		if m := maxAgeRe.FindStringSubmatch(v); m != nil {
			age, _ = strconv.Atoi(m[1])
		}
		switch {
		case age < 15552000:
			add(Check{ID: "hsts", Header: "Strict-Transport-Security", Status: "warn", Value: v, Detail: "short-max-age"}, 5)
		case strings.Contains(strings.ToLower(v), "preload") && strings.Contains(strings.ToLower(v), "includesubdomains"):
			add(Check{ID: "hsts", Header: "Strict-Transport-Security", Status: "pass", Value: v, Detail: "preload"}, 0)
		default:
			add(Check{ID: "hsts", Header: "Strict-Transport-Security", Status: "pass", Value: v, Detail: "ok"}, 0)
		}
	}

	// CSP
	csp, hasCSP := header(h, "Content-Security-Policy")
	if !hasCSP {
		detail := "missing"
		if _, ro := header(h, "Content-Security-Policy-Report-Only"); ro {
			detail = "report-only"
		}
		add(Check{ID: "csp", Header: "Content-Security-Policy", Status: "fail", Detail: detail}, 25)
	} else {
		low := strings.ToLower(csp)
		scripts := directive(low, "script-src")
		if scripts == "" {
			scripts = directive(low, "default-src")
		}
		switch {
		case scripts == "":
			add(Check{ID: "csp", Header: "Content-Security-Policy", Status: "warn", Value: csp, Detail: "no-script-src"}, 10)
		case strings.Contains(scripts, "'unsafe-inline'") && !strings.Contains(scripts, "'nonce-") && !strings.Contains(scripts, "'sha"):
			add(Check{ID: "csp", Header: "Content-Security-Policy", Status: "warn", Value: csp, Detail: "unsafe-inline"}, 10)
		case strings.Contains(scripts, "'unsafe-eval'"):
			add(Check{ID: "csp", Header: "Content-Security-Policy", Status: "warn", Value: csp, Detail: "unsafe-eval"}, 5)
		case hasToken(scripts, "*") || hasToken(scripts, "https:") || hasToken(scripts, "http:"):
			add(Check{ID: "csp", Header: "Content-Security-Policy", Status: "warn", Value: csp, Detail: "wildcard"}, 10)
		default:
			add(Check{ID: "csp", Header: "Content-Security-Policy", Status: "pass", Value: csp, Detail: "ok"}, 0)
		}
	}

	// Framing: CSP frame-ancestors supersedes X-Frame-Options.
	xfo, hasXFO := header(h, "X-Frame-Options")
	switch {
	case directive(strings.ToLower(csp), "frame-ancestors") != "":
		add(Check{ID: "framing", Header: "Content-Security-Policy", Status: "pass", Value: "frame-ancestors " + directive(strings.ToLower(csp), "frame-ancestors"), Detail: "frame-ancestors"}, 0)
	case hasXFO && (strings.EqualFold(strings.TrimSpace(xfo), "DENY") || strings.EqualFold(strings.TrimSpace(xfo), "SAMEORIGIN")):
		add(Check{ID: "framing", Header: "X-Frame-Options", Status: "pass", Value: xfo, Detail: "xfo"}, 0)
	default:
		add(Check{ID: "framing", Header: "X-Frame-Options", Status: "fail", Value: xfo, Detail: "missing"}, 15)
	}

	if v, ok := header(h, "X-Content-Type-Options"); ok && strings.EqualFold(strings.TrimSpace(v), "nosniff") {
		add(Check{ID: "nosniff", Header: "X-Content-Type-Options", Status: "pass", Value: v, Detail: "ok"}, 0)
	} else {
		add(Check{ID: "nosniff", Header: "X-Content-Type-Options", Status: "fail", Value: v, Detail: "missing"}, 10)
	}

	switch v, ok := header(h, "Referrer-Policy"); {
	case !ok:
		add(Check{ID: "referrer", Header: "Referrer-Policy", Status: "warn", Detail: "missing"}, 5)
	case strings.Contains(strings.ToLower(v), "unsafe-url") || strings.Contains(strings.ToLower(v), "no-referrer-when-downgrade"):
		add(Check{ID: "referrer", Header: "Referrer-Policy", Status: "warn", Value: v, Detail: "leaky"}, 5)
	default:
		add(Check{ID: "referrer", Header: "Referrer-Policy", Status: "pass", Value: v, Detail: "ok"}, 0)
	}

	if v, ok := header(h, "Permissions-Policy"); ok {
		add(Check{ID: "permissions", Header: "Permissions-Policy", Status: "pass", Value: v, Detail: "ok"}, 0)
	} else {
		add(Check{ID: "permissions", Header: "Permissions-Policy", Status: "warn", Detail: "missing"}, 5)
	}

	for _, iso := range []string{"Cross-Origin-Opener-Policy", "Cross-Origin-Resource-Policy", "Cross-Origin-Embedder-Policy"} {
		if v, ok := header(h, iso); ok {
			add(Check{ID: strings.ToLower(strings.ReplaceAll(iso, "-", "")), Header: iso, Status: "pass", Value: v, Detail: "ok"}, 0)
		} else {
			add(Check{ID: strings.ToLower(strings.ReplaceAll(iso, "-", "")), Header: iso, Status: "info", Detail: "missing"}, 0)
		}
	}

	// Information disclosure.
	for _, leak := range []string{"Server", "X-Powered-By", "X-AspNet-Version", "X-AspNetMvc-Version"} {
		if v, ok := header(h, leak); ok && (leak != "Server" || regexp.MustCompile(`\d`).MatchString(v)) {
			add(Check{ID: "leak", Header: leak, Status: "warn", Value: v, Detail: "version"}, 2)
		}
	}

	// Cookies.
	for _, c := range headers(h, "Set-Cookie") {
		low := strings.ToLower(c)
		name := strings.SplitN(c, "=", 2)[0]
		var missing []string
		if https && !strings.Contains(low, "; secure") {
			missing = append(missing, "Secure")
		}
		if !strings.Contains(low, "httponly") {
			missing = append(missing, "HttpOnly")
		}
		if !strings.Contains(low, "samesite") {
			missing = append(missing, "SameSite")
		}
		if len(missing) > 0 {
			add(Check{ID: "cookie", Header: "Set-Cookie", Status: "warn", Value: name, Detail: strings.Join(missing, ",")}, 3)
		}
	}
	return cs, max(score, 0)
}

func directive(csp, name string) string {
	for _, part := range strings.Split(csp, ";") {
		f := strings.Fields(part)
		if len(f) > 0 && f[0] == name {
			return strings.Join(f[1:], " ")
		}
	}
	return ""
}

func hasToken(list, tok string) bool {
	for _, f := range strings.Fields(list) {
		if f == tok {
			return true
		}
	}
	return false
}

// Grade maps a score to A+ … F.
func Grade(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 85:
		return "A"
	case score >= 70:
		return "B"
	case score >= 55:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}

func (t *SecHeadersTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	u := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: u, Method: "GET", FollowRedirects: true})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	final := hops[len(hops)-1]
	if final.Error != "" {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.Code(final.ErrorCode), Message: final.Error})
		return
	}
	checks, score := Analyze(final, strings.HasPrefix(final.URL, "https://"))
	httpapi.JSON(w, r, SecResult{URL: final.URL, Status: final.Status, Grade: Grade(score), Score: score, Checks: checks, Hop: final}, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: u}})
}
