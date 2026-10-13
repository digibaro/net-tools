// Package sse writes Server-Sent Events (docs/API_SPEC.md §4) with periodic heartbeats.
package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Heartbeat is the interval of ": ping" comments that keep proxies from timing out.
var Heartbeat = 15 * time.Second

// Writer serializes events onto one response. Safe for concurrent use.
type Writer struct {
	mu     sync.Mutex
	w      http.ResponseWriter
	rc     *http.ResponseController
	done   chan struct{}
	closed bool
}

// New starts an event stream on w.
func New(w http.ResponseWriter) *Writer {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	s := &Writer{w: w, rc: http.NewResponseController(w), done: make(chan struct{})}
	_ = s.rc.Flush()
	go s.heartbeat(Heartbeat) // read here, not in the goroutine
	return s
}

func (s *Writer) heartbeat(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-t.C:
			s.mu.Lock()
			if !s.closed {
				_, _ = fmt.Fprint(s.w, ": ping\n\n")
				_ = s.rc.Flush()
			}
			s.mu.Unlock()
		}
	}
}

// Event writes one named event with a JSON payload and flushes it.
func (s *Writer) Event(name string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sse: marshal %s: %w", name, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("sse: invalid event name %q", name)
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, b); err != nil {
		return fmt.Errorf("sse: write: %w", err)
	}
	if err := s.rc.Flush(); err != nil {
		return fmt.Errorf("sse: flush: %w", err)
	}
	return nil
}

// Close stops the heartbeat; further events are dropped.
func (s *Writer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.done)
	}
}
