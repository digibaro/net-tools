// Package tooltest mounts a single tool on the real router for handler tests.
package tooltest

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/execx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// Env is a router with one tool mounted.
type Env struct {
	t       testing.TB
	Handler http.Handler
}

// Options tweak the environment.
type Options struct {
	// AllowPrivate lets tools reach test servers on 127.0.0.1.
	AllowPrivate bool
	// Lookup overrides DNS resolution for targets (defaults to the system resolver).
	Lookup target.LookupFunc
	// Env overrides config variables.
	Env map[string]string
	// Exec overrides the command runner (fake binaries).
	Exec *execx.Runner
	// Deny adds always-denied prefixes (like DENY_CIDRS).
	Deny []string
}

// New mounts tool on a router built like production.
func New(t testing.TB, tool tools.Tool, o Options) *Env {
	t.Helper()
	env := map[string]string{"RATE_LIMIT_RPM": "10000", "RATE_LIMIT_HEAVY_RPM": "10000"}
	for k, v := range o.Env {
		env[k] = v
	}
	cfg, err := config.LoadFrom(func(k string) (string, bool) { v, ok := env[k]; return v, ok }, func(string) ([]byte, error) { return nil, io.EOF })
	if err != nil {
		t.Fatal(err)
	}
	pol := target.Policy{AllowPrivate: o.AllowPrivate}
	for _, d := range o.Deny {
		pol.Deny = append(pol.Deny, netip.MustParsePrefix(d))
	}
	res := target.NewResolver(pol)
	if o.Lookup != nil {
		res.Lookup = o.Lookup
	}
	h := httpapi.NewRouter(httpapi.Options{
		Config: cfg,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Tools:  []tools.Tool{tool},
		Target: res,
		Exec:   o.Exec,
	})
	return &Env{t: t, Handler: h}
}

// StaticLookup resolves every name to the given addresses.
func StaticLookup(addrs ...string) target.LookupFunc {
	return func(context.Context, string, string) ([]netip.Addr, error) {
		out := make([]netip.Addr, len(addrs))
		for i, a := range addrs {
			out[i] = netip.MustParseAddr(a)
		}
		return out, nil
	}
}

// Response is a decoded envelope.
type Response struct {
	Status int
	OK     bool            `json:"ok"`
	Data   json.RawMessage `json:"data"`
	Error  struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
	Raw *httptest.ResponseRecorder
}

// Get performs a GET and decodes the envelope.
func (e *Env) Get(path string) Response {
	e.t.Helper()
	return e.Do(httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
}

// Do performs a request and decodes the envelope (SSE responses are returned undecoded).
func (e *Env) Do(req *http.Request) Response {
	e.t.Helper()
	req.RemoteAddr = "198.51.100.10:40000"
	rec := httptest.NewRecorder()
	e.Handler.ServeHTTP(rec, req)
	r := Response{Status: rec.Code, Raw: rec}
	if ct := rec.Header().Get("Content-Type"); len(ct) >= 16 && ct[:16] == "application/json" {
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			e.t.Fatalf("invalid JSON: %v: %s", err, rec.Body.String())
		}
	}
	return r
}

// Decode unmarshals Data into v.
func (r Response) Decode(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Data, v); err != nil {
		t.Fatalf("decode data: %v: %s", err, r.Data)
	}
}

// ExpectError asserts an error envelope with code.
func (r Response) ExpectError(t testing.TB, code string) {
	t.Helper()
	if r.OK || r.Error.Code != code {
		t.Fatalf("status %d, want error %s; body: %s", r.Status, code, r.Raw.Body.String())
	}
}

// Must is a convenience for asserting success.
func (r Response) Must(t testing.TB) Response {
	t.Helper()
	if !r.OK {
		t.Fatalf("status %d, error %s: %s", r.Status, r.Error.Code, r.Error.Message)
	}
	return r
}
