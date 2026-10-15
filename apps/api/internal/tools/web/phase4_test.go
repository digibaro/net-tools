package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestDetect(t *testing.T) {
	h := []httpcore.Header{{Name: "Server", Value: "cloudflare"}, {Name: "CF-RAY", Value: "8a1b-AMS"}, {Name: "Set-Cookie", Value: "__cf_bm=abc; path=/"}, {Name: "Set-Cookie", Value: "visid_incap_123=x"}, {Name: "X-Goog-Meta", Value: "1"}}
	got := Detect(h, []string{"www.example.com.cdn.cloudflare.net"}, "C=US, O=Let's Encrypt, CN=R11")
	names := map[string]int{}
	for _, d := range got {
		names[d.Name] = len(d.Evidence)
	}
	if names["Cloudflare"] != 4 || names["Imperva / Incapsula"] != 1 || names["Let's Encrypt certificate"] != 1 || names["Google Cloud CDN / Front End"] != 1 {
		t.Errorf("got %+v", got)
	}
	if len(Detect([]httpcore.Header{{Name: "Server", Value: "nginx"}}, nil, "")) != 0 {
		t.Error("plain nginx detected as something")
	}
}

func TestCDNHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Amz-Cf-Id", "abc")
		w.Header().Set("X-Cache", "Hit from cloudfront")
	}))
	defer srv.Close()
	var res CDNResult
	tooltest.New(t, &CDNTool{Resolver: "9.9.9.9:9"}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/http/cdn?url="+url.QueryEscape(srv.URL)).Must(t).Decode(t, &res)
	if len(res.Detections) != 1 || res.Detections[0].Name != "Amazon CloudFront" || res.CacheStatus != "X-Cache: Hit from cloudfront" {
		t.Errorf("res = %+v", res)
	}
}

func cacheHop(status int, kv ...string) httpcore.Hop {
	h := httpcore.Hop{Status: status}
	for i := 0; i+1 < len(kv); i += 2 {
		h.Headers = append(h.Headers, httpcore.Header{Name: kv[i], Value: kv[i+1]})
	}
	return h
}

func findings(r CacheResult) string {
	var ids []string
	for _, f := range r.Findings {
		ids = append(ids, f.ID)
	}
	return strings.Join(ids, " ")
}

func TestAnalyzeCache(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	date := now.Format(http.TimeFormat)
	r := AnalyzeCache(cacheHop(200, "Cache-Control", "public, max-age=600, s-maxage=3600", "ETag", `"v1"`, "Date", date))
	if r.BrowserTTL != 600 || r.SharedTTL != 3600 || r.TTLSource != "max-age+s-maxage" || !r.SharedCachable || findings(r) != "" {
		t.Errorf("public = %+v (%s)", r, findings(r))
	}
	r = AnalyzeCache(cacheHop(200, "Cache-Control", "private, max-age=60", "Set-Cookie", "a=b", "Vary", "Cookie"))
	if r.SharedCachable || r.SharedTTL != -1 || r.BrowserTTL != 60 || findings(r) != "private noValidators varyFragment" {
		t.Errorf("private = %+v (%s)", r, findings(r))
	}
	r = AnalyzeCache(cacheHop(200, "Cache-Control", "max-age=300", "Set-Cookie", "session=1", "Last-Modified", date))
	if !strings.Contains(findings(r), "setCookieShared") {
		t.Errorf("set-cookie = %s", findings(r))
	}
	r = AnalyzeCache(cacheHop(200, "Date", date, "Last-Modified", now.Add(-100*time.Hour).Format(http.TimeFormat)))
	if r.TTLSource != "heuristic" || r.BrowserTTL != 36000 {
		t.Errorf("heuristic = %+v", r)
	}
	r = AnalyzeCache(cacheHop(200, "Cache-Control", "no-store", "Vary", "*"))
	if r.BrowserCachable || findings(r) != "noStore varyStar" {
		t.Errorf("no-store = %+v (%s)", r, findings(r))
	}
	r = AnalyzeCache(cacheHop(200, "Expires", "0", "Pragma", "no-cache"))
	if r.BrowserTTL != 0 || !strings.Contains(findings(r), "expiresInvalid") || !strings.Contains(findings(r), "pragmaOnly") {
		t.Errorf("expires 0 = %+v (%s)", r, findings(r))
	}
}

func TestCacheHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("ETag", `"x"`)
		if r.Header.Get("If-None-Match") == `"x"` {
			w.WriteHeader(http.StatusNotModified)
		}
	}))
	defer srv.Close()
	var res CacheResult
	tooltest.New(t, &CacheTool{}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/http/cache?url="+url.QueryEscape(srv.URL)).Must(t).Decode(t, &res)
	if res.Repeat == nil || res.Repeat.Sent != "If-None-Match" || res.Repeat.Status != 304 || res.BrowserTTL != 60 {
		t.Errorf("res = %+v %+v", res, res.Repeat)
	}
}

const mixedPage = `<!doctype html><html><head>
<link rel="stylesheet" href="http://cdn.example/a.css"><link rel="canonical" href="http://example.com/">
<link rel="icon" href="http://example.com/favicon.ico">
<script src="//cdn.example/ok.js"></script><script src="http://cdn.example/bad.js"></script>
<style>body { background: url("http://img.example/bg.png") }</style></head><body>
<img src="http://img.example/a.png" srcset="https://img.example/b.png 2x, http://img.example/c.png 3x">
<form action="http://example.com/login"></form><a href="http://example.com/page">x</a>
<iframe src="/relative"></iframe></body></html>`

func TestFindMixed(t *testing.T) {
	doc, _ := html.Parse(strings.NewReader(mixedPage))
	base, _ := url.Parse("https://example.com/")
	items, resources, upgrade := FindMixed(doc, base)
	kinds := map[string]int{}
	for _, it := range items {
		kinds[it.Kind]++
	}
	if kinds["active"] != 2 || kinds["passive"] != 4 || kinds["form"] != 1 || kinds["link"] != 1 || upgrade || resources < 10 {
		t.Errorf("kinds = %v (%d resources) items %+v", kinds, resources, items)
	}
}

func TestMixedHandler(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Security-Policy", "upgrade-insecure-requests")
		_, _ = w.Write([]byte(mixedPage))
	}))
	defer srv.Close()
	env := tooltest.New(t, &MixedTool{}, tooltest.Options{AllowPrivate: true})
	var res MixedResult
	env.Get("/api/v1/http/mixed-content?url="+url.QueryEscape(srv.URL)).Must(t).Decode(t, &res)
	if res.Active != 2 || res.Forms != 1 || !res.UpgradeInsecureRequests {
		t.Errorf("res = %+v", res)
	}
	plain := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer plain.Close()
	env.Get("/api/v1/http/mixed-content?url="+url.QueryEscape(plain.URL)).ExpectError(t, "INVALID_INPUT")
}
