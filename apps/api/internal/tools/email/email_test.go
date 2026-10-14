package email

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

type fakeDNS struct {
	txt map[string][]string
	mx  map[string][]*dns.MX
	a   map[string]int
}

func (f fakeDNS) TXT(_ context.Context, name string) ([]string, string, error) {
	if v, ok := f.txt[name]; ok {
		return v, "NOERROR", nil
	}
	return nil, "NXDOMAIN", nil
}

func (f fakeDNS) MX(_ context.Context, name string) ([]*dns.MX, string, error) {
	return f.mx[name], "NOERROR", nil
}

func (f fakeDNS) Exists(_ context.Context, name string, _ uint16) (int, string, error) {
	return f.a[name], "NOERROR", nil
}

func TestSPF(t *testing.T) {
	f := fakeDNS{txt: map[string][]string{
		"example.com":         {"v=spf1 include:_spf.provider.test mx a:mail.example.com ip4:192.0.2.0/24 -all", "google-site-verification=x"},
		"_spf.provider.test":  {"v=spf1 include:_a.provider.test include:_b.provider.test ~all"},
		"_a.provider.test":    {"v=spf1 ip4:198.51.100.0/24 ~all"},
		"_b.provider.test":    {"v=spf1 ip6:2001:db8::/32 ptr ~all"},
		"plus.test":           {"v=spf1 +all"},
		"dup.test":            {"v=spf1 -all", "v=spf1 ~all"},
		"loop.test":           {"v=spf1 include:loop.test -all"},
		"many.test":           {"v=spf1 a:1.t a:2.t a:3.t a:4.t a:5.t a:6.t a:7.t a:8.t a:9.t a:10.t a:11.t -all"},
		"broken-include.test": {"v=spf1 include:missing.test -all"},
	}, a: map[string]int{"example.com": 1, "mail.example.com": 1, "1.t": 1, "2.t": 1, "3.t": 1, "4.t": 1, "5.t": 1, "6.t": 1, "7.t": 1, "8.t": 1, "9.t": 1, "10.t": 1, "11.t": 1}}
	ctx := context.Background()

	r := CheckSPF(ctx, f, "example.com")
	if r.All != "-all" || r.Lookups != 6 || len(r.Problems) != 0 || !slices.Contains(r.Warnings, "ptr-deprecated") {
		t.Errorf("example.com = %+v", r)
	}
	if inc := r.Record.Terms[0].Include; inc == nil || len(inc.Terms) != 3 || inc.Terms[0].Include.Terms[0].Value != "198.51.100.0/24" {
		t.Errorf("include tree = %+v", r.Record.Terms[0])
	}
	for domain, want := range map[string]string{
		"plus.test": "plus-all", "dup.test": "multiple-records", "loop.test": "include-loop", "many.test": "too-many-lookups",
		"broken-include.test": "include-missing", "none.test": "nxdomain",
	} {
		if r := CheckSPF(ctx, f, domain); !slices.Contains(r.Problems, want) {
			t.Errorf("%s: problems %v, want %s", domain, r.Problems, want)
		}
	}
	if term := parseTerm("~ip6:2001:db8::/32"); term.Qualifier != "~" || term.Kind != "ip6" || term.Value != "2001:db8::/32" {
		t.Errorf("parseTerm = %+v", term)
	}
	if term := parseTerm("redirect=_spf.x.test"); term.Kind != "redirect" || term.Value != "_spf.x.test" {
		t.Errorf("parseTerm = %+v", term)
	}
}

func TestDMARC(t *testing.T) {
	f := fakeDNS{txt: map[string][]string{
		"_dmarc.example.com":                      {"v=DMARC1; p=quarantine; sp=reject; pct=50; rua=mailto:dmarc@example.com,mailto:x@reports.test; adkim=s"},
		"example.com._report._dmarc.reports.test": {"v=DMARC1"},
		"_dmarc.weak.test":                        {"v=DMARC1; p=none; rua=mailto:a@other.test"},
	}}
	r := CheckDMARC(context.Background(), f, "mail.example.com")
	if r.FoundAt != "example.com" || r.Policy != "quarantine" || r.SubPolicy != "reject" || r.Pct != 50 || r.ADKIM != "s" || r.ASPF != "r" {
		t.Fatalf("dmarc = %+v", r)
	}
	if r.RUA[0].External || !r.RUA[1].External || r.RUA[1].Authorized == nil || !*r.RUA[1].Authorized || !slices.Contains(r.Warnings, "partial-pct") {
		t.Errorf("rua = %+v warnings %v", r.RUA, r.Warnings)
	}
	w := CheckDMARC(context.Background(), f, "weak.test")
	if !slices.Contains(w.Warnings, "policy-none") || !slices.Contains(w.Problems, "unauthorized-external-report") {
		t.Errorf("weak = %+v", w)
	}
	if n := CheckDMARC(context.Background(), f, "none.test"); !slices.Contains(n.Problems, "no-record") {
		t.Errorf("none = %+v", n)
	}
}

func TestDKIM(t *testing.T) {
	pub := testRSAPublicKey(t, 2048)
	small := testRSAPublicKey(t, 1024)
	f := fakeDNS{txt: map[string][]string{
		"s1._domainkey.example.com":   {"v=DKIM1; k=rsa; p=" + pub},
		"old._domainkey.example.com":  {"v=DKIM1; k=rsa; t=y; p=" + small},
		"gone._domainkey.example.com": {"v=DKIM1; p="},
		"bad._domainkey.example.com":  {"v=DKIM1; p=!!!"},
	}}
	ctx := context.Background()
	if r := CheckDKIM(ctx, f, "example.com", "s1"); r.KeyBits != 2048 || r.KeyType != "rsa" || len(r.Problems)+len(r.Warnings) != 0 {
		t.Errorf("s1 = %+v", r)
	}
	if r := CheckDKIM(ctx, f, "example.com", "old"); r.KeyBits != 1024 || !r.Testing || !slices.Contains(r.Warnings, "short-key") {
		t.Errorf("old = %+v", r)
	}
	if r := CheckDKIM(ctx, f, "example.com", "gone"); !r.Revoked {
		t.Errorf("gone = %+v", r)
	}
	if r := CheckDKIM(ctx, f, "example.com", "bad"); !slices.Contains(r.Problems, "bad-key") {
		t.Errorf("bad = %+v", r)
	}
	if r := CheckDKIM(ctx, f, "example.com", "missing"); !slices.Contains(r.Problems, "no-record") {
		t.Errorf("missing = %+v", r)
	}
}

// fakeSMTP speaks just enough SMTP for a STARTTLS probe.
func fakeSMTP(t *testing.T) (int, *x509.CertPool) {
	t.Helper()
	ts := httptest.NewUnstartedServer(http.NotFoundHandler())
	ts.StartTLS()
	cert := ts.TLS.Certificates[0]
	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())
	ts.Close()
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
				_, _ = c.Write([]byte("220 mx.example.com ESMTP test\r\n"))
				_, _ = r.ReadString('\n')
				_, _ = c.Write([]byte("250-mx.example.com\r\n250-SIZE 35882577\r\n250 STARTTLS\r\n"))
				_, _ = r.ReadString('\n')
				_, _ = c.Write([]byte("220 2.0.0 Ready\r\n"))
				s := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
				_ = s.Handshake()
				_, _ = bufio.NewReader(s).ReadString('\n')
			}(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, pool
}

func TestMXProbe(t *testing.T) {
	port, pool := fakeSMTP(t)
	tlsx.RootsForTests = pool
	defer func() { tlsx.RootsForTests = nil }()
	p := ProbeSMTP(context.Background(), "127.0.0.1:"+strconv.Itoa(port), "example.com")
	if !p.Connected || !p.StartTLS || p.Banner != "mx.example.com ESMTP test" || !slices.Contains(p.Extensions, "SIZE 35882577") || p.TLS == nil || !p.TLS.Verified {
		t.Fatalf("probe = %+v tls=%+v", p, p.TLS)
	}

	tool := &MXTool{Port: port, res: fakeDNS{mx: map[string][]*dns.MX{"example.com": {{Preference: 10, Mx: "mx.example.com."}}, "nomail.test": {{Mx: "."}}}}}
	env := tooltest.New(t, tool, tooltest.Options{AllowPrivate: true, Lookup: tooltest.StaticLookup("127.0.0.1")})
	var res MXResult
	env.Get("/api/v1/email/mx?domain=example.com").Must(t).Decode(t, &res)
	if len(res.Hosts) != 1 || res.Hosts[0].Host != "mx.example.com" || len(res.Hosts[0].Probes) != 1 || !res.Hosts[0].Probes[0].StartTLS {
		t.Errorf("res = %+v", res)
	}
	env.Get("/api/v1/email/mx?domain=nomail.test").Must(t).Decode(t, &res)
	if !res.NullMX {
		t.Errorf("null MX = %+v", res)
	}
}

// MX hosts resolving to private addresses must not be probed.
func TestMXPolicy(t *testing.T) {
	tool := &MXTool{res: fakeDNS{mx: map[string][]*dns.MX{"example.com": {{Preference: 10, Mx: "mx.internal.test."}}}}}
	env := tooltest.New(t, tool, tooltest.Options{Lookup: tooltest.StaticLookup("10.0.0.25")})
	var res MXResult
	env.Get("/api/v1/email/mx?domain=example.com").Must(t).Decode(t, &res)
	if len(res.Hosts[0].Probes) != 0 || !strings.Contains(res.Hosts[0].Error, "private") {
		t.Errorf("host = %+v", res.Hosts[0])
	}
	env.Get("/api/v1/email/mx?domain=bad%20domain").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/email/mx?domain=localhost").ExpectError(t, "INVALID_INPUT")
}

// Every MX host is resolved through the public-profile policy before port 25 is touched.
func TestMXSSRF(t *testing.T) {
	hosts := map[string]string{
		"loopback.test.": "127.0.0.1", "metadata.test.": "169.254.169.254", "mapped.test.": "::ffff:127.0.0.1",
		"ula.test.": "fd00::25", "rfc1918.test.": "192.168.1.25", "cgnat.test.": "100.64.0.25", "nat64.test.": "64:ff9b::a00:1",
	}
	mx := map[string][]*dns.MX{}
	for h := range hosts {
		mx[strings.TrimSuffix(strings.SplitN(h, ".", 2)[0], ".")+".example"] = []*dns.MX{{Preference: 10, Mx: h}}
	}
	lookup := func(_ context.Context, _, host string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr(hosts[host+"."])}, nil
	}
	env := tooltest.New(t, &MXTool{res: fakeDNS{mx: mx}}, tooltest.Options{Lookup: lookup})
	for domain := range mx {
		var res MXResult
		env.Get("/api/v1/email/mx?domain="+domain).Must(t).Decode(t, &res)
		if len(res.Hosts) != 1 || res.Hosts[0].Error == "" || len(res.Hosts[0].Probes) != 0 {
			t.Errorf("%s: %+v", domain, res.Hosts)
		}
	}
}

func TestSPFStopsAtLookupLimit(t *testing.T) {
	txt := map[string][]string{}
	// A fan-out tree: each level includes 5 children; walking it fully would take thousands of queries.
	var build func(name string, depth int)
	build = func(name string, depth int) {
		rec := "v=spf1"
		for i := range 5 {
			child := fmt.Sprintf("c%d.%s", i, name)
			rec += " include:" + child
			if depth < 5 {
				build(child, depth+1)
			}
		}
		txt[name] = []string{rec + " -all"}
	}
	build("fan.test", 0)
	counting := &countingDNS{fakeDNS: fakeDNS{txt: txt}}
	r := CheckSPF(context.Background(), counting, "fan.test")
	if !slices.Contains(r.Problems, "too-many-lookups") || counting.queries > MaxLookups+2 {
		t.Errorf("queries = %d, problems %v", counting.queries, r.Problems)
	}
}

type countingDNS struct {
	fakeDNS
	queries int
}

func (c *countingDNS) TXT(ctx context.Context, name string) ([]string, string, error) {
	c.queries++
	return c.fakeDNS.TXT(ctx, name)
}

func (c *countingDNS) Exists(ctx context.Context, name string, qt uint16) (int, string, error) {
	c.queries++
	return c.fakeDNS.Exists(ctx, name, qt)
}
