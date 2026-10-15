package dnsencrypted

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// startDoT runs a DNS-over-TLS server with httptest's certificate (valid for 127.0.0.1 / example.com).
func startDoT(t *testing.T) (string, *x509.CertPool) {
	t.Helper()
	ts := httptest.NewUnstartedServer(nil)
	ts.StartTLS() // just to obtain a certificate and pool
	cert := ts.TLS.Certificates[0]
	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())
	ts.Close()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"dot"}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	srv := &dns.Server{Listener: ln, Net: "tcp-tls", Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		rr, _ := dns.NewRR(r.Question[0].Name + " 60 IN A 192.0.2.53")
		m.Answer = append(m.Answer, rr)
		_ = w.WriteMsg(m)
	})}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })
	return ln.Addr().(*net.TCPAddr).String(), pool
}

func TestDoT(t *testing.T) {
	addr, pool := startDoT(t)
	env := tooltest.New(t, &Tool{RootCAs: pool}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/encrypted?transport=dot&count=2&endpoint="+url.QueryEscape(addr)).Must(t).Decode(t, &res)
	if res.TLS == nil || !strings.HasPrefix(res.TLS.Version, "TLS 1.") || res.TLS.ALPN != "dot" {
		t.Fatalf("tls = %+v", res.TLS)
	}
	// Endpoint given as IP: httptest certificates include 127.0.0.1 as an IP SAN.
	if !res.TLS.Verified {
		t.Errorf("verification failed: %s", res.TLS.VerifyError)
	}
	if len(res.Queries) != 2 || res.Queries[0].Rcode != "NOERROR" || res.Answer[0].Data != "192.0.2.53" {
		t.Errorf("queries = %+v answer = %+v", res.Queries, res.Answer)
	}

	// Without the test root the certificate is untrusted, but still inspected.
	env2 := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	env2.Get("/api/v1/dns/encrypted?transport=dot&count=1&endpoint="+url.QueryEscape(addr)).Must(t).Decode(t, &res)
	if res.TLS.Verified || res.TLS.VerifyError == "" {
		t.Errorf("expected verification error: %+v", res.TLS)
	}
}

func TestValidationAndSSRF(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Lookup: tooltest.StaticLookup("10.9.9.9")})
	env.Get("/api/v1/dns/encrypted").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/encrypted?endpoint=1.1.1.1&count=9").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/encrypted?endpoint=1.1.1.1&transport=udp").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/encrypted?endpoint=127.0.0.1:853").ExpectError(t, "TARGET_PRIVATE_DENIED")
	env.Get("/api/v1/dns/encrypted?endpoint=169.254.169.254").ExpectError(t, "TARGET_BLOCKED")
	env.Get("/api/v1/dns/encrypted?endpoint=dot.internal.example").ExpectError(t, "TARGET_PRIVATE_DENIED")
	env.Get("/api/v1/dns/encrypted?transport=doh&endpoint="+url.QueryEscape("https://doh.internal.example/dns-query")).ExpectError(t, "TARGET_PRIVATE_DENIED")
}
