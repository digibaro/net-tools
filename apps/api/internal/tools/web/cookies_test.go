package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestAnalyzeCookie(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	https, _ := url.Parse("https://example.com/")
	plain, _ := url.Parse("http://example.com/")
	for line, want := range map[string]string{
		"sid=abcdef123456; Path=/; Secure; HttpOnly; SameSite=Lax": "",
		"sid=abcdef123456; Path=/; Secure; SameSite=Lax":           "w:session-not-httponly",
		"pref=1; SameSite=None":                                    "p:samesite-none-insecure",
		"__Host-id=1; Secure; Path=/; Domain=example.com":          "p:host-prefix",
		"__Secure-id=1; Path=/":                                    "p:secure-prefix",
		"theme=dark; Max-Age=63072000; Secure; SameSite=Lax":       "w:long-lived",
		"track=1; Domain=example.com; Secure; SameSite=Lax":        "w:domain-wide",
		"gone=; Max-Age=0; Secure; SameSite=Lax":                   "",
		"x=1; Secure":                                              "w:no-samesite",
	} {
		c, err := http.ParseSetCookie(line)
		if err != nil {
			t.Fatal(err)
		}
		got := AnalyzeCookie(c, https, now)
		switch {
		case want == "" && (len(got.Problems) > 0 || slices.ContainsFunc(got.Warnings, func(w string) bool { return w != "deleted" })):
			t.Errorf("%s: unexpected %v %v", line, got.Problems, got.Warnings)
		case strings.HasPrefix(want, "p:") && !slices.Contains(got.Problems, want[2:]):
			t.Errorf("%s: problems %v, want %s", line, got.Problems, want)
		case strings.HasPrefix(want, "w:") && !slices.Contains(got.Warnings, want[2:]):
			t.Errorf("%s: warnings %v, want %s", line, got.Warnings, want)
		}
	}
	c, _ := http.ParseSetCookie("a=1; HttpOnly; SameSite=Lax")
	if got := AnalyzeCookie(c, plain, now); !slices.Contains(got.Warnings, "plain-http") || !got.Session {
		t.Errorf("plain = %+v", got)
	}
	c, _ = http.ParseSetCookie("token=supersecretvalue123; Secure; HttpOnly; SameSite=Strict")
	if got := AnalyzeCookie(c, https, now); strings.Contains(got.Value, "supersecret") || got.ValueLength != 19 {
		t.Errorf("value not masked: %+v", got)
	}
}

func TestCookiesEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.SetCookie(w, &http.Cookie{Name: "first", Value: "1", Path: "/"})
			http.Redirect(w, r, "/home", http.StatusFound)
			return
		}
		w.Header().Add("Set-Cookie", "sid=abcdefghijk; Path=/; HttpOnly; SameSite=Lax")
		w.Header().Add("Set-Cookie", "pref=dark; SameSite=None")
	}))
	defer srv.Close()
	var out CookiesResult
	tooltest.New(t, &CookiesTool{}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/http/cookies?url="+url.QueryEscape(srv.URL+"/")).Must(t).Decode(t, &out)
	if len(out.Cookies) != 3 || out.Cookies[0].Name != "first" || !strings.HasSuffix(out.URL, "/home") || !slices.Contains(out.Cookies[2].Problems, "samesite-none-insecure") {
		t.Errorf("out = %+v", out)
	}
}
