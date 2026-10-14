// Package hstspreload implements WEB-10: a domain's status on the HSTS preload list and whether
// it currently meets the submission requirements (hstspreload.org API).
package hstspreload

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/providers"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Tool is web.hsts.
type Tool struct {
	deps tools.Deps
	// Base overrides https://hstspreload.org/api/v2 (tests).
	Base string
}

func (*Tool) ID() string                                { return "web.hsts" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://hstspreload.org/api/v2"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(30*time.Second)).Get("/http/hsts-preload", t.handle)
}

// Issue is a requirement the domain does not meet (error) or should fix (warning).
type Issue struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
	Message string `json:"message"`
}

// Result is the WEB-10 payload. Status is hstspreload.org's: unknown, pending, preloaded,
// rejected, removed, …
type Result struct {
	Domain          string  `json:"domain"`
	Status          string  `json:"status"`
	PreloadedDomain string  `json:"preloadedDomain"`
	Preloadable     bool    `json:"preloadable"`
	Errors          []Issue `json:"errors"`
	Warnings        []Issue `json:"warnings"`
	CheckError      string  `json:"checkError"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("domain", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	domain, err := target.NormalizeHostname(raw)
	if err != nil || !strings.Contains(domain, ".") {
		httpapi.Error(w, r, httpapi.InvalidInput("domain", "enter a domain name such as example.com"))
		return
	}
	v := url.Values{"domain": {domain}}.Encode()
	var status struct {
		Status          string `json:"status"`
		PreloadedDomain string `json:"preloadedDomain"`
	}
	var check struct {
		Errors   []Issue `json:"errors"`
		Warnings []Issue `json:"warnings"`
	}
	var statusErr, checkErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		statusErr = t.deps.Providers.GetJSON(r.Context(), providers.Request{Provider: "hstspreload.org", URL: t.Base + "/status?" + v, TTL: time.Hour}, &status)
	})
	// The preloadable check makes hstspreload.org fetch the site; it can be slow.
	wg.Go(func() {
		checkErr = t.deps.Providers.GetJSON(r.Context(), providers.Request{Provider: "hstspreload.org", URL: t.Base + "/preloadable?" + v, TTL: 10 * time.Minute, Timeout: 25 * time.Second}, &check)
	})
	wg.Wait()
	if statusErr != nil {
		httpapi.Error(w, r, statusErr)
		return
	}
	out := Result{Domain: domain, Status: status.Status, PreloadedDomain: status.PreloadedDomain, Errors: []Issue{}, Warnings: []Issue{}}
	if checkErr != nil {
		out.CheckError = httpapi.ToAPIError(checkErr).Message
	} else {
		if check.Errors != nil {
			out.Errors = check.Errors
		}
		if check.Warnings != nil {
			out.Warnings = check.Warnings
		}
		out.Preloadable = len(out.Errors) == 0
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "hstspreload.org"})
}
