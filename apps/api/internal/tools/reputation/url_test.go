package reputation

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeProviders answers like Safe Browsing, VirusTotal and urlscan.io, keyed on the domain:
// evil.example is flagged everywhere, new.example is unknown, the rest is clean.
func fakeProviders(t *testing.T, seen map[string]string, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.Path] = r.Header.Get("X-Apikey") + r.Header.Get("Api-Key") + r.URL.Query().Get("key")
		mu.Unlock()
		switch {
		case strings.HasPrefix(r.URL.Path, "/sb/threatMatches:find"):
			b, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || !strings.Contains(string(b), `"threatEntryTypes":["URL"]`) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if strings.Contains(string(b), "evil.example") {
				_, _ = w.Write([]byte(`{"matches":[{"threatType":"SOCIAL_ENGINEERING","platformType":"ANY_PLATFORM","threat":{"url":"http://evil.example/"}},{"threatType":"MALWARE","platformType":"ANY_PLATFORM"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		case strings.HasPrefix(r.URL.Path, "/vt/urls/"):
			raw, _ := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(r.URL.Path, "/vt/urls/"))
			if strings.Contains(string(raw), "evil.example") {
				_, _ = w.Write([]byte(`{"data":{"attributes":{"last_analysis_stats":{"malicious":7,"suspicious":1,"harmless":60,"undetected":20},"last_analysis_date":1791936000,"reputation":-12,"categories":{"Forcepoint ThreatSeeker":"phishing"}}}}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"NotFoundError","message":"URL not found"}}`))
		case r.URL.Path == "/vt/domains/example.com":
			_, _ = w.Write([]byte(`{"data":{"attributes":{"last_analysis_stats":{"malicious":0,"suspicious":0,"harmless":66,"undetected":24},"reputation":120}}}`))
		case strings.HasPrefix(r.URL.Path, "/vt/domains/"):
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == "/us/search/":
			q := r.URL.Query().Get("q")
			if q == "page.domain:new.example" {
				_, _ = w.Write([]byte(`{"results":[],"total":0}`))
				return
			}
			_, _ = w.Write([]byte(`{"results":[{"_id":"0190a1b2-0000-4000-8000-000000000001","task":{"time":"2026-10-13T08:00:00.000Z","url":"https://` + strings.TrimPrefix(q, "page.domain:") + `/"}}],"total":3}`))
		case strings.HasPrefix(r.URL.Path, "/us/result/"):
			_, _ = w.Write([]byte(`{"verdicts":{"overall":{"score":0,"malicious":false,"categories":[]}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestURLReputation(t *testing.T) {
	seen := map[string]string{}
	var mu sync.Mutex
	srv := fakeProviders(t, seen, &mu)
	defer srv.Close()
	tool := &URLTool{SafeBrowsingBase: srv.URL + "/sb", VirusTotalBase: srv.URL + "/vt", URLScanBase: srv.URL + "/us"}
	env := tooltest.New(t, tool, tooltest.Options{AllowPrivate: true, Env: map[string]string{"SAFEBROWSING_KEY": "gsb-key", "VIRUSTOTAL_KEY": "vt-key", "URLSCAN_KEY": "us-key"}})

	var res URLReputation
	r := env.Get("/api/v1/reputation/url?url=https://evil.example/login#frag").Must(t)
	r.Decode(t, &res)
	if res.Verdict != VerdictMalicious || res.URL != "https://evil.example/login" || res.Domain != "evil.example" || len(res.Providers) != 3 {
		t.Fatalf("res = %+v", res)
	}
	sb, vt, us := res.Providers[0], res.Providers[1], res.Providers[2]
	if sb.Verdict != VerdictMalicious || sb.Summary != "MALWARE, SOCIAL_ENGINEERING" {
		t.Errorf("sb = %+v", sb)
	}
	if vt.Verdict != VerdictMalicious || vt.Details["malicious"] != "7" || vt.Details["categories"] != "phishing" || !strings.HasPrefix(vt.Link, "https://www.virustotal.com/gui/url/") {
		t.Errorf("vt = %+v", vt)
	}
	if us.Verdict != VerdictClean || us.Details["scans"] != "3" || us.Link != "https://urlscan.io/result/0190a1b2-0000-4000-8000-000000000001/" {
		t.Errorf("us = %+v", us)
	}
	if strings.Contains(r.Raw.Body.String(), "-key") {
		t.Error("a provider key leaked into the response")
	}
	mu.Lock()
	if seen["/sb/threatMatches:find"] != "gsb-key" || seen["/us/search/"] != "us-key" {
		t.Errorf("keys = %v", seen)
	}
	mu.Unlock()

	env.Get("/api/v1/reputation/url?url=example.com").Must(t).Decode(t, &res)
	if res.Verdict != VerdictClean || res.Providers[1].Details["object"] != "domain" || res.Providers[1].Summary != "0/90 engines flag this domain" {
		t.Errorf("clean = %+v", res.Providers)
	}

	env.Get("/api/v1/reputation/url?url=new.example").Must(t).Decode(t, &res)
	if res.Verdict != VerdictClean || res.Providers[1].Verdict != VerdictUnknown || res.Providers[2].Verdict != VerdictUnknown {
		t.Errorf("unknown = %+v", res.Providers)
	}
	for _, bad := range []string{"ftp://x.example", "http://10.0.0.1/", "http://intranet/", "http://user:pw@x.example/", ""} {
		env.Get("/api/v1/reputation/url?url="+bad).ExpectError(t, "INVALID_INPUT")
	}
}

func TestURLReputationPartialKeys(t *testing.T) {
	seen := map[string]string{}
	var mu sync.Mutex
	srv := fakeProviders(t, seen, &mu)
	defer srv.Close()
	tool := &URLTool{SafeBrowsingBase: srv.URL + "/sb", VirusTotalBase: srv.URL + "/vt", URLScanBase: srv.URL + "/us"}
	env := tooltest.New(t, tool, tooltest.Options{AllowPrivate: true, Env: map[string]string{"SAFEBROWSING_KEY": "gsb-key"}})
	var res URLReputation
	env.Get("/api/v1/reputation/url?url=example.com").Must(t).Decode(t, &res)
	b, _ := json.Marshal(res.Providers)
	if res.Providers[0].Verdict != VerdictClean || res.Providers[1].Verdict != VerdictSkipped || res.Providers[2].Verdict != VerdictSkipped {
		t.Errorf("providers = %s", b)
	}
	mu.Lock()
	if _, called := seen["/us/search/"]; called {
		t.Error("urlscan called without a key")
	}
	mu.Unlock()
	if ok, reason := (&URLTool{}).Enabled(tools.Deps{Config: tooltest.Config(t, nil)}); ok || reason != tools.ReasonProviderKeyMissing {
		t.Errorf("no keys: %v %s", ok, reason)
	}
}
