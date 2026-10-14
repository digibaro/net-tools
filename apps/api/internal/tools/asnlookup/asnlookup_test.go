package asnlookup

import (
	"context"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func TestASNLookup(t *testing.T) {
	txt := func(_ context.Context, name string) ([]string, error) {
		return map[string][]string{
			"8.8.8.8.origin.asn.cymru.com": {"15169 | 8.8.8.0/24 | US | arin | 2023-12-28"},
			"AS15169.asn.cymru.com":        {"15169 | US | arin | 2000-03-30 | GOOGLE, US"},
		}[name], nil
	}
	env := tooltest.New(t, &Tool{TXT: txt}, tooltest.Options{})
	var res Result
	env.Get("/api/v1/asn/lookup?ip=8.8.8.8").Must(t).Decode(t, &res)
	if !res.Announced || res.Origins[0].ASN != 15169 || res.Origins[0].ASName != "GOOGLE, US" {
		t.Errorf("res = %+v", res)
	}
	env.Get("/api/v1/asn/lookup?ip=10.1.2.3").Must(t).Decode(t, &res)
	if res.Announced || len(res.Origins) != 0 {
		t.Errorf("private = %+v", res)
	}
	env.Get("/api/v1/asn/lookup?ip=example.com").ExpectError(t, "INVALID_INPUT")
}
