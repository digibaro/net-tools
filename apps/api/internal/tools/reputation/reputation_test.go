package reputation

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func fixtureServer(t *testing.T, route func(r *http.Request) (int, string)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, file := route(r)
		w.WriteHeader(status)
		if file != "" {
			b, _ := os.ReadFile(file)
			_, _ = w.Write(b)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestInternetDB(t *testing.T) {
	base := fixtureServer(t, func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/1.1.1.1":
			return 200, "testdata/internetdb-1.1.1.1.json"
		case "/203.0.113.10":
			return 200, "testdata/internetdb-vuln.json"
		}
		return 404, ""
	})
	env := tooltest.New(t, &InternetDBTool{Base: base}, tooltest.Options{AllowPrivate: true})
	var e Exposure
	env.Get("/api/v1/reputation/internetdb?ip=1.1.1.1").Must(t).Decode(t, &e)
	if !e.Found || len(e.Ports) != 11 || e.Hostnames[0] != "one.one.one.one" || len(e.Vulns) != 0 {
		t.Errorf("e = %+v", e)
	}
	env.Get("/api/v1/reputation/internetdb?ip=203.0.113.10").Must(t).Decode(t, &e)
	if len(e.Vulns) != 2 || e.Tags[0] != "eol-product" {
		t.Errorf("vuln = %+v", e)
	}
	env.Get("/api/v1/reputation/internetdb?ip=8.8.4.4").Must(t).Decode(t, &e)
	if e.Found || e.Ports == nil {
		t.Errorf("unknown = %+v", e)
	}
	for _, bad := range []string{"10.0.0.1", "127.0.0.1", "169.254.169.254", "::ffff:192.168.0.1", "nope"} {
		env.Get("/api/v1/reputation/internetdb?ip="+bad).ExpectError(t, "INVALID_INPUT")
	}
}

func TestAbuseIPDB(t *testing.T) {
	var key string
	base := fixtureServer(t, func(r *http.Request) (int, string) {
		key = r.Header.Get("Key")
		if r.URL.Path != "/check" || r.URL.Query().Get("ipAddress") != "203.0.113.66" {
			return 422, ""
		}
		return 200, "testdata/abuseipdb.json"
	})
	tool := &AbuseIPDBTool{Base: base}
	if ok, reason := tool.Enabled(tools.Deps{Config: tooltest.Config(t, nil)}); ok || reason != tools.ReasonProviderKeyMissing {
		t.Errorf("without key: %v %s", ok, reason)
	}
	env := tooltest.New(t, tool, tooltest.Options{AllowPrivate: true, Env: map[string]string{"ABUSEIPDB_KEY": "secret-key"}})
	var a Abuse
	env.Get("/api/v1/reputation/abuseipdb?ip=203.0.113.66").Must(t).Decode(t, &a)
	if a.Score != 87 || a.TotalReports != 42 || a.Reports[0].Comment != "SSH brute force" || a.Reports[0].Categories[1] != 22 || a.LastReported == "" || key != "secret-key" {
		t.Errorf("a = %+v key=%q", a, key)
	}
	r := env.Get("/api/v1/reputation/abuseipdb?ip=198.51.100.1")
	if r.OK || r.Error.Code != "PROVIDER_ERROR" || strings.Contains(r.Raw.Body.String(), "secret-key") {
		t.Errorf("error = %+v", r.Error)
	}
}
