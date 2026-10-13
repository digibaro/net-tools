package limits

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

var (
	ipA = netip.MustParseAddr("198.51.100.1")
	ipB = netip.MustParseAddr("198.51.100.2")
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewRateLimiter(60, 6)
	l.now = func() time.Time { return now }
	for i := range burst(60) {
		if ok, _ := l.Allow(Light, ipA); !ok {
			t.Fatalf("request %d denied within burst", i)
		}
	}
	ok, wait := l.Allow(Light, ipA)
	if ok || wait <= 0 || wait > time.Second {
		t.Fatalf("over burst: ok=%v wait=%s", ok, wait)
	}
	if ok, _ := l.Allow(Light, ipB); !ok {
		t.Error("clients must have independent buckets")
	}
	if ok, _ := l.Allow(Heavy, ipA); !ok {
		t.Error("classes must have independent buckets")
	}
	now = now.Add(time.Second)
	if ok, _ := l.Allow(Light, ipA); !ok {
		t.Error("token not refilled after 1 s at 60 rpm")
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	l := NewRateLimiter(6, 6)
	denied := 0
	h := l.Middleware(Light, func(*http.Request) netip.Addr { return ipA }, func(w http.ResponseWriter, _ *http.Request) {
		denied++
		w.WriteHeader(http.StatusTooManyRequests)
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	var last *httptest.ResponseRecorder
	for range 4 {
		last = httptest.NewRecorder()
		h.ServeHTTP(last, httptest.NewRequest(http.MethodGet, "/", nil))
	}
	if denied != 1 || last.Code != http.StatusTooManyRequests || last.Header().Get("Retry-After") == "" {
		t.Errorf("denied=%d code=%d retry=%q", denied, last.Code, last.Header().Get("Retry-After"))
	}
}

func TestPool(t *testing.T) {
	p := NewPool(3, 2)
	r1, err1 := p.Acquire(ipA)
	_, err2 := p.Acquire(ipA)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	if _, err := p.Acquire(ipA); !errors.Is(err, ErrBusy) {
		t.Error("per-IP limit not enforced")
	}
	r3, err := p.Acquire(ipB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Acquire(netip.MustParseAddr("198.51.100.3")); !errors.Is(err, ErrBusy) {
		t.Error("global limit not enforced")
	}
	r1()
	r1() // idempotent
	r3()
	if p.Active() != 1 {
		t.Errorf("active = %d", p.Active())
	}
	if _, err := p.Acquire(ipA); err != nil {
		t.Errorf("slot not released: %v", err)
	}
}

func TestTimeout(t *testing.T) {
	var deadline time.Time
	h := Timeout(50 * time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, _ = r.Context().Deadline()
		<-r.Context().Done()
	}))
	start := time.Now()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
	if deadline.IsZero() || time.Since(start) > time.Second {
		t.Errorf("deadline=%v elapsed=%s", deadline, time.Since(start))
	}
}
