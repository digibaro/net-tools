// Package cache is a small in-memory TTL cache with request coalescing, used for provider data
// (docs/API_SPEC.md §7).
package cache

import (
	"context"
	"sync"
	"time"
)

type entry[V any] struct {
	value   V
	expires time.Time
}

type call[V any] struct {
	done  chan struct{}
	value V
	err   error
}

// TTL caches values per key until they expire; at most Max entries are kept.
type TTL[V any] struct {
	Max int
	Now func() time.Time

	mu       sync.Mutex
	items    map[string]entry[V]
	inflight map[string]*call[V]
}

// New returns a cache bounded to max entries.
func New[V any](maxEntries int) *TTL[V] {
	return &TTL[V]{Max: maxEntries, Now: time.Now, items: map[string]entry[V]{}, inflight: map[string]*call[V]{}}
}

// Get returns a live entry.
func (c *TTL[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || !c.Now().Before(e.expires) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Set stores a value; when full, expired entries are dropped first, then the soonest-expiring one.
func (c *TTL[V]) Set(key string, v V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(key, v, ttl)
}

func (c *TTL[V]) setLocked(key string, v V, ttl time.Duration) {
	now := c.Now()
	if _, exists := c.items[key]; !exists && len(c.items) >= c.Max {
		var victim string
		var soonest time.Time
		for k, e := range c.items {
			if !now.Before(e.expires) {
				delete(c.items, k)
				continue
			}
			if victim == "" || e.expires.Before(soonest) {
				victim, soonest = k, e.expires
			}
		}
		if len(c.items) >= c.Max {
			delete(c.items, victim)
		}
	}
	c.items[key] = entry[V]{value: v, expires: now.Add(ttl)}
}

// Do returns the cached value or calls fn once per key, sharing the result with concurrent
// callers. Errors are not cached.
func (c *TTL[V]) Do(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) (V, error)) (V, error) {
	if v, ok := c.Get(key); ok {
		return v, nil
	}
	c.mu.Lock()
	if cl, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-cl.done:
			return cl.value, cl.err
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err() //nolint:wrapcheck // context error passes through
		}
	}
	cl := &call[V]{done: make(chan struct{})}
	c.inflight[key] = cl
	c.mu.Unlock()

	cl.value, cl.err = fn(ctx)
	c.mu.Lock()
	delete(c.inflight, key)
	if cl.err == nil {
		c.setLocked(key, cl.value, ttl)
	}
	c.mu.Unlock()
	close(cl.done)
	return cl.value, cl.err
}
