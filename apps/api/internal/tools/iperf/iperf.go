// Package iperf implements CON-08: a bandwidth test against an iperf3 server via the
// allow-listed iperf3 binary (--json-stream, one event per line).
package iperf

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/execx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// Bounds beyond IPERF_MAX_SECONDS.
const (
	MaxParallel   = 4
	MaxUDPMbps    = 1000
	connectTimeMs = 3000
)

// Tool is net.iperf.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string { return "net.iperf" }

func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	if ok, reason := tools.RequireFlag(d.Config.Features.Iperf); !ok {
		return ok, reason
	}
	if d.Exec == nil || !d.Exec.Available("iperf3") {
		return false, tools.ReasonBinaryMissing
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(time.Duration(d.Config.Limits.IperfMaxSeconds+15)*time.Second)).Get("/iperf/stream", t.handle)
}

// Options are the validated test parameters.
type Options struct {
	Port     int
	UDP      bool
	Seconds  int
	Reverse  bool
	Parallel int
	Mbps     int
}

// Args builds the iperf3 argv. ip must come from target.Resolve.
func Args(ip string, o Options) ([]string, error) {
	ipArg, err := execx.UserValue(ip)
	if err != nil {
		return nil, err //nolint:wrapcheck // validation
	}
	args := []string{"-c", ipArg, "-p", strconv.Itoa(o.Port), "-t", strconv.Itoa(o.Seconds), "-i", "1", "-P", strconv.Itoa(o.Parallel),
		"--connect-timeout", strconv.Itoa(connectTimeMs), "--json-stream"}
	if o.UDP {
		args = append(args, "-u", "-b", strconv.Itoa(o.Mbps)+"M")
	}
	if o.Reverse {
		args = append(args, "-R")
	}
	return args, nil
}

// Interval is the "interval" event.
type Interval struct {
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Mbps        float64 `json:"mbps"`
	Bytes       int64   `json:"bytes"`
	Retransmits *int    `json:"retransmits,omitempty"`
	Packets     *int    `json:"packets,omitempty"`
}

// Summary is the done payload.
type Summary struct {
	Protocol     string   `json:"protocol"`
	Reverse      bool     `json:"reverse"`
	Seconds      float64  `json:"seconds"`
	SentMbps     float64  `json:"sentMbps"`
	ReceivedMbps float64  `json:"receivedMbps"`
	SentBytes    int64    `json:"sentBytes"`
	Retransmits  *int     `json:"retransmits,omitempty"`
	JitterMs     *float64 `json:"jitterMs,omitempty"`
	LostPercent  *float64 `json:"lostPercent,omitempty"`
	LostPackets  *int     `json:"lostPackets,omitempty"`
	Server       string   `json:"server"`
	Version      string   `json:"version"`
}

type sum struct {
	Start         float64  `json:"start"`
	End           float64  `json:"end"`
	Seconds       float64  `json:"seconds"`
	Bytes         int64    `json:"bytes"`
	BitsPerSecond float64  `json:"bits_per_second"`
	Retransmits   *int     `json:"retransmits"`
	Packets       *int     `json:"packets"`
	JitterMs      *float64 `json:"jitter_ms"`
	LostPackets   *int     `json:"lost_packets"`
	LostPercent   *float64 `json:"lost_percent"`
}

type event struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

const mbit = 1e6

// Parser accumulates --json-stream lines.
type Parser struct {
	summary Summary
	ended   bool
	errMsg  string
}

// Line handles one line and returns an interval when it carried one.
func (p *Parser) Line(line string) *Interval {
	var e event
	if json.Unmarshal([]byte(line), &e) != nil {
		return nil
	}
	switch e.Event {
	case "start":
		var d struct {
			Version      string `json:"version"`
			ConnectingTo struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			} `json:"connecting_to"`
			TestStart struct {
				Protocol string `json:"protocol"`
				Reverse  int    `json:"reverse"`
			} `json:"test_start"`
		}
		if json.Unmarshal(e.Data, &d) == nil {
			p.summary.Protocol, p.summary.Reverse, p.summary.Version = d.TestStart.Protocol, d.TestStart.Reverse == 1, d.Version
			p.summary.Server = d.ConnectingTo.Host + ":" + strconv.Itoa(d.ConnectingTo.Port)
		}
	case "interval":
		var d struct {
			Sum sum `json:"sum"`
		}
		if json.Unmarshal(e.Data, &d) == nil {
			s := d.Sum
			return &Interval{Start: s.Start, End: s.End, Mbps: s.BitsPerSecond / mbit, Bytes: s.Bytes, Retransmits: s.Retransmits, Packets: s.Packets}
		}
	case "end":
		var d struct {
			Sum         *sum `json:"sum"` // UDP: receiver-side jitter and loss
			SumSent     *sum `json:"sum_sent"`
			SumReceived *sum `json:"sum_received"`
		}
		if json.Unmarshal(e.Data, &d) == nil && d.SumSent != nil {
			p.ended = true
			p.summary.Seconds, p.summary.SentMbps, p.summary.SentBytes, p.summary.Retransmits = d.SumSent.Seconds, d.SumSent.BitsPerSecond/mbit, d.SumSent.Bytes, d.SumSent.Retransmits
			if d.SumReceived != nil {
				p.summary.ReceivedMbps = d.SumReceived.BitsPerSecond / mbit
			}
			if d.Sum != nil {
				p.summary.JitterMs, p.summary.LostPercent, p.summary.LostPackets = d.Sum.JitterMs, d.Sum.LostPercent, d.Sum.LostPackets
			}
		}
	case "error":
		var msg string
		if json.Unmarshal(e.Data, &msg) == nil {
			p.errMsg = msg
		}
	}
	return nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	lim := t.deps.Config.Limits
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("server", "", true, 253)
		o := Options{
			Port:     q.Int("port", 5201, 1, 65535),
			UDP:      q.Enum("proto", "tcp", "tcp", "udp") == "udp",
			Seconds:  q.Int("duration", min(10, lim.IperfMaxSeconds), 1, lim.IperfMaxSeconds),
			Reverse:  q.Bool("reverse", false),
			Parallel: q.Int("parallel", 1, 1, MaxParallel),
			Mbps:     q.Int("bitrate", 10, 1, MaxUDPMbps),
		}
		family := q.Family()
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Family: family, Field: "server"})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		args, err := Args(tgt.IP.String(), o)
		if err != nil {
			return nil, err
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": raw, "used": tgt.IP}, "params": map[string]any{"port": o.Port, "udp": o.UDP, "duration": o.Seconds, "reverse": o.Reverse, "parallel": o.Parallel}}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return t.run(ctx, args, emit)
		}}, nil
	})
}

func (t *Tool) run(ctx context.Context, args []string, emit stream.Emit) (Summary, error) {
	var p Parser
	stderr, err := t.deps.Exec.Run(ctx, "iperf3", args, func(line string) {
		if iv := p.Line(line); iv != nil {
			emit("interval", iv)
		}
	})
	if p.errMsg != "" {
		return Summary{}, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "iperf3: " + p.errMsg}
	}
	if err != nil && ctx.Err() == nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		return Summary{}, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "iperf3 failed: " + msg, Err: err}
	}
	if !p.ended {
		if ctx.Err() != nil {
			return Summary{}, ctx.Err() //nolint:wrapcheck // stream reports TIMEOUT
		}
		return Summary{}, errors.New("iperf3 ended without a result")
	}
	return p.summary, nil
}
