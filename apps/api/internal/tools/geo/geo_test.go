package geo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"testing"

	"github.com/maxmind/mmdbwriter/mmdbtype"

	"github.com/digibaro/net-tools-docker/apps/api/internal/geoip/geoiptest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func noTXT(context.Context, string) ([]string, error) { return nil, nil }

func TestGeo(t *testing.T) {
	dir := t.TempDir()
	geoiptest.Write(t, dir, "GeoLite2-City.mmdb", "GeoLite2-City", map[string]mmdbtype.Map{
		"1.1.1.0/24": geoiptest.City("Sydney", "AU", "Australia", -33.87, 151.21),
	})
	geoiptest.Write(t, dir, "GeoLite2-ASN.mmdb", "GeoLite2-ASN", map[string]mmdbtype.Map{
		"1.1.1.0/24": geoiptest.ASN(13335, "CLOUDFLARENET"),
	})
	env := tooltest.New(t, &Tool{TXT: noTXT}, tooltest.Options{Env: map[string]string{"DATA_DIR": dir}})
	var res Result
	env.Get("/api/v1/geo?ip=1.1.1.1").Must(t).Decode(t, &res)
	if !res.Found || res.City != "Sydney" || res.CountryCode != "AU" || *res.ASN != 13335 || res.ASNSource != "mmdb" || res.Prefix != "1.1.1.0/24" || res.Database == "" {
		t.Errorf("res = %+v", res)
	}
	env.Get("/api/v1/geo?ip=10.0.0.1").Must(t).Decode(t, &res)
	if res.Public || res.Found {
		t.Errorf("private = %+v", res)
	}
	env.Get("/api/v1/geo?ip=8.8.8.8").Must(t).Decode(t, &res)
	if res.Found || !res.Public {
		t.Errorf("miss = %+v", res)
	}
	env.Get("/api/v1/geo?ip=example.com").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/geo?ip=1.1.1.1&source=ipinfo").ExpectError(t, "PROVIDER_KEY_MISSING")
	env.Get("/api/v1/geo?ip=1.1.1.1&source=ipapi").ExpectError(t, "INVALID_INPUT")
}

// Without a city database the tool is disabled with data_missing; the ASN falls back to Cymru.
func TestGeoDisabledAndCymru(t *testing.T) {
	tool := &Tool{}
	ok, reason := tool.Enabled(tools.Deps{Config: tooltest.Config(t, map[string]string{"DATA_DIR": t.TempDir()})})
	if ok || reason != tools.ReasonDataMissing {
		t.Errorf("enabled = %v %s", ok, reason)
	}
	txt := func(_ context.Context, name string) ([]string, error) {
		return map[string][]string{
			"8.8.8.8.origin.asn.cymru.com": {"15169 | 8.8.8.0/24 | US | arin | 2023-12-28"},
			"AS15169.asn.cymru.com":        {"15169 | US | arin | 2000-03-30 | GOOGLE, US"},
		}[name], nil
	}
	res := Lookup(context.Background(), nil, nil, txt, mustAddr("8.8.8.8"))
	if res.ASN == nil || *res.ASN != 15169 || res.ASNSource != "cymru" {
		t.Errorf("res = %+v", res)
	}
}

func mustAddr(s string) netip.Addr { return netip.MustParseAddr(s) }

func TestGeoIPInfo(t *testing.T) {
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		b, _ := os.ReadFile("testdata/ipinfo-8.8.8.8.json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	// No local database, token configured: ipinfo is the default source.
	tool := &Tool{TXT: noTXT, IPInfoBase: srv.URL}
	env := tooltest.New(t, tool, tooltest.Options{AllowPrivate: true, Env: map[string]string{"DATA_DIR": t.TempDir(), "IPINFO_TOKEN": "tok"}})
	var res Result
	env.Get("/api/v1/geo?ip=8.8.8.8").Must(t).Decode(t, &res)
	if !res.Found || res.City != "Mountain View" || *res.Latitude != 37.4056 || *res.ASN != 15169 || res.ASOrg != "Google LLC" || res.Source != "ipinfo" {
		t.Errorf("res = %+v", res)
	}
	if auth != "Bearer tok" || path != "/8.8.8.8/json" {
		t.Errorf("auth=%q path=%q", auth, path)
	}
	env.Get("/api/v1/geo?ip=8.8.8.8&source=local").ExpectError(t, "TOOL_DISABLED")

	// Database present, no token: asking for ipinfo explains the missing key.
	dir := t.TempDir()
	geoiptest.Write(t, dir, "GeoLite2-City.mmdb", "GeoLite2-City", map[string]mmdbtype.Map{"1.1.1.0/24": geoiptest.City("Sydney", "AU", "Australia", -33.87, 151.21)})
	local := tooltest.New(t, &Tool{TXT: noTXT, IPInfoBase: srv.URL}, tooltest.Options{AllowPrivate: true, Env: map[string]string{"DATA_DIR": dir}})
	local.Get("/api/v1/geo?ip=1.1.1.1&source=ipinfo").ExpectError(t, "PROVIDER_KEY_MISSING")
}
