package lan

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mdlayher/arp"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// ARP scan limits: subnets of at most /22 (1022 hosts), requests 1 ms apart, 2 s to collect.
const (
	MinARPPrefix = 22
	sendGap      = time.Millisecond
	collect      = 2 * time.Second
)

// Iface is a local interface with its connected IPv4 networks.
type Iface struct {
	Name     string
	Prefixes []netip.Prefix
}

// ARPConn is the subset of an ARP client the scan needs.
type ARPConn interface {
	Request(ip netip.Addr) error
	ReadReply() (netip.Addr, net.HardwareAddr, error)
	SetReadDeadline(t time.Time) error
	Close() error
}

type mdlayher struct{ c *arp.Client }

func (m mdlayher) Request(ip netip.Addr) error { return m.c.Request(ip) } //nolint:wrapcheck // passthrough
func (m mdlayher) SetReadDeadline(t time.Time) error {
	return m.c.SetReadDeadline(t) //nolint:wrapcheck // passthrough
}
func (m mdlayher) Close() error { return m.c.Close() } //nolint:wrapcheck // passthrough
func (m mdlayher) ReadReply() (netip.Addr, net.HardwareAddr, error) {
	for {
		p, _, err := m.c.Read()
		if err != nil {
			return netip.Addr{}, nil, err //nolint:wrapcheck // deadline ends the loop
		}
		if p.Operation == arp.OperationReply {
			return p.SenderIP, p.SenderHardwareAddr, nil
		}
	}
}

func systemIfaces() ([]Iface, error) {
	list, err := net.Interfaces()
	if err != nil {
		return nil, err //nolint:wrapcheck // internal
	}
	var out []Iface
	for _, ifi := range list {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifi.Addrs()
		var pfx []netip.Prefix
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if p, err := netip.ParsePrefix(ipn.String()); err == nil && p.Addr().Is4() {
					pfx = append(pfx, p.Masked())
				}
			}
		}
		if len(pfx) > 0 {
			out = append(out, Iface{Name: ifi.Name, Prefixes: pfx})
		}
	}
	return out, nil
}

func dialARP(name string) (ARPConn, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err //nolint:wrapcheck // internal
	}
	c, err := arp.Dial(ifi)
	if err != nil {
		return nil, err //nolint:wrapcheck // needs NET_RAW
	}
	return mdlayher{c}, nil
}

// ARPScanTool is LAN-02.
type ARPScanTool struct {
	deps   tools.Deps
	Ifaces func() ([]Iface, error)
	Dial   func(name string) (ARPConn, error)
}

func (*ARPScanTool) ID() string                                { return "lan.arpscan" }
func (*ARPScanTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *ARPScanTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Ifaces == nil {
		t.Ifaces = systemIfaces
	}
	if t.Dial == nil {
		t.Dial = dialARP
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(30*time.Second)).Get("/lan/arpscan/stream", t.handle)
}

// Host is the "host" event.
type Host struct {
	IP  string `json:"ip"`
	MAC string `json:"mac"`
}

// ARPSummary is the done payload.
type ARPSummary struct {
	Interface string `json:"interface"`
	Subnet    string `json:"subnet"`
	Scanned   int    `json:"scanned"`
	Hosts     []Host `json:"hosts"`
}

// Subnets lists the directly connected IPv4 networks (for the UI's suggestions).
func (t *ARPScanTool) subnet(raw string) (Iface, netip.Prefix, error) {
	p, err := netip.ParsePrefix(raw)
	if err != nil || !p.Addr().Is4() {
		return Iface{}, p, httpapi.InvalidInput("cidr", "enter an IPv4 subnet such as 192.168.1.0/24")
	}
	p = p.Masked()
	if p.Bits() < MinARPPrefix {
		return Iface{}, p, httpapi.InvalidInput("cidr", "subnets larger than /22 are not scanned")
	}
	ifaces, err := t.Ifaces()
	if err != nil {
		return Iface{}, p, err
	}
	for _, ifi := range ifaces {
		for _, conn := range ifi.Prefixes {
			if conn.Bits() <= p.Bits() && conn.Contains(p.Addr()) {
				return ifi, p, nil
			}
		}
	}
	return Iface{}, p, httpapi.InvalidInput("cidr", "the subnet is not directly connected to this server; ARP only works on the local network")
}

func (t *ARPScanTool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		raw := q.String("cidr", "", true, 43)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		ifi, p, err := t.subnet(raw)
		if err != nil {
			return nil, err
		}
		conn, err := t.Dial(ifi.Name)
		if err != nil {
			return nil, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "cannot open a raw socket for ARP (the api container needs NET_RAW and host networking)", Err: err}
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "interface": ifi.Name, "subnet": p.String()}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			defer conn.Close()
			return scan(ctx, conn, ifi.Name, p, emit), nil
		}}, nil
	})
}

func hostsOf(p netip.Prefix) []netip.Addr {
	var out []netip.Addr
	for a := p.Addr(); p.Contains(a); a = a.Next() {
		out = append(out, a)
	}
	if p.Bits() <= 30 && len(out) > 2 {
		out = out[1 : len(out)-1] // skip network and broadcast
	}
	return out
}

func scan(ctx context.Context, conn ARPConn, ifName string, p netip.Prefix, emit stream.Emit) ARPSummary {
	targets := hostsOf(p)
	sum := ARPSummary{Interface: ifName, Subnet: p.String(), Scanned: len(targets), Hosts: []Host{}}
	seen := map[netip.Addr]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sendDone := make(chan struct{})
	wg.Go(func() {
		defer close(sendDone)
		for _, a := range targets {
			if ctx.Err() != nil {
				return
			}
			_ = conn.Request(a)
			time.Sleep(sendGap)
		}
	})
	deadline := time.Time{}
	for {
		select {
		case <-sendDone:
			if deadline.IsZero() {
				deadline = time.Now().Add(collect)
			}
		default:
		}
		if !deadline.IsZero() && time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		ip, mac, err := conn.ReadReply()
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				break
			}
			continue
		}
		if !p.Contains(ip) {
			continue
		}
		mu.Lock()
		if !seen[ip] {
			seen[ip] = true
			h := Host{IP: ip.String(), MAC: mac.String()}
			sum.Hosts = append(sum.Hosts, h)
			emit("host", h)
		}
		mu.Unlock()
	}
	wg.Wait()
	slices.SortFunc(sum.Hosts, func(a, b Host) int {
		return netip.MustParseAddr(a.IP).Compare(netip.MustParseAddr(b.IP))
	})
	return sum
}
