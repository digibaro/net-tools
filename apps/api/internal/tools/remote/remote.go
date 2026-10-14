// Package remote runs measurements from Globalping probes worldwide (CON-10 and the
// "Worldwide" vantage of ping/traceroute/mtr/dns/http).
package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/providers"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// Limits.
const (
	MaxProbes    = 10
	MaxLocations = 5
	MaxRawOutput = 16 << 10
	pollEvery    = 500 * time.Millisecond
	maxDuration  = 60 * time.Second
)

// Types are the measurement types Globalping supports.
var Types = []string{"ping", "traceroute", "mtr", "dns", "http"}

// Tool is remote.globalping.
type Tool struct {
	deps tools.Deps
	// BaseURL overrides https://api.globalping.io/v1 (tests).
	BaseURL string
	// PollEvery overrides the polling interval (tests).
	PollEvery time.Duration
}

func (*Tool) ID() string                                { return "remote.globalping" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) } // token optional

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.BaseURL == "" {
		t.BaseURL = "https://api.globalping.io/v1"
	}
	if t.PollEvery == 0 {
		t.PollEvery = pollEvery
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(maxDuration+10*time.Second)).Get("/remote/stream", t.handle)
}

// Probe describes where a result comes from.
type Probe struct {
	Continent string   `json:"continent"`
	Country   string   `json:"country"`
	City      string   `json:"city"`
	ASN       int      `json:"asn"`
	Network   string   `json:"network"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Tags      []string `json:"tags"`
}

// Result is one probe's measurement (the "probe" SSE event). Result holds Globalping's
// type-specific result object (stats/timings/hops/answers/headers), minus raw bodies.
type Result struct {
	Index  int             `json:"index"`
	Probe  Probe           `json:"probe"`
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
}

// Meta is the meta event payload.
type Meta struct {
	Tool      string   `json:"tool"`
	Vantage   string   `json:"vantage"`
	Type      string   `json:"type"`
	Target    string   `json:"target"`
	Locations []string `json:"locations"`
	Limit     int      `json:"limit"`
	ID        string   `json:"id"`
	Probes    int      `json:"probes"`
}

var locationRe = regexp.MustCompile(`^[\p{L}\d][\p{L}\d .+\-]{0,63}$`)

// ParseLocations splits "Germany, US+AWS" into Globalping "magic" locations.
func ParseLocations(raw string) ([]string, error) {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !locationRe.MatchString(p) {
			return nil, httpapi.InvalidInput("locations", "locations are names such as Germany, Europe, AS13335 or US+AWS")
		}
		out = append(out, p)
	}
	if len(out) > MaxLocations {
		return nil, httpapi.InvalidInput("locations", "at most 5 locations")
	}
	return out, nil
}

// validTarget accepts host names and public IP literals; the probes resolve names themselves.
func validTarget(raw, typ string) (string, error) {
	s := strings.TrimSpace(raw)
	if ip, err := netip.ParseAddr(strings.Trim(s, "[]")); err == nil {
		if typ == "dns" {
			return "", httpapi.InvalidInput("target", "enter a domain name to query")
		}
		// Measurements run on the public Internet, so the public policy applies whatever the profile.
		if err := (target.Policy{}).Check(ip); err != nil {
			return "", err //nolint:wrapcheck // client-facing target error
		}
		return ip.String(), nil
	}
	h, err := target.NormalizeHostname(s)
	if err != nil || !strings.Contains(h, ".") {
		return "", httpapi.InvalidInput("target", "enter a public host name or IP address")
	}
	return h, nil
}

type createResp struct {
	ID          string `json:"id"`
	ProbesCount int    `json:"probesCount"`
}

type measurement struct {
	Status  string `json:"status"`
	Results []struct {
		Probe  Probe                      `json:"probe"`
		Result map[string]json.RawMessage `json:"result"`
	} `json:"results"`
}

func upstreamMessage(b []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &e) == nil {
		return e.Error.Message
	}
	return ""
}

func (t *Tool) headers() http.Header {
	h := http.Header{}
	if tok := t.deps.Config.Features.ProviderSecrets["GLOBALPING_TOKEN"]; tok != "" {
		h.Set("Authorization", "Bearer "+tok)
	}
	return h
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		typ := q.Enum("type", "ping", Types...)
		rawTarget := q.String("target", "", true, 253)
		rawLoc := q.String("locations", "", false, 400)
		limit := q.Int("limit", 3, 1, MaxProbes)
		dnsType := q.Enum("dnsType", "A", "A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA", "CAA", "PTR", "SRV", "HTTPS")
		proto := q.Enum("protocol", "ICMP", "ICMP", "TCP", "UDP")
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		tgt, err := validTarget(rawTarget, typ)
		if err != nil {
			return nil, err
		}
		locs, err := ParseLocations(rawLoc)
		if err != nil {
			return nil, err
		}
		body := map[string]any{"type": typ, "target": tgt, "limit": limit}
		if len(locs) > 0 {
			l := make([]map[string]string, len(locs))
			for i, m := range locs {
				l[i] = map[string]string{"magic": m}
			}
			body["locations"] = l
		}
		switch typ {
		case "ping":
			body["measurementOptions"] = map[string]any{"packets": 4}
		case "mtr":
			body["measurementOptions"] = map[string]any{"packets": 4, "protocol": proto}
		case "traceroute":
			body["measurementOptions"] = map[string]any{"protocol": proto}
		case "dns":
			body["measurementOptions"] = map[string]any{"query": map[string]string{"type": dnsType}}
		case "http":
			body["measurementOptions"] = map[string]any{"protocol": "HTTPS", "request": map[string]string{"method": "HEAD", "path": "/"}}
		}
		var created createResp
		err = t.deps.Providers.GetJSON(ctx, providers.Request{
			Provider: "Globalping", Method: http.MethodPost, URL: t.BaseURL + "/measurements",
			Header: t.headers(), Body: body, UpstreamMessage: upstreamMessage,
		}, &created)
		if err != nil {
			return nil, err //nolint:wrapcheck // provider errors are client-safe
		}
		meta := Meta{Tool: t.ID(), Vantage: "globalping", Type: typ, Target: tgt, Locations: locs, Limit: limit, ID: created.ID, Probes: created.ProbesCount}
		if meta.Locations == nil {
			meta.Locations = []string{}
		}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return t.poll(ctx, created.ID, emit)
		}}, nil
	})
}

// Summary is the done payload.
type Summary struct {
	ID       string `json:"id"`
	Probes   int    `json:"probes"`
	Finished bool   `json:"finished"`
}

func (t *Tool) poll(ctx context.Context, id string, emit stream.Emit) (Summary, error) {
	ctx, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()
	sent := map[int]bool{}
	tick := time.NewTicker(t.PollEvery)
	defer tick.Stop()
	for {
		var m measurement
		err := t.deps.Providers.GetJSON(ctx, providers.Request{Provider: "Globalping", URL: t.BaseURL + "/measurements/" + id, Header: t.headers(), TTL: -1}, &m)
		if err != nil {
			return Summary{}, err //nolint:wrapcheck // client-safe
		}
		for i, res := range m.Results {
			var status string
			_ = json.Unmarshal(res.Result["status"], &status)
			if sent[i] || status == "in-progress" {
				continue
			}
			sent[i] = true
			emit("probe", Result{Index: i, Probe: normalizeProbe(res.Probe), Status: status, Result: trim(res.Result)})
		}
		if m.Status != "in-progress" {
			return Summary{ID: id, Probes: len(m.Results), Finished: true}, nil
		}
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return Summary{ID: id, Probes: len(sent), Finished: false}, nil
			}
			return Summary{}, ctx.Err() //nolint:wrapcheck // cancellation
		case <-tick.C:
		}
	}
}

func normalizeProbe(p Probe) Probe {
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return p
}

// trim drops bulky fields (bodies, raw headers) and caps the raw output.
func trim(r map[string]json.RawMessage) json.RawMessage {
	delete(r, "rawBody")
	delete(r, "rawHeaders")
	if raw, ok := r["rawOutput"]; ok && len(raw) > MaxRawOutput {
		var s string
		if json.Unmarshal(raw, &s) == nil && len(s) > MaxRawOutput {
			b, _ := json.Marshal(s[:MaxRawOutput] + "\n…")
			r["rawOutput"] = b
		}
	}
	b, _ := json.Marshal(r)
	return b
}
