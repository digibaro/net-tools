package target

import (
	"context"
	"errors"
	"net/netip"
	"sync/atomic"
	"testing"
)

type fakeDNS map[string][]string

func (f fakeDNS) lookup(_ context.Context, network, host string) ([]netip.Addr, error) {
	ss, ok := f[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	var out []netip.Addr
	for _, s := range ss {
		a := netip.MustParseAddr(s)
		if (network == "ip4" && !a.Is4()) || (network == "ip6" && !a.Is6()) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

var dns = fakeDNS{
	"example.com":       {"93.184.215.14", "2606:2800:21f:cb07:6820:80da:af6b:8b2c"},
	"internal.test":     {"10.1.2.3"},
	"mixed.test":        {"93.184.215.14", "192.168.1.1"},
	"metadata.test":     {"169.254.169.254"},
	"v6only.test":       {"2001:4860:4860::8888"},
	"xn--mnchen-3ya.de": {"81.169.145.78"},
	"localhost":         {"127.0.0.1", "::1"},
}

func resolver(p Policy) *Resolver {
	return &Resolver{Policy: p, Lookup: dns.lookup}
}

func code(err error) string {
	var te *Error
	if errors.As(err, &te) {
		return te.Code
	}
	if err != nil {
		return "OTHER:" + err.Error()
	}
	return ""
}

func TestPolicyDenyRanges(t *testing.T) {
	strict := Policy{}
	relaxed := Policy{AllowPrivate: true}
	tests := []struct {
		addr            string
		strict, relaxed string
	}{
		// always denied
		{"0.0.0.0", CodeBlocked, CodeBlocked},
		{"0.1.2.3", CodeBlocked, CodeBlocked},
		{"::", CodeBlocked, CodeBlocked},
		{"169.254.169.254", CodeBlocked, CodeBlocked},
		{"fd00:ec2::254", CodeBlocked, CodeBlocked},
		{"100.100.100.200", CodeBlocked, CodeBlocked},
		{"168.63.129.16", CodeBlocked, CodeBlocked},
		{"224.0.0.1", CodeBlocked, CodeBlocked},
		{"239.255.255.250", CodeBlocked, CodeBlocked},
		{"ff02::1", CodeBlocked, CodeBlocked},
		{"255.255.255.255", CodeBlocked, CodeBlocked},
		{"240.0.0.1", CodeBlocked, CodeBlocked},
		// private unless allowed
		{"10.0.0.1", CodePrivateDenied, ""},
		{"172.16.0.1", CodePrivateDenied, ""},
		{"172.31.255.255", CodePrivateDenied, ""},
		{"192.168.1.1", CodePrivateDenied, ""},
		{"100.64.0.1", CodePrivateDenied, ""},
		{"127.0.0.1", CodePrivateDenied, ""},
		{"127.255.255.254", CodePrivateDenied, ""},
		{"169.254.1.1", CodePrivateDenied, ""},
		{"::1", CodePrivateDenied, ""},
		{"fc00::1", CodePrivateDenied, ""},
		{"fd12:3456::1", CodePrivateDenied, ""},
		{"fe80::1", CodePrivateDenied, ""},
		// IPv4 embedded in IPv6
		{"::ffff:127.0.0.1", CodePrivateDenied, ""},
		{"::ffff:10.0.0.1", CodePrivateDenied, ""},
		{"::ffff:169.254.169.254", CodeBlocked, CodeBlocked},
		{"::127.0.0.1", CodePrivateDenied, ""},
		{"2002:7f00:1::", CodePrivateDenied, ""},
		{"2002:a9fe:a9fe::1", CodeBlocked, CodeBlocked},
		{"64:ff9b::10.0.0.1", CodePrivateDenied, ""},
		{"64:ff9b::a9fe:a9fe", CodeBlocked, CodeBlocked},
		{"64:ff9b:1::c0a8:101", CodePrivateDenied, ""},
		// public
		{"8.8.8.8", "", ""},
		{"172.32.0.1", "", ""},
		{"2606:4700::1111", "", ""},
		{"::ffff:8.8.8.8", "", ""},
		{"2002:808:808::1", "", ""},
		{"64:ff9b::808:808", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			a := netip.MustParseAddr(tt.addr)
			if got := code(strict.Check(a)); got != tt.strict {
				t.Errorf("strict: got %q, want %q", got, tt.strict)
			}
			if got := code(relaxed.Check(a)); got != tt.relaxed {
				t.Errorf("relaxed: got %q, want %q", got, tt.relaxed)
			}
		})
	}
}

func TestPolicyDenyAllowSelf(t *testing.T) {
	p := Policy{
		AllowPrivate: true,
		Deny:         []netip.Prefix{netip.MustParsePrefix("203.0.113.0/24")},
		Self:         []netip.Addr{netip.MustParseAddr("172.18.0.3")},
	}
	if code(p.Check(netip.MustParseAddr("203.0.113.9"))) != CodeBlocked {
		t.Error("DENY_CIDRS not applied")
	}
	if code(p.Check(netip.MustParseAddr("172.18.0.3"))) != "" {
		t.Error("self address must be allowed when private targets are allowed")
	}
	p.AllowPrivate = false
	if code(p.Check(netip.MustParseAddr("172.18.0.3"))) != CodePrivateDenied {
		t.Error("self address must be denied in public profile")
	}

	allow := Policy{Allow: []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")}}
	if code(allow.Check(netip.MustParseAddr("198.51.100.7"))) != "" || code(allow.Check(netip.MustParseAddr("8.8.8.8"))) != CodeBlocked {
		t.Error("ALLOW_CIDRS mode not enforced")
	}
	if code(allow.Check(netip.MustParseAddr("::ffff:198.51.100.7"))) != "" {
		t.Error("ALLOW_CIDRS must match mapped addresses")
	}
}

func TestResolveParsing(t *testing.T) {
	r := resolver(Policy{})
	tests := []struct {
		input    string
		opts     Opts
		wantCode string
		wantHost string
		wantIP   string
		wantPort int
	}{
		{"example.com", Opts{}, "", "example.com", "93.184.215.14", 0},
		{"EXAMPLE.COM.", Opts{}, "", "example.com", "93.184.215.14", 0},
		{"  example.com  ", Opts{}, "", "example.com", "93.184.215.14", 0},
		{"example.com", Opts{Family: 6}, "", "example.com", "2606:2800:21f:cb07:6820:80da:af6b:8b2c", 0},
		{"münchen.de", Opts{}, "", "xn--mnchen-3ya.de", "81.169.145.78", 0},
		{"MÜNCHEN.DE", Opts{}, "", "xn--mnchen-3ya.de", "81.169.145.78", 0},
		{"ｍünchen.de", Opts{}, "", "xn--mnchen-3ya.de", "81.169.145.78", 0}, // fullwidth m normalized
		{"8.8.8.8", Opts{}, "", "8.8.8.8", "8.8.8.8", 0},
		{"[2001:4860:4860::8888]", Opts{}, "", "2001:4860:4860::8888", "2001:4860:4860::8888", 0},
		{"example.com:443", Opts{AllowPort: true}, "", "example.com", "93.184.215.14", 443},
		{"[2001:4860:4860::8888]:853", Opts{AllowPort: true}, "", "2001:4860:4860::8888", "2001:4860:4860::8888", 853},
		{"2001:4860:4860::8888", Opts{AllowPort: true}, "", "2001:4860:4860::8888", "2001:4860:4860::8888", 0},
		// invalid input
		{"", Opts{}, CodeInvalid, "", "", 0},
		{"http://example.com", Opts{}, CodeInvalid, "", "", 0},
		{"user@example.com", Opts{}, CodeInvalid, "", "", 0},
		{"example.com/path", Opts{}, CodeInvalid, "", "", 0},
		{"exa mple.com", Opts{}, CodeInvalid, "", "", 0},
		{"fe80::1%eth0", Opts{}, CodeInvalid, "", "", 0},
		{"2130706433", Opts{}, CodeInvalid, "", "", 0},
		{"0177.0.0.1", Opts{}, CodeInvalid, "", "", 0},
		{"0x7f.1", Opts{}, CodeInvalid, "", "", 0},
		{"0x7f000001", Opts{}, CodeInvalid, "", "", 0},
		{"127.1", Opts{}, CodeInvalid, "", "", 0},
		{"127.0.0.1.", Opts{}, CodeInvalid, "", "", 0},
		{"foo.123", Opts{}, CodeInvalid, "", "", 0},
		{"-bad-.example", Opts{}, CodeInvalid, "", "", 0},
		{"example.com:0", Opts{AllowPort: true}, CodeInvalid, "", "", 0},
		{"example.com:65536", Opts{AllowPort: true}, CodeInvalid, "", "", 0},
		{"example.com:080", Opts{AllowPort: true}, CodeInvalid, "", "", 0},
		{"8.8.8.8", Opts{Family: 6}, CodeInvalid, "", "", 0},
		// resolution and policy
		{"nonexistent.test", Opts{}, CodeUnresolvable, "", "", 0},
		{"v6only.test", Opts{Family: 4}, CodeUnresolvable, "", "", 0},
		{"internal.test", Opts{}, CodePrivateDenied, "", "", 0},
		{"localhost", Opts{}, CodePrivateDenied, "", "", 0},
		{"mixed.test", Opts{}, CodePrivateDenied, "", "", 0},
		{"metadata.test", Opts{}, CodeBlocked, "", "", 0},
		{"127.0.0.1", Opts{}, CodePrivateDenied, "", "", 0},
		{"[::ffff:7f00:1]", Opts{}, CodePrivateDenied, "", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Resolve(context.Background(), tt.input, tt.opts)
			if c := code(err); c != tt.wantCode {
				t.Fatalf("code = %q (%v), want %q", c, err, tt.wantCode)
			}
			if err != nil {
				return
			}
			if got.Host != tt.wantHost || got.IP.String() != tt.wantIP || got.Port != tt.wantPort {
				t.Errorf("got host=%q ip=%s port=%d", got.Host, got.IP, got.Port)
			}
		})
	}
}

func TestResolveRelaxedProfile(t *testing.T) {
	r := resolver(Policy{AllowPrivate: true})
	got, err := r.Resolve(context.Background(), "internal.test", Opts{})
	if err != nil || got.IP.String() != "10.1.2.3" {
		t.Fatalf("got %v %v", got, err)
	}
	if _, err := r.Resolve(context.Background(), "metadata.test", Opts{}); code(err) != CodeBlocked {
		t.Errorf("metadata must stay blocked when private targets are allowed: %v", err)
	}
}

// DNS rebinding: the first answer is public, later answers private. Resolve pins the first
// answer; a later Resolve (e.g. after an HTTP redirect) sees the private address and is refused.
func TestResolveRebinding(t *testing.T) {
	var calls atomic.Int32
	r := &Resolver{Lookup: func(_ context.Context, _, _ string) ([]netip.Addr, error) {
		if calls.Add(1) == 1 {
			return []netip.Addr{netip.MustParseAddr("93.184.215.14")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}}
	first, err := r.Resolve(context.Background(), "rebind.test", Opts{})
	if err != nil || first.IP.String() != "93.184.215.14" {
		t.Fatalf("first resolve: %v %v", first, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("Resolve must look up exactly once, got %d", calls.Load())
	}
	if first.AddrPort(443) != "93.184.215.14:443" {
		t.Errorf("AddrPort = %s", first.AddrPort(443))
	}
	if _, err := r.Resolve(context.Background(), "rebind.test", Opts{}); code(err) != CodePrivateDenied {
		t.Errorf("second resolve must be denied, got %v", err)
	}
}

func TestResolveCancelled(t *testing.T) {
	r := &Resolver{Lookup: func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Resolve(ctx, "slow.test", Opts{}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestErrorDetails(t *testing.T) {
	_, err := resolver(Policy{}).Resolve(context.Background(), "internal.test", Opts{Field: "host"})
	var te *Error
	if !errors.As(err, &te) || te.APICode() != CodePrivateDenied || te.APIDetails()["address"] != "10.1.2.3" {
		t.Fatalf("err = %#v", err)
	}
	_, err = resolver(Policy{}).Resolve(context.Background(), "", Opts{Field: "host"})
	if !errors.As(err, &te) || te.APIDetails()["field"] != "host" {
		t.Fatalf("err = %#v", err)
	}
}
