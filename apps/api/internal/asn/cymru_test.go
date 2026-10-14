package asn

import (
	"context"
	"net/netip"
	"testing"
)

func fake(m map[string][]string) TXTFunc {
	return func(_ context.Context, name string) ([]string, error) { return m[name], nil }
}

func TestQueryName(t *testing.T) {
	if got := QueryName(netip.MustParseAddr("1.1.1.1")); got != "1.1.1.1.origin.asn.cymru.com" {
		t.Error(got)
	}
	if got := QueryName(netip.MustParseAddr("192.0.2.10")); got != "10.2.0.192.origin.asn.cymru.com" {
		t.Error(got)
	}
	if got := QueryName(netip.MustParseAddr("2606:4700::1")); got != "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.7.4.6.0.6.2.origin6.asn.cymru.com" {
		t.Error(got)
	}
}

func TestLookup(t *testing.T) {
	txt := fake(map[string][]string{
		"1.1.1.1.origin.asn.cymru.com": {"13335 | 1.1.1.0/24 | AU | apnic | 2011-08-11"},
		"AS13335.asn.cymru.com":        {"13335 | US | arin | 2010-07-14 | CLOUDFLARENET, US"},
		"5.5.5.5.origin.asn.cymru.com": {"64500 64501 | 5.5.0.0/16 | DE | ripencc | "},
		"AS64500.asn.cymru.com":        {"64500 | DE | ripencc |  | EXAMPLE-A, DE"},
	})
	o, err := Lookup(context.Background(), txt, netip.MustParseAddr("1.1.1.1"))
	if err != nil || len(o) != 1 || o[0].ASN != 13335 || o[0].Prefix != "1.1.1.0/24" || o[0].ASName != "CLOUDFLARENET, US" || o[0].Registry != "apnic" {
		t.Fatalf("o = %+v %v", o, err)
	}
	o, _ = Lookup(context.Background(), txt, netip.MustParseAddr("5.5.5.5"))
	if len(o) != 2 || o[1].ASN != 64501 || o[0].ASName != "EXAMPLE-A, DE" || o[1].ASName != "" {
		t.Errorf("multi-origin = %+v", o)
	}
	if o, _ := Lookup(context.Background(), txt, netip.MustParseAddr("10.0.0.1")); len(o) != 0 {
		t.Errorf("unannounced = %+v", o)
	}
}
