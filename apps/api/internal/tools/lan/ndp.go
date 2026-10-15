package lan

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mdlayher/ndp"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

var (
	allNodes   = netip.MustParseAddr("ff02::1")
	allRouters = netip.MustParseAddr("ff02::2")
)

// ICMPv6Conn sends and receives raw ICMPv6 messages on one interface (injectable for tests).
type ICMPv6Conn interface {
	WriteTo(b []byte, dst netip.Addr) error
	ReadFrom(b []byte) (int, netip.Addr, error)
	SetReadDeadline(t time.Time) error
	Close() error
	// Local is the link-local source address and MAC of the interface.
	Local() (netip.Addr, net.HardwareAddr)
}

type icmp6Conn struct {
	pc  *ipv6.PacketConn
	c   *icmp.PacketConn
	ifi *net.Interface
	src netip.Addr
}

func (c *icmp6Conn) WriteTo(b []byte, dst netip.Addr) error {
	cm := &ipv6.ControlMessage{HopLimit: 255, IfIndex: c.ifi.Index} // NDP requires hop limit 255
	_, err := c.pc.WriteTo(b, cm, &net.IPAddr{IP: dst.AsSlice(), Zone: c.ifi.Name})
	return err //nolint:wrapcheck // send errors are counted, not shown
}

func (c *icmp6Conn) ReadFrom(b []byte) (int, netip.Addr, error) {
	n, _, src, err := c.pc.ReadFrom(b)
	if err != nil {
		return 0, netip.Addr{}, err //nolint:wrapcheck // deadline ends the loop
	}
	a, _ := src.(*net.IPAddr)
	ip, _ := netip.AddrFromSlice(a.IP)
	return n, ip.Unmap(), nil
}

func (c *icmp6Conn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t) //nolint:wrapcheck // passthrough
}
func (c *icmp6Conn) Close() error                          { return c.c.Close() } //nolint:wrapcheck // passthrough
func (c *icmp6Conn) Local() (netip.Addr, net.HardwareAddr) { return c.src, c.ifi.HardwareAddr }

// openICMPv6 opens a raw ICMPv6 socket (NET_RAW) bound to the interface's link-local address.
func openICMPv6(iface string) (ICMPv6Conn, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface: %w", err)
	}
	addrs, _ := ifi.Addrs()
	var ll netip.Addr
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if ip, ok := netip.AddrFromSlice(n.IP); ok && ip.Is6() && ip.IsLinkLocalUnicast() {
				ll = ip
				break
			}
		}
	}
	if !ll.IsValid() {
		return nil, fmt.Errorf("%s has no IPv6 link-local address", iface)
	}
	c, err := icmp.ListenPacket("ip6:ipv6-icmp", ll.String()+"%"+iface)
	if err != nil {
		return nil, fmt.Errorf("raw socket: %w", err)
	}
	pc := c.IPv6PacketConn()
	_ = pc.SetMulticastHopLimit(255)
	_ = pc.SetHopLimit(255)
	_ = pc.SetMulticastInterface(ifi)
	_ = pc.SetControlMessage(ipv6.FlagInterface, true)
	return &icmp6Conn{pc: pc, c: c, ifi: ifi, src: ll}, nil
}

// solicitedNode is the solicited-node multicast group of addr (RFC 4291 §2.7.1).
func solicitedNode(addr netip.Addr) netip.Addr {
	a := addr.As16()
	return netip.AddrFrom16([16]byte{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0xff, a[13], a[14], a[15]})
}

// NDPHost is the "host" event: an IPv6 neighbour on the link.
type NDPHost struct {
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Router bool   `json:"router"`
}

// NDPPrefix is one prefix from a router advertisement.
type NDPPrefix struct {
	Prefix     string `json:"prefix"`
	OnLink     bool   `json:"onLink"`
	SLAAC      bool   `json:"slaac"`
	ValidSec   int64  `json:"validSeconds"`
	PreferrSec int64  `json:"preferredSeconds"`
}

// NDPRouter is the "router" event: a router advertisement.
type NDPRouter struct {
	IP          string      `json:"ip"`
	MAC         string      `json:"mac"`
	LifetimeSec int64       `json:"lifetimeSeconds"`
	Managed     bool        `json:"managed"`
	Other       bool        `json:"other"`
	Preference  string      `json:"preference"`
	HopLimit    int         `json:"hopLimit"`
	MTU         int         `json:"mtu"`
	Prefixes    []NDPPrefix `json:"prefixes"`
	DNS         []string    `json:"dns"`
	Search      []string    `json:"search"`
}

// NDPSummary is the done payload.
type NDPSummary struct {
	Interface string      `json:"interface"`
	Source    string      `json:"source"`
	Hosts     []NDPHost   `json:"hosts"`
	Routers   []NDPRouter `json:"routers"`
}

func lla(opts []ndp.Option, dir ndp.Direction) string {
	for _, o := range opts {
		if l, ok := o.(*ndp.LinkLayerAddress); ok && l.Direction == dir {
			return l.Addr.String()
		}
	}
	return ""
}

// RouterFromRA converts a parsed router advertisement.
func RouterFromRA(src netip.Addr, ra *ndp.RouterAdvertisement) NDPRouter {
	r := NDPRouter{IP: src.String(), LifetimeSec: int64(ra.RouterLifetime / time.Second), Managed: ra.ManagedConfiguration, Other: ra.OtherConfiguration,
		Preference: strings.ToLower(ra.RouterSelectionPreference.String()), HopLimit: int(ra.CurrentHopLimit), Prefixes: []NDPPrefix{}, DNS: []string{}, Search: []string{},
		MAC: lla(ra.Options, ndp.Source)}
	for _, o := range ra.Options {
		switch v := o.(type) {
		case *ndp.PrefixInformation:
			r.Prefixes = append(r.Prefixes, NDPPrefix{Prefix: netip.PrefixFrom(v.Prefix, int(v.PrefixLength)).String(), OnLink: v.OnLink, SLAAC: v.AutonomousAddressConfiguration,
				ValidSec: int64(v.ValidLifetime / time.Second), PreferrSec: int64(v.PreferredLifetime / time.Second)})
		case *ndp.MTU:
			r.MTU = int(v.MTU)
		case *ndp.RecursiveDNSServer:
			for _, s := range v.Servers {
				r.DNS = append(r.DNS, s.String())
			}
		case *ndp.DNSSearchList:
			r.Search = append(r.Search, v.DomainNames...)
		}
	}
	return r
}

func echoRequest(id int) []byte {
	b, _ := (&icmp.Message{Type: ipv6.ICMPTypeEchoRequest, Body: &icmp.Echo{ID: id, Seq: 1, Data: []byte("nettools-ndp")}}).Marshal(nil)
	return b
}

// NDPTool is LAN-08: IPv6 neighbours and routers.
type NDPTool struct {
	deps     tools.Deps
	Open     func(iface string) (ICMPv6Conn, error)
	ReadFile func(string) ([]byte, error)
}

func (*NDPTool) ID() string                                { return "lan.ndp" }
func (*NDPTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *NDPTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Open == nil {
		t.Open = openICMPv6
	}
	if t.ReadFile == nil {
		t.ReadFile = os.ReadFile
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(20*time.Second)).Get("/lan/ndp/stream", t.handle)
}

func (t *NDPTool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		iface := q.String("interface", "", false, 32)
		seconds := q.Int("seconds", 4, 2, 10)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		if iface == "" {
			b, _ := t.ReadFile("/proc/net/route")
			routes := ParseDefaultRoutes(b)
			if len(routes) == 0 {
				return nil, httpapi.InvalidInput("interface", "no default route; name the interface to probe")
			}
			iface = routes[0].Interface
		}
		conn, err := t.Open(iface)
		if err != nil {
			return nil, httpapi.InvalidInput("interface", "cannot probe IPv6 on "+iface+" (no link-local address, unknown interface, or no NET_RAW / host networking)")
		}
		src, _ := conn.Local()
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "interface": iface, "source": src.String(), "seconds": seconds}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			defer conn.Close()
			sum := probeNDP(ctx, conn, time.Duration(seconds)*time.Second, emit)
			sum.Interface = iface
			return sum, nil
		}}, nil
	})
}

func probeNDP(ctx context.Context, conn ICMPv6Conn, dur time.Duration, emit stream.Emit) NDPSummary {
	self, mac := conn.Local()
	sum := NDPSummary{Source: self.String(), Hosts: []NDPHost{}, Routers: []NDPRouter{}}
	var idb [2]byte
	_, _ = rand.Read(idb[:])
	id := int(binary.BigEndian.Uint16(idb[:]))
	srcOpt := []ndp.Option{&ndp.LinkLayerAddress{Direction: ndp.Source, Addr: mac}}
	if rs, err := ndp.MarshalMessage(&ndp.RouterSolicitation{Options: srcOpt}); err == nil {
		_ = conn.WriteTo(rs, allRouters)
	}
	_ = conn.WriteTo(echoRequest(id), allNodes)

	hosts := map[netip.Addr]*NDPHost{}
	var order []netip.Addr
	routers := map[netip.Addr]bool{}
	solicit := func(ip netip.Addr) {
		if ns, err := ndp.MarshalMessage(&ndp.NeighborSolicitation{TargetAddress: ip, Options: srcOpt}); err == nil {
			_ = conn.WriteTo(ns, solicitedNode(ip))
		}
	}
	addHost := func(ip netip.Addr) *NDPHost {
		if h, ok := hosts[ip]; ok {
			return h
		}
		h := &NDPHost{IP: ip.String()}
		hosts[ip] = h
		order = append(order, ip)
		return h
	}
	deadline := time.Now().Add(dur)
	resent := false
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) && ctx.Err() == nil && len(order) < 512 {
		if !resent && time.Until(deadline) < dur/2 {
			_ = conn.WriteTo(echoRequest(id), allNodes) // multicast is lossy: ask twice
			resent = true
		}
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, src, err := conn.ReadFrom(buf)
		if err != nil || n < 4 || src == self {
			continue
		}
		switch buf[0] {
		case byte(ipv6.ICMPTypeEchoReply):
			m, err := icmp.ParseMessage(58, buf[:n])
			if e, ok := m.Body.(*icmp.Echo); err == nil && ok && e.ID == id {
				if _, known := hosts[src]; !known {
					addHost(src)
					solicit(src)
				}
			}
		case byte(ipv6.ICMPTypeNeighborAdvertisement), byte(ipv6.ICMPTypeRouterAdvertisement):
			m, err := ndp.ParseMessage(buf[:n])
			if err != nil {
				continue
			}
			switch v := m.(type) {
			case *ndp.NeighborAdvertisement:
				h := addHost(v.TargetAddress)
				if mac := lla(v.Options, ndp.Target); mac != "" && h.MAC == "" {
					h.MAC, h.Router = mac, v.Router
					emit("host", *h)
				}
			case *ndp.RouterAdvertisement:
				if !routers[src] {
					routers[src] = true
					rt := RouterFromRA(src, v)
					sum.Routers = append(sum.Routers, rt)
					emit("router", rt)
					h := addHost(src)
					h.Router = true
					if h.MAC == "" && rt.MAC != "" {
						h.MAC = rt.MAC
						emit("host", *h)
					}
				}
			}
		}
	}
	for _, ip := range order {
		sum.Hosts = append(sum.Hosts, *hosts[ip])
	}
	slices.SortFunc(sum.Hosts, func(a, b NDPHost) int {
		return netip.MustParseAddr(a.IP).Compare(netip.MustParseAddr(b.IP))
	})
	return sum
}
