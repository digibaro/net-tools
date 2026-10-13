// Package dnstrace implements DNS-04: iterative resolution from the root (dig +trace).
package dnstrace

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// MaxSteps bounds the walk (loops, absurd delegation chains).
const MaxSteps = 16

// RootServers are the IPv4 root server addresses (IANA root hints, a–m).
var RootServers = []string{
	"198.41.0.4", "170.247.170.2", "192.33.4.12", "199.7.91.13", "192.203.230.10", "192.5.5.241",
	"192.112.36.4", "198.97.190.53", "192.36.148.17", "192.58.128.30", "193.0.14.129", "199.7.83.42", "202.12.27.33",
}

// Tool is the delegation trace.
type Tool struct {
	deps tools.Deps
	// Roots and Port override the root hints (tests).
	Roots []string
	Port  uint16
}

func (*Tool) ID() string                                { return "dns.trace" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Get("/dns/trace/stream", t.handle)
}

// Step is one server queried during the walk (SSE "step" event).
type Step struct {
	Step     int           `json:"step"`
	Zone     string        `json:"zone"`
	Server   string        `json:"server"`
	Address  string        `json:"address"`
	Rcode    string        `json:"rcode"`
	AA       bool          `json:"aa"`
	RTTMs    float64       `json:"rttMs"`
	Answer   []dnsx.Record `json:"answer"`
	Referral []string      `json:"referral"`
	Note     string        `json:"note"`
}

// Summary is the done payload.
type Summary struct {
	Steps  int    `json:"steps"`
	Rcode  string `json:"rcode"`
	Answer int    `json:"answers"`
}

type nameserver struct {
	name string
	addr netip.Addr
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(_ context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("name", "", true, 253)
		typ := q.Enum("type", "A", dnslookup.RecordTypes...)
		if err := q.Err(); err != nil {
			return nil, err
		}
		qtype, _ := dnsx.ParseType(typ)
		name, err := dnslookup.ParseName(raw, qtype)
		if err != nil {
			return nil, err
		}
		return &stream.Job{
			Meta: map[string]any{"tool": t.ID(), "vantage": "server", "params": map[string]string{"name": name, "type": typ}},
			Run: func(ctx context.Context, emit stream.Emit) (any, error) {
				return t.walk(ctx, name, qtype, emit)
			},
		}, nil
	})
}

func (t *Tool) walk(ctx context.Context, name string, qtype uint16, emit stream.Emit) (Summary, error) {
	roots := t.Roots
	if roots == nil {
		roots = RootServers
	}
	servers := make([]nameserver, 0, len(roots))
	for i, ip := range roots {
		servers = append(servers, nameserver{name: string(rune('a'+i%26)) + ".root-servers.net.", addr: netip.MustParseAddr(ip)})
	}
	zone := "."
	sum := Summary{}
	for step := 1; step <= MaxSteps; step++ {
		msg, used, rtt, err := t.ask(ctx, servers, name, qtype)
		if err != nil {
			return sum, err
		}
		st := Step{Step: step, Zone: zone, Server: used.name, Address: used.addr.String(), Rcode: dnsx.RcodeName(msg.Rcode), AA: msg.Authoritative, RTTMs: dnslookup.Ms(rtt), Answer: dnsx.Records(msg.Answer), Referral: []string{}}
		sum.Steps, sum.Rcode, sum.Answer = step, st.Rcode, len(st.Answer)

		if msg.Rcode != dns.RcodeSuccess || len(msg.Answer) > 0 || msg.Authoritative {
			emit("step", st)
			return sum, nil
		}
		next, nextZone := referral(msg)
		st.Referral = names(next)
		if len(next) == 0 || !dns.IsSubDomain(zone, nextZone) || nextZone == zone {
			st.Note = "no usable referral (lame or misconfigured server)"
			emit("step", st)
			return sum, nil
		}
		if missingGlue(next) {
			st.Note = "referral without glue; nameserver addresses resolved separately"
			next = t.resolveGlue(ctx, next)
		}
		emit("step", st)
		servers, zone = next, nextZone
	}
	return sum, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "delegation chain longer than 16 steps"}
}

// ask tries up to three servers; each address must pass the target policy.
func (t *Tool) ask(ctx context.Context, servers []nameserver, name string, qtype uint16) (*dns.Msg, nameserver, time.Duration, error) {
	port := t.Port
	if port == 0 {
		port = 53
	}
	var lastErr error
	tries := 0
	for _, ns := range servers {
		if !ns.addr.IsValid() {
			continue
		}
		if err := t.deps.Target.Policy.Check(ns.addr); err != nil {
			lastErr = err
			continue
		}
		if tries == 3 {
			break
		}
		tries++
		m := dnsx.NewQuery(name, qtype, false)
		m.RecursionDesired = false
		qctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		msg, rtt, err := dnsx.Exchange(qctx, dnsx.Server{Label: ns.name, Addr: netip.AddrPortFrom(ns.addr, port), Transport: dnsx.UDP}, m)
		cancel()
		if err == nil {
			return msg, ns, rtt, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ns, 0, ctx.Err() //nolint:wrapcheck // cancellation
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no nameserver address available")
	}
	return nil, nameserver{}, 0, dnslookup.Upstream(lastErr)
}

// referral extracts the delegated zone, its NS names and glue (IPv4 preferred).
func referral(m *dns.Msg) ([]nameserver, string) {
	var zone string
	var out []nameserver
	for _, rr := range m.Ns {
		if ns, ok := rr.(*dns.NS); ok {
			zone = ns.Hdr.Name
			out = append(out, nameserver{name: strings.ToLower(ns.Ns)})
		}
	}
	glue := map[string][]netip.Addr{}
	for _, rr := range m.Extra {
		switch v := rr.(type) {
		case *dns.A:
			if a, ok := netip.AddrFromSlice(v.A.To4()); ok {
				glue[strings.ToLower(v.Hdr.Name)] = append(glue[strings.ToLower(v.Hdr.Name)], a)
			}
		case *dns.AAAA:
			if a, ok := netip.AddrFromSlice(v.AAAA); ok {
				glue[strings.ToLower(v.Hdr.Name)] = append(glue[strings.ToLower(v.Hdr.Name)], a)
			}
		}
	}
	var withGlue []nameserver
	for _, ns := range out {
		addrs := glue[ns.name]
		slices.SortStableFunc(addrs, func(a, b netip.Addr) int {
			if a.Is4() == b.Is4() {
				return 0
			}
			if a.Is4() {
				return -1
			}
			return 1
		})
		if len(addrs) == 0 {
			withGlue = append(withGlue, ns)
			continue
		}
		for _, a := range addrs {
			withGlue = append(withGlue, nameserver{name: ns.name, addr: a})
		}
	}
	return withGlue, zone
}

func missingGlue(ns []nameserver) bool {
	return !slices.ContainsFunc(ns, func(n nameserver) bool { return n.addr.IsValid() })
}

func (t *Tool) resolveGlue(ctx context.Context, ns []nameserver) []nameserver {
	sys, err := dnsx.SystemServer(dnsx.UDP)
	if err != nil {
		return ns
	}
	var out []nameserver
	for _, n := range ns[:min(len(ns), 4)] {
		addrs, _ := dnsx.Addresses(ctx, sys, n.name)
		for _, a := range addrs {
			out = append(out, nameserver{name: n.name, addr: a})
		}
	}
	return out
}

func names(ns []nameserver) []string {
	var out []string
	for _, n := range ns {
		if !slices.Contains(out, n.name) {
			out = append(out, n.name)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}
