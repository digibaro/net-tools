// Package rttstats summarizes round-trip samples for ping-like tools.
package rttstats

import (
	"math"
	"time"
)

// Summary is the "done" payload of ping-like streams.
type Summary struct {
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
	LossPct  float64 `json:"lossPct"`
	MinMs    float64 `json:"minMs"`
	AvgMs    float64 `json:"avgMs"`
	MaxMs    float64 `json:"maxMs"`
	StdDevMs float64 `json:"stddevMs"`
	// JitterMs is the mean absolute difference between consecutive RTTs.
	JitterMs float64 `json:"jitterMs"`
}

// Ms converts to milliseconds rounded to 3 decimals.
func Ms(d time.Duration) float64 { return round(float64(d.Microseconds()) / 1000) }

func round(v float64) float64 { return math.Round(v*1000) / 1000 }

// Summarize computes statistics; rtts are the received samples in order.
func Summarize(sent int, rtts []time.Duration) Summary {
	s := Summary{Sent: sent, Received: len(rtts)}
	if sent > 0 {
		s.LossPct = round(float64(sent-len(rtts)) / float64(sent) * 100)
	}
	if len(rtts) == 0 {
		return s
	}
	ms := make([]float64, len(rtts))
	sum := 0.0
	s.MinMs, s.MaxMs = math.Inf(1), math.Inf(-1)
	for i, d := range rtts {
		ms[i] = float64(d.Microseconds()) / 1000
		sum += ms[i]
		s.MinMs, s.MaxMs = math.Min(s.MinMs, ms[i]), math.Max(s.MaxMs, ms[i])
	}
	avg := sum / float64(len(ms))
	variance, jitter := 0.0, 0.0
	for i, v := range ms {
		variance += (v - avg) * (v - avg)
		if i > 0 {
			jitter += math.Abs(v - ms[i-1])
		}
	}
	s.AvgMs, s.StdDevMs = round(avg), round(math.Sqrt(variance/float64(len(ms))))
	if len(ms) > 1 {
		s.JitterMs = round(jitter / float64(len(ms)-1))
	}
	s.MinMs, s.MaxMs = round(s.MinMs), round(s.MaxMs)
	return s
}
