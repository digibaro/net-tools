package ctsearch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestCTSearch(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		if r.URL.Query().Get("q") == "%.slow.example" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		b, _ := os.ReadFile("testdata/example.com.json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	now := func() time.Time { return time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC) }
	env := tooltest.New(t, &Tool{Base: srv.URL, Now: now}, tooltest.Options{AllowPrivate: true})
	var out Result
	env.Get("/api/v1/ct?domain=*.Example.COM").Must(t).Decode(t, &out)
	if out.Domain != "example.com" || out.Total == 0 || len(out.Certs) != out.Total || !slices.Contains(out.Names, "example.com") || out.Issuers[0].Count == 0 || out.Certs[0].URL == "" {
		t.Errorf("out = %+v", out)
	}
	for i := 1; i < len(out.Certs); i++ {
		if out.Certs[i].NotBefore > out.Certs[i-1].NotBefore {
			t.Fatal("not sorted newest first")
		}
	}
	if query == "" || !strings.Contains(query, "exclude=expired") || !strings.Contains(query, "deduplicate=Y") {
		t.Errorf("query = %s", query)
	}
	env.Get("/api/v1/ct?domain=example.com&includeExpired=true").Must(t)
	if strings.Contains(query, "exclude=expired") {
		t.Errorf("expired not included: %s", query)
	}
	env.Get("/api/v1/ct?domain=localhost").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/ct?domain=slow.example").ExpectError(t, "PROVIDER_ERROR")
}

func TestDedupeAndIssuer(t *testing.T) {
	tool := &Tool{Now: time.Now}
	out := tool.summarize("example.com", []entry{
		{ID: 1, Serial: "aa", IssuerName: "C=US, O=Let's Encrypt, CN=R11", NameValue: "example.com\nwww.example.com\nother.test", NotBefore: "2026-09-01T00:00:00", NotAfter: "2026-12-01T00:00:00"},
		{ID: 2, Serial: "aa", IssuerName: "C=US, O=Let's Encrypt, CN=R11", NameValue: "example.com", NotBefore: "2026-09-01T00:00:00"},
		{ID: 3, Serial: "bb", IssuerName: "CN=Other CA, O=X", NameValue: "api.example.com", NotBefore: "2026-08-01T00:00:00", NotAfter: "2020-01-01T00:00:00"},
	})
	if out.Total != 2 || out.Issuers[0].Name != "Other CA" && out.Issuers[0].Name != "R11" || !slices.Equal(out.Names, []string{"api.example.com", "example.com", "www.example.com"}) || !out.Certs[1].Expired {
		t.Errorf("out = %+v", out)
	}
}
