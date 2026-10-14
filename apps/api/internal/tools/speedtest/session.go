package speedtest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// Session limits: one test (ping, download and upload phases) fits easily in both.
const (
	SessionTTL         = 45 * time.Second
	SessionMaxRequests = 600
	maxOpenSessions    = 1000
	TokenHeader        = "X-Speedtest-Token" //nolint:gosec // header name, not a credential
)

type session struct {
	expires  time.Time
	requests int
}

// sessions tracks short-lived speed-test tokens (in memory; a restart only ends running tests).
type sessions struct {
	mu   sync.Mutex
	open map[string]*session
	now  func() time.Time
}

func newSessions() *sessions {
	return &sessions{open: map[string]*session{}, now: time.Now}
}

func (s *sessions) create(now time.Time) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: %w", err)
	}
	tok := hex.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.open { // prune expired sessions
		if now.After(v.expires) {
			delete(s.open, k)
		}
	}
	if len(s.open) >= maxOpenSessions {
		return "", fmt.Errorf("too many open sessions")
	}
	s.open[tok] = &session{expires: now.Add(SessionTTL)}
	return tok, nil
}

// use counts one request against tok; false when unknown, expired or used up.
func (s *sessions) use(tok string) bool {
	if tok == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.open[tok]
	if !ok || s.now().After(v.expires) || v.requests >= SessionMaxRequests {
		return false
	}
	v.requests++
	return true
}

// orLimit lets requests with a valid session token through and sends the rest to limit.
func (s *sessions) orLimit(limit tools.Middleware) tools.Middleware {
	return func(next http.Handler) http.Handler {
		limited := limit(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.use(r.Header.Get(TokenHeader)) {
				next.ServeHTTP(w, r)
				return
			}
			limited.ServeHTTP(w, r)
		})
	}
}
