package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func hop(h map[string]string, extra ...httpcore.Header) httpcore.Hop {
	out := httpcore.Hop{Status: 200, StatusText: "OK"}
	for k, v := range h {
		out.Headers = append(out.Headers, httpcore.Header{Name: k, Value: v})
	}
	out.Headers = append(out.Headers, extra...)
	return out
}

func status(cs []Check, id string) string {
	for _, c := range cs {
		if c.ID == id {
			return c.Status + ":" + c.Detail
		}
	}
	return ""
}

func TestAnalyzeSecurityHeaders(t *testing.T) {
	good := hop(map[string]string{
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains; preload",
		"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'sha256-abc'; frame-ancestors 'none'",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "camera=()",
	})
	cs, score := Analyze(good, true)
	if score != 100 || Grade(score) != "A+" || status(cs, "csp") != "pass:ok" || status(cs, "framing") != "pass:frame-ancestors" || status(cs, "hsts") != "pass:preload" {
		t.Errorf("good: %d %+v", score, cs)
	}
	bad := hop(map[string]string{"Content-Security-Policy": "script-src * 'unsafe-inline'", "Server": "nginx/1.25.3", "X-Powered-By": "PHP/8.3"}, httpcore.Header{Name: "Set-Cookie", Value: "sid=1; Path=/"})
	cs, score = Analyze(bad, false)
	if Grade(score) == "A" || status(cs, "hsts") != "fail:not-https" || status(cs, "csp") != "warn:unsafe-inline" || status(cs, "framing") != "fail:missing" || status(cs, "leak") == "" || status(cs, "cookie") != "warn:HttpOnly,SameSite" {
		t.Errorf("bad: %d %+v", score, cs)
	}
}

func TestCORSEvaluation(t *testing.T) {
	if NeedsPreflight("GET", nil) || !NeedsPreflight("PUT", nil) || !NeedsPreflight("GET", []string{"Authorization"}) {
		t.Error("NeedsPreflight")
	}
	pre := hop(map[string]string{"Access-Control-Allow-Origin": "https://app.example", "Access-Control-Allow-Methods": "GET, PUT", "Access-Control-Allow-Headers": "authorization", "Access-Control-Allow-Credentials": "true"})
	for _, f := range EvaluatePreflight(pre, "https://app.example", "PUT", []string{"Authorization"}, true) {
		if !f.OK {
			t.Errorf("finding %+v", f)
		}
	}
	wild := hop(map[string]string{"Access-Control-Allow-Origin": "*"})
	if fs := EvaluatePreflight(wild, "https://app.example", "GET", nil, true); fs[1].OK {
		t.Error("wildcard with credentials must fail")
	}
	act := EvaluateActual(hop(map[string]string{"Access-Control-Allow-Origin": "https://app.example"}), "https://app.example", false)
	if !act[0].OK || act[1].Code != "vary-origin" || act[1].OK {
		t.Errorf("actual %+v", act)
	}
}

func TestEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/final", http.StatusMovedPermanently)
		case "/echo":
			w.Header().Set("X-Got-Method", r.Method)
			w.Header().Set("X-Got-Header", r.Header.Get("X-Test"))
			b, _ := io.ReadAll(r.Body)
			_, _ = w.Write(b)
		case "/api":
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
				w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Vary", "Origin")
		default:
			w.Header().Set("X-Content-Type-Options", "nosniff")
			_, _ = io.WriteString(w, "final")
		}
	}))
	defer srv.Close()
	opts := tooltest.Options{AllowPrivate: true}

	var hops HopsResult
	tooltest.New(t, &RedirectsTool{}, opts).Get("/api/v1/http/redirects?url="+url.QueryEscape(srv.URL+"/redirect")).Must(t).Decode(t, &hops)
	if len(hops.Hops) != 2 || hops.Hops[0].Status != 301 || hops.Hops[1].Status != 200 {
		t.Errorf("redirects = %+v", hops.Hops)
	}

	tooltest.New(t, &TimingTool{}, opts).Get("/api/v1/http/timing?url="+url.QueryEscape(srv.URL+"/redirect")).Must(t).Decode(t, &hops)
	if len(hops.Hops) != 1 || hops.Hops[0].Timing.TotalMs <= 0 {
		t.Errorf("timing = %+v", hops.Hops)
	}

	body, _ := json.Marshal(RequestBody{URL: srv.URL + "/echo", Method: "PUT", Headers: map[string]string{"X-Test": "yes"}, Body: "payload"})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/http/request", bytes.NewReader(body))
	tooltest.New(t, &RequestTool{}, opts).Do(req).Must(t).Decode(t, &hops)
	h := hops.Hops[0]
	if h.BodyPreview != "payload" || !strings.Contains(headerDump(h), "X-Got-Method=PUT") || !strings.Contains(headerDump(h), "X-Got-Header=yes") {
		t.Errorf("request = %+v", h)
	}

	var sec SecResult
	tooltest.New(t, &SecHeadersTool{}, opts).Get("/api/v1/http/security-headers?url="+url.QueryEscape(srv.URL+"/redirect")).Must(t).Decode(t, &sec)
	if sec.Grade != "F" || status(sec.Checks, "nosniff") != "pass:ok" {
		t.Errorf("sec = %s %+v", sec.Grade, sec.Checks)
	}

	var cors CORSResult
	tooltest.New(t, &CORSTool{}, opts).Get("/api/v1/http/cors?method=DELETE&origin=https://app.example&url="+url.QueryEscape(srv.URL+"/api")).Must(t).Decode(t, &cors)
	if !cors.NeedsPreflight || !cors.Allowed || cors.Actual != nil {
		t.Errorf("cors = %+v", cors)
	}
}

func headerDump(h httpcore.Hop) string {
	var b strings.Builder
	for _, x := range h.Headers {
		b.WriteString(x.Name + "=" + x.Value + "\n")
	}
	return b.String()
}

func TestWebValidationAndSSRF(t *testing.T) {
	opts := tooltest.Options{Lookup: tooltest.StaticLookup("10.0.0.80")}
	for path, code := range map[string]string{
		"/api/v1/http/redirects?url=" + url.QueryEscape("http://127.0.0.1/"):                    "TARGET_PRIVATE_DENIED",
		"/api/v1/http/redirects?url=" + url.QueryEscape("http://169.254.169.254/latest/"):       "TARGET_BLOCKED",
		"/api/v1/http/redirects?url=" + url.QueryEscape("http://intranet.example/"):             "TARGET_PRIVATE_DENIED",
		"/api/v1/http/redirects?url=" + url.QueryEscape("http://[::ffff:127.0.0.1]/"):           "TARGET_PRIVATE_DENIED",
		"/api/v1/http/redirects?url=" + url.QueryEscape("http://2130706433/"):                   "INVALID_INPUT",
		"/api/v1/http/redirects?url=" + url.QueryEscape("file:///etc/passwd"):                   "INVALID_INPUT",
		"/api/v1/http/redirects?url=" + url.QueryEscape("http://user:pw@example.com/"):          "INVALID_INPUT",
		"/api/v1/http/redirects?url=" + url.QueryEscape("gopher://example.com/"):                "INVALID_INPUT",
		"/api/v1/http/timing?url=" + url.QueryEscape("http://0.0.0.0:8081/"):                    "TARGET_BLOCKED",
		"/api/v1/http/security-headers?url=" + url.QueryEscape("http://localhost/"):             "TARGET_PRIVATE_DENIED",
		"/api/v1/http/cors?origin=https://a.example&url=" + url.QueryEscape("http://10.1.1.1/"): "TARGET_PRIVATE_DENIED",
		"/api/v1/http/cors?origin=not-an-origin&url=https://example.com":                        "INVALID_INPUT",
	} {
		var tool tools.Tool
		switch {
		case strings.Contains(path, "/redirects"):
			tool = &RedirectsTool{}
		case strings.Contains(path, "/timing"):
			tool = &TimingTool{}
		case strings.Contains(path, "/security-headers"):
			tool = &SecHeadersTool{}
		default:
			tool = &CORSTool{}
		}
		tooltest.New(t, tool, opts).Get(path).ExpectError(t, code)
	}
	body, _ := json.Marshal(RequestBody{URL: "http://127.0.0.1:8081/api/healthz"})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/http/request", bytes.NewReader(body))
	tooltest.New(t, &RequestTool{}, opts).Do(req).ExpectError(t, "TARGET_PRIVATE_DENIED")
	body, _ = json.Marshal(RequestBody{URL: "https://example.com", Method: "TRACE"})
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/http/request", bytes.NewReader(body))
	tooltest.New(t, &RequestTool{}, opts).Do(req).ExpectError(t, "INVALID_INPUT")
	body, _ = json.Marshal(RequestBody{URL: "https://example.com", Headers: map[string]string{"Transfer-Encoding": "chunked"}})
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/http/request", bytes.NewReader(body))
	tooltest.New(t, &RequestTool{}, opts).Do(req).ExpectError(t, "INVALID_INPUT")
}
