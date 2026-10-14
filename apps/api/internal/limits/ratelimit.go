// Package limits enforces per-client rate limits, a concurrency pool for heavy jobs and
// per-route timeouts (docs/SECURITY.md §4).
package limits

import (
	"context"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Class selects a rate-limit bucket.
type Class int

const (
	// Light is for quick request/response tools (RATE_LIMIT_RPM).
	Light Class = iota
	// Heavy is for streaming, scanning and provider tools (RATE_LIMIT_HEAVY_RPM).
	Heavy
)

type visitor struct {
	limiter *rate.Limiter
	seen    time.Time
}

// RateLimiter is a token bucket per (class, client IP).
type RateLimiter struct {
	mu       sync.Mutex
	rpm      map[Class]int
	visitors map[Class]map[netip.Addr]*visitor
	now      func() time.Time
}

// NewRateLimiter creates limiters allowing rpm requests per minute with a burst of rpm/6 (min 3).
func NewRateLimiter(lightRPM, heavyRPM int) *RateLimiter {
	return &RateLimiter{
		rpm:      map[Class]int{Light: lightRPM, Heavy: heavyRPM},
		visitors: map[Class]map[netip.Addr]*visitor{Light: {}, Heavy: {}},
		now:      time.Now,
	}
}

func burst(rpm int) int {
	return max(3, rpm/6)
}

// Allow consumes a token; when denied it returns the wait until the next token.
func (l *RateLimiter) Allow(class Class, ip netip.Addr) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	vs := l.visitors[class]
	v, ok := vs[ip]
	if !ok {
		rpm := l.rpm[class]
		v = &visitor{limiter: rate.NewLimiter(rate.Limit(float64(rpm)/60), burst(rpm))}
		vs[ip] = v
		if len(vs) > 10_000 {
			l.evict(class, now)
		}
	}
	v.seen = now
	r := v.limiter.ReserveN(now, 1)
	if delay := r.DelayFrom(now); delay > 0 {
		r.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// evict drops visitors idle for more than 10 minutes (called with mu held).
func (l *RateLimiter) evict(class Class, now time.Time) {
	for ip, v := range l.visitors[class] {
		if now.Sub(v.seen) > 10*time.Minute {
			delete(l.visitors[class], ip)
		}
	}
}

// Middleware rejects over-limit requests with 429 and Retry-After. clientIP extracts the
// client address; onDenied writes the error envelope.
func (l *RateLimiter) Middleware(class Class, clientIP func(*http.Request) netip.Addr, onDenied func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, wait := l.Allow(class, clientIP(r))
			if !ok {
				secs := int(wait.Seconds()) + 1
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				onDenied(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout bounds a route's context.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
