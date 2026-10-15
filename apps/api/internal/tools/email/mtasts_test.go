package email

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestMXMatches(t *testing.T) {
	for pattern, cases := range map[string]map[string]bool{
		"mx1.example.com":    {"mx1.example.com.": true, "mx2.example.com": false},
		"*.mail.example.com": {"a.mail.example.com": true, "mail.example.com": false, "a.b.mail.example.com": false},
	} {
		for host, want := range cases {
			if MXMatches(pattern, host) != want {
				t.Errorf("%s ~ %s != %v", pattern, host, want)
			}
		}
	}
}

func TestMTASTS(t *testing.T) {
	res := fakeDNS{
		txt: map[string][]string{
			"_mta-sts.good.test":      {"v=STSv1; id=20261014"},
			"_smtp._tls.good.test":    {"v=TLSRPTv1; rua=mailto:tlsrpt@good.test"},
			"default._bimi.good.test": {"v=BIMI1; l=https://good.test/logo.svg; a=https://good.test/vmc.pem"},
			"_dmarc.good.test":        {"v=DMARC1; p=reject; rua=mailto:d@good.test"},
			"_mta-sts.bad.test":       {"v=STSv1; id=1"},
			"default._bimi.bad.test":  {"v=BIMI1; l=http://bad.test/logo.png"},
			"_dmarc.bad.test":         {"v=DMARC1; p=none"},
		},
		mx: map[string][]*dns.MX{
			"good.test": {{Mx: "mx1.mail.good.test."}, {Mx: "mx2.mail.good.test."}},
			"bad.test":  {{Mx: "mx.bad.test."}, {Mx: "other.provider.test."}},
		},
	}
	files := map[string]Fetched{
		"https://mta-sts.good.test/.well-known/mta-sts.txt": {Status: 200, ContentType: "text/plain", Body: "version: STSv1\nmode: enforce\nmx: *.mail.good.test\nmax_age: 604800\n"},
		"https://good.test/logo.svg":                        {Status: 200, ContentType: "image/svg+xml", Body: `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"></svg>`},
		"https://mta-sts.bad.test/.well-known/mta-sts.txt":  {Status: 200, ContentType: "text/html", Body: "version: STSv1\nmode: testing\nmx: mx.bad.test\nmax_age: 3600\n", CertError: "certificate has expired"},
	}
	fetch := func(_ context.Context, u string) (Fetched, error) {
		if f, ok := files[u]; ok {
			return f, nil
		}
		return Fetched{}, errors.New("unreachable")
	}
	env := tooltest.New(t, &MTASTSTool{res: res, fetch: fetch}, tooltest.Options{})
	var out MTASTSResult
	env.Get("/api/v1/email/mtasts?domain=good.test").Must(t).Decode(t, &out)
	if out.MTASTS.Mode != "enforce" || out.MTASTS.ID != "20261014" || len(out.MTASTS.Problems) != 0 || len(out.TLSRPT.RUA) != 1 || len(out.BIMI.Problems) != 0 || len(out.BIMI.Warnings) != 0 {
		t.Errorf("good = %+v", out)
	}
	env.Get("/api/v1/email/mtasts?domain=bad.test").Must(t).Decode(t, &out)
	s := out.MTASTS
	for _, want := range []string{"policy-certificate", "mx-not-covered"} {
		if !slices.Contains(s.Problems, want) {
			t.Errorf("bad mta-sts missing %s: %+v", want, s)
		}
	}
	if !slices.Contains(s.Warnings, "mode-testing") || !slices.Contains(s.Warnings, "short-max-age") || !slices.Contains(s.Warnings, "policy-content-type") || s.Unmatched[0] != "other.provider.test" {
		t.Errorf("bad warnings = %+v", s)
	}
	if !slices.Contains(out.TLSRPT.Problems, "no-record") || !slices.Contains(out.BIMI.Problems, "logo-not-https-svg") || !slices.Contains(out.BIMI.Problems, "dmarc-not-enforced") {
		t.Errorf("bad tlsrpt/bimi = %+v %+v", out.TLSRPT, out.BIMI)
	}
	env.Get("/api/v1/email/mtasts?domain=none.test").Must(t).Decode(t, &out)
	if !slices.Contains(out.MTASTS.Problems, "no-record") {
		t.Errorf("none = %+v", out.MTASTS)
	}
}
