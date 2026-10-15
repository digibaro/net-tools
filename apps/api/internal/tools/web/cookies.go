package web

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// CookiesTool is WEB-08.
type CookiesTool struct {
	deps tools.Deps
	Now  func() time.Time
}

func (*CookiesTool) ID() string                                { return "web.cookies" }
func (*CookiesTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *CookiesTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(45*time.Second)).Get("/http/cookies", t.handle)
}

// Cookie is one Set-Cookie header, analyzed. Value is masked: cookies can carry session tokens.
type Cookie struct {
	SetBy       string   `json:"setBy"`
	Name        string   `json:"name"`
	Value       string   `json:"value"`
	ValueLength int      `json:"valueLength"`
	Domain      string   `json:"domain"`
	Path        string   `json:"path"`
	Expires     string   `json:"expires"`
	MaxAge      *int     `json:"maxAge"`
	Session     bool     `json:"session"`
	Secure      bool     `json:"secure"`
	HTTPOnly    bool     `json:"httpOnly"`
	SameSite    string   `json:"sameSite"`
	Partitioned bool     `json:"partitioned"`
	Problems    []string `json:"problems"`
	Warnings    []string `json:"warnings"`
}

// CookiesResult is the WEB-08 payload.
type CookiesResult struct {
	URL     string   `json:"url"`
	Cookies []Cookie `json:"cookies"`
}

var sessionName = regexp.MustCompile(`(?i)sess|sid|auth|token|jwt|login|remember`)

func mask(v string) string {
	if len(v) <= 8 {
		return strings.Repeat("•", len(v))
	}
	return v[:6] + "…"
}

// AnalyzeCookie applies the browser rules (RFC 6265bis) that most often break cookies.
func AnalyzeCookie(c *http.Cookie, setBy *url.URL, now time.Time) Cookie {
	out := Cookie{SetBy: setBy.String(), Name: c.Name, Value: mask(c.Value), ValueLength: len(c.Value), Domain: c.Domain, Path: c.Path, Secure: c.Secure, HTTPOnly: c.HttpOnly, Partitioned: c.Partitioned, Problems: []string{}, Warnings: []string{}}
	switch c.SameSite {
	case http.SameSiteStrictMode:
		out.SameSite = "Strict"
	case http.SameSiteLaxMode:
		out.SameSite = "Lax"
	case http.SameSiteNoneMode:
		out.SameSite = "None"
	}
	if c.MaxAge != 0 {
		m := c.MaxAge
		out.MaxAge = &m
	}
	var expiry time.Time
	switch {
	case c.MaxAge > 0:
		expiry = now.Add(time.Duration(c.MaxAge) * time.Second)
	case c.MaxAge < 0:
		out.Warnings = append(out.Warnings, "deleted") // Max-Age<=0 removes the cookie
	case !c.Expires.IsZero():
		expiry = c.Expires
	default:
		out.Session = true
	}
	if !expiry.IsZero() {
		out.Expires = expiry.UTC().Format(time.RFC3339)
		if expiry.Sub(now) > 400*24*time.Hour {
			out.Warnings = append(out.Warnings, "long-lived")
		}
	}
	https := setBy.Scheme == "https"
	if c.SameSite == http.SameSiteNoneMode && !c.Secure {
		out.Problems = append(out.Problems, "samesite-none-insecure")
	}
	if strings.HasPrefix(c.Name, "__Secure-") && (!c.Secure || !https) {
		out.Problems = append(out.Problems, "secure-prefix")
	}
	if strings.HasPrefix(c.Name, "__Host-") && (!c.Secure || !https || c.Domain != "" || c.Path != "/") {
		out.Problems = append(out.Problems, "host-prefix")
	}
	if len(c.Name)+len(c.Value) > 4096 {
		out.Problems = append(out.Problems, "too-large")
	}
	if https && !c.Secure {
		out.Warnings = append(out.Warnings, "not-secure")
	}
	if !https {
		out.Warnings = append(out.Warnings, "plain-http")
	}
	if !c.HttpOnly && sessionName.MatchString(c.Name) {
		out.Warnings = append(out.Warnings, "session-not-httponly")
	}
	if c.SameSite == 0 || c.SameSite == http.SameSiteDefaultMode { // absent or empty: browsers default to Lax
		out.Warnings = append(out.Warnings, "no-samesite")
	}
	if c.Domain != "" {
		out.Warnings = append(out.Warnings, "domain-wide")
	}
	return out
}

func (t *CookiesTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	u := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: u, FollowRedirects: true})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := CookiesResult{URL: hops[len(hops)-1].URL, Cookies: []Cookie{}}
	now := t.Now()
	for _, h := range hops {
		hu, err := url.Parse(h.URL)
		if err != nil {
			continue
		}
		for _, hd := range h.Headers {
			if !strings.EqualFold(hd.Name, "Set-Cookie") || len(out.Cookies) >= 100 {
				continue
			}
			c, err := http.ParseSetCookie(hd.Value)
			if err != nil {
				out.Cookies = append(out.Cookies, Cookie{SetBy: h.URL, Name: "(unparseable)", Value: mask(hd.Value), ValueLength: len(hd.Value), Problems: []string{"invalid"}, Warnings: []string{}})
				continue
			}
			out.Cookies = append(out.Cookies, AnalyzeCookie(c, hu, now))
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: u}})
}
