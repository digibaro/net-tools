package httpcore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/target"
)

func client(deny ...string) *Client {
	p := target.Policy{AllowPrivate: true}
	for _, d := range deny {
		p.Deny = append(p.Deny, netip.MustParsePrefix(d))
	}
	return &Client{Resolver: target.NewResolver(p)}
}

func TestRedirectChainAndMethodRewrite(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Method", r.Method)
		_, _ = io.WriteString(w, "hello "+r.Header.Get("User-Agent"))
	}))
	defer final.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/middle", http.StatusFound)
			return
		}
		http.Redirect(w, r, final.URL+"/end", http.StatusSeeOther)
	}))
	defer first.Close()

	hops, err := client().Do(context.Background(), Request{URL: first.URL + "/start", Method: "POST", Body: "x=1", FollowRedirects: true, MaxBody: 1 << 20, PreviewBytes: 100, UserAgent: "NetTools/test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 3 || hops[0].Status != 302 || hops[0].Location != "/middle" || hops[1].Status != 303 || hops[2].Status != 200 {
		t.Fatalf("hops = %+v", hops)
	}
	if hops[2].Method != "GET" || hops[2].BodyPreview != "hello NetTools/test" || hops[0].Method != "POST" {
		t.Errorf("final = %+v", hops[2])
	}
	if hops[2].Timing.TotalMs <= 0 || hops[2].RemoteAddr == "" {
		t.Errorf("timing = %+v", hops[2].Timing)
	}
}

// A redirect to a denied address must be refused at that hop, not fetched (SSRF).
func TestRedirectIntoDeniedRange(t *testing.T) {
	hit := false
	inner := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }))
	defer inner.Close()
	innerURL := strings.Replace(inner.URL, "127.0.0.1", "127.0.0.2", 1)
	outer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, innerURL, http.StatusFound)
	}))
	defer outer.Close()
	hops, err := client("127.0.0.2/32").Do(context.Background(), Request{URL: outer.URL, Method: "GET", FollowRedirects: true, MaxBody: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 2 || hops[1].ErrorCode != "TARGET_BLOCKED" || hit {
		t.Errorf("hops = %+v hit=%v", hops, hit)
	}
	// The first hop itself denied: an error, not a chain.
	if _, err := client("127.0.0.1/32").Do(context.Background(), Request{URL: outer.URL, Method: "GET"}); err == nil || !strings.Contains(err.Error(), "TARGET_BLOCKED") {
		t.Errorf("err = %v", err)
	}
}

func TestRedirectLoopIsBounded(t *testing.T) {
	loop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Path, http.StatusFound)
	}))
	defer loop.Close()
	hops, err := client().Do(context.Background(), Request{URL: loop.URL + "/x", Method: "GET", FollowRedirects: true, MaxBody: 1024})
	if err != nil || len(hops) != 12 || hops[11].ErrorCode != "TOO_MANY_REDIRECTS" {
		t.Errorf("len=%d err=%v last=%+v", len(hops), err, hops[len(hops)-1])
	}
}

func TestBodyCapAndHeaders(t *testing.T) {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("a", 5000))
	}))
	defer big.Close()
	hops, err := client().Do(context.Background(), Request{URL: big.URL, Method: "GET", MaxBody: 1000, PreviewBytes: 100})
	if err != nil || hops[0].BodyBytes != 1000 || !hops[0].BodyTruncated || len(hops[0].BodyPreview) != 100 {
		t.Errorf("hop = %+v %v", hops[0].BodyBytes, err)
	}
	for _, bad := range []map[string]string{{"Content-Length": "1"}, {"X-A": "b\r\nInjected: 1"}, {"Bad Name": "x"}} {
		if ValidateHeaders(bad) == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	if ValidateHeaders(map[string]string{"Authorization": "Bearer x", "Host": "vhost.example"}) != nil {
		t.Error("normal headers rejected")
	}
}
