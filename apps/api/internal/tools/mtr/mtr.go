// Package mtr implements CON-03 (traceroute) and CON-04 (MTR) by streaming `mtr --raw`.
package mtr

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/execx"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// Mode selects traceroute (3 probes per hop) or mtr (many cycles with running statistics).
type Mode int

const (
	Traceroute Mode = iota
	MTR
)

// Tool is traceroute or MTR.
type Tool struct {
	Mode Mode
	deps tools.Deps
	// UpdateEvery throttles per-hop updates in MTR mode.
	UpdateEvery time.Duration
}

func (t *Tool) ID() string {
	if t.Mode == MTR {
		return "net.mtr"
	}
	return "net.traceroute"
}

func (t *Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	if d.Exec == nil || !d.Exec.Available("mtr") {
		return false, tools.ReasonBinaryMissing
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	path := "/traceroute/stream"
	if t.Mode == MTR {
		path = "/mtr/stream"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(d.Config.Limits.StreamTimeoutMax)).Get(path, t.handle)
}

// Args builds the mtr argv. The only target-derived value is the pinned IP, formatted by netip.
func Args(ip string, proto string, port, cycles, maxHops int, interval time.Duration) ([]string, error) {
	ipArg, err := execx.UserValue(ip)
	if err != nil {
		return nil, err //nolint:wrapcheck // validation
	}
	args := []string{"--raw", "--report-cycles", strconv.Itoa(cycles), "--interval", strconv.FormatFloat(interval.Seconds(), 'f', 1, 64), "--max-ttl", strconv.Itoa(maxHops), "--gracetime", "2"}
	switch proto {
	case "udp":
		args = append(args, "--udp")
	case "tcp":
		args = append(args, "--tcp", "--port", strconv.Itoa(port))
	}
	return append(args, "--", ipArg), nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	lim := t.deps.Config.Limits
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("target", "", true, 300)
		proto := q.Enum("proto", "icmp", "icmp", "udp", "tcp")
		port := q.Int("port", 443, 1, 65535)
		maxHops := q.Int("maxHops", lim.TracerouteMaxHops, 1, lim.TracerouteMaxHops)
		cycles := 3
		if t.Mode == MTR {
			cycles = q.Int("cycles", min(10, lim.MTRMaxCycles), 1, lim.MTRMaxCycles)
		}
		family := q.Family()
		if err := q.Err(); err != nil {
			return nil, err
		}
		tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Family: family})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		interval := time.Second
		if t.Mode == Traceroute {
			interval = 200 * time.Millisecond
		}
		args, err := Args(tgt.IP.String(), proto, port, cycles, maxHops, interval)
		if err != nil {
			return nil, err
		}
		return &stream.Job{
			Meta: map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": raw, "resolved": tgt.Resolved, "used": tgt.IP}, "params": map[string]any{"proto": proto, "maxHops": maxHops, "cycles": cycles}},
			Run: func(ctx context.Context, emit stream.Emit) (any, error) {
				return t.run(ctx, args, emit)
			},
		}, nil
	})
}

// Summary is the done payload: the final hop table.
type Summary struct {
	Hops []Hop `json:"hops"`
}

func (t *Tool) run(ctx context.Context, args []string, emit stream.Emit) (Summary, error) {
	p := NewParser()
	every := t.UpdateEvery
	if every == 0 {
		every = 500 * time.Millisecond
	}
	th := &throttle{last: map[int]time.Time{}, every: every}
	_, err := t.deps.Exec.Run(ctx, "mtr", args, func(line string) {
		if hop := p.Line(line); hop > 0 && (t.Mode == Traceroute || th.allow(hop, time.Now())) {
			if h := p.Get(hop); h.IP != "" || t.Mode == MTR {
				emit("hop", h)
			}
		}
	})
	if err != nil && ctx.Err() == nil {
		return Summary{}, &mtrError{err}
	}
	return Summary{Hops: p.Hops()}, nil
}

type mtrError struct{ err error }

func (e *mtrError) Error() string              { return "mtr failed: " + e.err.Error() }
func (e *mtrError) Unwrap() error              { return e.err }
func (e *mtrError) APICode() string            { return "INTERNAL" }
func (e *mtrError) APIMessage() string         { return "the trace could not be run on this server" }
func (e *mtrError) APIDetails() map[string]any { return nil }
