package lan

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// mDNS constants (RFC 6762/6763).
const (
	servicesName = "_services._dns-sd._udp.local."
	MaxInstances = 300
)

var mdnsGroup = netip.MustParseAddrPort("224.0.0.251:5353")

// PacketConn is the socket the multicast tools use (injectable for tests).
type PacketConn interface {
	WriteTo(p []byte, addr net.Addr) (int, error)
	ReadFrom(p []byte) (int, net.Addr, error)
	SetReadDeadline(t time.Time) error
	Close() error
}

func listenUDP4(ctx context.Context) (PacketConn, error) {
	var lc net.ListenConfig
	pc, err := lc.ListenPacket(ctx, "udp4", ":0")
	if err != nil {
		return nil, err //nolint:wrapcheck // surfaced as INTERNAL
	}
	return pc, nil
}

// Service is the "service" event: one discovered instance.
type Service struct {
	Instance  string   `json:"instance"`
	Type      string   `json:"type"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Addresses []string `json:"addresses"`
	TXT       []string `json:"txt"`
	From      string   `json:"from"`
}

// MDNSTool is LAN-03.
type MDNSTool struct {
	deps   tools.Deps
	Listen func(ctx context.Context) (PacketConn, error)
}

func (*MDNSTool) ID() string                                { return "lan.mdns" }
func (*MDNSTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *MDNSTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Listen == nil {
		t.Listen = listenUDP4
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(20*time.Second)).Get("/lan/mdns/stream", t.handle)
}

func query(names []string, qtype uint16) []byte {
	m := new(dns.Msg)
	m.Id = 0
	for _, n := range names {
		m.Question = append(m.Question, dns.Question{Name: n, Qtype: qtype, Qclass: dns.ClassINET | 1<<15}) // QU: unicast reply
	}
	b, _ := m.Pack()
	return b
}

type browser struct {
	mu        sync.Mutex
	types     map[string]bool
	instances map[string]*Service
	addrs     map[string][]string
	emitted   map[string]bool
}

func (b *browser) absorb(msg *dns.Msg, from string, wantType string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var newTypes []string
	for _, rr := range append(msg.Answer, msg.Extra...) {
		switch v := rr.(type) {
		case *dns.PTR:
			if strings.EqualFold(v.Hdr.Name, servicesName) {
				if !b.types[v.Ptr] && (wantType == "" || strings.EqualFold(v.Ptr, wantType)) {
					b.types[v.Ptr] = true
					newTypes = append(newTypes, v.Ptr)
				}
				continue
			}
			if _, ok := b.instances[v.Ptr]; !ok && len(b.instances) < MaxInstances {
				b.instances[v.Ptr] = &Service{Instance: v.Ptr, Type: v.Hdr.Name, Addresses: []string{}, TXT: []string{}, From: from}
			}
		case *dns.SRV:
			if s, ok := b.instances[v.Hdr.Name]; ok {
				s.Host, s.Port = v.Target, int(v.Port)
			}
		case *dns.TXT:
			if s, ok := b.instances[v.Hdr.Name]; ok && len(s.TXT) == 0 {
				for _, x := range v.Txt {
					if x != "" && len(s.TXT) < 30 {
						s.TXT = append(s.TXT, x)
					}
				}
			}
		case *dns.A:
			b.addAddr(v.Hdr.Name, v.A.String())
		case *dns.AAAA:
			b.addAddr(v.Hdr.Name, v.AAAA.String())
		}
	}
	return newTypes
}

func (b *browser) addAddr(host, a string) {
	if !slices.Contains(b.addrs[host], a) {
		b.addrs[host] = append(b.addrs[host], a)
	}
}

// ready returns instances with a resolved host that were not emitted yet.
func (b *browser) ready() []Service {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Service
	for k, s := range b.instances {
		if s.Host == "" || b.emitted[k] {
			continue
		}
		s.Addresses = slices.Clone(b.addrs[s.Host])
		if len(s.Addresses) == 0 && len(s.TXT) == 0 {
			continue
		}
		b.emitted[k] = true
		out = append(out, *s)
	}
	return out
}

func (t *MDNSTool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		service := q.String("service", "", false, 100)
		seconds := q.Int("seconds", 5, 1, 10)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		if service != "" {
			service = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(service)), ".local") + ".local."
			if !strings.HasPrefix(service, "_") || !strings.Contains(service, "._tcp.") && !strings.Contains(service, "._udp.") {
				return nil, httpapi.InvalidInput("service", "service types look like _http._tcp")
			}
		}
		pc, err := t.Listen(ctx)
		if err != nil {
			return nil, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "cannot open a UDP socket", Err: err}
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "seconds": seconds, "service": service}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			defer pc.Close()
			return browse(ctx, pc, service, time.Duration(seconds)*time.Second, emit), nil
		}}, nil
	})
}

// MDNSSummary is the done payload.
type MDNSSummary struct {
	Types    []string  `json:"types"`
	Services []Service `json:"services"`
}

func browse(ctx context.Context, pc PacketConn, service string, dur time.Duration, emit stream.Emit) MDNSSummary {
	b := &browser{types: map[string]bool{}, instances: map[string]*Service{}, addrs: map[string][]string{}, emitted: map[string]bool{}}
	dst := net.UDPAddrFromAddrPort(mdnsGroup)
	if service != "" {
		b.types[service] = true
		_, _ = pc.WriteTo(query([]string{service}, dns.TypePTR), dst)
	} else {
		_, _ = pc.WriteTo(query([]string{servicesName}, dns.TypePTR), dst)
	}
	deadline := time.Now().Add(dur)
	nextResend := time.Now().Add(time.Second)
	buf := make([]byte, 9000)
	var all []Service
	for time.Now().Before(deadline) && ctx.Err() == nil {
		_ = pc.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, from, err := pc.ReadFrom(buf)
		if err == nil {
			msg := new(dns.Msg)
			if msg.Unpack(buf[:n]) == nil && msg.Response {
				src := ""
				if ua, ok := from.(*net.UDPAddr); ok {
					src = ua.IP.String()
				}
				if newTypes := b.absorb(msg, src, service); len(newTypes) > 0 {
					_, _ = pc.WriteTo(query(newTypes, dns.TypePTR), dst)
				}
			}
		}
		for _, s := range b.ready() {
			all = append(all, s)
			emit("service", s)
		}
		if time.Now().After(nextResend) { // ask again for SRV/TXT/A of instances still unresolved
			b.mu.Lock()
			var names []string
			for k, s := range b.instances {
				if s.Host == "" && len(names) < 20 {
					names = append(names, k)
				}
			}
			b.mu.Unlock()
			if len(names) > 0 {
				_, _ = pc.WriteTo(query(names, dns.TypeANY), dst)
			}
			nextResend = time.Now().Add(time.Second)
		}
	}
	sum := MDNSSummary{Types: []string{}, Services: all}
	if sum.Services == nil {
		sum.Services = []Service{}
	}
	for ty := range b.types {
		sum.Types = append(sum.Types, ty)
	}
	slices.Sort(sum.Types)
	slices.SortFunc(sum.Services, func(a, b Service) int { return strings.Compare(a.Type+a.Instance, b.Type+b.Instance) })
	return sum
}
