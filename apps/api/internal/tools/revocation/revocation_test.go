package revocation

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ocsp"

	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

type pki struct {
	ca     *x509.Certificate
	caKey  crypto.Signer
	leaf   *x509.Certificate
	tlsCrt tls.Certificate
}

func newPKI(t *testing.T, base string, serial int64) pki {
	t.Helper()
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour * 24), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature}
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := x509.ParseCertificate(caDER)
	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	leafTpl := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "example.test"}, DNSNames: []string{"example.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour * 24),
		OCSPServer: []string{base + "/ocsp"}, CRLDistributionPoints: []string{base + "/ca.crl"}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTpl, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(leafDER)
	return pki{ca: ca, caKey: caKey, leaf: leaf, tlsCrt: tls.Certificate{Certificate: [][]byte{leafDER, caDER}, PrivateKey: leafKey}}
}

// revocation server: OCSP (GET form) and CRL, revoking serial `revoked` (0 = none).
func revocationServer(t *testing.T, p *pki, revoked int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/ocsp/"):
			raw, _ := url.PathUnescape(strings.TrimPrefix(r.URL.EscapedPath(), "/ocsp/"))
			der, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			req, err := ocsp.ParseRequest(der)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			tpl := ocsp.Response{Status: ocsp.Good, SerialNumber: req.SerialNumber, ThisUpdate: time.Now().Add(-time.Minute), NextUpdate: time.Now().Add(time.Hour)}
			if req.SerialNumber.Int64() == revoked {
				tpl.Status, tpl.RevokedAt, tpl.RevocationReason = ocsp.Revoked, time.Now().Add(-time.Minute), ocsp.KeyCompromise
			}
			resp, _ := ocsp.CreateResponse(p.ca, p.ca, tpl, p.caKey)
			_, _ = w.Write(resp)
		case r.URL.Path == "/ca.crl":
			var entries []x509.RevocationListEntry
			if revoked != 0 {
				entries = append(entries, x509.RevocationListEntry{SerialNumber: big.NewInt(revoked), RevocationTime: time.Now().Add(-time.Minute), ReasonCode: 1})
			}
			crl, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{Number: big.NewInt(1), ThisUpdate: time.Now().Add(-time.Minute), NextUpdate: time.Now().Add(time.Hour), RevokedCertificateEntries: entries}, p.ca, p.caKey)
			if err != nil {
				t.Error(err)
			}
			_, _ = w.Write(crl)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func tlsServer(t *testing.T, c tls.Certificate) int {
	t.Helper()
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{c}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_ = c.(*tls.Conn).Handshake()
				time.Sleep(50 * time.Millisecond)
				c.Close()
			}(conn)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func run(t *testing.T, revoke bool) Result {
	t.Helper()
	// The revocation server's URL must be known before issuing, so start it with a placeholder PKI.
	var p pki
	srv := revocationServer(t, &p, 0)
	if revoke {
		srv.Close()
		srv = revocationServer(t, &p, 42)
	}
	defer srv.Close()
	p = newPKI(t, srv.URL, 42)
	port := tlsServer(t, p.tlsCrt)
	var out Result
	tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/tls/revocation?host=127.0.0.1&port="+strconv.Itoa(port)).Must(t).Decode(t, &out)
	return out
}

func TestRevocation(t *testing.T) {
	good := run(t, false)
	if good.Status != "good" || len(good.OCSP) != 1 || good.OCSP[0].Status != "good" || len(good.CRL) != 1 || good.CRL[0].Status != "good" || good.Issuer != "Test CA" {
		t.Errorf("good = %+v", good)
	}
	bad := run(t, true)
	if bad.Status != "revoked" || bad.OCSP[0].Status != "revoked" || bad.OCSP[0].Reason != "key compromise" || bad.CRL[0].Status != "revoked" || bad.CRL[0].Entries != 1 {
		t.Errorf("revoked = %+v", bad)
	}
}

func TestRevocationPolicy(t *testing.T) {
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/tls/revocation?host=127.0.0.1").ExpectError(t, "TARGET_PRIVATE_DENIED")
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/tls/revocation?host=169.254.169.254").ExpectError(t, "TARGET_BLOCKED")
}

// OCSP and CRL URLs come from the certificate: the policy refuses private ones.
func TestRevocationURLPolicy(t *testing.T) {
	p := newPKI(t, "http://127.0.0.1:9", 7)
	tool := &Tool{deps: tools.Deps{Target: target.NewResolver(target.Policy{}), UserAgent: "t"}, Now: time.Now}
	ocsps, crls := tool.check(context.Background(), p.leaf, p.ca, nil)
	if len(ocsps) != 1 || !strings.Contains(ocsps[0].Error, "private") || len(crls) != 1 || !strings.Contains(crls[0].Error, "private") {
		t.Errorf("ocsp=%+v crl=%+v", ocsps, crls)
	}
}
