package lan

import (
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mdlayher/ndp"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeLink answers like a LAN with a router (fe80::1) and a host (fe80::2).
type fakeLink struct {
	mu      sync.Mutex
	pending []struct {
		b   []byte
		src netip.Addr
	}
	closed bool
}

var (
	selfLL   = netip.MustParseAddr("fe80::aa")
	routerLL = netip.MustParseAddr("fe80::1")
	hostLL   = netip.MustParseAddr("fe80::2")
	routerHW = net.HardwareAddr{0x02, 0, 0, 0, 0, 0x01}
	hostHW   = net.HardwareAddr{0x02, 0, 0, 0, 0, 0x02}
)

func (f *fakeLink) queue(b []byte, src netip.Addr) {
	f.pending = append(f.pending, struct {
		b   []byte
		src netip.Addr
	}{b, src})
}

func (f *fakeLink) WriteTo(b []byte, dst netip.Addr) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch b[0] {
	case byte(ipv6.ICMPTypeEchoRequest):
		m, _ := icmp.ParseMessage(58, b)
		e := m.Body.(*icmp.Echo) //nolint:forcetypeassert // test
		for _, src := range []netip.Addr{selfLL, routerLL, hostLL} {
			r, _ := (&icmp.Message{Type: ipv6.ICMPTypeEchoReply, Body: &icmp.Echo{ID: e.ID, Seq: e.Seq}}).Marshal(nil)
			f.queue(r, src)
		}
	case byte(ipv6.ICMPTypeNeighborSolicitation):
		m, _ := ndp.ParseMessage(b)
		ns := m.(*ndp.NeighborSolicitation) //nolint:forcetypeassert // test
		if dst != solicitedNode(ns.TargetAddress) {
			return errors.New("NS not sent to the solicited-node group")
		}
		hw := map[netip.Addr]net.HardwareAddr{routerLL: routerHW, hostLL: hostHW}[ns.TargetAddress]
		na, _ := ndp.MarshalMessage(&ndp.NeighborAdvertisement{Solicited: true, Router: ns.TargetAddress == routerLL, TargetAddress: ns.TargetAddress,
			Options: []ndp.Option{&ndp.LinkLayerAddress{Direction: ndp.Target, Addr: hw}}})
		f.queue(na, ns.TargetAddress)
	case byte(ipv6.ICMPTypeRouterSolicitation):
		if dst != allRouters {
			return errors.New("RS not sent to ff02::2")
		}
		ra, _ := ndp.MarshalMessage(&ndp.RouterAdvertisement{CurrentHopLimit: 64, OtherConfiguration: true, RouterLifetime: 1800 * time.Second,
			Options: []ndp.Option{
				&ndp.LinkLayerAddress{Direction: ndp.Source, Addr: routerHW},
				&ndp.MTU{MTU: 1492},
				&ndp.PrefixInformation{PrefixLength: 64, OnLink: true, AutonomousAddressConfiguration: true, ValidLifetime: 86400 * time.Second, PreferredLifetime: 14400 * time.Second, Prefix: netip.MustParseAddr("2001:db8:1::")},
				&ndp.RecursiveDNSServer{Lifetime: time.Hour, Servers: []netip.Addr{netip.MustParseAddr("2001:db8:1::53")}},
				&ndp.DNSSearchList{Lifetime: time.Hour, DomainNames: []string{"home.arpa"}},
			}})
		f.queue(ra, routerLL)
	}
	return nil
}

func (f *fakeLink) ReadFrom(b []byte) (int, netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.pending) == 0 {
		return 0, netip.Addr{}, timeoutErr{}
	}
	p := f.pending[0]
	f.pending = f.pending[1:]
	return copy(b, p.b), p.src, nil
}
func (f *fakeLink) SetReadDeadline(time.Time) error { return nil }
func (f *fakeLink) Close() error                    { f.closed = true; return nil }
func (f *fakeLink) Local() (netip.Addr, net.HardwareAddr) {
	return selfLL, net.HardwareAddr{0x02, 0, 0, 0, 0, 0xaa}
}

func TestNDP(t *testing.T) {
	link := &fakeLink{}
	tool := &NDPTool{
		ReadFile: func(string) ([]byte, error) { return []byte(procRoute), nil },
		Open:     func(string) (ICMPv6Conn, error) { return link, nil },
	}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	body := env.Get("/api/v1/lan/ndp/stream?seconds=2").Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("body = %s", body)
	}
	var d struct {
		Summary NDPSummary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	s := d.Summary
	if s.Interface != "eth0" || s.Source != "fe80::aa" || len(s.Hosts) != 2 || !link.closed {
		t.Fatalf("summary = %+v", s)
	}
	if s.Hosts[0].IP != "fe80::1" || s.Hosts[0].MAC != routerHW.String() || !s.Hosts[0].Router || s.Hosts[1].MAC != hostHW.String() || s.Hosts[1].Router {
		t.Errorf("hosts = %+v", s.Hosts)
	}
	if len(s.Routers) != 1 {
		t.Fatalf("routers = %+v", s.Routers)
	}
	r := s.Routers[0]
	if r.MTU != 1492 || !r.Other || r.Managed || r.LifetimeSec != 1800 || r.Prefixes[0].Prefix != "2001:db8:1::/64" || !r.Prefixes[0].SLAAC || r.DNS[0] != "2001:db8:1::53" || r.Search[0] != "home.arpa" || r.MAC != routerHW.String() {
		t.Errorf("router = %+v", r)
	}
	if b := env.Get("/api/v1/lan/ndp/stream?seconds=60").Raw.Body.String(); !strings.Contains(b, "INVALID_INPUT") {
		t.Error(b)
	}
}

func TestSolicitedNode(t *testing.T) {
	if got := solicitedNode(netip.MustParseAddr("2001:db8::1:800:200e:8c6c")); got.String() != "ff02::1:ff0e:8c6c" {
		t.Errorf("got %s", got)
	}
}
