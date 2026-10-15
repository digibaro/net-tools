package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

func TestAuditTarget(t *testing.T) {
	for raw, want := range map[string]string{
		"target=example.com&count=5":                       "example.com",
		"host=mx.example.com&domain=example.com":           "mx.example.com",
		"url=https://user:pw@example.com/a?token=s3cret#x": "https://example.com/a",
		"set=AS-EXAMPLE":                                   "AS-EXAMPLE",
		"seconds=5":                                        "",
		"target=" + strings.Repeat("a", 300):               strings.Repeat("a", 200),
	} {
		q, _ := url.ParseQuery(raw)
		if got := AuditTarget(q); got != want {
			t.Errorf("%s: got %q, want %q", raw, got, want)
		}
	}
}

func TestAuditLogsOneLinePerRequest(t *testing.T) {
	var buf bytes.Buffer
	h := Audit("net.ping", func(s string) string { return "hashed:" + s })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/ping/stream?target=example.com", strings.NewReader(`{"community":"s3cret"}`))
	ctx := context.WithValue(r.Context(), loggerKey, slog.New(slog.NewJSONHandler(&buf, nil)))
	ctx = context.WithValue(ctx, clientIPKey, netip.MustParseAddr("192.0.2.7"))
	h.ServeHTTP(httptest.NewRecorder(), r.WithContext(ctx))

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("not one JSON line: %q", buf.String())
	}
	if line["msg"] != "audit" || line["tool"] != "net.ping" || line["target"] != "example.com" || line["status"] != float64(http.StatusTeapot) || line["clientIp"] != "hashed:192.0.2.7" || line["method"] != "POST" {
		t.Errorf("line = %v", line)
	}
	if strings.Contains(buf.String(), "s3cret") {
		t.Error("request body leaked into the audit log")
	}
}
