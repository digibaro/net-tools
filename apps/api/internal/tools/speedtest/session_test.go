package speedtest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestSessionBypassesPerRequestLimit(t *testing.T) {
	clock := time.Now()
	s := newSessions()
	s.now = func() time.Time { return clock }
	env := tooltest.New(t, &Tool{sessions: s}, tooltest.Options{Env: map[string]string{"SPEEDTEST_ENABLED": "true", "RATE_LIMIT_RPM": "5", "RATE_LIMIT_HEAVY_RPM": "2"}})
	req := func(method, path, tok string) int {
		r := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(make([]byte, 1024)))
		if tok != "" {
			r.Header.Set(TokenHeader, tok)
		}
		return env.Do(r).Raw.Code
	}

	var sess Session
	env.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/speedtest/session", nil)).Must(t).Decode(t, &sess)
	if len(sess.Token) != 32 || sess.MaxRequests != SessionMaxRequests {
		t.Fatalf("session = %+v", sess)
	}
	// A whole test's worth of pieces passes with the token although RATE_LIMIT_RPM is 5.
	for i := range 50 {
		if c := req(http.MethodPost, "/api/v1/speedtest/upload", sess.Token); c != http.StatusOK {
			t.Fatalf("upload %d with token: HTTP %d", i, c)
		}
	}
	if c := req(http.MethodGet, "/api/v1/speedtest/download?bytes=1048576", sess.Token); c != http.StatusOK {
		t.Fatalf("download with token: HTTP %d", c)
	}
	// Without a token the normal limit applies.
	limited := false
	for range 10 {
		if req(http.MethodGet, "/api/v1/speedtest/ping", "") == http.StatusTooManyRequests {
			limited = true
		}
	}
	if !limited {
		t.Error("requests without a token were not rate limited")
	}
	// An unknown token is the same as none (the bucket is already empty).
	if c := req(http.MethodGet, "/api/v1/speedtest/ping", "deadbeef"); c != http.StatusTooManyRequests {
		t.Errorf("unknown token: HTTP %d", c)
	}
	// Expired sessions fall back to the limit.
	clock = clock.Add(SessionTTL + time.Second)
	if c := req(http.MethodGet, "/api/v1/speedtest/ping", sess.Token); c != http.StatusTooManyRequests {
		t.Errorf("expired token: HTTP %d", c)
	}
}

func TestSessionRequestCap(t *testing.T) {
	s := newSessions()
	tok, err := s.create(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for i := range SessionMaxRequests {
		if !s.use(tok) {
			t.Fatalf("request %d refused", i)
		}
	}
	if s.use(tok) {
		t.Error("request beyond the cap accepted")
	}
}

func TestSessionCreationIsRateLimited(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Env: map[string]string{"SPEEDTEST_ENABLED": "true", "RATE_LIMIT_HEAVY_RPM": "2"}})
	codes := []int{}
	for range 4 {
		codes = append(codes, env.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/speedtest/session", nil)).Raw.Code)
	}
	if codes[0] != http.StatusOK || codes[3] != http.StatusTooManyRequests {
		t.Errorf("codes = %v", codes)
	}
}
