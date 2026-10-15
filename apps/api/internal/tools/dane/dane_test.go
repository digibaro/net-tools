package dane

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func spkiHash(c *x509.Certificate) string {
	s := sha256.Sum256(c.RawSubjectPublicKeyInfo)
	return hex.EncodeToString(s[:])
}

func TestDANEHTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)
	leaf := srv.Certificate()
	good := spkiHash(leaf)
	port := srv.Listener.Addr().(*net.TCPAddr).Port

	zone := func(ad bool, data string) string {
		return dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			m.AuthenticatedData = ad
			if r.Question[0].Name == "_443._tcp.www.example.test." {
				rr, _ := dns.NewRR("_443._tcp.www.example.test. 300 IN TLSA 3 1 1 " + data)
				m.Answer = append(m.Answer, rr)
			}
			_ = w.WriteMsg(m)
		}).String()
	}
	lookup := func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	env := tooltest.New(t, &Tool{Port: port}, tooltest.Options{AllowPrivate: true, Lookup: lookup})
	get := func(resolver string) Host {
		var res Result
		env.Get("/api/v1/tls/dane?service=https&host=www.example.test&resolver="+url.QueryEscape(resolver)).Must(t).Decode(t, &res)
		return res.Hosts[0]
	}
	if h := get(zone(true, good)); h.Verdict != "valid" || !h.Records[0].Match || h.Records[0].MatchedCert != 0 || h.TLS == nil {
		t.Errorf("valid = %+v", h)
	}
	if h := get(zone(true, "00"+good[2:])); h.Verdict != "mismatch" {
		t.Errorf("mismatch = %+v", h)
	}
	if h := get(zone(false, good)); h.Verdict != "insecure" {
		t.Errorf("insecure = %+v", h)
	}
	env.Get("/api/v1/tls/dane?host=example.test&service=ftp").ExpectError(t, "INVALID_INPUT")
}

func TestMatches(t *testing.T) {
	c := &x509.Certificate{Raw: []byte("cert"), RawSubjectPublicKeyInfo: []byte("spki")}
	if !Matches(TLSA{Selector: 0, MatchingType: 0, Data: hex.EncodeToString([]byte("cert"))}, c) {
		t.Error("full cert exact")
	}
	if Matches(TLSA{Selector: 1, MatchingType: 2, Data: "00"}, c) {
		t.Error("sha512 matched garbage")
	}
	recs := []TLSA{{Usage: 3, Selector: 1, MatchingType: 0, Data: hex.EncodeToString([]byte("spki-ca"))}, {Usage: 2, Selector: 1, MatchingType: 0, Data: hex.EncodeToString([]byte("spki-ca"))}}
	MatchChain(recs, []*x509.Certificate{c, {RawSubjectPublicKeyInfo: []byte("spki-ca")}})
	if recs[0].Match || !recs[1].Match || recs[1].MatchedCert != 1 {
		t.Errorf("chain = %+v", recs)
	}
}
