package lan

import (
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"
	"golang.org/x/net/bpf"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeV6 answers a SOLICIT with ADVERTISEs from two servers plus one for another transaction.
type fakeV6 struct {
	mu      sync.Mutex
	replies [][]byte
	srcs    []netip.Addr
	drop    int
}

func advertise(sol *dhcpv6.Message, serverMAC net.HardwareAddr, addr string, prefix string, xid *dhcpv6.TransactionID) []byte {
	mods := []dhcpv6.Modifier{
		dhcpv6.WithServerID(&dhcpv6.DUIDLL{HWType: iana.HWTypeEthernet, LinkLayerAddr: serverMAC}),
		dhcpv6.WithDNS(net.ParseIP("2001:db8::53")),
		dhcpv6.WithDomainSearchList("home.arpa"),
		dhcpv6.WithIANA(dhcpv6.OptIAAddress{IPv6Addr: net.ParseIP(addr), PreferredLifetime: time.Hour, ValidLifetime: 2 * time.Hour}),
	}
	_, pfx, _ := net.ParseCIDR(prefix)
	mods = append(mods, dhcpv6.WithIAPD([4]byte{0, 0, 0, 1}, &dhcpv6.OptIAPrefix{Prefix: pfx, PreferredLifetime: time.Hour, ValidLifetime: 4 * time.Hour}))
	adv, err := dhcpv6.NewAdvertiseFromSolicit(sol, mods...)
	if err != nil {
		panic(err)
	}
	if xid != nil {
		adv.TransactionID = *xid
	}
	return adv.ToBytes()
}

func (f *fakeV6) Send(b []byte) error {
	d, err := dhcpv6.FromBytes(b)
	sol, ok := d.(*dhcpv6.Message)
	if err != nil || !ok || sol.MessageType != dhcpv6.MessageTypeSolicit || len(sol.Options.IAPD()) != 1 || len(sol.Options.IANA()) != 1 {
		return errors.New("expected a SOLICIT with IA_NA and IA_PD")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.drop > 0 { // the first SOLICIT is lost
		f.drop--
		return nil
	}
	other := dhcpv6.TransactionID{9, 9, 9}
	f.replies = [][]byte{
		advertise(sol, net.HardwareAddr{2, 0, 0, 0, 0, 1}, "2001:db8:1::100", "2001:db8:100::/56", nil),
		advertise(sol, net.HardwareAddr{2, 0, 0, 0, 0, 1}, "2001:db8:1::100", "2001:db8:100::/56", nil), // retransmission
		advertise(sol, net.HardwareAddr{2, 0, 0, 0, 0, 66}, "fd00::66", "fd00:66::/48", nil),
		advertise(sol, net.HardwareAddr{2, 0, 0, 0, 0, 77}, "fd00::77", "fd00:77::/48", &other),
	}
	f.srcs = []netip.Addr{netip.MustParseAddr("fe80::1"), netip.MustParseAddr("fe80::1"), netip.MustParseAddr("fe80::66"), netip.MustParseAddr("fe80::77")}
	return nil
}

func (f *fakeV6) Receive(b []byte) (int, netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.replies) == 0 {
		return 0, netip.Addr{}, timeoutErr{}
	}
	r, src := f.replies[0], f.srcs[0]
	f.replies, f.srcs = f.replies[1:], f.srcs[1:]
	return copy(b, r), src, nil
}
func (f *fakeV6) SetReadDeadline(time.Time) error { return nil }
func (f *fakeV6) Close() error                    { return nil }
func (f *fakeV6) MAC() net.HardwareAddr           { return net.HardwareAddr{2, 0, 0, 0, 0, 0xaa} }

func TestDHCPv6(t *testing.T) {
	tool := &DHCPv6Tool{
		ReadFile: func(string) ([]byte, error) { return []byte(procRoute), nil },
		Open:     func(string) (DHCPv6Link, error) { return &fakeV6{drop: 1}, nil },
	}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	var res DHCPv6Result
	env.Get("/api/v1/lan/dhcpv6?seconds=3").Must(t).Decode(t, &res)
	if res.Interface != "eth0" || res.Servers != 2 || len(res.Offers) != 2 {
		t.Fatalf("res = %+v", res)
	}
	o := res.Offers[0]
	if o.Server != "fe80::1" || o.Addresses[0].Value != "2001:db8:1::100" || o.Addresses[0].ValidSec != 7200 || o.Prefixes[0].Value != "2001:db8:100::/56" || o.DNS[0] != "2001:db8::53" || o.Search[0] != "home.arpa" || o.DUID == "" {
		t.Errorf("offer = %+v", o)
	}
	env.Get("/api/v1/lan/dhcpv6?seconds=60").ExpectError(t, "INVALID_INPUT")
}

func TestDHCPv6Filter(t *testing.T) {
	prog, err := dhcpv6Filter()
	if err != nil {
		t.Fatal(err)
	}
	insts, _ := bpf.Disassemble(prog)
	vm, err := bpf.NewVM(insts)
	if err != nil {
		t.Fatal(err)
	}
	pkt := func(next byte, srcPort uint16) []byte {
		b := make([]byte, 60)
		b[0], b[6] = 0x60, next
		binary.BigEndian.PutUint16(b[40:], srcPort)
		return b
	}
	for name, c := range map[string]struct {
		pkt  []byte
		pass bool
	}{"dhcpv6 server": {pkt(17, 547), true}, "dns": {pkt(17, 53), false}, "tcp": {pkt(6, 547), false}} {
		if n, err := vm.Run(c.pkt); err != nil || (n > 0) != c.pass {
			t.Errorf("%s: %d %v", name, n, err)
		}
	}
}
