// Package tcping implements CON-02: repeated TCP connects to host:port over SSE.
package tcping

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// Reply is an SSE "reply" (connected) or "timeout" (failed) event.
type Reply struct {
	Seq   int     `json:"seq"`
	RTTMs float64 `json:"rttMs,omitempty"`
	From  string  `json:"from"`
	Error string  `json:"error,omitempty"`
}

// Tool is TCP ping.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "net.tcping" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(d.Config.Limits.StreamTimeoutMax)).Get("/tcping/stream", t.handle)
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	lim := t.deps.Config.Limits
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("target", "", true, 300)
		port := q.Int("port", 443, 1, 65535)
		count := q.Int("count", min(5, lim.PingMaxCount), 1, lim.PingMaxCount)
		intervalMs := q.Int("interval", 1000, int(lim.PingMinInterval.Milliseconds()), 5000)
		family := q.Family()
		if err := q.Err(); err != nil {
			return nil, err
		}
		tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Family: family})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		addr := tgt.AddrPort(port)
		return &stream.Job{
			Meta: map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": raw, "resolved": tgt.Resolved, "used": addr}, "params": map[string]int{"port": port, "count": count, "interval": intervalMs}},
			Run: func(ctx context.Context, emit stream.Emit) (any, error) {
				var rtts []time.Duration
				sent := 0
				tick := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
				defer tick.Stop()
				for i := range count {
					if i > 0 {
						select {
						case <-ctx.Done():
							return rttstats.Summarize(sent, rtts), nil
						case <-tick.C:
						}
					}
					sent++
					rtt, err := connect(ctx, addr, 3*time.Second)
					if err != nil {
						emit("timeout", Reply{Seq: i, From: addr, Error: reason(err)})
						continue
					}
					rtts = append(rtts, rtt)
					emit("reply", Reply{Seq: i, RTTMs: rttstats.Ms(rtt), From: addr})
				}
				return rttstats.Summarize(sent, rtts), nil
			},
		}, nil
	})
}

func connect(ctx context.Context, addr string, timeout time.Duration) (time.Duration, error) {
	d := net.Dialer{Timeout: timeout}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err //nolint:wrapcheck // classified by reason()
	}
	rtt := time.Since(start)
	_ = c.Close()
	return rtt, nil
}

// reason classifies a dial error for display.
func reason(err error) string {
	var ne net.Error
	msg := err.Error()
	switch {
	case errors.As(err, &ne) && ne.Timeout():
		return "timeout"
	case strings.Contains(msg, "refused"):
		return "connection refused"
	case strings.Contains(msg, "unreachable"):
		return "unreachable"
	default:
		return "failed"
	}
}
