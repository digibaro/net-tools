// Package pmtu implements CON-07: the largest packet that reaches a target without
// fragmentation, found by bisecting ICMP echo sizes with the Don't-Fragment bit set.
package pmtu

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"
	probing "github.com/prometheus-community/pro-bing"

	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// Header overheads: IP + ICMP.
const (
	overhead4 = 20 + 8
	overhead6 = 40 + 8
)

// Prober reports whether an echo with this ICMP payload size (DF set) gets a reply.
type Prober func(ctx context.Context, ip netip.Addr, payload int) bool

// ProBing is the production prober: two DF echoes, 1.5 s.
func ProBing(ctx context.Context, ip netip.Addr, payload int) bool {
	p := probing.New(ip.String())
	p.SetIPAddr(&net.IPAddr{IP: net.IP(ip.AsSlice())})
	p.SetPrivileged(false)
	p.SetDoNotFragment(true)
	p.Count, p.Interval, p.Size, p.Timeout = 2, 300*time.Millisecond, payload, 1500*time.Millisecond
	if err := p.RunWithContext(ctx); err != nil {
		return false // EMSGSIZE: larger than the local or cached path MTU
	}
	return p.Statistics().PacketsRecv > 0
}

// Tool is net.pmtu.
type Tool struct {
	deps   tools.Deps
	Prober Prober
}

func (*Tool) ID() string                                { return "net.pmtu" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Prober == nil {
		t.Prober = ProBing
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Get("/pmtu/stream", t.handle)
}

// Probe is the "probe" event.
type Probe struct {
	MTU int  `json:"mtu"`
	OK  bool `json:"ok"`
}

// Result is the done summary.
type Result struct {
	Target    string `json:"target"`
	Address   string `json:"address"`
	PathMTU   int    `json:"pathMtu"`
	MSS       int    `json:"mss"`
	Reachable bool   `json:"reachable"`
	Probes    int    `json:"probes"`
	Note      string `json:"note"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("target", "", true, 253)
		maxMTU := q.Int("max", 1500, 1280, 9000)
		family := q.Enum("family", "0", "0", "4", "6")
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		fam := map[string]int{"0": 0, "4": 4, "6": 6}[family]
		tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Family: fam, Field: "target"})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": raw, "address": tgt.IP.String()}, "max": maxMTU}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return t.search(ctx, raw, tgt.IP, maxMTU, emit), nil
		}}, nil
	})
}

// search bisects the MTU between the protocol minimum and maxMTU.
func (t *Tool) search(ctx context.Context, raw string, ip netip.Addr, maxMTU int, emit stream.Emit) Result {
	over, low := overhead4, 68
	if ip.Is6() {
		over, low = overhead6, 1280
	}
	res := Result{Target: raw, Address: ip.String()}
	try := func(mtu int) bool {
		ok := t.Prober(ctx, ip, mtu-over)
		res.Probes++
		emit("probe", Probe{MTU: mtu, OK: ok})
		return ok
	}
	if !try(low) {
		res.Note = "the target does not answer ICMP echo, so the path MTU cannot be measured"
		return res
	}
	res.Reachable = true
	if try(maxMTU) {
		res.PathMTU = maxMTU
	} else {
		lo, hi := low, maxMTU // lo works, hi fails
		for hi-lo > 1 && ctx.Err() == nil {
			mid := (lo + hi) / 2
			if try(mid) {
				lo = mid
			} else {
				hi = mid
			}
		}
		res.PathMTU = lo
	}
	res.MSS = res.PathMTU - over + 8 - 20 // TCP MSS: MTU − IP header − 20-byte TCP header
	switch {
	case res.PathMTU == maxMTU:
		res.Note = "no fragmentation up to the maximum tested size"
	case res.PathMTU < 1500 && !ip.Is6():
		res.Note = "smaller than Ethernet's 1500: a tunnel (PPPoE, VPN, GRE) on the path reduces the MTU"
	}
	return res
}
