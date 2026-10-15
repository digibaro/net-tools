package netx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
)

func TestHTTPClientDialsPinnedIPOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/elsewhere")
		_, _ = io.WriteString(w, "host="+r.Host)
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()
	ap := netip.MustParseAddrPort(strings.TrimPrefix(srv.URL, "http://"))
	tgt := target.Target{Host: "pinned.example", IP: ap.Addr()}
	c := HTTPClient(tgt, ClientOptions{})
	// The URL names a host that does not resolve; the pinned dialer must still reach the server.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://does-not-resolve.invalid:"+itoa(ap.Port())+"/", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "host=does-not-resolve.invalid") {
		t.Errorf("body = %q", body)
	}
	if c.Transport.(*http.Transport).Proxy != nil {
		t.Error("proxy env must be ignored")
	}
}

func itoa(p uint16) string { return strconv.Itoa(int(p)) }

func TestParseHTTPURL(t *testing.T) {
	ok := map[string]string{
		"example.com":                   "https://example.com",
		"http://example.com/a?b=1#frag": "http://example.com/a?b=1",
		"https://[2001:db8::1]:8443/":   "https://[2001:db8::1]:8443/",
	}
	for in, want := range ok {
		u, err := ParseHTTPURL(in)
		if err != nil || u.String() != want {
			t.Errorf("%q: %v %v", in, u, err)
		}
	}
	for _, bad := range []string{"", "ftp://example.com", "file:///etc/passwd", "https://user:pw@example.com", "http://", "gopher://x"} {
		if _, err := ParseHTTPURL(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestPortAndReadLimited(t *testing.T) {
	for in, want := range map[string]int{"http://a": 80, "https://a": 443, "https://a:8443": 8443} {
		u, _ := url.Parse(in)
		if Port(u) != want {
			t.Errorf("%s: %d", in, Port(u))
		}
	}
	b, trunc, err := ReadLimited(strings.NewReader("abcdef"), 4)
	if err != nil || string(b) != "abcd" || !trunc {
		t.Errorf("%q %v %v", b, trunc, err)
	}
	b, trunc, _ = ReadLimited(strings.NewReader("ab"), 4)
	if string(b) != "ab" || trunc {
		t.Errorf("%q %v", b, trunc)
	}
}

func TestResolveURLAppliesPolicy(t *testing.T) {
	r := &target.Resolver{Lookup: func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("10.0.0.1")}, nil
	}}
	u, _ := url.Parse("https://intranet.example/")
	if _, err := ResolveURL(context.Background(), r, u); err == nil || !strings.Contains(err.Error(), "TARGET_PRIVATE_DENIED") {
		t.Errorf("err = %v", err)
	}
}
