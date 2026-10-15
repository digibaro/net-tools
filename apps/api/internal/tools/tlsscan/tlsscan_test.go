package tlsscan

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestWeaknesses(t *testing.T) {
	if w := Weaknesses("TLS_RSA_WITH_3DES_EDE_CBC_SHA"); !slices.Equal(w, []string{"3des", "no-forward-secrecy", "cbc"}) {
		t.Error(w)
	}
	if w := Weaknesses("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"); len(w) != 0 {
		t.Error(w)
	}
}

func scanOf(t *testing.T, cfg *tls.Config) Summary {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.NotFoundHandler())
	srv.TLS = cfg
	srv.StartTLS()
	defer srv.Close()
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	body := env.Get("/api/v1/tls/scan/stream?host=" + strings.TrimPrefix(srv.URL, "https://")).Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("no done event: %s", body)
	}
	var d struct {
		Summary Summary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "event: probe") {
		t.Error("no probe events")
	}
	return d.Summary
}

func TestScanModernServer(t *testing.T) {
	s := scanOf(t, &tls.Config{MinVersion: tls.VersionTLS12})
	if !slices.Equal(s.Versions, []string{"TLS 1.2", "TLS 1.3"}) || slices.Contains(s.Problems, "legacy-versions") || len(s.Ciphers["TLS 1.2"]) == 0 || !slices.Contains(s.Groups, "X25519") {
		t.Errorf("modern = %+v", s)
	}
}

func TestScanLegacyServer(t *testing.T) {
	s := scanOf(t, &tls.Config{MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS12, CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}})
	if !slices.Contains(s.Versions, "TLS 1.0") || !slices.Contains(s.Problems, "legacy-versions") || !slices.Contains(s.Problems, "no-forward-secrecy") || !slices.Contains(s.Warnings, "no-tls13") || s.Rating != "weak" {
		t.Errorf("legacy = %+v", s)
	}
	if !slices.Equal(s.Ciphers["TLS 1.2"], []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_RSA_WITH_AES_128_CBC_SHA"}) {
		t.Errorf("ciphers = %v", s.Ciphers)
	}
}

func TestScanPolicy(t *testing.T) {
	body := tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/tls/scan/stream?host=10.0.0.1").Raw.Body.String()
	if !strings.Contains(body, "TARGET_PRIVATE_DENIED") {
		t.Error(body)
	}
}
