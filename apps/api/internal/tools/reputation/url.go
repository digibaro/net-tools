package reputation

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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

// ---------- REP-05 URL / domain reputation ----------

// Provider key names.
const (
	keySafeBrowsing = "SAFEBROWSING_KEY"
	keyVirusTotal   = "VIRUSTOTAL_KEY"
	keyURLScan      = "URLSCAN_KEY"
)

// URLTool is rep.url: Google Safe Browsing, VirusTotal and urlscan.io lookups (passive: nothing
// is submitted for scanning). Enabled when at least one of the keys is configured.
type URLTool struct {
	deps tools.Deps
	// Base URLs (tests override them).
	SafeBrowsingBase, VirusTotalBase, URLScanBase string
}

func (*URLTool) ID() string { return "rep.url" }
func (*URLTool) Enabled(d tools.Deps) (bool, tools.Reason) {
	for _, k := range []string{keySafeBrowsing, keyVirusTotal, keyURLScan} {
		if d.Config.HasSecret(k) {
			return true, ""
		}
	}
	return false, tools.ReasonProviderKeyMissing
}

func (t *URLTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.SafeBrowsingBase == "" {
		t.SafeBrowsingBase = "https://safebrowsing.googleapis.com/v4"
	}
	if t.VirusTotalBase == "" {
		t.VirusTotalBase = "https://www.virustotal.com/api/v3"
	}
	if t.URLScanBase == "" {
		t.URLScanBase = "https://urlscan.io/api/v1"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(20*time.Second)).Get("/reputation/url", t.handle)
}

// Verdict values.
const (
	VerdictClean      = "clean"
	VerdictSuspicious = "suspicious"
	VerdictMalicious  = "malicious"
	VerdictUnknown    = "unknown" // provider has no data
	VerdictSkipped    = "skipped" // no key configured
	VerdictError      = "error"
)

// ProviderVerdict is one provider's answer.
type ProviderVerdict struct {
	Provider string            `json:"provider"`
	Verdict  string            `json:"verdict"`
	Summary  string            `json:"summary"`
	Details  map[string]string `json:"details"`
	Link     string            `json:"link"`
	Error    string            `json:"error"`
}

// URLReputation is the REP-05 payload.
type URLReputation struct {
	Input     string            `json:"input"`
	URL       string            `json:"url"`
	Domain    string            `json:"domain"`
	Verdict   string            `json:"verdict"`
	Providers []ProviderVerdict `json:"providers"`
}

// ParseURLInput accepts an http(s) URL or a bare domain and returns the URL to check and its host.
func ParseURLInput(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	bad := httpapi.InvalidInput("url", "enter a URL (https://example.com/path) or a domain name")
	if raw == "" || len(raw) > 2048 {
		return "", "", bad
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return "", "", bad
	}
	host, err := target.NormalizeHostname(u.Hostname())
	if err != nil || !strings.Contains(host, ".") {
		return "", "", httpapi.InvalidInput("url", "only public domain names can be checked (no IP addresses or internal names)")
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), host, nil
}

func (t *URLTool) key(name string) string { return t.deps.Config.Features.ProviderSecrets[name] }

func (t *URLTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	u, host, err := ParseURLInput(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	checks := []struct {
		name, key string
		run       func(context.Context, string, string) ProviderVerdict
	}{
		{"Google Safe Browsing", keySafeBrowsing, t.safeBrowsing},
		{"VirusTotal", keyVirusTotal, t.virusTotal},
		{"urlscan.io", keyURLScan, t.urlscan},
	}
	out := make([]ProviderVerdict, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		if t.key(c.key) == "" {
			out[i] = ProviderVerdict{Provider: c.name, Verdict: VerdictSkipped, Details: map[string]string{}}
			continue
		}
		wg.Go(func() {
			v := c.run(ctx, u, host)
			v.Provider = c.name
			if v.Details == nil {
				v.Details = map[string]string{}
			}
			out[i] = v
		})
	}
	wg.Wait()
	httpapi.JSON(w, r, URLReputation{Input: raw, URL: u, Domain: host, Verdict: overall(out), Providers: out}, &httpapi.Meta{Tool: t.ID(), Vantage: "providers"})
}

// overall is the worst verdict any provider gave.
func overall(vs []ProviderVerdict) string {
	rank := map[string]int{VerdictMalicious: 4, VerdictSuspicious: 3, VerdictClean: 2, VerdictUnknown: 1}
	best := VerdictUnknown
	for _, v := range vs {
		if rank[v.Verdict] > rank[best] {
			best = v.Verdict
		}
	}
	return best
}

func providerError(err error) ProviderVerdict {
	msg := "lookup failed"
	if m, ok := err.(interface{ APIMessage() string }); ok { //nolint:errorlint // client-safe message
		msg = m.APIMessage()
	}
	return ProviderVerdict{Verdict: VerdictError, Error: msg}
}

// Safe Browsing v4 threatMatches:find.
func (t *URLTool) safeBrowsing(ctx context.Context, u, _ string) ProviderVerdict {
	body := map[string]any{
		"client": map[string]string{"clientId": "nettools", "clientVersion": "1"},
		"threatInfo": map[string]any{
			"threatTypes":      []string{"MALWARE", "SOCIAL_ENGINEERING", "UNWANTED_SOFTWARE", "POTENTIALLY_HARMFUL_APPLICATION"},
			"platformTypes":    []string{"ANY_PLATFORM"},
			"threatEntryTypes": []string{"URL"},
			"threatEntries":    []map[string]string{{"url": u}},
		},
	}
	var resp struct {
		Matches []struct {
			ThreatType   string `json:"threatType"`
			PlatformType string `json:"platformType"`
		} `json:"matches"`
	}
	req := providers.Request{Provider: "Google Safe Browsing", Method: http.MethodPost, URL: t.SafeBrowsingBase + "/threatMatches:find?key=" + url.QueryEscape(t.key(keySafeBrowsing)), Body: body}
	if err := t.deps.Providers.GetJSON(ctx, req, &resp); err != nil {
		return providerError(err)
	}
	if len(resp.Matches) == 0 {
		return ProviderVerdict{Verdict: VerdictClean, Summary: "not on any Safe Browsing list"}
	}
	types := map[string]bool{}
	for _, m := range resp.Matches {
		types[m.ThreatType] = true
	}
	list := make([]string, 0, len(types))
	for k := range types {
		list = append(list, k)
	}
	sort.Strings(list)
	return ProviderVerdict{Verdict: VerdictMalicious, Summary: strings.Join(list, ", "), Details: map[string]string{"threatTypes": strings.Join(list, ", ")}}
}

type vtObject struct {
	Data struct {
		Attributes struct {
			Stats struct {
				Malicious  int `json:"malicious"`
				Suspicious int `json:"suspicious"`
				Harmless   int `json:"harmless"`
				Undetected int `json:"undetected"`
			} `json:"last_analysis_stats"`
			LastAnalysisDate int64             `json:"last_analysis_date"`
			Reputation       int               `json:"reputation"`
			Categories       map[string]string `json:"categories"`
		} `json:"attributes"`
	} `json:"data"`
}

// VirusTotal v3: URL object (id = unpadded base64url of the URL), falling back to the domain.
func (t *URLTool) virusTotal(ctx context.Context, u, host string) ProviderVerdict {
	h := http.Header{"X-Apikey": {t.key(keyVirusTotal)}}
	get := func(path string) (vtObject, error) {
		var o vtObject
		err := t.deps.Providers.GetJSON(ctx, providers.Request{Provider: "VirusTotal", URL: t.VirusTotalBase + path, Header: h, TTL: 30 * time.Minute}, &o)
		return o, err //nolint:wrapcheck // provider errors are client-safe
	}
	id := base64.RawURLEncoding.EncodeToString([]byte(u))
	kind, link := "URL", "https://www.virustotal.com/gui/url/"+id
	o, err := get("/urls/" + id)
	if providers.IsNotFound(err) {
		kind, link = "domain", "https://www.virustotal.com/gui/domain/"+host
		o, err = get("/domains/" + url.PathEscape(host))
	}
	if providers.IsNotFound(err) {
		return ProviderVerdict{Verdict: VerdictUnknown, Summary: "not in VirusTotal"}
	}
	if err != nil {
		return providerError(err)
	}
	a := o.Data.Attributes
	s := a.Stats
	v := VerdictClean
	switch {
	case s.Malicious >= 2:
		v = VerdictMalicious
	case s.Malicious == 1 || s.Suspicious > 0:
		v = VerdictSuspicious
	case s.Harmless+s.Undetected == 0:
		v = VerdictUnknown
	}
	d := map[string]string{"object": kind, "malicious": strconv.Itoa(s.Malicious), "suspicious": strconv.Itoa(s.Suspicious), "harmless": strconv.Itoa(s.Harmless), "undetected": strconv.Itoa(s.Undetected), "reputation": strconv.Itoa(a.Reputation)}
	if a.LastAnalysisDate > 0 {
		d["lastAnalysis"] = time.Unix(a.LastAnalysisDate, 0).UTC().Format(time.RFC3339)
	}
	cats := map[string]bool{}
	for _, c := range a.Categories {
		cats[c] = true
	}
	if len(cats) > 0 {
		list := make([]string, 0, len(cats))
		for c := range cats {
			list = append(list, c)
		}
		sort.Strings(list)
		d["categories"] = strings.Join(list, ", ")
	}
	total := s.Malicious + s.Suspicious + s.Harmless + s.Undetected
	return ProviderVerdict{Verdict: v, Summary: strconv.Itoa(s.Malicious) + "/" + strconv.Itoa(total) + " engines flag this " + kind, Details: d, Link: link}
}

// urlscan.io: search existing public scans of the domain (no new scan is submitted), then read
// the newest result's verdict.
func (t *URLTool) urlscan(ctx context.Context, _ string, host string) ProviderVerdict {
	h := http.Header{"Api-Key": {t.key(keyURLScan)}}
	var search struct {
		Results []struct {
			ID   string `json:"_id"`
			Task struct {
				Time string `json:"time"`
				URL  string `json:"url"`
			} `json:"task"`
		} `json:"results"`
		Total int `json:"total"`
	}
	q := url.Values{"q": {"page.domain:" + host}, "size": {"10"}}
	if err := t.deps.Providers.GetJSON(ctx, providers.Request{Provider: "urlscan.io", URL: t.URLScanBase + "/search/?" + q.Encode(), Header: h, TTL: 30 * time.Minute}, &search); err != nil {
		return providerError(err)
	}
	if len(search.Results) == 0 {
		return ProviderVerdict{Verdict: VerdictUnknown, Summary: "no public scans of this domain"}
	}
	latest := search.Results[0]
	d := map[string]string{"scans": strconv.Itoa(search.Total), "latestScan": latest.Task.Time, "latestURL": latest.Task.URL}
	link := "https://urlscan.io/result/" + url.PathEscape(latest.ID) + "/"
	var res struct {
		Verdicts struct {
			Overall struct {
				Score      int      `json:"score"`
				Malicious  bool     `json:"malicious"`
				Categories []string `json:"categories"`
			} `json:"overall"`
		} `json:"verdicts"`
	}
	if err := t.deps.Providers.GetJSON(ctx, providers.Request{Provider: "urlscan.io", URL: t.URLScanBase + "/result/" + url.PathEscape(latest.ID) + "/", Header: h, TTL: time.Hour}, &res); err != nil {
		return ProviderVerdict{Verdict: VerdictUnknown, Summary: strconv.Itoa(search.Total) + " public scans; verdict unavailable", Details: d, Link: link}
	}
	o := res.Verdicts.Overall
	d["score"] = strconv.Itoa(o.Score)
	if len(o.Categories) > 0 {
		d["categories"] = strings.Join(o.Categories, ", ")
	}
	v := VerdictClean
	switch {
	case o.Malicious:
		v = VerdictMalicious
	case o.Score > 0:
		v = VerdictSuspicious
	}
	return ProviderVerdict{Verdict: v, Summary: "latest of " + strconv.Itoa(search.Total) + " public scans, score " + strconv.Itoa(o.Score), Details: d, Link: link}
}
