package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tests := []struct {
		name     string
		incoming string
		reuse    bool
	}{
		{"none", "", false},
		{"valid upstream id", "abc123-DEF_456.7", true},
		{"too short", "abc", false},
		{"injection", "abcdefgh\nlevel=error", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			h := RequestID(logger)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seen = GetRequestID(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header[RequestIDHeader] = []string{tt.incoming}
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			got := rec.Header().Get(RequestIDHeader)
			if got == "" || got != seen {
				t.Fatalf("header %q, context %q", got, seen)
			}
			if (got == tt.incoming) != tt.reuse {
				t.Errorf("got %q, reuse=%v", got, tt.reuse)
			}
		})
	}
}

func TestStatusRecorderFlushes(t *testing.T) {
	rec := httptest.NewRecorder()
	var w http.ResponseWriter = &statusRecorder{ResponseWriter: rec}
	f, ok := w.(http.Flusher)
	if !ok {
		t.Fatal("statusRecorder must implement http.Flusher for SSE")
	}
	f.Flush()
	if !rec.Flushed {
		t.Error("Flush not forwarded")
	}
}
