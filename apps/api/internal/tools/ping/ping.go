// Package ping implements CON-01: ICMP echo with live replies over SSE.
package ping

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	probing "github.com/prometheus-community/pro-bing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// Reply is an SSE "reply" event.
type Reply struct {
	Seq   int     `json:"seq"`
	RTTMs float64 `json:"rttMs"`
	TTL   int     `json:"ttl"`
	From  string  `json:"from"`
	Bytes int     `json:"bytes"`
}

// Options for one ping run.
type Options struct {
	Count    int
	Interval time.Duration
	Size     int
}

// Engine sends echo requests to ip and calls onReply per answer; it returns the number sent.
type Engine func(ctx context.Context, ip netip.Addr, o Options, onReply func(Reply, time.Duration)) (sent int, err error)

// ProBing is the production engine: unprivileged ICMP (ping_group_range) via pro-bing.
func ProBing(ctx context.Context, ip netip.Addr, o Options, onReply func(Reply, time.Duration)) (int, error) {
	p := probing.New(ip.String())
	p.SetIPAddr(&net.IPAddr{IP: net.IP(ip.AsSlice())})
	p.SetPrivileged(false)
	p.Count, p.Interval, p.Size = o.Count, o.Interval, o.Size
	p.Timeout = time.Duration(o.Count)*o.Interval + 2*time.Second
	p.OnRecv = func(pkt *probing.Packet) {
		onReply(Reply{Seq: pkt.Seq, RTTMs: rttstats.Ms(pkt.Rtt), TTL: pkt.TTL, From: pkt.IPAddr.String(), Bytes: pkt.Nbytes}, pkt.Rtt)
	}
	err := p.RunWithContext(ctx)
	return p.Statistics().PacketsSent, err //nolint:wrapcheck // surfaced as INTERNAL/TIMEOUT
}

// Tool is ping.
type Tool struct {
	deps   tools.Deps
	Engine Engine
}

func (*Tool) ID() string                                { return "net.ping" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Engine == nil {
		t.Engine = ProBing
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(d.Config.Limits.StreamTimeoutMax)).Get("/ping/stream", t.handle)
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	lim := t.deps.Config.Limits
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("target", "", true, 300)
		count := q.Int("count", min(5, lim.PingMaxCount), 1, lim.PingMaxCount)
		intervalMs := q.Int("interval", 1000, int(lim.PingMinInterval.Milliseconds()), 5000)
		size := q.Int("size", 56, 16, 1472)
		family := q.Family()
		if err := q.Err(); err != nil {
			return nil, err
		}
		tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Family: family})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		o := Options{Count: count, Interval: time.Duration(intervalMs) * time.Millisecond, Size: size}
		return &stream.Job{
			Meta: map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": raw, "resolved": tgt.Resolved, "used": tgt.IP}, "params": map[string]int{"count": count, "interval": intervalMs, "size": size}},
			Run: func(ctx context.Context, emit stream.Emit) (any, error) {
				var rtts []time.Duration
				sent, err := t.Engine(ctx, tgt.IP, o, func(rep Reply, rtt time.Duration) {
					rtts = append(rtts, rtt)
					emit("reply", rep)
				})
				if err != nil && ctx.Err() == nil {
					return nil, err
				}
				return rttstats.Summarize(sent, rtts), nil
			},
		}, nil
	})
}
