// Package target is the only way to turn user input into an address to dial (docs/SECURITY.md §2):
// parse and normalize, resolve, apply the private/deny/allow policy to every resolved address,
// and pin one address so tools never re-resolve (DNS rebinding defense).
package target

import (
	"net/netip"
)

// Policy decides which addresses may be dialed.
type Policy struct {
	// AllowPrivate permits RFC 1918, CGNAT, loopback, link-local and ULA targets.
	AllowPrivate bool
	// Deny is an extra, always-applied deny list (DENY_CIDRS).
	Deny []netip.Prefix
	// Allow, if non-empty, restricts targets to these prefixes (ALLOW_CIDRS).
	Allow []netip.Prefix
	// Self lists the server's own addresses, which are treated as private.
	Self []netip.Addr
}

func mustPrefixes(ss ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(ss))
	for i, s := range ss {
		out[i] = netip.MustParsePrefix(s)
	}
	return out
}

// Always denied, whatever the profile: unroutable/special and cloud metadata endpoints.
var alwaysDeny = mustPrefixes(
	"0.0.0.0/8",          // "this network"; 0.0.0.0 reaches localhost on Linux
	"::/128",             // unspecified
	"169.254.169.254/32", // AWS/GCP/Azure/OpenStack metadata
	"fd00:ec2::254/128",  // AWS metadata (IPv6)
	"100.100.100.200/32", // Alibaba Cloud metadata
	"168.63.129.16/32",   // Azure wire server
	"224.0.0.0/4",        // multicast
	"ff00::/8",           // multicast
	"240.0.0.0/4",        // reserved (includes 255.255.255.255 broadcast)
)

// Denied unless AllowPrivate.
var privateRanges = mustPrefixes(
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", // RFC 1918
	"100.64.0.0/10",  // CGNAT
	"127.0.0.0/8",    // loopback
	"169.254.0.0/16", // link-local
	"::1/128",        // loopback
	"fc00::/7",       // ULA
	"fe80::/10",      // link-local
)

var (
	sixToFour = netip.MustParsePrefix("2002::/16")
	nat64     = netip.MustParsePrefix("64:ff9b::/96")
	nat64Loc  = netip.MustParsePrefix("64:ff9b:1::/48")
	v4compat  = netip.MustParsePrefix("::/96")
)

// embeddedIPv4 returns IPv4 addresses carried inside an IPv6 address (IPv4-mapped,
// IPv4-compatible, 6to4, NAT64), which must pass the same policy as the address itself.
func embeddedIPv4(a netip.Addr) []netip.Addr {
	if !a.Is6() {
		return nil
	}
	b := a.As16()
	var out []netip.Addr
	switch {
	case a.Is4In6():
		out = append(out, a.Unmap())
	case v4compat.Contains(a) && a != netip.IPv6Unspecified() && a != netip.IPv6Loopback():
		out = append(out, netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}))
	}
	if sixToFour.Contains(a) {
		out = append(out, netip.AddrFrom4([4]byte{b[2], b[3], b[4], b[5]}))
	}
	if nat64.Contains(a) || nat64Loc.Contains(a) {
		out = append(out, netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}))
	}
	return out
}

func inAny(a netip.Addr, ps []netip.Prefix) bool {
	for _, p := range ps {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// Check returns nil if addr may be dialed, or a *Error (TARGET_BLOCKED / TARGET_PRIVATE_DENIED).
func (p Policy) Check(addr netip.Addr) error {
	addr = addr.WithZone("")
	candidates := append([]netip.Addr{addr}, embeddedIPv4(addr)...)
	for _, a := range candidates {
		if !a.IsValid() {
			return blocked(addr, "invalid address")
		}
		if inAny(a, alwaysDeny) || inAny(a, p.Deny) {
			return blocked(addr, "address is on the deny list")
		}
		if !p.AllowPrivate && (inAny(a, privateRanges) || p.isSelf(a)) {
			return &Error{Code: CodePrivateDenied, Message: "private, loopback and link-local targets are not allowed on this server", Addr: addr}
		}
	}
	if len(p.Allow) > 0 && !inAny(addr.Unmap(), p.Allow) {
		return blocked(addr, "address is outside the allowed ranges")
	}
	return nil
}

func (p Policy) isSelf(a netip.Addr) bool {
	for _, s := range p.Self {
		if s.WithZone("").Unmap() == a.Unmap() {
			return true
		}
	}
	return false
}

func blocked(addr netip.Addr, msg string) error {
	return &Error{Code: CodeBlocked, Message: msg, Addr: addr}
}
