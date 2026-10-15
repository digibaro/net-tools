package hstspreload

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestHSTSPreload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == "broken.example" && strings.HasSuffix(r.URL.Path, "/preloadable") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if domain == "broken.example" {
			domain = "example.com"
		}
		b, err := os.ReadFile("testdata/" + strings.TrimPrefix(r.URL.Path, "/") + "-" + domain + ".json")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	env := tooltest.New(t, &Tool{Base: srv.URL}, tooltest.Options{AllowPrivate: true})
	var out Result
	env.Get("/api/v1/http/hsts-preload?domain=GitHub.com").Must(t).Decode(t, &out)
	if out.Status != "preloaded" || !out.Preloadable || out.PreloadedDomain != "github.com" {
		t.Errorf("github = %+v", out)
	}
	env.Get("/api/v1/http/hsts-preload?domain=example.com").Must(t).Decode(t, &out)
	if out.Status != "unknown" || out.Preloadable || len(out.Errors) != 2 || out.Errors[0].Code != "response.no_header" {
		t.Errorf("example = %+v", out)
	}
	env.Get("/api/v1/http/hsts-preload?domain=broken.example").Must(t).Decode(t, &out)
	if out.CheckError == "" || out.Preloadable {
		t.Errorf("check failure = %+v", out)
	}
	env.Get("/api/v1/http/hsts-preload?domain=localhost").ExpectError(t, "INVALID_INPUT")
}
