// Package httpcore performs inspected HTTP requests for the web tools: pinned dialing, redirect
// chains re-resolved through the target policy per hop, httptrace timings and bounded bodies.
package httpcore

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/netx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
)

// Methods allowed for inspection requests.
var Methods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// MaxRequestBody bounds the user-supplied request body.
const MaxRequestBody = 64 << 10

// forbiddenHeaders are managed by the client or would confuse connection handling.
var forbiddenHeaders = map[string]bool{
	"Connection": true, "Content-Length": true, "Transfer-Encoding": true, "Upgrade": true, "Te": true,
	"Trailer": true, "Proxy-Connection": true, "Keep-Alive": true, "Expect": true,
}

// Header is a single response or request header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Timing breaks one request down like curl -w.
type Timing struct {
	ResolveMs  float64 `json:"resolveMs"`
	ConnectMs  float64 `json:"connectMs"`
	TLSMs      float64 `json:"tlsMs"`
	TTFBMs     float64 `json:"ttfbMs"`
	TransferMs float64 `json:"transferMs"`
	TotalMs    float64 `json:"totalMs"`
}

// TLSInfo summarizes the negotiated TLS session.
type TLSInfo struct {
	Version     string `json:"version"`
	Cipher      string `json:"cipher"`
	ALPN        string `json:"alpn"`
	ServerName  string `json:"serverName"`
	Subject     string `json:"subject"`
	Issuer      string `json:"issuer"`
	NotAfter    string `json:"notAfter"`
	VerifyError string `json:"verifyError"`
}

// Hop is one request/response in a chain.
type Hop struct {
	URL           string   `json:"url"`
	Method        string   `json:"method"`
	RemoteAddr    string   `json:"remoteAddr"`
	Status        int      `json:"status"`
	StatusText    string   `json:"statusText"`
	Proto         string   `json:"proto"`
	Headers       []Header `json:"headers"`
	Location      string   `json:"location"`
	Timing        Timing   `json:"timing"`
	TLS           *TLSInfo `json:"tls"`
	BodyPreview   string   `json:"bodyPreview"`
	BodyBytes     int      `json:"bodyBytes"`
	BodyTruncated bool     `json:"bodyTruncated"`
	ContentType   string   `json:"contentType"`
	// Error is set when this hop could not be fetched (e.g. a redirect to a denied address).
	Error     string `json:"error"`
	ErrorCode string `json:"errorCode"`
}

// Request describes what to fetch.
type Request struct {
	URL             string
	Method          string
	Headers         map[string]string
	Body            string
	FollowRedirects bool
	MaxRedirects    int
	// MaxBody bounds the response body read per hop (HTTP_MAX_BODY_BYTES).
	MaxBody int64
	// PreviewBytes bounds the body text returned (0 = no body in the result).
	PreviewBytes int
	// Insecure skips certificate verification (the verify result is still reported).
	Insecure  bool
	UserAgent string
	Timeout   time.Duration
}

// Client fetches with the target policy.
type Client struct {
	Resolver *target.Resolver
	// TLSRoots overrides the system trust store (tests).
	TLSRoots *tls.Config
}

// ValidateHeaders checks user request headers.
func ValidateHeaders(h map[string]string) error {
	if len(h) > 30 {
		return errors.New("at most 30 request headers")
	}
	for k, v := range h {
		ck := http.CanonicalHeaderKey(strings.TrimSpace(k))
		if ck == "" || strings.ContainsAny(ck, " :\r\n") || strings.ContainsAny(v, "\r\n") || len(v) > 4096 {
			return fmt.Errorf("invalid header %q", k)
		}
		if forbiddenHeaders[ck] {
			return fmt.Errorf("header %s cannot be set", ck)
		}
	}
	return nil
}

// Do runs the request (and redirects if asked). The first hop's errors are returned as error;
// later hops record their error in the chain so the user sees where it stopped.
func (c *Client) Do(ctx context.Context, req Request) ([]Hop, error) {
	u, err := netx.ParseHTTPURL(req.URL)
	if err != nil {
		return nil, err //nolint:wrapcheck // client-facing
	}
	if req.MaxRedirects <= 0 || req.MaxRedirects > netx.MaxRedirects {
		req.MaxRedirects = netx.MaxRedirects
	}
	method := req.Method
	body := req.Body
	var hops []Hop
	for i := 0; ; i++ {
		hop, next, err := c.one(ctx, u, method, body, req)
		if err != nil {
			if i == 0 {
				return nil, err
			}
			hop.Error, hop.ErrorCode = errMessage(err)
			hops = append(hops, hop)
			return hops, nil
		}
		hops = append(hops, hop)
		if next == nil || !req.FollowRedirects {
			return hops, nil
		}
		if i >= req.MaxRedirects {
			hops = append(hops, Hop{URL: next.String(), Headers: []Header{}, Error: "too many redirects (limit 10)", ErrorCode: "TOO_MANY_REDIRECTS"})
			return hops, nil
		}
		// 303, and 301/302 for non-GET/HEAD, switch to GET without a body (browser behaviour).
		if hop.Status == http.StatusSeeOther || ((hop.Status == http.StatusMovedPermanently || hop.Status == http.StatusFound) && method != http.MethodGet && method != http.MethodHead) {
			method, body = http.MethodGet, ""
		}
		u = next
	}
}

func errMessage(err error) (string, string) {
	var c interface {
		APICode() string
		APIMessage() string
	}
	if errors.As(err, &c) {
		return c.APIMessage(), c.APICode()
	}
	return "request failed", "PROVIDER_ERROR"
}

func (c *Client) one(ctx context.Context, u *url.URL, method, body string, req Request) (Hop, *url.URL, error) {
	hop := Hop{URL: u.String(), Method: method, Headers: []Header{}}
	resolveStart := time.Now()
	tgt, err := netx.ResolveURL(ctx, c.Resolver, u)
	if err != nil {
		return hop, nil, err //nolint:wrapcheck // client-facing target error
	}
	hop.Timing.ResolveMs = rttstats.Ms(time.Since(resolveStart))
	hop.RemoteAddr = tgt.AddrPort(0)

	opts := netx.ClientOptions{Timeout: req.Timeout, InsecureSkipVerify: true}
	if c.TLSRoots != nil {
		opts.TLSConfig = c.TLSRoots
	}
	client := netx.HTTPClient(tgt, opts)

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	hreq, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return hop, nil, &target.Error{Code: target.CodeInvalid, Message: "invalid request", Field: "url", Err: err}
	}
	for k, v := range req.Headers {
		hreq.Header.Set(k, v)
	}
	if hreq.Header.Get("User-Agent") == "" {
		hreq.Header.Set("User-Agent", req.UserAgent)
	}

	var connectStart, connectDone, tlsStart, tlsDone, firstByte time.Time
	trace := &httptrace.ClientTrace{
		ConnectStart:         func(string, string) { connectStart = time.Now() },
		ConnectDone:          func(string, string, error) { connectDone = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { tlsDone = time.Now() },
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}
	start := time.Now()
	resp, err := client.Do(hreq.WithContext(httptrace.WithClientTrace(ctx, trace)))
	if err != nil {
		if ctx.Err() != nil {
			return hop, nil, ctx.Err() //nolint:wrapcheck // timeout/cancel mapped by httpapi
		}
		return hop, nil, &fetchError{err: err}
	}
	defer resp.Body.Close()
	data, truncated, rerr := netx.ReadLimited(resp.Body, req.MaxBody)
	end := time.Now()

	hop.Status, hop.StatusText, hop.Proto = resp.StatusCode, http.StatusText(resp.StatusCode), resp.Proto
	hop.ContentType = resp.Header.Get("Content-Type")
	hop.BodyBytes, hop.BodyTruncated = len(data), truncated || rerr != nil
	if req.PreviewBytes > 0 {
		preview := data
		if len(preview) > req.PreviewBytes {
			preview, hop.BodyTruncated = preview[:req.PreviewBytes], true
		}
		hop.BodyPreview = string(bytes.ToValidUTF8(preview, []byte("�")))
	}
	for name, values := range resp.Header {
		for _, v := range values {
			hop.Headers = append(hop.Headers, Header{Name: name, Value: v})
		}
	}
	sort.SliceStable(hop.Headers, func(a, b int) bool { return hop.Headers[a].Name < hop.Headers[b].Name })

	since := func(a, b time.Time) float64 {
		if a.IsZero() || b.IsZero() {
			return 0
		}
		return rttstats.Ms(b.Sub(a))
	}
	hop.Timing.ConnectMs = since(connectStart, connectDone)
	hop.Timing.TLSMs = since(tlsStart, tlsDone)
	hop.Timing.TTFBMs = since(start, firstByte)
	hop.Timing.TransferMs = since(firstByte, end)
	hop.Timing.TotalMs = rttstats.Ms(end.Sub(start)) + hop.Timing.ResolveMs

	if resp.TLS != nil {
		hop.TLS = tlsInfo(resp.TLS, tgt)
	}
	var next *url.URL
	if loc := resp.Header.Get("Location"); loc != "" && resp.StatusCode >= 300 && resp.StatusCode < 400 {
		hop.Location = loc
		if n, err := u.Parse(loc); err == nil {
			next = n
		}
	}
	return hop, next, nil
}

func tlsInfo(st *tls.ConnectionState, tgt target.Target) *TLSInfo {
	info := &TLSInfo{Version: tls.VersionName(st.Version), Cipher: tls.CipherSuiteName(st.CipherSuite), ALPN: st.NegotiatedProtocol, ServerName: st.ServerName}
	if len(st.PeerCertificates) == 0 {
		return info
	}
	leaf := st.PeerCertificates[0]
	info.Subject, info.Issuer, info.NotAfter = leaf.Subject.String(), leaf.Issuer.String(), leaf.NotAfter.UTC().Format(time.RFC3339)
	name := tgt.Host
	if err := VerifyChain(st, name); err != nil {
		info.VerifyError = err.Error()
	}
	return info
}

type fetchError struct{ err error }

func (e *fetchError) Error() string      { return e.err.Error() }
func (e *fetchError) Unwrap() error      { return e.err }
func (e *fetchError) APICode() string    { return "PROVIDER_ERROR" }
func (e *fetchError) APIMessage() string { return "the request failed: " + simplify(e.err) }
func (e *fetchError) APIDetails() map[string]any {
	return nil
}

func simplify(err error) string {
	s := err.Error()
	for _, k := range []string{"connection refused", "no such host", "i/o timeout", "connection reset", "tls:", "EOF", "certificate"} {
		if i := strings.Index(s, k); i >= 0 {
			return s[i:]
		}
	}
	return "connection error"
}
