package whois

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

// fakeWhois answers each line with answer(query).
func fakeWhois(t *testing.T, answer func(q string) string) string {
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
				q, _ := bufio.NewReader(c).ReadString('\n')
				_, _ = c.Write([]byte(answer(strings.TrimSpace(q))))
			}(c)
		}
	}()
	return ln.Addr().String()
}

func TestNormalizeAndReferral(t *testing.T) {
	for in, want := range map[string]string{"Example.COM.": "domain:example.com", "8.8.8.8": "ip:8.8.8.8", "[2001:db8::1]": "ip:2001:db8::1", "as13335": "asn:AS13335", "13335": "asn:AS13335"} {
		q, k, err := Normalize(in)
		if err != nil || k+":"+q != want {
			t.Errorf("%s → %s:%s %v", in, k, q, err)
		}
	}
	for _, bad := range []string{"", "localhost", "a b"} {
		if _, _, err := Normalize(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	for text, want := range map[string]string{
		"refer:        whois.verisign-grs.com\n":             "whois.verisign-grs.com",
		"   Registrar WHOIS Server: whois.markmonitor.com\n": "whois.markmonitor.com",
		"ReferralServer:  whois://whois.ripe.net\n":          "whois.ripe.net",
		"ReferralServer: rwhois://rwhois.example:4321/\n":    "rwhois.example:4321",
		"nothing here": "",
	} {
		if got := Referral(text); got != want {
			t.Errorf("Referral(%q) = %q, want %q", text, got, want)
		}
	}
	if queryFor("whois.arin.net", "8.8.8.8", "ip") != "n + 8.8.8.8" {
		t.Error("ARIN query syntax")
	}
}

func TestWhoisReferralChain(t *testing.T) {
	registrar := fakeWhois(t, func(string) string {
		return "Domain Name: EXAMPLE.TEST\nRegistrar: Example Registrar, Inc.\nCreation Date: 1995-08-14T04:00:00Z\nRegistrar Registration Expiration Date: 2030-08-13T04:00:00Z\nDomain Status: clientTransferProhibited https://icann.org/epp\nName Server: NS1.EXAMPLE.TEST\nName Server: ns2.example.test\n"
	})
	registry := fakeWhois(t, func(string) string {
		return "Domain Name: EXAMPLE.TEST\nRegistrar WHOIS Server: " + registrar + "\nRegistry Expiry Date: 2030-08-13T04:00:00Z\n"
	})
	root := fakeWhois(t, func(string) string { return "% IANA WHOIS server\nrefer:        " + registry + "\n" })
	env := tooltest.New(t, &Tool{Root: root, anyPort: true}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/whois?query=example.test").Must(t).Decode(t, &res)
	if len(res.Chain) != 3 || res.Kind != "domain" {
		t.Fatalf("chain = %+v", res.Chain)
	}
	p := res.Parsed
	if p.Registrar != "Example Registrar, Inc." || p.Created != "1995-08-14T04:00:00Z" || p.Expires != "2030-08-13T04:00:00Z" || len(p.Nameservers) != 2 || p.Nameservers[0] != "ns1.example.test" || p.Status[0] != "clientTransferProhibited" {
		t.Errorf("parsed = %+v", p)
	}
}

// A referral to a private address must not be followed in the public profile.
func TestWhoisReferralPolicy(t *testing.T) {
	root := fakeWhois(t, func(string) string { return "refer: 10.0.0.43\n" })
	env := tooltest.New(t, &Tool{Root: root, anyPort: true}, tooltest.Options{AllowPrivate: true, Deny: []string{"10.0.0.0/8"}})
	var res Result
	env.Get("/api/v1/whois?query=example.test").Must(t).Decode(t, &res)
	if len(res.Chain) != 2 || !strings.Contains(res.Chain[1].Error, "deny list") || res.Chain[1].Text != "" {
		t.Errorf("chain = %+v", res.Chain)
	}
	strict := tooltest.New(t, &Tool{Root: "127.0.0.1:43"}, tooltest.Options{})
	strict.Get("/api/v1/whois?query=example.test").ExpectError(t, "PROVIDER_ERROR")
	strict.Get("/api/v1/whois?query=not%20valid").ExpectError(t, "INVALID_INPUT")
}

// Referrals are untrusted data: every one goes through the public-profile policy (asserted by
// comparing with the policy's own error, so a failed connection cannot pass the test) and the
// WHOIS port allow-list.
func TestWhoisReferralSSRF(t *testing.T) {
	res := target.NewResolver(target.Policy{})
	res.Lookup = tooltest.StaticLookup("10.0.0.43") // a "public" name that resolves privately (rebinding)
	tool := &Tool{deps: tools.Deps{Target: res}}
	for _, referral := range []string{"127.0.0.1", "169.254.169.254", "[::ffff:127.0.0.1]", "[fd00::43]", "0x7f.0.0.1", "10.1.2.3", "whois.rebind.example", "[64:ff9b::a00:1]"} {
		resp := tool.ask(context.Background(), referral, "example.test")
		_, perr := res.Resolve(context.Background(), referral, target.Opts{AllowPort: true, Field: "server"})
		if perr == nil || resp.Text != "" || resp.Error != httpapi.ToAPIError(perr).Message {
			t.Errorf("referral %s: %+v (policy: %v)", referral, resp, perr)
		}
	}
	res.Lookup = tooltest.StaticLookup("93.184.215.14")
	if resp := tool.ask(context.Background(), "whois.example.com:22", "example.test"); !strings.Contains(resp.Error, "non-WHOIS port") {
		t.Errorf("port 22: %+v", resp)
	}
}
