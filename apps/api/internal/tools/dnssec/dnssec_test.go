package dnssec

import (
	"crypto"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

var now = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

type zone struct {
	name     string
	ksk, zsk *dns.DNSKEY
	kp, zp   crypto.Signer
}

func newKey(t *testing.T, name string, flags uint16) (*dns.DNSKEY, crypto.Signer) {
	k := &dns.DNSKEY{Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeDNSKEY, Class: dns.ClassINET, Ttl: 3600}, Flags: flags, Protocol: 3, Algorithm: dns.ECDSAP256SHA256}
	p, err := k.Generate(256)
	if err != nil {
		t.Fatal(err)
	}
	return k, p.(crypto.Signer)
}

func newZone(t *testing.T, name string) *zone {
	z := &zone{name: name}
	z.ksk, z.kp = newKey(t, name, 257)
	z.zsk, z.zp = newKey(t, name, 256)
	return z
}

func (z *zone) signWith(t *testing.T, key *dns.DNSKEY, priv crypto.Signer, set []dns.RR) []dns.RR {
	h := set[0].Header()
	sig := &dns.RRSIG{
		Hdr:         dns.RR_Header{Name: h.Name, Rrtype: dns.TypeRRSIG, Class: dns.ClassINET, Ttl: h.Ttl},
		TypeCovered: h.Rrtype, Algorithm: key.Algorithm, Labels: uint8(dns.CountLabel(h.Name)), OrigTtl: h.Ttl, //nolint:gosec // test
		Expiration: uint32(now.Add(24 * time.Hour).Unix()), Inception: uint32(now.Add(-time.Hour).Unix()), //nolint:gosec // test
		KeyTag: key.KeyTag(), SignerName: z.name,
	}
	if err := sig.Sign(priv, set); err != nil {
		t.Fatal(err)
	}
	return append(append([]dns.RR{}, set...), sig)
}

func (z *zone) sign(t *testing.T, set ...dns.RR) []dns.RR { return z.signWith(t, z.zsk, z.zp, set) }

func (z *zone) dnskeys(t *testing.T) []dns.RR {
	return z.signWith(t, z.ksk, z.kp, []dns.RR{z.ksk, z.zsk})
}

func (z *zone) ds() *dns.DS { return z.ksk.ToDS(dns.SHA256) }

func rr(s string) dns.RR {
	r, err := dns.NewRR(s)
	if err != nil {
		panic(err)
	}
	return r
}

type reply struct {
	rcode      int
	answer, ns []dns.RR
}

func hierarchy(t *testing.T) (map[string]reply, []*dns.DS) {
	root, tld, good, bogus := newZone(t, "."), newZone(t, "test."), newZone(t, "good.test."), newZone(t, "bogus.test.")
	stranger, _ := newKey(t, "bogus.test.", 257)
	soa := func(z string) dns.RR {
		return rr(z + " 3600 IN SOA ns." + strings.TrimPrefix(z, ".") + " host. 1 7200 3600 1209600 300")
	}
	tampered := good.sign(t, rr("badsig.good.test. 300 IN A 192.0.2.66"))
	tampered[0] = rr("badsig.good.test. 300 IN A 192.0.2.67") // data changed after signing
	m := map[string]reply{
		". DNSKEY":             {answer: root.dnskeys(t)},
		". SOA":                {answer: root.sign(t, rr(". 3600 IN SOA a.root. host. 1 1 1 1 1"))},
		"test. DS":             {answer: root.sign(t, tld.ds())},
		"test. DNSKEY":         {answer: tld.dnskeys(t)},
		"test. SOA":            {answer: tld.sign(t, soa("test."))},
		"good.test. DS":        {answer: tld.sign(t, good.ds())},
		"good.test. DNSKEY":    {answer: good.dnskeys(t)},
		"good.test. SOA":       {answer: good.sign(t, soa("good.test."))},
		"www.good.test. A":     {answer: good.sign(t, rr("www.good.test. 300 IN A 192.0.2.10"))},
		"badsig.good.test. A":  {answer: tampered},
		"nx.good.test. A":      {rcode: dns.RcodeNameError, ns: append(good.sign(t, rr("good.test. 300 IN NSEC www.good.test. A NS SOA RRSIG NSEC DNSKEY")), soa("good.test."))},
		"insecure.test. DS":    {ns: tld.sign(t, rr("insecure.test. 300 IN NSEC x.test. NS RRSIG NSEC"))},
		"insecure.test. SOA":   {answer: []dns.RR{soa("insecure.test.")}},
		"www.insecure.test. A": {answer: []dns.RR{rr("www.insecure.test. 300 IN A 192.0.2.20")}},
		"bogus.test. DS":       {answer: tld.sign(t, stranger.ToDS(dns.SHA256))},
		"bogus.test. DNSKEY":   {answer: bogus.dnskeys(t)},
		"bogus.test. SOA":      {answer: bogus.sign(t, soa("bogus.test."))},
		"www.bogus.test. A":    {answer: bogus.sign(t, rr("www.bogus.test. 300 IN A 192.0.2.30"))},
	}
	return m, []*dns.DS{root.ds()}
}

func TestDNSSEC(t *testing.T) {
	data, anchors := hierarchy(t)
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		q := r.Question[0]
		m := new(dns.Msg)
		m.SetReply(r)
		if rep, ok := data[strings.ToLower(q.Name)+" "+dns.TypeToString[q.Qtype]]; ok {
			m.Rcode, m.Answer, m.Ns = rep.rcode, rep.answer, rep.ns
		} else if q.Qtype != dns.TypeSOA && q.Qtype != dns.TypeDS {
			m.Rcode = dns.RcodeNameError
		}
		_ = w.WriteMsg(m)
	})
	srv := dnsx.Server{Label: "fake", Addr: addr, Transport: dnsx.TCP}
	env := tooltest.New(t, &Tool{Server: &srv, Anchors: anchors, Now: func() time.Time { return now }}, tooltest.Options{AllowPrivate: true})

	check := func(q, want string, wantZones int, at string) Result {
		t.Helper()
		var res Result
		env.Get("/api/v1/dns/dnssec?"+q).Must(t).Decode(t, &res)
		if res.Status != want || len(res.Steps) != wantZones || (at != "" && !strings.Contains(res.Message, "at "+at)) {
			t.Errorf("%s: status %s (%s), %d steps", q, res.Status, res.Message, len(res.Steps))
			for _, s := range res.Steps {
				t.Logf("  %s %s %s %+v", s.Zone, s.Status, s.Message, s.Signatures)
			}
			t.Logf("  answer %+v", res.Answer)
		}
		return res
	}
	res := check("name=www.good.test", Secure, 3, "")
	if !res.Steps[0].DS[0].Matched || res.Answer.Records[0].Data != "192.0.2.10" || !res.Answer.Signatures[0].Valid {
		t.Errorf("secure details = %+v", res)
	}
	check("name=nx.good.test", Secure, 3, "")
	check("name=www.insecure.test", Insecure, 3, "insecure.test.")
	res = check("name=www.bogus.test", Bogus, 3, "bogus.test.")
	if res.Steps[2].DS[0].Matched {
		t.Error("stranger DS matched")
	}
	check("name=badsig.good.test", Bogus, 3, "")
	env.Get("/api/v1/dns/dnssec?name=bad%20name").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/dns/dnssec?name=x.test&type=ANY").ExpectError(t, "INVALID_INPUT")

	// Expired signatures are bogus.
	late := tooltest.New(t, &Tool{Server: &srv, Anchors: anchors, Now: func() time.Time { return now.Add(48 * time.Hour) }}, tooltest.Options{AllowPrivate: true})
	var lr Result
	late.Get("/api/v1/dns/dnssec?name=www.good.test").Must(t).Decode(t, &lr)
	if lr.Status != Bogus || !strings.Contains(lr.Steps[0].Signatures[0].Error, "validity") {
		t.Errorf("expired = %+v", lr.Steps[0])
	}
}
