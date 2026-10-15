// Package ednscomp implements DNS-18: EDNS compliance of a zone's authoritative servers, after
// the ISC EDNS compatibility tests (ednscomp): unknown versions, options and flags must be
// handled as RFC 6891 requires, or resolvers fall back and DNSSEC / large answers break.
package ednscomp

import (
	"context"
	"encoding/hex"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// MaxServers bounds how many nameservers of a zone are tested.
const MaxServers = 6

// Tool is the EDNS compliance test.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.ednscomp" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(45*time.Second)).Get("/dns/edns-compliance", t.handle)
}

// Check is one test against one server.
type Check struct {
	ID     string `json:"id"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// Server is the result for one nameserver address.
type Server struct {
	Name   string  `json:"name"`
	Addr   string  `json:"addr"`
	Error  string  `json:"error"`
	Checks []Check `json:"checks"`
	Passed int     `json:"passed"`
	NSID   string  `json:"nsid"`
	Cookie bool    `json:"cookie"`
}

// Result is the response payload.
type Result struct {
	Zone    string   `json:"zone"`
	Servers []Server `json:"servers"`
}

const (
	unknownOption = 100    // an unassigned EDNS option code
	unknownFlag   = 0x4000 // an undefined EDNS header flag (next to DO)
)

type test struct {
	id    string
	tcp   bool
	build func(m *dns.Msg)
	judge func(r *dns.Msg) (bool, string)
}

func opt(m *dns.Msg) *dns.OPT { return m.IsEdns0() }

func edns(m *dns.Msg) { m.SetEdns0(1232, false) }

func addOption(m *dns.Msg, o dns.EDNS0) { opt(m).Option = append(opt(m).Option, o) }

func noerror(r *dns.Msg) (bool, string) {
	if r.Rcode != dns.RcodeSuccess {
		return false, "expected NOERROR, got " + dnsx.RcodeName(r.Rcode)
	}
	return true, ""
}

func noerrorWithOPT(r *dns.Msg) (bool, string) {
	if ok, d := noerror(r); !ok {
		return ok, d
	}
	if o := opt(r); o == nil {
		return false, "no OPT record in the answer"
	} else if o.Version() != 0 {
		return false, "answer has EDNS version " + strconv.Itoa(int(o.Version()))
	}
	return true, ""
}

func badvers(r *dns.Msg) (bool, string) {
	if r.Rcode != dns.RcodeBadVers {
		return false, "expected BADVERS for EDNS version 1, got " + dnsx.RcodeName(r.Rcode)
	}
	if o := opt(r); o == nil || o.Version() != 0 {
		return false, "BADVERS must carry an OPT record with version 0"
	}
	if len(r.Answer) > 0 {
		return false, "BADVERS answer must not contain records"
	}
	return true, ""
}

// Tests in the order of the ISC ednscomp report.
var tests = []test{
	{id: "dns", build: func(*dns.Msg) {}, judge: func(r *dns.Msg) (bool, string) {
		if ok, d := noerror(r); !ok {
			return ok, d
		}
		if opt(r) != nil {
			return false, "OPT record returned to a query without EDNS"
		}
		return true, ""
	}},
	{id: "edns", build: edns, judge: noerrorWithOPT},
	{id: "edns1", build: func(m *dns.Msg) { edns(m); opt(m).SetVersion(1) }, judge: badvers},
	{id: "ednsopt", build: func(m *dns.Msg) { edns(m); addOption(m, &dns.EDNS0_LOCAL{Code: unknownOption, Data: []byte{}}) }, judge: func(r *dns.Msg) (bool, string) {
		if ok, d := noerrorWithOPT(r); !ok {
			return ok, d
		}
		if slices.ContainsFunc(opt(r).Option, func(o dns.EDNS0) bool { return o.Option() == unknownOption }) {
			return false, "unknown option echoed back"
		}
		return true, ""
	}},
	{id: "edns1opt", build: func(m *dns.Msg) {
		edns(m)
		opt(m).SetVersion(1)
		addOption(m, &dns.EDNS0_LOCAL{Code: unknownOption, Data: []byte{}})
	}, judge: badvers},
	{id: "do", build: func(m *dns.Msg) { m.SetEdns0(1232, true) }, judge: func(r *dns.Msg) (bool, string) {
		if ok, d := noerrorWithOPT(r); !ok {
			return ok, d
		}
		if !opt(r).Do() {
			return false, "DO bit not echoed"
		}
		return true, ""
	}},
	{id: "ednsflags", build: func(m *dns.Msg) { edns(m); opt(m).Hdr.Ttl |= unknownFlag }, judge: func(r *dns.Msg) (bool, string) {
		if ok, d := noerrorWithOPT(r); !ok {
			return ok, d
		}
		if opt(r).Hdr.Ttl&0x7FFF != 0 {
			return false, "undefined EDNS flag echoed back (must be cleared)"
		}
		return true, ""
	}},
	{id: "optlist", build: func(m *dns.Msg) {
		edns(m)
		addOption(m, &dns.EDNS0_NSID{Code: dns.EDNS0NSID})
		addOption(m, &dns.EDNS0_EXPIRE{Code: dns.EDNS0EXPIRE})
		addOption(m, &dns.EDNS0_COOKIE{Code: dns.EDNS0COOKIE, Cookie: "0102030405060708"})
		addOption(m, &dns.EDNS0_SUBNET{Code: dns.EDNS0SUBNET, Family: 1, SourceNetmask: 0, Address: net.IPv4zero})
	}, judge: noerrorWithOPT},
	{id: "tcp", tcp: true, build: edns, judge: noerrorWithOPT},
}

func exchange(ctx context.Context, addr string, tcp bool, m *dns.Msg) (*dns.Msg, error) {
	c := &dns.Client{Net: "udp", Timeout: 3 * time.Second, UDPSize: 1232}
	if tcp {
		c = &dns.Client{Net: "tcp", Timeout: 4 * time.Second}
	}
	r, _, err := c.ExchangeContext(ctx, m, addr)
	return r, err //nolint:wrapcheck // reported per check
}

// RunServer runs every test against addr for zone.
func RunServer(ctx context.Context, name, addr, zone string) Server {
	s := Server{Name: name, Addr: addr, Checks: []Check{}}
	for _, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(zone), dns.TypeSOA)
		m.RecursionDesired = false
		tc.build(m)
		r, err := exchange(ctx, addr, tc.tcp, m)
		c := Check{ID: tc.id}
		switch {
		case err != nil:
			c.Detail = "no answer (timeout or connection refused)"
		default:
			c.Pass, c.Detail = tc.judge(r)
			if o := opt(r); tc.id == "optlist" && o != nil {
				for _, o := range o.Option {
					switch v := o.(type) {
					case *dns.EDNS0_NSID:
						s.NSID = decodeNSID(v.Nsid)
					case *dns.EDNS0_COOKIE:
						s.Cookie = len(v.Cookie) > 16
					}
				}
			}
		}
		if c.Pass {
			s.Passed++
		}
		s.Checks = append(s.Checks, c)
	}
	return s
}

func decodeNSID(h string) string {
	b, err := hex.DecodeString(h)
	if err != nil {
		return h
	}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return h
		}
	}
	return string(b)
}
func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	rawZone := q.String("zone", "", true, 253)
	server := q.String("server", "", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	zone, err := target.NormalizeHostname(strings.TrimSpace(rawZone))
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("zone", "not a valid domain name"))
		return
	}
	type ns struct{ name, spec string }
	var list []ns
	if server != "" {
		list = append(list, ns{server, server})
	} else {
		sys, err := dnsx.SystemServer(dnsx.UDP)
		if err != nil {
			httpapi.Error(w, r, dnslookup.Upstream(err))
			return
		}
		msg, _, err := dnsx.Query(r.Context(), sys, zone, dns.TypeNS)
		if err != nil {
			httpapi.Error(w, r, dnslookup.Upstream(err))
			return
		}
		for _, rr := range msg.Answer {
			if n, ok := rr.(*dns.NS); ok && len(list) < MaxServers {
				h := strings.TrimSuffix(n.Ns, ".")
				list = append(list, ns{h, h})
			}
		}
		if len(list) == 0 {
			httpapi.Error(w, r, httpapi.InvalidInput("zone", "no NS records found for "+zone+"; is it a zone apex?"))
			return
		}
	}
	res := Result{Zone: zone, Servers: make([]Server, len(list))}
	ctx := r.Context()
	var wg sync.WaitGroup
	for i, n := range list {
		wg.Go(func() {
			srv, err := dnsx.ResolveServer(ctx, t.deps.Target, n.spec, dnsx.UDP, 4)
			if err != nil {
				res.Servers[i] = Server{Name: n.name, Error: httpapi.ToAPIError(err).Message, Checks: []Check{}}
				return
			}
			res.Servers[i] = RunServer(ctx, n.name, srv.Addr.String(), zone)
		})
	}
	wg.Wait()
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
