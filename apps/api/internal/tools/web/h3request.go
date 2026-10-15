package web

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
)

// H3Request is a real HTTP/3 GET to the pinned address, then a second one to test session
// resumption and 0-RTT (WEB-05).
type H3Request struct {
	OK      bool    `json:"ok"`
	Status  int     `json:"status"`
	Server  string  `json:"server"`
	TTFBMs  float64 `json:"ttfbMs"`
	Resumed bool    `json:"resumed"`
	ZeroRTT bool    `json:"zeroRtt"`
	// RepeatTTFBMs is the time to first byte of the resumed request (0-RTT when accepted).
	RepeatTTFBMs float64 `json:"repeatTtfbMs"`
	Error        string  `json:"error"`
}

// maxH3Body bounds what is read of each HTTP/3 response.
const maxH3Body = 1 << 20

func h3Get(ctx context.Context, addr netip.AddrPort, u *url.URL, ua string) H3Request {
	out := H3Request{}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tlsCfg := &tls.Config{ServerName: u.Hostname(), NextProtos: []string{http3.NextProtoH3}, InsecureSkipVerify: true, MinVersion: tls.VersionTLS13, ClientSessionCache: tls.NewLRUClientSessionCache(4)} //nolint:gosec // inspection only
	var (
		mu   sync.Mutex
		last *quic.Conn
	)
	dial := func(ctx context.Context, _ string, tc *tls.Config, qc *quic.Config) (*quic.Conn, error) {
		c, err := quic.DialAddrEarly(ctx, addr.String(), tc, qc)
		mu.Lock()
		last = c
		mu.Unlock()
		return c, err //nolint:wrapcheck // reported as text
	}
	type reply struct {
		status int
		server string
		ttfb   float64
	}
	round := func(method string) (reply, error) {
		tr := &http3.Transport{TLSClientConfig: tlsCfg, QUICConfig: &quic.Config{HandshakeIdleTimeout: 5 * time.Second}, Dial: dial}
		defer tr.Close()
		// The URL host only sets SNI and :authority; dial connects to the policy-checked address.
		req, err := http.NewRequestWithContext(ctx, method, u.String(), nil) //nolint:gosec // dial is pinned to addr
		if err != nil {
			return reply{}, err //nolint:wrapcheck // reported as text
		}
		req.Header.Set("User-Agent", ua)
		start := time.Now()
		resp, err := tr.RoundTrip(req)
		if err != nil {
			return reply{}, err //nolint:wrapcheck // reported as text
		}
		defer resp.Body.Close()
		rep := reply{status: resp.StatusCode, server: resp.Header.Get("Server"), ttfb: rttstats.Ms(time.Since(start))}
		_, _ = io.CopyN(io.Discard, resp.Body, maxH3Body) // lets the session ticket arrive
		return rep, nil
	}
	first, err := round(http.MethodGet)
	if err != nil {
		out.Error = "HTTP/3 request failed: " + err.Error()
		return out
	}
	out.OK, out.Status, out.Server, out.TTFBMs = true, first.status, first.server, first.ttfb
	if again, err := round(http3.MethodGet0RTT); err == nil {
		mu.Lock()
		cs := last.ConnectionState()
		mu.Unlock()
		out.Resumed, out.ZeroRTT, out.RepeatTTFBMs = cs.TLS.DidResume, cs.Used0RTT, again.ttfb
	}
	return out
}
