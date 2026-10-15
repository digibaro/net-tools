package netx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
)

// MaxProviderBody caps provider responses (docs/SECURITY.md §6).
const MaxProviderBody = 5 << 20

// MaxProviderBodyOverride bounds WithBodyLimit (large published lists such as Azure Service Tags).
const MaxProviderBodyOverride = 32 << 20

type bodyLimitKey struct{}

// WithBodyLimit raises the response cap for requests made with ctx (at most MaxProviderBodyOverride).
func WithBodyLimit(ctx context.Context, n int64) context.Context {
	return context.WithValue(ctx, bodyLimitKey{}, n)
}

// PolicyClient returns an HTTP client for calls whose hosts come from data rather than the user
// (RDAP bootstrap, providers). Every dial resolves the host through the target policy and
// connects to the checked address, so redirects and bootstrap targets are covered too.
func PolicyClient(res *target.Resolver, userAgent string, timeout time.Duration) *http.Client {
	d := &net.Dialer{Timeout: timeout}
	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("dial: %w", err)
			}
			p, _ := strconv.Atoi(port)
			tgt, err := res.Resolve(ctx, host, target.Opts{})
			if err != nil {
				return nil, err //nolint:wrapcheck // policy error surfaces to the client
			}
			conn, err := d.DialContext(ctx, network, tgt.AddrPort(p))
			if err != nil {
				return nil, fmt.Errorf("dial: %w", err)
			}
			return conn, nil
		},
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:    timeout,
		ResponseHeaderTimeout:  timeout,
		MaxResponseHeaderBytes: 64 << 10,
		ForceAttemptHTTP2:      true,
	}
	return &http.Client{
		Transport: &limitTransport{base: tr, ua: userAgent, max: MaxProviderBody},
		Timeout:   timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

type limitTransport struct {
	base http.RoundTripper
	ua   string
	max  int64
}

func (l *limitTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if l.ua != "" {
		r.Header.Set("User-Agent", l.ua)
	}
	resp, err := l.base.RoundTrip(r)
	if err != nil {
		return nil, err //nolint:wrapcheck // transport errors pass through
	}
	limit := l.max
	if n, ok := r.Context().Value(bodyLimitKey{}).(int64); ok && n > 0 && n <= MaxProviderBodyOverride {
		limit = n
	}
	resp.Body = &limitedBody{ReadCloser: resp.Body, r: io.LimitReader(resp.Body, limit)}
	return resp, nil
}

type limitedBody struct {
	io.ReadCloser
	r io.Reader
}

func (b *limitedBody) Read(p []byte) (int, error) { return b.r.Read(p) } //nolint:wrapcheck // passthrough
