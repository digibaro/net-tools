// Package latmatrix implements CON-15: ping several targets in parallel and compare them.
package latmatrix

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/ping"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// MaxTargets bounds the fan-out.
const MaxTargets = 10

// Tool is net.latmatrix.
type Tool struct {
	deps   tools.Deps
	Engine ping.Engine
}

func (*Tool) ID() string                                { return "net.latmatrix" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Engine == nil {
		t.Engine = ping.ProBing
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(d.Config.Limits.StreamTimeoutMax)).Get("/latency-matrix/stream", t.handle)
}

// Row is the "row" event: one target's statistics.
type Row struct {
	Target  string           `json:"target"`
	Address string           `json:"address"`
	Stats   rttstats.Summary `json:"stats"`
	RTTs    []*float64       `json:"rtts"`
	Error   string           `json:"error"`
}

// Summary is the done payload: rows ordered by average RTT (failures last).
type Summary struct {
	Rows []Row `json:"rows"`
}

func splitTargets(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
		if p = strings.TrimSpace(p); p != "" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(context.Context) (*stream.Job, error) {
		lim := t.deps.Config.Limits
		q := params.New(r.URL.Query())
		raw := q.String("targets", "", true, 2000)
		count := q.Int("count", 5, 1, lim.PingMaxCount)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		targets := splitTargets(raw)
		if len(targets) == 0 || len(targets) > MaxTargets {
			return nil, httpapi.InvalidInput("targets", "enter 1 to 10 host names or IP addresses")
		}
		interval := max(lim.PingMinInterval, 500*time.Millisecond)
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "targets": targets, "count": count}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			rows := make([]Row, len(targets))
			var mu sync.Mutex
			var wg sync.WaitGroup
			for i, tg := range targets {
				wg.Go(func() {
					rows[i] = t.measure(ctx, tg, count, interval)
					mu.Lock()
					emit("row", rows[i])
					mu.Unlock()
				})
			}
			wg.Wait()
			sorted := slices.Clone(rows)
			slices.SortStableFunc(sorted, func(a, b Row) int {
				okA, okB := a.Stats.Received > 0, b.Stats.Received > 0
				switch {
				case okA != okB:
					if okA {
						return -1
					}
					return 1
				case a.Stats.AvgMs < b.Stats.AvgMs:
					return -1
				case a.Stats.AvgMs > b.Stats.AvgMs:
					return 1
				}
				return 0
			})
			return Summary{Rows: sorted}, nil
		}}, nil
	})
}

// measure resolves one target through the policy (a refusal becomes the row's error) and pings it.
func (t *Tool) measure(ctx context.Context, raw string, count int, interval time.Duration) Row {
	row := Row{Target: raw, RTTs: make([]*float64, count)}
	tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Field: "targets"})
	if err != nil {
		row.Error = httpapi.ToAPIError(err).Message
		return row
	}
	row.Address = tgt.IP.String()
	var rtts []time.Duration
	var mu sync.Mutex
	sent, err := t.Engine(ctx, tgt.IP, ping.Options{Count: count, Interval: interval, Size: 56}, func(rep ping.Reply, d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		rtts = append(rtts, d)
		if rep.Seq >= 0 && rep.Seq < count {
			v := rep.RTTMs
			row.RTTs[rep.Seq] = &v
		}
	})
	if err != nil && ctx.Err() == nil && sent == 0 {
		row.Error = "ping failed"
	}
	row.Stats = rttstats.Summarize(max(sent, count), rtts)
	return row
}
