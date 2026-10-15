package web

import (
	"net/http"
	"slices"
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

// CacheTool is WEB-17: how browsers and shared caches (CDNs, proxies) may cache a response
// (RFC 9111), plus a conditional repeat request to test validators and cache hits.
type CacheTool struct{ deps tools.Deps }

func (*CacheTool) ID() string                                { return "web.cache" }
func (*CacheTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *CacheTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(40*time.Second)).Get("/http/cache", t.handle)
}

// Directive is one Cache-Control directive.
type Directive struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CacheFinding is one observation; Severity is info, warn or bad.
type CacheFinding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

// Repeat is the conditional second request.
type Repeat struct {
	Sent        string `json:"sent"`
	Status      int    `json:"status"`
	CacheStatus string `json:"cacheStatus"`
	Age         string `json:"age"`
}

// CacheResult is the WEB-17 payload.
type CacheResult struct {
	URL             string         `json:"url"`
	Status          int            `json:"status"`
	CacheControl    string         `json:"cacheControl"`
	Directives      []Directive    `json:"directives"`
	BrowserTTL      int64          `json:"browserTtl"`
	SharedTTL       int64          `json:"sharedTtl"`
	TTLSource       string         `json:"ttlSource"`
	BrowserCachable bool           `json:"browserCacheable"`
	SharedCachable  bool           `json:"sharedCacheable"`
	Age             string         `json:"age"`
	ETag            string         `json:"etag"`
	LastModified    string         `json:"lastModified"`
	Expires         string         `json:"expires"`
	Vary            string         `json:"vary"`
	CacheStatus     string         `json:"cacheStatus"`
	Repeat          *Repeat        `json:"repeat"`
	Findings        []CacheFinding `json:"findings"`
}

// ParseCacheControl splits a Cache-Control value into lower-case directives.
func ParseCacheControl(v string) []Directive {
	out := []Directive{}
	for _, part := range strings.Split(v, ",") {
		k, val, _ := strings.Cut(strings.TrimSpace(part), "=")
		if k = strings.ToLower(strings.TrimSpace(k)); k != "" {
			out = append(out, Directive{Name: k, Value: strings.Trim(strings.TrimSpace(val), `"`)})
		}
	}
	return out
}

func ccDirective(ds []Directive, name string) (string, bool) {
	for _, d := range ds {
		if d.Name == name {
			return d.Value, true
		}
	}
	return "", false
}

// Statuses cacheable without explicit freshness (RFC 9110 §15.1).
var heuristicStatuses = []int{200, 203, 204, 206, 300, 301, 308, 404, 405, 410, 414, 501}

// AnalyzeCache fills the policy fields and findings from the response headers.
func AnalyzeCache(h httpcore.Hop) CacheResult {
	res := CacheResult{URL: h.URL, Status: h.Status, Findings: []CacheFinding{}, BrowserTTL: -1, SharedTTL: -1, TTLSource: "none"}
	add := func(id, sev, detail string) {
		res.Findings = append(res.Findings, CacheFinding{ID: id, Severity: sev, Detail: detail})
	}
	res.CacheControl = strings.Join(headers(h, "Cache-Control"), ", ")
	res.Directives = ParseCacheControl(res.CacheControl)
	res.Age, _ = header(h, "Age")
	res.ETag, _ = header(h, "ETag")
	res.LastModified, _ = header(h, "Last-Modified")
	res.Expires, _ = header(h, "Expires")
	res.Vary = strings.Join(headers(h, "Vary"), ", ")
	res.CacheStatus = cacheStatus(h)
	ds := res.Directives
	_, noStore := ccDirective(ds, "no-store")
	_, private := ccDirective(ds, "private")
	_, noCache := ccDirective(ds, "no-cache")
	_, public := ccDirective(ds, "public")
	res.BrowserCachable = !noStore
	res.SharedCachable = !noStore && !private
	dateStr, _ := header(h, "Date")
	date, dateErr := http.ParseTime(dateStr)
	if dateErr != nil {
		date = time.Now()
	}
	seconds := func(name string) (int64, bool) {
		v, ok := ccDirective(ds, name)
		if !ok {
			return 0, false
		}
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil && n >= 0
	}
	switch {
	case noStore:
	case func() bool { _, ok := seconds("max-age"); return ok }():
		res.BrowserTTL, _ = seconds("max-age")
		res.SharedTTL, res.TTLSource = res.BrowserTTL, "max-age"
	case res.Expires != "":
		if exp, err := http.ParseTime(res.Expires); err == nil {
			res.BrowserTTL = max(0, int64(exp.Sub(date)/time.Second))
		} else {
			res.BrowserTTL = 0 // invalid Expires means already expired
			add("expiresInvalid", "warn", res.Expires)
		}
		res.SharedTTL, res.TTLSource = res.BrowserTTL, "expires"
	case res.LastModified != "" && slices.Contains(heuristicStatuses, h.Status):
		if lm, err := http.ParseTime(res.LastModified); err == nil && date.After(lm) {
			res.BrowserTTL = int64(date.Sub(lm)/time.Second) / 10
			res.SharedTTL, res.TTLSource = res.BrowserTTL, "heuristic"
			add("heuristic", "warn", "")
		}
	}
	if s, ok := seconds("s-maxage"); ok && !noStore {
		res.SharedTTL, res.TTLSource = s, res.TTLSource+"+s-maxage"
		res.TTLSource = strings.TrimPrefix(res.TTLSource, "none+")
	}
	if private {
		res.SharedTTL = -1
	}
	switch {
	case noStore:
		add("noStore", "info", "")
	case noCache:
		add("noCache", "info", "")
	case res.TTLSource == "max-age" && res.BrowserTTL == 0:
		add("maxAgeZero", "info", "")
	case res.TTLSource == "none":
		add("noLifetime", "warn", "")
	}
	if private {
		add("private", "info", "")
	}
	if public && private {
		add("publicPrivate", "bad", "")
	}
	if _, ok := ccDirective(ds, "immutable"); ok {
		add("immutable", "info", "")
	}
	if !noStore && res.ETag == "" && res.LastModified == "" {
		add("noValidators", "warn", "")
	}
	vary := strings.ToLower(res.Vary)
	if strings.Contains(vary, "*") {
		add("varyStar", "bad", res.Vary)
	} else if strings.Contains(vary, "cookie") || strings.Contains(vary, "user-agent") {
		add("varyFragment", "warn", res.Vary)
	}
	if res.SharedCachable && res.SharedTTL > 0 && len(headers(h, "Set-Cookie")) > 0 {
		add("setCookieShared", "bad", "")
	}
	if p, ok := header(h, "Pragma"); ok && res.CacheControl == "" && strings.Contains(strings.ToLower(p), "no-cache") {
		add("pragmaOnly", "warn", "")
	}
	return res
}

func (t *CacheTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: raw, Method: "GET", FollowRedirects: true})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	last := hops[len(hops)-1]
	res := AnalyzeCache(last)
	// Repeat with validators: a 304 shows revalidation works; a HIT shows a shared cache stored it.
	cond := map[string]string{}
	switch {
	case res.ETag != "":
		cond["If-None-Match"] = res.ETag
	case res.LastModified != "":
		cond["If-Modified-Since"] = res.LastModified
	}
	if again, err := fetch(r.Context(), t.deps, httpcore.Request{URL: last.URL, Method: "GET", Headers: cond}); err == nil {
		h2 := again[len(again)-1]
		rep := &Repeat{Status: h2.Status, CacheStatus: cacheStatus(h2)}
		rep.Age, _ = header(h2, "Age")
		for k := range cond {
			rep.Sent = k
		}
		res.Repeat = rep
		if rep.Sent != "" && h2.Status != http.StatusNotModified {
			res.Findings = append(res.Findings, CacheFinding{ID: "no304", Severity: "warn", Detail: rep.Sent})
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}
