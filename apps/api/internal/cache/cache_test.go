package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTTLAndEviction(t *testing.T) {
	now := time.Unix(0, 0)
	c := New[int](2)
	c.Now = func() time.Time { return now }
	c.Set("a", 1, time.Minute)
	c.Set("b", 2, 2*time.Minute)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatal("a missing")
	}
	c.Set("c", 3, time.Hour) // evicts a (soonest expiry)
	if _, ok := c.Get("a"); ok {
		t.Error("a not evicted")
	}
	now = now.Add(3 * time.Minute)
	if _, ok := c.Get("b"); ok {
		t.Error("b not expired")
	}
	if v, _ := c.Get("c"); v != 3 {
		t.Error("c lost")
	}
}

func TestDoCoalescesAndSkipsErrors(t *testing.T) {
	c := New[int](10)
	var calls atomic.Int32
	release := make(chan struct{})
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			v, err := c.Do(context.Background(), "k", time.Minute, func(context.Context) (int, error) {
				calls.Add(1)
				<-release
				return 42, nil
			})
			if err != nil || v != 42 {
				t.Errorf("v=%d err=%v", v, err)
			}
		})
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("calls = %d", calls.Load())
	}
	boom := errors.New("boom")
	if _, err := c.Do(context.Background(), "e", time.Minute, func(context.Context) (int, error) { return 0, boom }); !errors.Is(err, boom) {
		t.Error(err)
	}
	if _, ok := c.Get("e"); ok {
		t.Error("error was cached")
	}
}
