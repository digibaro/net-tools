package web

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// CORSTool is WEB-06: send a preflight with a given Origin/method/headers and evaluate it.
type CORSTool struct{ deps tools.Deps }

func (*CORSTool) ID() string                                { return "web.cors" }
func (*CORSTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *CORSTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(30*time.Second)).Get("/http/cors", t.handle)
}

// Finding is one CORS evaluation result; Code is translated by the UI.
type Finding struct {
	Code  string `json:"code"`
	OK    bool   `json:"ok"`
	Value string `json:"value"`
}

// CORSResult is the response payload.
type CORSResult struct {
	Allowed         bool          `json:"allowed"`
	NeedsPreflight  bool          `json:"needsPreflight"`
	Findings        []Finding     `json:"findings"`
	Preflight       *httpcore.Hop `json:"preflight"`
	Actual          *httpcore.Hop `json:"actual"`
	RequestedMethod string        `json:"requestedMethod"`
	RequestHeaders  []string      `json:"requestHeaders"`
}

var simpleMethods = []string{"GET", "HEAD", "POST"}
var simpleHeaders = []string{"accept", "accept-language", "content-language"}

// NeedsPreflight reports whether the browser would send a preflight (Fetch standard).
func NeedsPreflight(method string, hdrs []string) bool {
	if !slices.Contains(simpleMethods, method) {
		return true
	}
	for _, h := range hdrs {
		if !slices.Contains(simpleHeaders, strings.ToLower(h)) {
			return true
		}
	}
	return false
}

func listContains(list, item string) bool {
	for _, x := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(x), item) || strings.TrimSpace(x) == "*" {
			return true
		}
	}
	return false
}

// EvaluatePreflight checks a preflight response against the request.
func EvaluatePreflight(h httpcore.Hop, origin, method string, hdrs []string, credentials bool) []Finding {
	var fs []Finding
	fs = append(fs, Finding{Code: "preflight-status", OK: h.Status >= 200 && h.Status < 300, Value: h.StatusText})
	acao, _ := header(h, "Access-Control-Allow-Origin")
	fs = append(fs, Finding{Code: "allow-origin", OK: acao == origin || (acao == "*" && !credentials), Value: acao})
	if credentials {
		acac, _ := header(h, "Access-Control-Allow-Credentials")
		fs = append(fs, Finding{Code: "allow-credentials", OK: acac == "true" && acao != "*", Value: acac})
	}
	if !slices.Contains(simpleMethods, method) {
		acam, _ := header(h, "Access-Control-Allow-Methods")
		fs = append(fs, Finding{Code: "allow-methods", OK: listContains(acam, method) && (!credentials || strings.TrimSpace(acam) != "*"), Value: acam})
	}
	if len(hdrs) > 0 {
		acah, _ := header(h, "Access-Control-Allow-Headers")
		ok := true
		for _, x := range hdrs {
			if !slices.Contains(simpleHeaders, strings.ToLower(x)) && !listContains(acah, x) {
				ok = false
			}
		}
		fs = append(fs, Finding{Code: "allow-headers", OK: ok, Value: acah})
	}
	if ma, ok := header(h, "Access-Control-Max-Age"); ok {
		fs = append(fs, Finding{Code: "max-age", OK: true, Value: ma})
	}
	return fs
}

// EvaluateActual checks the CORS headers on the actual response.
func EvaluateActual(h httpcore.Hop, origin string, credentials bool) []Finding {
	acao, _ := header(h, "Access-Control-Allow-Origin")
	fs := []Finding{{Code: "actual-allow-origin", OK: acao == origin || (acao == "*" && !credentials), Value: acao}}
	if acao == origin && acao != "*" {
		vary := strings.Join(headers(h, "Vary"), ", ")
		fs = append(fs, Finding{Code: "vary-origin", OK: listContains(vary, "Origin"), Value: vary})
	}
	if acao == "null" {
		fs = append(fs, Finding{Code: "null-origin", OK: false, Value: acao})
	}
	return fs
}

func (t *CORSTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	u := q.String("url", "", true, 2048)
	origin := q.String("origin", "", true, 300)
	method := q.Enum("method", "GET", httpcore.Methods...)
	hdrList := q.String("headers", "", false, 1000)
	credentials := q.Bool("credentials", false)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ou, err := url.Parse(origin)
	if err != nil || (ou.Scheme != "http" && ou.Scheme != "https") || ou.Host == "" || (ou.Path != "" && ou.Path != "/") || ou.RawQuery != "" {
		httpapi.Error(w, r, httpapi.InvalidInput("origin", "origin must look like https://app.example.com"))
		return
	}
	origin = ou.Scheme + "://" + ou.Host
	var hdrs []string
	for _, h := range strings.Split(hdrList, ",") {
		if h = strings.TrimSpace(h); h != "" {
			if strings.ContainsAny(h, " :\r\n") || len(hdrs) >= 20 {
				httpapi.Error(w, r, httpapi.InvalidInput("headers", "comma-separated header names, at most 20"))
				return
			}
			hdrs = append(hdrs, h)
		}
	}
	res := CORSResult{NeedsPreflight: NeedsPreflight(method, hdrs), RequestedMethod: method, RequestHeaders: hdrs, Findings: []Finding{}}
	if hdrs == nil {
		res.RequestHeaders = []string{}
	}
	ctx := r.Context()
	if res.NeedsPreflight {
		pre := map[string]string{"Origin": origin, "Access-Control-Request-Method": method}
		if len(hdrs) > 0 {
			pre["Access-Control-Request-Headers"] = strings.ToLower(strings.Join(hdrs, ","))
		}
		hops, err := fetch(ctx, t.deps, httpcore.Request{URL: u, Method: "OPTIONS", Headers: pre})
		if err != nil {
			httpapi.Error(w, r, err)
			return
		}
		res.Preflight = &hops[0]
		res.Findings = append(res.Findings, EvaluatePreflight(hops[0], origin, method, hdrs, credentials)...)
	}
	// The actual request is only sent for safe methods so the tester never causes side effects.
	if method == "GET" || method == "HEAD" {
		hops, err := fetch(ctx, t.deps, httpcore.Request{URL: u, Method: method, Headers: map[string]string{"Origin": origin}})
		if err != nil {
			httpapi.Error(w, r, err)
			return
		}
		res.Actual = &hops[0]
		res.Findings = append(res.Findings, EvaluateActual(hops[0], origin, credentials)...)
	}
	res.Allowed = len(res.Findings) > 0
	for _, f := range res.Findings {
		if !f.OK && f.Code != "max-age" && f.Code != "vary-origin" {
			res.Allowed = false
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: u}})
}
