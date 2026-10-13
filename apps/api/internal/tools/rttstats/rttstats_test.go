package rttstats

import (
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	ms := func(v ...int) []time.Duration {
		out := make([]time.Duration, len(v))
		for i, x := range v {
			out[i] = time.Duration(x) * time.Millisecond
		}
		return out
	}
	s := Summarize(4, ms(10, 20, 30))
	want := Summary{Sent: 4, Received: 3, LossPct: 25, MinMs: 10, AvgMs: 20, MaxMs: 30, StdDevMs: 8.165, JitterMs: 10}
	if s != want {
		t.Errorf("got %+v, want %+v", s, want)
	}
	if s := Summarize(3, nil); s.LossPct != 100 || s.Received != 0 {
		t.Errorf("all lost: %+v", s)
	}
	if s := Summarize(0, nil); s.LossPct != 0 {
		t.Errorf("nothing sent: %+v", s)
	}
}
