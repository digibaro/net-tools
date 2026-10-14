package web

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quic-go/quic-go"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/netx"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/rttstats"
)

// ProtocolsTool is WEB-05: HTTP/2 (ALPN) and HTTP/3 (Alt-Svc + a real QUIC handshake).
type ProtocolsTool struct {
	deps tools.Deps
	// H3Port forces the QUIC port (tests); otherwise Alt-Svc or 443.
	H3Port int
}

func (*ProtocolsTool) ID() string                                { return "web.protocols" }
func (*ProtocolsTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *ProtocolsTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(40*time.Second)).Get("/http/protocols", t.handle)
}

// AltSvc is one Alt-Svc alternative (RFC 7838).
type AltSvc struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	MaxAge   int    `json:"maxAge"`
}

// HTTP3 is the QUIC handshake attempt.
type HTTP3 struct {
	Advertised  bool    `json:"advertised"`
	Port        int     `json:"port"`
	OK          bool    `json:"ok"`
	HandshakeMs float64 `json:"handshakeMs"`
	QUICVersion string  `json:"quicVersion"`
	TLSVersion  string  `json:"tlsVersion"`
	Error       string  `json:"error"`
}

// ProtocolsResult is the WEB-05 payload.
type ProtocolsResult struct {
	URL         string   `json:"url"`
	Address     string   `json:"address"`
	ALPN        string   `json:"alpn"`
	TLSVersion  string   `json:"tlsVersion"`
	HTTP2       bool     `json:"http2"`
	HTTPVersion string   `json:"httpVersion"`
	AltSvc      []AltSvc `json:"altSvc"`
	HTTP3       HTTP3    `json:"http3"`
}

// ParseAltSvc parses an Alt-Svc header value ("clear" yields nothing).
func ParseAltSvc(v string) []AltSvc {
	out := []AltSvc{}
	for _, part := range strings.Split(v, ",") {
		fields := strings.Split(part, ";")
		proto, auth, ok := strings.Cut(strings.TrimSpace(fields[0]), "=")
		if !ok {
			continue
		}
		auth = strings.Trim(auth, `"`)
		host, port, err := net.SplitHostPort(auth)
		if err != nil {
			continue
		}
		p, _ := strconv.Atoi(port)
		a := AltSvc{Protocol: proto, Host: host, Port: p, MaxAge: 86400}
		for _, f := range fields[1:] {
			if k, val, ok := strings.Cut(strings.TrimSpace(f), "="); ok && k == "ma" {
				a.MaxAge, _ = strconv.Atoi(val)
			}
		}
		out = append(out, a)
	}
	return out
}

func (t *ProtocolsTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	u, err := netx.ParseHTTPURL(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if u.Scheme != "https" {
		httpapi.Error(w, r, httpapi.InvalidInput("url", "HTTP/2 and HTTP/3 are negotiated over HTTPS; use an https:// URL"))
		return
	}
	ctx := r.Context()
	tgt, err := netx.ResolveURL(ctx, t.deps.Target, u)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	port := netx.Port(u)
	out := ProtocolsResult{URL: u.String(), Address: tgt.AddrPort(port), AltSvc: []AltSvc{}}

	// 1. ALPN over TLS to the pinned address.
	d := tls.Dialer{NetDialer: &net.Dialer{Timeout: 8 * time.Second}, Config: &tls.Config{ServerName: u.Hostname(), NextProtos: []string{"h2", "http/1.1"}, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}} //nolint:gosec // only negotiation is inspected
	conn, err := d.DialContext(ctx, "tcp", out.Address)
	if err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInvalidInput, Message: "TLS connection failed", Details: map[string]any{"field": "url"}, Err: err})
		return
	}
	st := conn.(*tls.Conn).ConnectionState() //nolint:forcetypeassert // tls.Dialer returns *tls.Conn
	_ = conn.Close()
	out.ALPN, out.TLSVersion = st.NegotiatedProtocol, tls.VersionName(st.Version)
	out.HTTP2 = st.NegotiatedProtocol == "h2"

	// 2. One request for Alt-Svc and the protocol actually used.
	if hops, err := fetch(ctx, t.deps, httpcore.Request{URL: u.String(), Method: "GET"}); err == nil {
		last := hops[len(hops)-1]
		out.HTTPVersion = last.Proto
		if v, ok := header(last, "Alt-Svc"); ok {
			out.AltSvc = ParseAltSvc(v)
		}
	}

	// 3. QUIC handshake with ALPN h3 to the same pinned IP (a different Alt-Svc host is not followed).
	h3 := HTTP3{Port: 443}
	for _, a := range out.AltSvc {
		if a.Protocol == "h3" && (a.Host == "" || strings.EqualFold(a.Host, u.Hostname())) && a.Port > 0 {
			h3.Advertised, h3.Port = true, a.Port
			break
		}
	}
	if t.H3Port != 0 {
		h3.Port = t.H3Port
	}
	quicHandshake(ctx, tgt.IP, u.Hostname(), &h3)
	out.HTTP3 = h3
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Used: out.Address}})
}

func quicHandshake(ctx context.Context, ip netip.Addr, host string, h3 *HTTP3) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	pc, err := net.ListenUDP("udp", nil)
	if err != nil {
		h3.Error = "cannot open a UDP socket"
		return
	}
	defer pc.Close()
	start := time.Now()
	addr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(ip, uint16(h3.Port)))                                                                                                                                      //nolint:gosec // port 1-65535 from Alt-Svc or 443
	conn, err := quic.Dial(ctx, pc, addr, &tls.Config{ServerName: host, NextProtos: []string{"h3"}, InsecureSkipVerify: true, MinVersion: tls.VersionTLS13}, &quic.Config{HandshakeIdleTimeout: 5 * time.Second}) //nolint:gosec // only the handshake is inspected
	if err != nil {
		h3.Error = "no QUIC handshake (UDP blocked, or HTTP/3 not offered)"
		return
	}
	cs := conn.ConnectionState()
	h3.OK, h3.HandshakeMs = cs.TLS.NegotiatedProtocol == "h3", rttstats.Ms(time.Since(start))
	h3.QUICVersion, h3.TLSVersion = cs.Version.String(), tls.VersionName(cs.TLS.Version)
	_ = conn.CloseWithError(0, "")
}
