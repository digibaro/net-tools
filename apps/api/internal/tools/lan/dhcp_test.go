package lan

import (
	"errors"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

const procRoute = "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n" +
	"wlan0\t00000000\t0101A8C0\t0003\t0\t0\t600\t00000000\t0\t0\t0\n" +
	"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n" +
	"eth0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n"

func TestParseRoutesAndARP(t *testing.T) {
	r := ParseDefaultRoutes([]byte(procRoute))
	if len(r) != 2 || r[0].Interface != "eth0" || r[0].Gateway.String() != "192.168.1.1" || r[1].Metric != 600 {
		t.Errorf("routes = %+v", r)
	}
	arp := ParseARPTable([]byte("IP address       HW type     Flags       HW address            Mask     Device\n" +
		"192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:01     *        eth0\n" +
		"192.168.1.9      0x1         0x0         00:00:00:00:00:00     *        eth0\n"))
	if len(arp) != 1 || arp[netip.MustParseAddr("192.168.1.1")] != "aa:bb:cc:dd:ee:01" {
		t.Errorf("arp = %v", arp)
	}
}

type fakeDHCP struct {
	mu      sync.Mutex
	pending [][]byte
	from    []string
}

func offer(req *dhcpv4.DHCPv4, server, yours string, xidOverride bool) []byte {
	mods := []dhcpv4.Modifier{
		dhcpv4.WithMessageType(dhcpv4.MessageTypeOffer), dhcpv4.WithYourIP(net.ParseIP(yours)),
		dhcpv4.WithOption(dhcpv4.OptServerIdentifier(net.ParseIP(server))), dhcpv4.WithOption(dhcpv4.OptRouter(net.ParseIP(server))),
		dhcpv4.WithOption(dhcpv4.OptDNS(net.ParseIP("1.1.1.1"))), dhcpv4.WithOption(dhcpv4.OptSubnetMask(net.CIDRMask(24, 32))),
		dhcpv4.WithOption(dhcpv4.OptIPAddressLeaseTime(24 * time.Hour)), dhcpv4.WithOption(dhcpv4.OptDomainName("lan")),
	}
	if xidOverride {
		mods = append(mods, dhcpv4.WithTransactionID(dhcpv4.TransactionID{1, 2, 3, 4}))
	}
	m, err := dhcpv4.NewReplyFromRequest(req, mods...)
	if err != nil {
		panic(err)
	}
	return m.ToBytes()
}

func (f *fakeDHCP) WriteTo(p []byte, addr net.Addr) (int, error) {
	req, err := dhcpv4.FromBytes(p)
	if err != nil || req.MessageType() != dhcpv4.MessageTypeDiscover || addr.(*net.UDPAddr).Port != 67 || !req.IsBroadcast() {
		return 0, errors.New("expected a broadcast DISCOVER to port 67")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = [][]byte{
		offer(req, "192.168.1.1", "192.168.1.50", false),
		offer(req, "192.168.1.1", "192.168.1.50", false), // duplicate
		offer(req, "192.168.1.66", "192.168.1.200", false),
		offer(req, "10.0.0.1", "10.0.0.9", true), // someone else's transaction
	}
	f.from = []string{"192.168.1.1", "192.168.1.1", "192.168.1.66", "10.0.0.1"}
	return len(p), nil
}

func (f *fakeDHCP) ReadFrom(p []byte) (int, net.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.pending) == 0 {
		return 0, nil, timeoutErr{}
	}
	b, from := f.pending[0], f.from[0]
	f.pending, f.from = f.pending[1:], f.from[1:]
	return copy(p, b), &net.UDPAddr{IP: net.ParseIP(from), Port: 67}, nil
}

func (f *fakeDHCP) SetReadDeadline(time.Time) error  { return nil }
func (f *fakeDHCP) Close() error                     { return nil }
func (f *fakeDHCP) LocalAddr() net.Addr              { return &net.UDPAddr{Port: 68} }
func (f *fakeDHCP) SetDeadline(time.Time) error      { return nil }
func (f *fakeDHCP) SetWriteDeadline(time.Time) error { return nil }

func TestDHCP(t *testing.T) {
	var opened string
	tool := &DHCPTool{
		ReadFile: func(string) ([]byte, error) { return []byte(procRoute), nil },
		Open: func(iface string) (net.PacketConn, net.HardwareAddr, error) {
			opened = iface
			if iface == "bogus0" {
				return nil, nil, errors.New("no such interface")
			}
			mac, _ := net.ParseMAC("02:00:00:00:00:01")
			return &fakeDHCP{}, mac, nil
		},
	}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	var res DHCPResult
	env.Get("/api/v1/lan/dhcp?seconds=1").Must(t).Decode(t, &res)
	if opened != "eth0" || res.Interface != "eth0" || res.Servers != 2 || len(res.Offers) != 2 {
		t.Fatalf("res = %+v", res)
	}
	o := res.Offers[0]
	if o.Server != "192.168.1.1" || o.OfferedIP != "192.168.1.50" || o.SubnetMask != "255.255.255.0" || o.Routers[0] != "192.168.1.1" || o.DNS[0] != "1.1.1.1" || o.LeaseSeconds != 86400 || o.Domain != "lan" {
		t.Errorf("offer = %+v", o)
	}
	env.Get("/api/v1/lan/dhcp?interface=bogus0").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/lan/dhcp?seconds=99").ExpectError(t, "INVALID_INPUT")
}
