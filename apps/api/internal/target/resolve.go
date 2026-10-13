package target

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

// LookupFunc resolves a hostname (matches (*net.Resolver).LookupNetIP).
type LookupFunc func(ctx context.Context, network, host string) ([]netip.Addr, error)

// Resolver validates, resolves and policy-checks targets.
type Resolver struct {
	Policy  Policy
	Lookup  LookupFunc
	Timeout time.Duration
}

// NewResolver uses the system resolver with the documented 3 s timeout.
func NewResolver(p Policy) *Resolver {
	return &Resolver{Policy: p, Lookup: net.DefaultResolver.LookupNetIP, Timeout: 3 * time.Second}
}

// Opts controls parsing.
type Opts struct {
	// Family restricts resolution: 0 = any, 4 or 6.
	Family int
	// AllowPort accepts host:port / [v6]:port; Port is then set (0 if absent).
	AllowPort bool
	// Field names the input in validation errors (default "target").
	Field string
}

// Target is a validated, resolved and pinned destination.
type Target struct {
	Input string
	// Host is the normalized hostname (ASCII, lowercase, no trailing dot) or the IP literal.
	Host string
	IsIP bool
	Port int
	// Resolved holds every address the name resolved to (all passed the policy).
	Resolved []netip.Addr
	// IP is the pinned address to dial. Never re-resolve Host.
	IP netip.Addr
}

// AddrPort joins the pinned IP with port (or Target.Port when port is 0).
func (t Target) AddrPort(port int) string {
	if port == 0 {
		port = t.Port
	}
	return netip.AddrPortFrom(t.IP, uint16(port)).String() //nolint:gosec // ports validated to 0-65535
}

var (
	// Inputs that are not strict dotted-quad but that inet_aton/URL parsers would still read as
	// IPv4 (2130706433, 0177.0.0.1, 0x7f.1, 127.1): rejected outright to avoid ambiguity.
	numericHost = regexp.MustCompile(`^(?:0x[0-9a-f]*|[0-9]+)(?:\.(?:0x[0-9a-f]*|[0-9]+))*\.?$`)
	label       = regexp.MustCompile(`^[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?$`)
	idnaProfile = idna.New(idna.MapForLookup(), idna.BidiRule(), idna.ValidateLabels(true), idna.StrictDomainName(false))
)

// Resolve parses input, resolves it and applies the policy. It is the only sanctioned way for
// handlers to obtain an address to dial.
func (r *Resolver) Resolve(ctx context.Context, input string, o Opts) (Target, error) {
	field := o.Field
	if field == "" {
		field = "target"
	}
	t, err := parse(input, o, field)
	if err != nil {
		return Target{}, err
	}
	if t.IsIP {
		ip := t.Resolved[0]
		if err := familyMatches(ip, o.Family); err != nil {
			return Target{}, invalid(field, err.Error())
		}
		if err := r.Policy.Check(ip); err != nil {
			return Target{}, err
		}
		t.IP = ip
		return t, nil
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	lctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	network := map[int]string{4: "ip4", 6: "ip6"}[o.Family]
	if network == "" {
		network = "ip"
	}
	addrs, err := r.Lookup(lctx, network, t.Host)
	if err != nil || len(addrs) == 0 {
		if ctx.Err() != nil {
			return Target{}, ctx.Err() //nolint:wrapcheck // caller cancellation passes through
		}
		return Target{}, &Error{Code: CodeUnresolvable, Message: "the name could not be resolved", Field: field, Err: err}
	}
	var keep []netip.Addr
	for _, a := range addrs {
		a = a.WithZone("")
		if familyMatches(a, o.Family) != nil {
			continue
		}
		// Every resolved address must pass: a name mixing public and private answers is refused.
		if err := r.Policy.Check(a); err != nil {
			return Target{}, err
		}
		keep = append(keep, a)
	}
	if len(keep) == 0 {
		return Target{}, &Error{Code: CodeUnresolvable, Message: "no address of the requested family", Field: field}
	}
	t.Resolved = keep
	t.IP = keep[0]
	return t, nil
}

func familyMatches(a netip.Addr, family int) error {
	switch {
	case family == 4 && !a.Unmap().Is4():
		return errors.New("an IPv4 address is required")
	case family == 6 && (!a.Is6() || a.Is4In6()):
		return errors.New("an IPv6 address is required")
	}
	return nil
}

func parse(input string, o Opts, field string) (Target, error) {
	raw := strings.TrimSpace(input)
	t := Target{Input: raw}
	switch {
	case raw == "":
		return t, invalid(field, "a host name or IP address is required")
	case len(raw) > 300:
		return t, invalid(field, "input is too long")
	case strings.Contains(raw, "://") || strings.ContainsAny(raw, "/?#@\\ \t"):
		return t, invalid(field, "enter a host name or IP address, not a URL")
	}

	host := raw
	if o.AllowPort {
		h, port, err := splitHostPort(raw)
		if err != nil {
			return t, invalid(field, err.Error())
		}
		host, t.Port = h, port
	} else if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	if strings.Contains(host, "%") {
		return t, invalid(field, "IPv6 zone identifiers are not supported")
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		t.Host, t.IsIP, t.Resolved = ip.String(), true, []netip.Addr{ip}
		return t, nil
	}
	if strings.Contains(host, ":") {
		return t, invalid(field, "not a valid IPv6 address")
	}
	if numericHost.MatchString(strings.ToLower(host)) {
		return t, invalid(field, "ambiguous numeric address; use dotted-decimal IPv4 such as 192.0.2.1")
	}

	name, err := normalizeHostname(host)
	if err != nil {
		return t, invalid(field, err.Error())
	}
	t.Host = name
	return t, nil
}

func splitHostPort(s string) (string, int, error) {
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", 0, errors.New("unbalanced [ in IPv6 address")
		}
		host, rest := s[1:end], s[end+1:]
		if rest == "" {
			return host, 0, nil
		}
		if !strings.HasPrefix(rest, ":") {
			return "", 0, errors.New("unexpected characters after ]")
		}
		p, err := parsePort(rest[1:])
		return host, p, err
	}
	if strings.Count(s, ":") > 1 {
		return s, 0, nil // bare IPv6 without port
	}
	if i := strings.LastIndex(s, ":"); i >= 0 {
		p, err := parsePort(s[i+1:])
		return s[:i], p, err
	}
	return s, 0, nil
}

func parsePort(s string) (int, error) {
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 65535 || (len(s) > 1 && s[0] == '0') {
		return 0, errors.New("port must be a number from 1 to 65535")
	}
	return p, nil
}

func normalizeHostname(h string) (string, error) {
	h = strings.TrimSuffix(h, ".")
	ascii, err := idnaProfile.ToASCII(h)
	if err != nil {
		return "", errors.New("not a valid host name")
	}
	ascii = strings.ToLower(ascii)
	if len(ascii) == 0 || len(ascii) > 253 {
		return "", errors.New("host name must be 1-253 characters")
	}
	labels := strings.Split(ascii, ".")
	for _, l := range labels {
		if !label.MatchString(l) {
			return "", errors.New("not a valid host name")
		}
	}
	// A purely numeric last label would be read as an IPv4 address by many parsers.
	if numericHost.MatchString(labels[len(labels)-1]) {
		return "", errors.New("not a valid host name")
	}
	return ascii, nil
}
