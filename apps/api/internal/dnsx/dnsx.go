// Package dnsx sends DNS queries over UDP, TCP, DoT and DoH to policy-checked, pinned servers.
package dnsx

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/netx"
	"github.com/digibaro/net-tools/apps/api/internal/target"
)

// Transport is how a query is sent.
type Transport string

const (
	UDP Transport = "udp"
	TCP Transport = "tcp"
	DoT Transport = "dot"
	DoH Transport = "doh"
)

// ValidTransport reports whether s names a transport.
func ValidTransport(s string) bool {
	switch Transport(s) {
	case UDP, TCP, DoT, DoH:
		return true
	}
	return false
}

// Server is a resolved DNS server endpoint.
type Server struct {
	// Label is shown to users, e.g. "Cloudflare (1.1.1.1)".
	Label     string
	Addr      netip.AddrPort
	Transport Transport
	// TLSName is the DoT/DoH certificate name.
	TLSName string
	// DoHURL is the DoH endpoint.
	DoHURL string
	// RootCAs overrides the system trust store for DoT/DoH (tests).
	RootCAs *x509.CertPool
}

// tlsServerName is the name verified on the certificate (the IP itself for IP endpoints).
func (s Server) tlsServerName() string {
	if s.TLSName != "" {
		return s.TLSName
	}
	return s.Addr.Addr().String()
}

// Preset is a well-known public resolver.
type Preset struct {
	ID, Label, TLSName, DoHURL string
	V4, V6                     netip.Addr
}

// Presets are offered in the UI; "system" is handled separately.
var Presets = []Preset{
	{ID: "cloudflare", Label: "Cloudflare", V4: netip.MustParseAddr("1.1.1.1"), V6: netip.MustParseAddr("2606:4700:4700::1111"), TLSName: "one.one.one.one", DoHURL: "https://cloudflare-dns.com/dns-query"},
	{ID: "google", Label: "Google", V4: netip.MustParseAddr("8.8.8.8"), V6: netip.MustParseAddr("2001:4860:4860::8888"), TLSName: "dns.google", DoHURL: "https://dns.google/dns-query"},
	{ID: "quad9", Label: "Quad9", V4: netip.MustParseAddr("9.9.9.9"), V6: netip.MustParseAddr("2620:fe::fe"), TLSName: "dns.quad9.net", DoHURL: "https://dns.quad9.net/dns-query"},
	{ID: "opendns", Label: "OpenDNS", V4: netip.MustParseAddr("208.67.222.222"), V6: netip.MustParseAddr("2620:119:35::35"), TLSName: "dns.opendns.com", DoHURL: "https://doh.opendns.com/dns-query"},
}

// FindPreset returns the preset with id.
func FindPreset(id string) (Preset, bool) {
	for _, p := range Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

func defaultPort(t Transport) uint16 {
	switch t {
	case DoT:
		return 853
	case DoH:
		return 443
	default:
		return 53
	}
}

// ResolvConf is read for the "system" resolver (admin-configured, so not subject to the
// private-target policy: inside Docker it is usually 127.0.0.11).
var ResolvConf = "/etc/resolv.conf"

// SystemOverride, when set, replaces the resolv.conf server (tests only).
var SystemOverride *netip.AddrPort

// SystemServer returns the first nameserver from resolv.conf.
func SystemServer(t Transport) (Server, error) {
	if t == DoT || t == DoH {
		return Server{}, &target.Error{Code: target.CodeInvalid, Message: "the system resolver supports only UDP and TCP", Field: "transport"}
	}
	if SystemOverride != nil {
		return Server{Label: "System (" + SystemOverride.String() + ")", Addr: *SystemOverride, Transport: t}, nil
	}
	cfg, err := dns.ClientConfigFromFile(ResolvConf)
	if err != nil || len(cfg.Servers) == 0 {
		return Server{}, fmt.Errorf("system resolver: %w", errors.Join(err, errors.New("no nameserver configured")))
	}
	ip, err := netip.ParseAddr(cfg.Servers[0])
	if err != nil {
		return Server{}, fmt.Errorf("system resolver: %w", err)
	}
	port, _ := strconv.Atoi(cfg.Port)
	if port == 0 {
		port = 53
	}
	return Server{Label: "System (" + ip.String() + ")", Addr: netip.AddrPortFrom(ip, uint16(port)), Transport: t}, nil //nolint:gosec // port from resolv.conf, 1-65535
}

// ResolveServer turns user input into a Server. spec is "system", a preset id, "ip[:port]",
// "host[:port]" (DoT) or an https URL (DoH). Non-system servers go through the target policy.
func ResolveServer(ctx context.Context, r *target.Resolver, spec string, t Transport, family int) (Server, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "system" {
		return SystemServer(t)
	}
	if p, ok := FindPreset(spec); ok {
		ip := p.V4
		if family == 6 {
			ip = p.V6
		}
		s := Server{Label: fmt.Sprintf("%s (%s)", p.Label, ip), Addr: netip.AddrPortFrom(ip, defaultPort(t)), Transport: t, TLSName: p.TLSName}
		if t == DoH {
			s.DoHURL = p.DoHURL
		}
		return s, nil
	}
	if t == DoH {
		u, err := netx.ParseHTTPURL(spec)
		if err != nil {
			return Server{}, err //nolint:wrapcheck // client-facing validation error
		}
		if u.Scheme != "https" {
			return Server{}, &target.Error{Code: target.CodeInvalid, Message: "DoH requires an https URL", Field: "resolver"}
		}
		if u.Path == "" || u.Path == "/" {
			u.Path = "/dns-query"
		}
		tgt, err := netx.ResolveURL(ctx, r, u)
		if err != nil {
			return Server{}, err //nolint:wrapcheck // client-facing target error
		}
		return Server{Label: u.String(), Addr: netip.AddrPortFrom(tgt.IP, uint16(tgt.Port)), Transport: t, TLSName: tgt.Host, DoHURL: u.String()}, nil //nolint:gosec // port validated
	}
	tgt, err := r.Resolve(ctx, spec, target.Opts{AllowPort: true, Family: family, Field: "resolver"})
	if err != nil {
		return Server{}, err //nolint:wrapcheck // client-facing target error
	}
	port := uint16(tgt.Port) //nolint:gosec // validated 1-65535
	if port == 0 {
		port = defaultPort(t)
	}
	s := Server{Label: net.JoinHostPort(tgt.IP.String(), strconv.Itoa(int(port))), Addr: netip.AddrPortFrom(tgt.IP, port), Transport: t}
	if !tgt.IsIP {
		s.Label = fmt.Sprintf("%s (%s)", tgt.Host, s.Label)
		s.TLSName = tgt.Host
	}
	return s, nil
}

// NewQuery builds a recursive query with EDNS0 (4096) and optional DO bit.
func NewQuery(name string, qtype uint16, dnssec bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), qtype)
	m.RecursionDesired = true
	m.SetEdns0(4096, dnssec)
	return m
}

// Exchange sends m to s and returns the answer and round-trip time. UDP answers that are
// truncated are retried over TCP.
func Exchange(ctx context.Context, s Server, m *dns.Msg) (*dns.Msg, time.Duration, error) {
	switch s.Transport {
	case DoH:
		return exchangeDoH(ctx, s, m)
	case DoT:
		c := &dns.Client{Net: "tcp-tls", Timeout: 5 * time.Second, TLSConfig: &tls.Config{ServerName: s.tlsServerName(), RootCAs: s.RootCAs, MinVersion: tls.VersionTLS12}}
		return exchange(ctx, c, s, m)
	case TCP:
		return exchange(ctx, &dns.Client{Net: "tcp", Timeout: 5 * time.Second}, s, m)
	default:
		r, rtt, err := exchange(ctx, &dns.Client{Net: "udp", Timeout: 3 * time.Second, UDPSize: 4096}, s, m)
		if err == nil && r.Truncated {
			return exchange(ctx, &dns.Client{Net: "tcp", Timeout: 5 * time.Second}, s, m)
		}
		return r, rtt, err
	}
}

func exchange(ctx context.Context, c *dns.Client, s Server, m *dns.Msg) (*dns.Msg, time.Duration, error) {
	r, rtt, err := c.ExchangeContext(ctx, m, s.Addr.String())
	if err != nil {
		return nil, rtt, fmt.Errorf("query %s: %w", s.Label, err)
	}
	return r, rtt, nil
}

func exchangeDoH(ctx context.Context, s Server, m *dns.Msg) (*dns.Msg, time.Duration, error) {
	packed, err := m.Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("pack query: %w", err)
	}
	u, err := url.Parse(s.DoHURL)
	if err != nil {
		return nil, 0, fmt.Errorf("doh url: %w", err)
	}
	tgt := target.Target{Host: s.TLSName, IP: s.Addr.Addr(), Port: int(s.Addr.Port())}
	client := netx.HTTPClient(tgt, netx.ClientOptions{Timeout: 8 * time.Second, TLSConfig: &tls.Config{RootCAs: s.RootCAs, MinVersion: tls.VersionTLS12}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(packed))
	if err != nil {
		return nil, 0, fmt.Errorf("doh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("query %s: %w", s.Label, err)
	}
	defer resp.Body.Close()
	body, _, err := netx.ReadLimited(resp.Body, 64<<10)
	rtt := time.Since(start)
	if err != nil {
		return nil, rtt, err //nolint:wrapcheck // already wrapped
	}
	if resp.StatusCode != http.StatusOK {
		return nil, rtt, fmt.Errorf("query %s: HTTP %d", s.Label, resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	r := new(dns.Msg)
	if err := r.Unpack(body); err != nil {
		return nil, rtt, fmt.Errorf("query %s: invalid DNS response: %w", s.Label, err)
	}
	return r, rtt, nil
}

// Record is a JSON-friendly resource record.
type Record struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   uint32 `json:"ttl"`
	Class string `json:"class"`
	Data  string `json:"data"`
}

// Records converts RRs, dropping OPT pseudo-records.
func Records(rrs []dns.RR) []Record {
	out := make([]Record, 0, len(rrs))
	for _, rr := range rrs {
		h := rr.Header()
		if h.Rrtype == dns.TypeOPT {
			continue
		}
		full := rr.String()
		data := strings.TrimPrefix(full, h.String())
		out = append(out, Record{Name: h.Name, Type: dns.TypeToString[h.Rrtype], TTL: h.Ttl, Class: dns.ClassToString[h.Class], Data: strings.TrimSpace(data)})
	}
	return out
}

// ParseType accepts a record type name (A, MX, …) case-insensitively.
func ParseType(s string) (uint16, bool) {
	t, ok := dns.StringToType[strings.ToUpper(strings.TrimSpace(s))]
	return t, ok
}

// RcodeName returns e.g. "NOERROR", "NXDOMAIN".
func RcodeName(rcode int) string {
	if n, ok := dns.RcodeToString[rcode]; ok {
		return n
	}
	return strconv.Itoa(rcode)
}

// IsResolvConfMissing reports whether the system resolver is unavailable.
func IsResolvConfMissing(err error) bool { return errors.Is(err, os.ErrNotExist) }

// Query sends a single question and returns the reply (rcode is in the message).
func Query(ctx context.Context, s Server, name string, qtype uint16) (*dns.Msg, time.Duration, error) {
	return Exchange(ctx, s, NewQuery(name, qtype, false))
}

// Addresses resolves A and AAAA for name via s.
func Addresses(ctx context.Context, s Server, name string) ([]netip.Addr, error) {
	var out []netip.Addr
	for _, qt := range []uint16{dns.TypeA, dns.TypeAAAA} {
		m, _, err := Query(ctx, s, name, qt)
		if err != nil {
			return out, err
		}
		for _, rr := range m.Answer {
			switch v := rr.(type) {
			case *dns.A:
				if a, ok := netip.AddrFromSlice(v.A.To4()); ok {
					out = append(out, a)
				}
			case *dns.AAAA:
				if a, ok := netip.AddrFromSlice(v.AAAA); ok {
					out = append(out, a)
				}
			}
		}
	}
	return out, nil
}
