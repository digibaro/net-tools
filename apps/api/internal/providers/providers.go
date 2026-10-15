// Package providers calls public third-party APIs (docs/SECURITY.md §6): every request goes
// through the target policy, has a 10 s timeout and a 5 MiB cap, identifies itself with the
// NetTools user agent, is cached, and never leaks keys or key-bearing URLs into errors or logs.
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/cache"
	"github.com/digibaro/net-tools-docker/apps/api/internal/netx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
)

// Timeout bounds every provider call unless a request asks for more (at most MaxTimeout).
const (
	Timeout    = 10 * time.Second
	MaxTimeout = 30 * time.Second
)

// DefaultTTL applies when a request sets none (RIPEstat, PeeringDB, RDAP: 10 min).
const DefaultTTL = 10 * time.Minute

// Client is shared by all provider-backed tools.
type Client struct {
	HTTP  *http.Client
	cache *cache.TTL[[]byte]
}

// New builds a client whose dials go through res.
func New(res *target.Resolver, userAgent string) *Client {
	// The transport allows MaxTimeout; each request's context applies its own (shorter) limit.
	return &Client{HTTP: netx.PolicyClient(res, userAgent, MaxTimeout), cache: cache.New[[]byte](2000)}
}

// Request describes one GET.
type Request struct {
	// Provider names the upstream in error messages ("RIPEstat").
	Provider string
	URL      string
	Header   http.Header
	// TTL for the cached body; 0 → DefaultTTL, negative → not cached.
	TTL time.Duration
	// CacheKey defaults to URL; set it when the URL or headers carry a key.
	CacheKey string
	// Timeout overrides the 10 s default for slow providers (e.g. crt.sh), capped at MaxTimeout.
	Timeout time.Duration
	// Method defaults to GET; any other method is never cached.
	Method string
	// Body is sent as JSON when set.
	Body any
	// UpstreamMessage extracts a client-safe message from a 4xx error body (e.g. "no probes
	// found in this location"); return "" to keep the generic message.
	UpstreamMessage func(body []byte) string
}

// codeProviderError mirrors httpapi.CodeProviderError (this package cannot import httpapi).
const codeProviderError = "PROVIDER_ERROR"

// Error is a client-safe provider failure. Its text never contains the request URL.
type Error struct {
	Provider string
	Message  string
	Err      error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// APICode implements httpapi.Coder.
func (e *Error) APICode() string { return codeProviderError }

// APIMessage implements httpapi.Coder.
func (e *Error) APIMessage() string { return e.Message }

// APIDetails implements httpapi.Coder.
func (e *Error) APIDetails() map[string]any { return map[string]any{"provider": e.Provider} }

// StatusError is an upstream HTTP error; callers can inspect Status (e.g. 404 = not found).
type StatusError struct {
	Provider string
	Status   int
	// Detail is the provider's own explanation, when Request.UpstreamMessage supplied one.
	Detail string
}

func (e *StatusError) Error() string { return fmt.Sprintf("%s returned HTTP %d", e.Provider, e.Status) }

// APICode implements httpapi.Coder.
func (e *StatusError) APICode() string { return codeProviderError }

// APIMessage implements httpapi.Coder.
func (e *StatusError) APIMessage() string {
	if e.Detail != "" {
		return e.Provider + ": " + e.Detail
	}
	switch e.Status {
	case http.StatusTooManyRequests:
		return e.Provider + " is rate limiting this server; try again later"
	case http.StatusUnauthorized, http.StatusForbidden:
		return e.Provider + " rejected the request (check the configured API key)"
	}
	return fmt.Sprintf("%s returned an error (HTTP %d)", e.Provider, e.Status)
}

// APIDetails implements httpapi.Coder.
func (e *StatusError) APIDetails() map[string]any {
	return map[string]any{"provider": e.Provider, "status": e.Status}
}

// IsNotFound reports an upstream 404.
func IsNotFound(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && se.Status == http.StatusNotFound
}

// Get returns the response body (from cache when fresh).
func (c *Client) Get(ctx context.Context, r Request) ([]byte, error) {
	key := r.CacheKey
	if key == "" {
		key = r.URL
	}
	fetch := func(ctx context.Context) ([]byte, error) { return c.fetch(ctx, r) }
	if r.TTL < 0 || (r.Method != "" && r.Method != http.MethodGet) {
		return fetch(ctx)
	}
	ttl := r.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}
	return c.cache.Do(ctx, key, ttl, fetch)
}

// GetJSON decodes the body into out.
func (c *Client) GetJSON(ctx context.Context, r Request, out any) error {
	b, err := c.Get(ctx, r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return &Error{Provider: r.Provider, Message: r.Provider + " returned an unexpected response", Err: err}
	}
	return nil
}

func (c *Client) fetch(ctx context.Context, r Request) ([]byte, error) {
	timeout := Timeout
	if r.Timeout > 0 {
		timeout = min(r.Timeout, MaxTimeout)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if r.Body != nil {
		b, err := json.Marshal(r.Body)
		if err != nil {
			return nil, fmt.Errorf("%s: encode body: %w", r.Provider, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.URL, body)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", r.Provider, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range r.Header {
		req.Header[k] = v
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, transportError(r.Provider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		eb, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		se := &StatusError{Provider: r.Provider, Status: resp.StatusCode}
		if r.UpstreamMessage != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			se.Detail = truncate(r.UpstreamMessage(eb), 300)
		}
		return nil, se
	}
	b, err := io.ReadAll(resp.Body) // capped by PolicyClient (5 MiB)
	if err != nil {
		return nil, transportError(r.Provider, err)
	}
	return b, nil
}

// transportError drops the URL (which may contain a key) from net/http errors.
func transportError(provider string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	var coder interface{ APICode() string }
	if errors.As(err, &coder) {
		return err // policy errors (e.g. a provider host resolving privately) keep their code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Provider: provider, Message: provider + " did not answer in time", Err: err}
	}
	return &Error{Provider: provider, Message: provider + " could not be reached", Err: err}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
