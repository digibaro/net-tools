package tlsinspect

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func tlsServer(t *testing.T) int {
	t.Helper()
	ts := httptest.NewUnstartedServer(http.NotFoundHandler())
	ts.StartTLS()
	t.Cleanup(ts.Close)
	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())
	tlsx.RootsForTests = pool
	t.Cleanup(func() { tlsx.RootsForTests = nil })
	return ts.Listener.Addr().(*net.TCPAddr).Port
}

func TestInspect(t *testing.T) {
	port := tlsServer(t)
	env := tooltest.New(t, &InspectTool{}, tooltest.Options{AllowPrivate: true})
	var s tlsx.Session
	env.Get("/api/v1/tls/inspect?host=127.0.0.1&sni=example.com&port="+strconv.Itoa(port)).Must(t).Decode(t, &s)
	if !s.Verified || s.ServerName != "example.com" || len(s.Chain) == 0 || s.ConnectedTo != "127.0.0.1:"+strconv.Itoa(port) {
		t.Errorf("session = %+v", s)
	}
	env.Get("/api/v1/tls/inspect?host=127.0.0.1:"+strconv.Itoa(port)).Must(t).Decode(t, &s)
	if !s.Verified || s.ServerName != "127.0.0.1" {
		t.Errorf("IP SAN verification: %+v", s)
	}
	env.Get("/api/v1/tls/inspect?host=127.0.0.1&port=1").ExpectError(t, "PROVIDER_ERROR")
}

func TestExpiry(t *testing.T) {
	port := tlsServer(t)
	env := tooltest.New(t, &ExpiryTool{}, tooltest.Options{AllowPrivate: true, Env: map[string]string{"BULK_MAX_TARGETS": "3"}})
	body, _ := json.Marshal(map[string]any{"targets": []string{"127.0.0.1:" + strconv.Itoa(port), "127.0.0.1:1"}})
	r := env.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/tls/expiry", bytes.NewReader(body))).Must(t)
	var out struct{ Rows []ExpiryRow }
	r.Decode(t, &out)
	if out.Rows[0].DaysLeft == nil || *out.Rows[0].DaysLeft <= 0 || out.Rows[1].ErrorCode != "PROVIDER_ERROR" {
		t.Errorf("rows = %+v", out.Rows)
	}
	body, _ = json.Marshal(map[string]any{"targets": []string{"a.example", "b.example", "c.example", "d.example"}})
	env.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/tls/expiry", bytes.NewReader(body))).ExpectError(t, "INVALID_INPUT")
}

func TestTLSSSRF(t *testing.T) {
	strict := tooltest.New(t, &InspectTool{}, tooltest.Options{Lookup: tooltest.StaticLookup("10.0.4.43")})
	for q, code := range map[string]string{
		"host=127.0.0.1":                  "TARGET_PRIVATE_DENIED",
		"host=intranet.example":           "TARGET_PRIVATE_DENIED",
		"host=169.254.169.254:443":        "TARGET_BLOCKED",
		"host=[::1]:8443":                 "TARGET_PRIVATE_DENIED",
		"host=example.com&port=0":         "INVALID_INPUT",
		"host=example.com&starttls=nntp":  "INVALID_INPUT",
		"host=example.com&sni=bad%20name": "INVALID_INPUT",
	} {
		strict.Get("/api/v1/tls/inspect?"+q).ExpectError(t, code)
	}
	expiry := tooltest.New(t, &ExpiryTool{}, tooltest.Options{})
	body, _ := json.Marshal(map[string]any{"targets": []string{"127.0.0.1:443"}})
	var out struct{ Rows []ExpiryRow }
	expiry.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/tls/expiry", bytes.NewReader(body))).Must(t).Decode(t, &out)
	if out.Rows[0].ErrorCode != "TARGET_PRIVATE_DENIED" || !strings.Contains(out.Rows[0].Error, "private") {
		t.Errorf("bulk SSRF row = %+v", out.Rows[0])
	}
}
