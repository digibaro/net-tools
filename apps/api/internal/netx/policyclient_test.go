package netx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/target"
)

func TestPolicyClientSSRF(t *testing.T) {
	strict := PolicyClient(target.NewResolver(target.Policy{}), "test", 2*time.Second)
	for _, u := range []string{"http://127.0.0.1:9/", "http://169.254.169.254/latest/meta-data/", "http://[::ffff:127.0.0.1]:9/", "http://[fd00::1]/", "http://0x7f000001/", "http://10.0.0.1/"} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
		resp, err := strict.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		var te *target.Error
		if !errors.As(err, &te) {
			t.Errorf("%s: err = %v, want a policy error", u, err)
		}
	}
}

// Redirects are dialed through the policy too, and bodies are capped.
func TestPolicyClientRedirectAndLimit(t *testing.T) {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://127.0.0.2:9/secret", http.StatusFound)
			return
		}
		if r.Header.Get("User-Agent") != "test-ua" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(strings.Repeat("x", MaxProviderBody+100)))
	}))
	defer big.Close()
	res := target.NewResolver(target.Policy{AllowPrivate: true, Deny: []netip.Prefix{netip.MustParsePrefix("127.0.0.2/32")}})
	c := PolicyClient(res, "test-ua", 2*time.Second)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, big.URL+"/redirect", nil)
	resp, err := c.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	var te *target.Error
	if !errors.As(err, &te) || te.Code != target.CodeBlocked && te.Code != target.CodePrivateDenied {
		t.Errorf("redirect: err = %v", err)
	}

	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, big.URL+"/big", nil)
	resp, err = c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || len(b) != MaxProviderBody {
		t.Errorf("status %d, read %d bytes", resp.StatusCode, len(b))
	}
}
