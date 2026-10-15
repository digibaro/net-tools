// Package observatory adds the Mozilla HTTP Observatory grade as an option of the security
// headers analyzer (WEB-04).
package observatory

import (
	"encoding/json"
	"net/http"
	"net/url"
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

// Tool is web.observatory.
type Tool struct {
	deps tools.Deps
	// Base overrides https://observatory-api.mdn.mozilla.net/api/v2 (tests).
	Base string
}

func (*Tool) ID() string                                { return "web.observatory" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Base == "" {
		t.Base = "https://observatory-api.mdn.mozilla.net/api/v2"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/http/observatory", t.handle)
}

// Result is the Observatory grade.
type Result struct {
	Host        string `json:"host"`
	Grade       string `json:"grade"`
	Score       int    `json:"score"`
	TestsPassed int    `json:"testsPassed"`
	TestsFailed int    `json:"testsFailed"`
	ScannedAt   string `json:"scannedAt"`
	DetailsURL  string `json:"detailsUrl"`
}

func upstreamMessage(b []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(b, &e) == nil {
		return e.Message
	}
	return ""
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("host", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	host, err := target.NormalizeHostname(raw)
	if err != nil || !strings.Contains(host, ".") {
		httpapi.Error(w, r, httpapi.InvalidInput("host", "enter a public host name such as example.com"))
		return
	}
	var d struct {
		Grade       *string `json:"grade"`
		Score       int     `json:"score"`
		TestsPassed int     `json:"tests_passed"`
		TestsFailed int     `json:"tests_failed"`
		ScannedAt   string  `json:"scanned_at"`
		DetailsURL  string  `json:"details_url"`
		Error       *string `json:"error"`
	}
	// POST starts a scan or returns the recent one (the Observatory caches per host for minutes).
	err = t.deps.Providers.GetJSON(r.Context(), providers.Request{
		Provider: "Mozilla Observatory", Method: http.MethodPost, URL: t.Base + "/scan?" + url.Values{"host": {host}}.Encode(),
		Timeout: providers.MaxTimeout, UpstreamMessage: upstreamMessage,
	}, &d)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if d.Error != nil || d.Grade == nil {
		msg := "the scan did not produce a grade"
		if d.Error != nil {
			msg = *d.Error
		}
		httpapi.Error(w, r, &providers.Error{Provider: "Mozilla Observatory", Message: "Mozilla Observatory: " + msg})
		return
	}
	httpapi.JSON(w, r, Result{Host: host, Grade: *d.Grade, Score: d.Score, TestsPassed: d.TestsPassed, TestsFailed: d.TestsFailed, ScannedAt: d.ScannedAt, DetailsURL: d.DetailsURL}, &httpapi.Meta{Tool: t.ID(), Vantage: "observatory"})
}
