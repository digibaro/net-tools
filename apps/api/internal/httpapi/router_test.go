package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/config"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
)

type fakeTool struct {
	id      string
	enabled bool
	reason  tools.Reason
}

func (f fakeTool) ID() string { return f.id }
func (f fakeTool) Enabled(tools.Deps) (bool, tools.Reason) {
	return f.enabled, f.reason
}

func (f fakeTool) Mount(r chi.Router, _ tools.Deps) {
	r.Get("/fake/"+f.id, func(w http.ResponseWriter, r *http.Request) {
		JSON(w, r, map[string]string{"id": f.id}, &Meta{Tool: f.id})
	})
}

func testConfig(t *testing.T, env map[string]string) config.Config {
	t.Helper()
	cfg, err := config.LoadFrom(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}, func(string) ([]byte, error) { return nil, errors.New("no files") })
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return cfg
}

func newTestRouter(t *testing.T, o Options) http.Handler {
	t.Helper()
	if o.Logger == nil {
		o.Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	if o.Config.Profile == "" {
		o.Config = testConfig(t, nil)
	}
	return NewRouter(o)
}

func do(t *testing.T, h http.Handler, req *http.Request) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return rec, body
}

func get(t *testing.T, h http.Handler, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	return do(t, h, httptest.NewRequest(http.MethodGet, path, nil))
}

func TestHealthz(t *testing.T) {
	h := newTestRouter(t, Options{})
	rec, body := get(t, h, "/api/healthz")
	if rec.Code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("missing X-Request-Id")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestReadyz(t *testing.T) {
	tests := []struct {
		name   string
		checks []ReadyCheck
		status int
	}{
		{"no checks", nil, http.StatusOK},
		{"passing", []ReadyCheck{{Name: "a", Check: func(context.Context) error { return nil }}}, http.StatusOK},
		{"failing", []ReadyCheck{{Name: "geoip", Check: func(context.Context) error { return errors.New("/secret/path missing") }}}, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(t, Options{Ready: tt.checks})
			rec, body := get(t, h, "/api/readyz")
			if rec.Code != tt.status {
				t.Fatalf("status = %d, body %v", rec.Code, body)
			}
			if strings.Contains(rec.Body.String(), "/secret/path") {
				t.Error("readiness output leaks error details")
			}
		})
	}
}

func TestVersion(t *testing.T) {
	h := newTestRouter(t, Options{Build: BuildInfo{Version: "1.2.3", Commit: "abc", BuildDate: "2026-10-13", GoVersion: "go1.27"}})
	rec, body := get(t, h, "/api/v1/version")
	if rec.Code != http.StatusOK || body["ok"] != true {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	data := body["data"].(map[string]any)
	for k, want := range map[string]string{"version": "1.2.3", "commit": "abc", "buildDate": "2026-10-13", "goVersion": "go1.27"} {
		if data[k] != want {
			t.Errorf("%s = %v, want %s", k, data[k], want)
		}
	}
}

func TestCapabilitiesEmpty(t *testing.T) {
	h := newTestRouter(t, Options{Tools: nil})
	rec, body := get(t, h, "/api/v1/capabilities")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	data := body["data"].(map[string]any)
	if got := data["tools"].(map[string]any); len(got) != 0 {
		t.Errorf("tools = %v, want empty", got)
	}
	mode := data["mode"].(map[string]any)
	if mode["profile"] != "public" || mode["allowPrivateTargets"] != false || mode["lan"] != false {
		t.Errorf("mode = %v", mode)
	}
	limits := data["limits"].(map[string]any)
	if limits["pingMaxCount"] != float64(20) || limits["tracerouteMaxHops"] != float64(30) || limits["portscanMaxPorts"] != float64(1024) {
		t.Errorf("limits = %v", limits)
	}
}

func TestCapabilitiesAndMounting(t *testing.T) {
	h := newTestRouter(t, Options{
		Config: testConfig(t, map[string]string{"NETTOOLS_PROFILE": "homelab"}),
		Tools: []tools.Tool{
			fakeTool{id: "on", enabled: true},
			fakeTool{id: "off", enabled: false, reason: tools.ReasonDisabledByAdmin},
		},
	})
	_, body := get(t, h, "/api/v1/capabilities")
	got := body["data"].(map[string]any)["tools"].(map[string]any)
	if on := got["on"].(map[string]any); on["enabled"] != true || on["reason"] != nil {
		t.Errorf("on = %v", on)
	}
	if off := got["off"].(map[string]any); off["enabled"] != false || off["reason"] != "disabled_by_admin" {
		t.Errorf("off = %v", off)
	}
	if mode := body["data"].(map[string]any)["mode"].(map[string]any); mode["allowPrivateTargets"] != true {
		t.Errorf("mode = %v", mode)
	}

	rec, body := get(t, h, "/api/v1/fake/on")
	if rec.Code != http.StatusOK || body["meta"].(map[string]any)["tool"] != "on" {
		t.Errorf("enabled tool: %d %v", rec.Code, body)
	}
	rec, _ = get(t, h, "/api/v1/fake/off")
	if rec.Code != http.StatusNotFound {
		t.Errorf("disabled tool must not be mounted, got %d", rec.Code)
	}
}

func TestErrorEnvelopes(t *testing.T) {
	h := newTestRouter(t, Options{})
	rec, body := get(t, h, "/api/v1/nope")
	if rec.Code != http.StatusNotFound || body["ok"] != false {
		t.Fatalf("got %d %v", rec.Code, body)
	}
	if code := body["error"].(map[string]any)["code"]; code != "NOT_FOUND" {
		t.Errorf("code = %v", code)
	}

	rec, body = do(t, h, httptest.NewRequest(http.MethodPost, "/api/v1/version", nil))
	if rec.Code != http.StatusMethodNotAllowed || body["error"].(map[string]any)["code"] != "METHOD_NOT_ALLOWED" {
		t.Errorf("got %d %v", rec.Code, body)
	}
}

func TestPanicIsInternalError(t *testing.T) {
	h := newTestRouter(t, Options{Tools: []tools.Tool{panicTool{}}})
	rec, body := get(t, h, "/api/v1/panic")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	e := body["error"].(map[string]any)
	if e["code"] != "INTERNAL" || strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

type panicTool struct{}

func (panicTool) ID() string                              { return "panic" }
func (panicTool) Enabled(tools.Deps) (bool, tools.Reason) { return true, "" }
func (panicTool) Mount(r chi.Router, _ tools.Deps) {
	r.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("boom") })
}

func TestEcho(t *testing.T) {
	cfg := testConfig(t, map[string]string{"TRUSTED_PROXIES": "172.16.0.0/12"})
	h := newTestRouter(t, Options{Config: cfg})

	tests := []struct {
		name       string
		remote     string
		xff        string
		proto      string
		wantIP     string
		wantFamily float64
		wantScheme string
	}{
		{"direct v4", "203.0.113.9:5555", "", "", "203.0.113.9", 4, "http"},
		{"untrusted peer ignores XFF", "203.0.113.9:5555", "198.51.100.1", "https", "203.0.113.9", 4, "http"},
		{"trusted proxy", "172.18.0.2:4000", "198.51.100.1", "https", "198.51.100.1", 4, "https"},
		{"trusted chain skips trusted hops", "172.18.0.2:4000", "2001:db8::1, 172.19.0.5", "http", "2001:db8::1", 6, "http"},
		{"spoofed left entry ignored", "172.18.0.2:4000", "1.1.1.1, 198.51.100.7", "", "198.51.100.7", 4, "http"},
		{"garbage XFF falls back to peer", "172.18.0.2:4000", "not-an-ip", "", "172.18.0.2", 4, "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/echo", nil)
			req.RemoteAddr = tt.remote
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			req.Header.Set("Authorization", "Bearer secret")
			req.Header.Set("User-Agent", "test-agent")
			rec, body := do(t, h, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d", rec.Code)
			}
			data := body["data"].(map[string]any)
			if data["ip"] != tt.wantIP || data["family"] != tt.wantFamily || data["scheme"] != tt.wantScheme {
				t.Errorf("ip=%v family=%v scheme=%v", data["ip"], data["family"], data["scheme"])
			}
			if data["userAgent"] != "test-agent" {
				t.Errorf("userAgent = %v", data["userAgent"])
			}
			if strings.Contains(rec.Body.String(), "Bearer secret") {
				t.Error("Authorization header leaked")
			}
		})
	}
}

func TestToAPIError(t *testing.T) {
	wrapped := errors.Join(errors.New("ctx"), InvalidInput("name", "bad name"))
	if got := ToAPIError(wrapped); got.Code != CodeInvalidInput || got.Details["field"] != "name" {
		t.Errorf("got %+v", got)
	}
	plain := errors.New("db password=hunter2")
	got := ToAPIError(plain)
	if got.Code != CodeInternal || strings.Contains(got.Message, "hunter2") || !errors.Is(got, plain) {
		t.Errorf("got %+v", got)
	}
	for code, want := range map[Code]int{CodeTargetPrivateDenied: 403, CodeRateLimited: 429, CodeBusy: 503, Code("X"): 500} {
		if code.Status() != want {
			t.Errorf("%s.Status() = %d, want %d", code, code.Status(), want)
		}
	}
}

func TestToAPIErrorCoderAndTimeout(t *testing.T) {
	_, err := (&target.Resolver{}).Resolve(context.Background(), "http://x", target.Opts{Field: "host"})
	got := ToAPIError(fmt.Errorf("wrapped: %w", err))
	if got.Code != CodeInvalidInput || got.Details["field"] != "host" || got.Code.Status() != 400 {
		t.Errorf("got %+v", got)
	}
	if got := ToAPIError(fmt.Errorf("x: %w", context.DeadlineExceeded)); got.Code != CodeTimeout {
		t.Errorf("timeout mapped to %s", got.Code)
	}
}

func TestIPFormatter(t *testing.T) {
	if got := ipFormatter(false)("192.0.2.1"); got != "192.0.2.1" {
		t.Errorf("plain = %q", got)
	}
	h := ipFormatter(true)("192.0.2.1")
	if h == "192.0.2.1" || len(h) != 16 {
		t.Errorf("hashed = %q", h)
	}
}
