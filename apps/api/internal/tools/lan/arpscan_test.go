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

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// fakeARP answers requests for the addresses in alive (plus one stray reply outside the subnet).
type fakeARP struct {
	mu      sync.Mutex
	alive   map[netip.Addr]string
	replies chan [2]string
	closed  bool
}

func newFake(alive map[string]string) *fakeARP {
	f := &fakeARP{alive: map[netip.Addr]string{}, replies: make(chan [2]string, 2048)}
	for ip, mac := range alive {
		f.alive[netip.MustParseAddr(ip)] = mac
	}
	f.replies <- [2]string{"10.9.9.9", "de:ad:be:ef:00:01"} // stray reply, not in the scanned subnet
	return f
}

func (f *fakeARP) Request(ip netip.Addr) error {
	if mac, ok := f.alive[ip]; ok {
		f.replies <- [2]string{ip.String(), mac}
		f.replies <- [2]string{ip.String(), mac} // duplicates are common
	}
	return nil
}

func (f *fakeARP) ReadReply() (netip.Addr, net.HardwareAddr, error) {
	select {
	case r := <-f.replies:
		mac, _ := net.ParseMAC(r[1])
		return netip.MustParseAddr(r[0]), mac, nil
	case <-time.After(20 * time.Millisecond):
		return netip.Addr{}, nil, timeoutErr{}
	}
}

func (f *fakeARP) SetReadDeadline(time.Time) error { return nil }
func (f *fakeARP) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func TestARPScan(t *testing.T) {
	fake := newFake(map[string]string{"192.168.1.1": "aa:bb:cc:00:00:01", "192.168.1.20": "aa:bb:cc:00:00:14"})
	tool := &ARPScanTool{
		Ifaces: func() ([]Iface, error) {
			return []Iface{{Name: "eth0", Prefixes: []netip.Prefix{netip.MustParsePrefix("192.168.1.0/24")}}}, nil
		},
		Dial: func(name string) (ARPConn, error) {
			if name != "eth0" {
				return nil, errors.New("wrong interface")
			}
			return fake, nil
		},
	}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	body := env.Get("/api/v1/lan/arpscan/stream?cidr=192.168.1.0/27").Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("body = %s", body)
	}
	var d struct {
		Summary ARPSummary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	s := d.Summary
	if s.Interface != "eth0" || s.Scanned != 30 || len(s.Hosts) != 2 || s.Hosts[0].IP != "192.168.1.1" || s.Hosts[1].MAC != "aa:bb:cc:00:00:14" || strings.Count(body, "event: host") != 2 || !fake.closed {
		t.Errorf("summary = %+v", s)
	}
	for q, want := range map[string]string{"cidr=10.0.0.0/24": "not directly connected", "cidr=192.168.0.0/16": "larger than /22", "cidr=nope": "IPv4 subnet", "cidr=2001:db8::/120": "IPv4 subnet"} {
		if b := env.Get("/api/v1/lan/arpscan/stream?" + q).Raw.Body.String(); !strings.Contains(b, want) {
			t.Errorf("%s: %s", q, b)
		}
	}
}
