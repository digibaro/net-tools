// Package netx builds network clients that only ever dial a pinned, policy-checked address.
package netx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
)

// MaxRedirects bounds redirect chains (docs/SECURITY.md §2.4).
const MaxRedirects = 10

// PinnedDialer ignores the host in the dial address and connects to ip on the requested port.
func PinnedDialer(ip netip.Addr, timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("pinned dial: %w", err)
		}
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err != nil {
			return nil, fmt.Errorf("pinned dial: %w", err)
		}
		return conn, nil
	}
}

// ClientOptions tunes HTTPClient.
type ClientOptions struct {
	Timeout            time.Duration
	InsecureSkipVerify bool
	// DisableHTTP2 forces HTTP/1.1 (useful for exact header inspection).
	DisableHTTP2 bool
	// TLSConfig, if set, is cloned and used (ServerName is still forced to the target host).
	TLSConfig *tls.Config
}

// HTTPClient returns a client that dials only t.IP, never follows redirects, ignores proxy
// environment variables and sends SNI for the original host.
func HTTPClient(t target.Target, o ClientOptions) *http.Client {
	if o.Timeout <= 0 {
		o.Timeout = 15 * time.Second
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS10} //nolint:gosec // inspection tools must reach legacy servers
	if o.TLSConfig != nil {
		tlsCfg = o.TLSConfig.Clone()
	}
	if !t.IsIP {
		tlsCfg.ServerName = t.Host
	}
	tlsCfg.InsecureSkipVerify = o.InsecureSkipVerify //nolint:gosec // opt-in per request for inspection
	tr := &http.Transport{
		Proxy:                  nil,
		DialContext:            PinnedDialer(t.IP, o.Timeout),
		TLSClientConfig:        tlsCfg,
		TLSHandshakeTimeout:    o.Timeout,
		ResponseHeaderTimeout:  o.Timeout,
		DisableKeepAlives:      true,
		MaxResponseHeaderBytes: 256 << 10,
		ForceAttemptHTTP2:      !o.DisableHTTP2,
	}
	if o.DisableHTTP2 {
		tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	}
	return &http.Client{
		Transport:     tr,
		Timeout:       o.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// URLError is a validation error for user URLs.
type URLError struct{ msg string }

func (e *URLError) Error() string              { return e.msg }
func (e *URLError) APICode() string            { return "INVALID_INPUT" }
func (e *URLError) APIMessage() string         { return e.msg }
func (e *URLError) APIDetails() map[string]any { return map[string]any{"field": "url"} }
func urlErr(format string, a ...any) error     { return &URLError{msg: fmt.Sprintf(format, a...)} }

var _ interface{ APICode() string } = (*URLError)(nil)

// ParseHTTPURL validates a user URL: http(s) only, no userinfo, host required. A missing scheme
// defaults to https.
func ParseHTTPURL(input string) (*url.URL, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return nil, urlErr("a URL is required")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, urlErr("not a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, urlErr("only http and https URLs are supported")
	}
	if u.User != nil {
		return nil, urlErr("URLs with credentials are not supported")
	}
	if u.Hostname() == "" {
		return nil, urlErr("the URL has no host")
	}
	u.Fragment = ""
	return u, nil
}

// Port returns the URL's port or the scheme default.
func Port(u *url.URL) int {
	if p := u.Port(); p != "" {
		n, _ := strconv.Atoi(p)
		return n
	}
	if u.Scheme == "http" {
		return 80
	}
	return 443
}

// ResolveURL validates u's host through the target policy.
func ResolveURL(ctx context.Context, r *target.Resolver, u *url.URL) (target.Target, error) {
	host := u.Host
	if u.Port() == "" {
		host = net.JoinHostPort(u.Hostname(), strconv.Itoa(Port(u)))
	}
	t, err := r.Resolve(ctx, host, target.Opts{AllowPort: true, Field: "url"})
	if err != nil {
		return target.Target{}, err //nolint:wrapcheck // target errors are client-facing as-is
	}
	return t, nil
}

// ReadLimited reads at most n bytes and reports whether the body was truncated.
func ReadLimited(r io.Reader, n int64) ([]byte, bool, error) {
	b, err := io.ReadAll(io.LimitReader(r, n+1))
	if err != nil && !errors.Is(err, io.EOF) {
		return b, false, fmt.Errorf("read body: %w", err)
	}
	if int64(len(b)) > n {
		return b[:n], true, nil
	}
	return b, false, nil
}
