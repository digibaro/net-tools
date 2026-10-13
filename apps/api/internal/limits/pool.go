package limits

import (
	"errors"
	"net/netip"
	"sync"
)

// ErrBusy is returned when the global or per-client concurrency limit is reached.
var ErrBusy = errors.New("server is busy")

// Pool bounds simultaneous heavy jobs globally and per client. Acquire never blocks: callers
// get ErrBusy immediately so a client cannot queue unbounded work.
type Pool struct {
	mu     sync.Mutex
	global int
	perIP  int
	active int
	byIP   map[netip.Addr]int
}

// NewPool creates a pool (CONCURRENCY_GLOBAL, CONCURRENCY_PER_IP).
func NewPool(global, perIP int) *Pool {
	return &Pool{global: global, perIP: perIP, byIP: map[netip.Addr]int{}}
}

// Acquire reserves a slot for ip; call the returned release exactly once (it is idempotent).
func (p *Pool) Acquire(ip netip.Addr) (func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active >= p.global || p.byIP[ip] >= p.perIP {
		return nil, ErrBusy
	}
	p.active++
	p.byIP[ip]++
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.active--
			if p.byIP[ip]--; p.byIP[ip] <= 0 {
				delete(p.byIP, ip)
			}
		})
	}, nil
}

// Active returns the number of running jobs.
func (p *Pool) Active() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}
