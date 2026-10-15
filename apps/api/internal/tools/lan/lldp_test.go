package lan

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/bpf"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func tlv(typ int, v []byte) []byte {
	b := make([]byte, 2, 2+len(v))
	binary.BigEndian.PutUint16(b, uint16(typ<<9|len(v))) //nolint:gosec // test data
	return append(b, v...)
}

// lldpFrame is a switch port announcement as a Cisco/Juniper/Aruba switch sends it.
func lldpFrame(vlanTag bool) []byte {
	f := append([]byte{}, lldpMAC...)
	f = append(f, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55) // source: the switch port
	if vlanTag {
		f = append(f, 0x81, 0x00, 0x00, 0x0a)
	}
	f = append(f, 0x88, 0xcc)
	f = append(f, tlv(1, append([]byte{4}, 0x00, 0x11, 0x22, 0x33, 0x44, 0x00))...) // chassis: MAC
	f = append(f, tlv(2, append([]byte{5}, "GigabitEthernet1/0/24"...))...)         // port: ifName
	f = append(f, tlv(3, []byte{0, 120})...)                                        // TTL
	f = append(f, tlv(4, []byte("uplink to rack 3"))...)                            // port description
	f = append(f, tlv(5, []byte("core-sw1.example.net"))...)                        // system name
	f = append(f, tlv(6, []byte("Cisco IOS Software, C9300"))...)                   // system description
	f = append(f, tlv(7, []byte{0x00, 0x14, 0x00, 0x14})...)                        // capabilities: bridge+router
	f = append(f, tlv(8, []byte{5, 1, 192, 0, 2, 1, 2, 0, 0, 0, 1, 0})...)          // management: 192.0.2.1
	f = append(f, tlv(127, []byte{0x00, 0x80, 0xc2, 1, 0x00, 0x14})...)             // 802.1 PVID 20
	f = append(f, tlv(127, []byte{0x00, 0x12, 0x0f, 4, 0x23, 0x28})...)             // 802.3 max frame 9000
	return append(f, 0, 0)                                                          // end
}

func cdpTLV(typ uint16, v []byte) []byte {
	b := make([]byte, 4, 4+len(v))
	binary.BigEndian.PutUint16(b, typ)
	binary.BigEndian.PutUint16(b[2:], uint16(4+len(v))) //nolint:gosec // test data
	return append(b, v...)
}

func cdpFrame() []byte {
	f := append([]byte{}, cdpMAC...)
	f = append(f, 0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x00, 0x60) // source, 802.3 length
	f = append(f, 0xaa, 0xaa, 0x03, 0x00, 0x00, 0x0c, 0x20, 0x00) // LLC/SNAP, Cisco OUI, CDP PID
	f = append(f, 0x02, 180, 0x00, 0x00)                          // version, TTL, checksum
	f = append(f, cdpTLV(0x0001, []byte("access-sw2"))...)
	addr := []byte{0, 0, 0, 1, 1, 1, 0xcc, 0, 4, 198, 51, 100, 7} // 1 address: NLPID IPv4 198.51.100.7
	f = append(f, cdpTLV(0x0002, addr)...)
	f = append(f, cdpTLV(0x0003, []byte("FastEthernet0/3"))...)
	f = append(f, cdpTLV(0x0004, []byte{0, 0, 0, 0x28})...) // switch + IGMP
	f = append(f, cdpTLV(0x0005, []byte("Cisco IOS 15.2(7)E"))...)
	f = append(f, cdpTLV(0x0006, []byte("cisco WS-C2960X-24TS-L"))...)
	return append(f, cdpTLV(0x000a, []byte{0x00, 0x01})...)
}

func TestParseLLDP(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		n, ok := ParseLLDP(lldpFrame(tagged))
		if !ok || n.ChassisID != "00:11:22:33:44:00" || n.PortID != "GigabitEthernet1/0/24" || n.TTL != 120 || n.SystemName != "core-sw1.example.net" ||
			!slices.Equal(n.Capabilities, []string{"bridge", "router"}) || n.Management[0] != "192.0.2.1" || n.VLAN != 20 || n.MaxFrame != 9000 || n.SourceMAC != "00:11:22:33:44:55" {
			t.Errorf("tagged=%v: %+v", tagged, n)
		}
	}
	if _, ok := ParseLLDP(cdpFrame()); ok {
		t.Error("CDP parsed as LLDP")
	}
}

func TestParseCDP(t *testing.T) {
	n, ok := ParseCDP(cdpFrame())
	if !ok || n.SystemName != "access-sw2" || n.PortID != "FastEthernet0/3" || n.Platform != "cisco WS-C2960X-24TS-L" || n.Management[0] != "198.51.100.7" ||
		!slices.Equal(n.Capabilities, []string{"switch", "igmp"}) || n.VLAN != 1 || n.TTL != 180 {
		t.Errorf("cdp = %+v", n)
	}
	if _, ok := ParseCDP(lldpFrame(false)); ok {
		t.Error("LLDP parsed as CDP")
	}
}

func TestDiscoveryFilter(t *testing.T) {
	prog, err := discoveryFilter()
	if err != nil {
		t.Fatal(err)
	}
	insts, _ := bpf.Disassemble(prog)
	vm, err := bpf.NewVM(insts)
	if err != nil {
		t.Fatal(err)
	}
	ipv4 := append(append(append([]byte{}, 0, 1, 2, 3, 4, 5), 6, 7, 8, 9, 10, 11), 0x08, 0x00, 0x45, 0, 0, 20)
	for name, c := range map[string]struct {
		frame []byte
		pass  bool
	}{"lldp": {lldpFrame(false), true}, "lldp+vlan": {lldpFrame(true), true}, "cdp": {cdpFrame(), true}, "ipv4": {ipv4, false}} {
		n, err := vm.Run(c.frame)
		if err != nil || (n > 0) != c.pass {
			t.Errorf("%s: %d %v", name, n, err)
		}
	}
}

type fakeFrames struct {
	mu     sync.Mutex
	frames [][]byte
	closed bool
}

func (f *fakeFrames) ReadFrom(b []byte) (int, net.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.frames) == 0 {
		return 0, nil, timeoutErr{}
	}
	n := copy(b, f.frames[0])
	f.frames = f.frames[1:]
	return n, nil, nil
}
func (f *fakeFrames) SetReadDeadline(time.Time) error { return nil }
func (f *fakeFrames) Close() error                    { f.closed = true; return nil }

func TestLLDPTool(t *testing.T) {
	fake := &fakeFrames{frames: [][]byte{lldpFrame(false), lldpFrame(false), cdpFrame(), {1, 2, 3}}}
	tool := &LLDPTool{
		ReadFile: func(string) ([]byte, error) { return []byte(procRoute), nil },
		Open: func(iface string) (FrameConn, error) {
			if iface != "eth0" {
				return nil, errors.New("no such interface")
			}
			return fake, nil
		},
	}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	body := env.Get("/api/v1/lan/lldp/stream?seconds=5").Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("body = %s", body)
	}
	var d struct {
		Summary LLDPSummary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	if d.Summary.Interface != "eth0" || len(d.Summary.Neighbors) != 2 || strings.Count(body, "event: neighbor") != 2 || !fake.closed {
		t.Errorf("summary = %+v", d.Summary)
	}
	for q := range map[string]bool{"interface=bogus0": true, "seconds=600": true} {
		if b := env.Get("/api/v1/lan/lldp/stream?" + q).Raw.Body.String(); !strings.Contains(b, "INVALID_INPUT") {
			t.Errorf("%s: %s", q, b)
		}
	}
}
