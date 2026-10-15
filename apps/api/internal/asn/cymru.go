// Package asn maps IP addresses to origin ASNs via Team Cymru's DNS interface.
package asn

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
)

// Origin is one announcement covering the IP.
type Origin struct {
	ASN       uint32 `json:"asn"`
	Prefix    string `json:"prefix"`
	Country   string `json:"country"`
	Registry  string `json:"registry"`
	Allocated string `json:"allocated"`
	ASName    string `json:"asName"`
}

// TXTFunc returns TXT records for a name (injectable for tests).
type TXTFunc func(ctx context.Context, name string) ([]string, error)

// SystemTXT queries the system resolver.
func SystemTXT(ctx context.Context, name string) ([]string, error) {
	srv, err := dnsx.SystemServer(dnsx.UDP)
	if err != nil {
		return nil, err //nolint:wrapcheck // surfaced as INTERNAL
	}
	m, _, err := dnsx.Query(ctx, srv, name, dns.TypeTXT)
	if err != nil {
		return nil, err //nolint:wrapcheck // caller maps
	}
	var out []string
	for _, rr := range m.Answer {
		if t, ok := rr.(*dns.TXT); ok {
			out = append(out, strings.Join(t.Txt, ""))
		}
	}
	return out, nil
}

// QueryName is the Cymru origin name for ip.
func QueryName(ip netip.Addr) string {
	ip = ip.Unmap()
	if ip.Is4() {
		b := ip.As4()
		return fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com", b[3], b[2], b[1], b[0])
	}
	hex := fmt.Sprintf("%032x", ip.As16())
	nibbles := make([]string, 0, 32)
	for i := len(hex) - 1; i >= 0; i-- {
		nibbles = append(nibbles, hex[i:i+1])
	}
	return strings.Join(nibbles, ".") + ".origin6.asn.cymru.com"
}

func fields(s string) []string {
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Lookup returns all origins for ip (several when multiple ASNs announce it). An empty result
// means the address is not announced in BGP.
func Lookup(ctx context.Context, txt TXTFunc, ip netip.Addr) ([]Origin, error) {
	recs, err := txt(ctx, QueryName(ip))
	if err != nil {
		return nil, err
	}
	var out []Origin
	names := map[uint32]string{}
	for _, r := range recs {
		f := fields(r) // "13335 | 1.1.1.0/24 | AU | apnic | 2011-08-11"
		if len(f) < 5 {
			continue
		}
		for _, a := range strings.Fields(f[0]) {
			n, err := strconv.ParseUint(a, 10, 32)
			if err != nil {
				continue
			}
			out = append(out, Origin{ASN: uint32(n), Prefix: f[1], Country: f[2], Registry: f[3], Allocated: f[4]})
		}
	}
	for i := range out {
		name, ok := names[out[i].ASN]
		if !ok {
			name, _ = ASName(ctx, txt, out[i].ASN)
			names[out[i].ASN] = name
		}
		out[i].ASName = name
	}
	return out, nil
}

// ASName returns the AS description, e.g. "CLOUDFLARENET, US".
func ASName(ctx context.Context, txt TXTFunc, asn uint32) (string, error) {
	recs, err := txt(ctx, "AS"+strconv.FormatUint(uint64(asn), 10)+".asn.cymru.com")
	if err != nil {
		return "", err
	}
	for _, r := range recs {
		if f := fields(r); len(f) >= 5 {
			return f[4], nil
		}
	}
	return "", errors.New("no AS name")
}

var nonRoutable = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("2001:db8::/32"),
}

// IsPublic reports whether ip can have an origin AS (global unicast, not private/special).
func IsPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, p := range nonRoutable {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}
