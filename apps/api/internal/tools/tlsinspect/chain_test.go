package tlsinspect

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

type testCA struct {
	root, inter, leaf *x509.Certificate
}

func mkCert(t *testing.T, tmpl, parent *x509.Certificate, pub, signer any) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, signer)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := x509.ParseCertificate(der)
	return c
}

func newCA(t *testing.T) testCA {
	t.Helper()
	k := func() *ecdsa.PrivateKey { k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader); return k }
	rk, ik, lk := k(), k(), k()
	now := time.Now()
	base := func(n int64, cn string, ca bool) *x509.Certificate {
		return &x509.Certificate{SerialNumber: big.NewInt(n), Subject: pkix.Name{CommonName: cn}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), IsCA: ca, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	}
	rt := base(1, "Test Root", true)
	root := mkCert(t, rt, rt, &rk.PublicKey, rk)
	it := base(2, "Test Intermediate", true)
	it.IssuingCertificateURL = []string{"http://aia.test/root.cer"}
	inter := mkCert(t, it, root, &ik.PublicKey, rk)
	lt := base(3, "www.example.test", false)
	lt.DNSNames = []string{"www.example.test"}
	lt.IssuingCertificateURL = []string{"http://aia.test/inter.cer"}
	leaf := mkCert(t, lt, inter, &lk.PublicKey, ik)
	return testCA{root, inter, leaf}
}

func pemOf(cs ...*x509.Certificate) string {
	var b strings.Builder
	for _, c := range cs {
		_ = pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
	}
	return b.String()
}

func TestChainValidate(t *testing.T) {
	ca := newCA(t)
	pool := x509.NewCertPool()
	pool.AddCert(ca.root)
	tlsx.RootsForTests = pool
	defer func() { tlsx.RootsForTests = nil }()
	fetch := func(_ context.Context, url string) ([]byte, error) {
		switch url {
		case "http://aia.test/root.cer":
			return ca.root.Raw, nil
		case "http://aia.test/inter.cer":
			return ca.inter.Raw, nil
		}
		return nil, errors.New("404")
	}
	tool := &ChainTool{Fetch: fetch}
	env := tooltest.New(t, tool, tooltest.Options{})
	post := func(pemText, host string) ChainResult {
		var res ChainResult
		env.Post("/api/v1/tls/chain-validate", map[string]string{"pem": pemText, "host": host}).Must(t).Decode(t, &res)
		return res
	}
	r := post(pemOf(ca.leaf, ca.inter), "www.example.test")
	if !r.Trusted || !r.Complete || !r.Ordered || len(r.Path) != 3 || r.Certs[r.Path[2]].Role != "root" || r.Certs[r.Path[2]].Source != "aia" || r.HostOK == nil || !*r.HostOK {
		t.Errorf("complete = %+v", r)
	}
	if r.Certs[0].SignatureOK == nil || !*r.Certs[0].SignatureOK {
		t.Errorf("leaf signature = %+v", r.Certs[0])
	}
	r = post(pemOf(ca.inter, ca.leaf), "other.example")
	if r.Ordered || !r.Trusted || r.Certs[1].Role != "leaf" || *r.HostOK {
		t.Errorf("reversed = %+v", r)
	}
	r = post(pemOf(ca.leaf), "")
	if !r.Trusted || r.Complete || !strings.Contains(strings.Join(r.Notes, " "), "missing-intermediates") {
		t.Errorf("missing intermediate = %+v", r)
	}
	tlsx.RootsForTests = x509.NewCertPool()
	r = post(pemOf(ca.leaf, ca.inter, ca.root), "")
	if r.Trusted || r.VerifyError == "" {
		t.Errorf("untrusted = %+v", r)
	}
	env.Post("/api/v1/tls/chain-validate", map[string]string{"pem": "nothing"}).ExpectError(t, "INVALID_INPUT")
}

func TestParseAIAPKCS7(t *testing.T) {
	// certs-only PKCS#7 as produced by "openssl crl2pkcs7 -nocrl": ContentInfo{signedData, SignedData{…, [0] certificates}}
	ca := newCA(t)
	certs := append([]byte{0xa0}, derLen(len(ca.inter.Raw))...)
	certs = append(certs, ca.inter.Raw...)
	sd := seq(append(append(append([]byte{0x02, 0x01, 0x01}, 0x31, 0x00), seq([]byte{0x06, 0x09, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x07, 0x01})...), certs...))
	ci := seq(append([]byte{0x06, 0x09, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x07, 0x02}, append(append([]byte{0xa0}, derLen(len(sd))...), sd...)...))
	got, err := ParseAIA(ci)
	if err != nil || len(got) != 1 || got[0].Subject.CommonName != "Test Intermediate" {
		t.Errorf("p7c = %v %v", got, err)
	}
}

func derLen(n int) []byte {
	if n < 128 {
		return []byte{byte(n)}
	}
	return []byte{0x82, byte(n >> 8), byte(n)}
}

func seq(b []byte) []byte { return append(append([]byte{0x30}, derLen(len(b))...), b...) }
