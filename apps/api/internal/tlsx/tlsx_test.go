package tlsx

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	ts := httptest.NewUnstartedServer(http.NotFoundHandler())
	ts.StartTLS()
	defer ts.Close()
	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())
	return ts.TLS.Certificates[0], pool
}

// serve accepts one connection: optional SMTP STARTTLS dialog, then TLS.
func serve(t *testing.T, cert tls.Certificate, smtp bool) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				if smtp {
					r := bufio.NewReader(c)
					_, _ = c.Write([]byte("220 mx.example ESMTP\r\n"))
					_, _ = r.ReadString('\n')
					_, _ = c.Write([]byte("250-mx.example\r\n250 STARTTLS\r\n"))
					_, _ = r.ReadString('\n')
					_, _ = c.Write([]byte("220 go ahead\r\n"))
				}
				s := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
				_ = s.Handshake()
				time.Sleep(50 * time.Millisecond)
			}(c)
		}
	}()
	return ln.Addr().String()
}

func TestHandshakeAndVerify(t *testing.T) {
	cert, pool := testCert(t)
	RootsForTests = pool
	defer func() { RootsForTests = nil }()
	addr := serve(t, cert, false)
	s, err := Handshake(context.Background(), addr, Options{ServerName: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Verified || !strings.HasPrefix(s.Version, "TLS 1.") || len(s.Chain) == 0 || s.Chain[0].DaysLeft <= 0 || !strings.Contains(s.Chain[0].KeyAlgorithm, "RSA") && !strings.Contains(s.Chain[0].KeyAlgorithm, "ECDSA") {
		t.Fatalf("session = %+v", s)
	}
	s, _ = Handshake(context.Background(), addr, Options{ServerName: "wrong.example"})
	if s.Verified || !strings.Contains(s.VerifyError, "not valid for wrong.example") {
		t.Errorf("hostname mismatch not reported: %+v", s.VerifyError)
	}
	RootsForTests = x509.NewCertPool()
	s, _ = Handshake(context.Background(), addr, Options{ServerName: "example.com"})
	if s.Verified || !strings.Contains(s.VerifyError, "not trusted") {
		t.Errorf("untrusted chain not reported: %+v", s.VerifyError)
	}
	s, _ = Handshake(context.Background(), addr, Options{ServerName: "example.com", Now: time.Now().AddDate(100, 0, 0)})
	if s.Chain[0].DaysLeft >= 0 {
		t.Errorf("days left in the future: %d", s.Chain[0].DaysLeft)
	}
}

func TestSMTPStartTLS(t *testing.T) {
	cert, pool := testCert(t)
	RootsForTests = pool
	defer func() { RootsForTests = nil }()
	addr := serve(t, cert, true)
	s, err := Handshake(context.Background(), addr, Options{ServerName: "example.com", StartTLS: "smtp"})
	if err != nil || !s.Verified || s.StartTLS != "smtp" {
		t.Fatalf("starttls: %+v %v", s, err)
	}
	if _, err := Handshake(context.Background(), serve(t, cert, false), Options{StartTLS: "smtp", Timeout: time.Second}); err == nil {
		t.Error("expected STARTTLS failure against a plain TLS server")
	}
}

// serveXMPP speaks the RFC 6120 stream negotiation up to <proceed/>, then TLS.
func serveXMPP(t *testing.T, cert tls.Certificate, offer bool) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				r := bufio.NewReader(c)
				_, _ = r.ReadString('>') // <?xml …?>
				_, _ = r.ReadString('>') // <stream:stream …>
				features := "<stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'/></stream:features>"
				if offer {
					features = "<stream:features><starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'><required/></starttls></stream:features>"
				}
				_, _ = c.Write([]byte("<?xml version='1.0'?><stream:stream from='example.com' id='1' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>" + features))
				if !offer {
					return
				}
				_, _ = r.ReadString('>')
				_, _ = c.Write([]byte("<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>"))
				s := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
				_ = s.Handshake()
				time.Sleep(50 * time.Millisecond)
			}(c)
		}
	}()
	return ln.Addr().String()
}

func TestXMPPStartTLS(t *testing.T) {
	cert, pool := testCert(t)
	RootsForTests = pool
	defer func() { RootsForTests = nil }()
	s, err := Handshake(context.Background(), serveXMPP(t, cert, true), Options{ServerName: "example.com", StartTLS: "xmpp"})
	if err != nil || !s.Verified || s.StartTLS != "xmpp" {
		t.Fatalf("xmpp: %+v %v", s, err)
	}
	if _, err := Handshake(context.Background(), serveXMPP(t, cert, false), Options{ServerName: "example.com", StartTLS: "xmpp", Timeout: time.Second}); err == nil || !strings.Contains(err.Error(), "does not offer STARTTLS") {
		t.Errorf("no starttls offered: %v", err)
	}
}

// A server that floods plaintext before STARTTLS is cut off at maxPlaintext.
func TestStartTLSPlaintextLimit(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		line := []byte("220-" + strings.Repeat("x", 1000) + "\r\n")
		for range 200 {
			if _, err := c.Write(line); err != nil {
				return
			}
		}
	}()
	if _, err := Handshake(context.Background(), ln.Addr().String(), Options{StartTLS: "smtp", Timeout: 3 * time.Second}); err == nil {
		t.Error("flood accepted")
	}
}
