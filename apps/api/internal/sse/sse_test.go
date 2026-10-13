package sse

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	rec := httptest.NewRecorder()
	s := New(rec)
	if err := s.Event("reply", map[string]any{"seq": 1, "rttMs": 1.5}); err != nil {
		t.Fatal(err)
	}
	if err := s.Event("bad\nname", 1); err == nil {
		t.Error("newline in event name accepted")
	}
	s.Close()
	_ = s.Event("after", 1)
	body := rec.Body.String()
	if body != "event: reply\ndata: {\"rttMs\":1.5,\"seq\":1}\n\n" {
		t.Errorf("body = %q", body)
	}
	if rec.Header().Get("Content-Type") != "text/event-stream; charset=utf-8" || rec.Header().Get("X-Accel-Buffering") != "no" {
		t.Errorf("headers = %v", rec.Header())
	}
	if !rec.Flushed {
		t.Error("not flushed")
	}
}

func TestHeartbeat(t *testing.T) {
	old := Heartbeat
	Heartbeat = 10 * time.Millisecond
	defer func() { Heartbeat = old }()
	rec := httptest.NewRecorder()
	s := New(rec)
	time.Sleep(50 * time.Millisecond)
	s.Close()
	s.mu.Lock()
	body := rec.Body.String()
	s.mu.Unlock()
	if !strings.Contains(body, ": ping\n\n") {
		t.Errorf("no heartbeat in %q", body)
	}
}
