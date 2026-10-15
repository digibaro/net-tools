package lan

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mdlayher/packet"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/stream"
)

// Neighbour discovery multicast MACs and ethertypes.
var (
	lldpMAC = net.HardwareAddr{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e}
	cdpMAC  = net.HardwareAddr{0x01, 0x00, 0x0c, 0xcc, 0xcc, 0xcc}
)

const (
	etherLLDP   = 0x88cc
	etherVLAN   = 0x8100
	maxNeighbor = 64
)

// Neighbor is the "neighbor" event: one switch / router port that announced itself.
type Neighbor struct {
	Protocol          string   `json:"protocol"`
	SourceMAC         string   `json:"sourceMac"`
	ChassisID         string   `json:"chassisId"`
	PortID            string   `json:"portId"`
	PortDescription   string   `json:"portDescription"`
	SystemName        string   `json:"systemName"`
	SystemDescription string   `json:"systemDescription"`
	Platform          string   `json:"platform"`
	Capabilities      []string `json:"capabilities"`
	Enabled           []string `json:"enabled"`
	Management        []string `json:"management"`
	VLAN              int      `json:"vlan"`
	MaxFrame          int      `json:"maxFrame"`
	TTL               int      `json:"ttl"`
}

var lldpCaps = []string{"other", "repeater", "bridge", "wlan-ap", "router", "telephone", "docsis", "station", "c-vlan", "s-vlan", "tpmr"}

func capNames(bits uint16) []string {
	out := []string{}
	for i, n := range lldpCaps {
		if bits&(1<<i) != 0 {
			out = append(out, n)
		}
	}
	return out
}

func printable(b []byte) string {
	s := strings.TrimRight(string(b), "\x00")
	for _, r := range s {
		if r < 0x20 && r != '\n' && r != '\t' {
			return fmt.Sprintf("%x", b)
		}
	}
	return s
}

func lldpID(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	sub, v := b[0], b[1:]
	switch {
	case (sub == 4 || sub == 3) && len(v) == 6: // MAC address (chassis 4 / port 3)
		return net.HardwareAddr(v).String()
	case sub == 5 && len(v) >= 5 && (v[0] == 1 || v[0] == 2): // network address: AFI + address
		if a, ok := netip.AddrFromSlice(v[1:]); ok {
			return a.String()
		}
	}
	return printable(v)
}

// ParseLLDP decodes an Ethernet frame carrying LLDP (IEEE 802.1AB); ok is false for anything else.
func ParseLLDP(frame []byte) (Neighbor, bool) {
	if len(frame) < 16 {
		return Neighbor{}, false
	}
	off, et := 12, binary.BigEndian.Uint16(frame[12:])
	if et == etherVLAN && len(frame) >= 18 {
		off, et = 16, binary.BigEndian.Uint16(frame[16:])
	}
	if et != etherLLDP {
		return Neighbor{}, false
	}
	n := Neighbor{Protocol: "LLDP", SourceMAC: net.HardwareAddr(frame[6:12]).String(), Capabilities: []string{}, Enabled: []string{}, Management: []string{}}
	b := frame[off+2:]
	for len(b) >= 2 {
		h := binary.BigEndian.Uint16(b)
		typ, l := int(h>>9), int(h&0x1ff)
		if typ == 0 || len(b) < 2+l {
			break
		}
		v := b[2 : 2+l]
		b = b[2+l:]
		switch typ {
		case 1:
			n.ChassisID = lldpID(v)
		case 2:
			n.PortID = lldpID(v)
		case 3:
			if l >= 2 {
				n.TTL = int(binary.BigEndian.Uint16(v))
			}
		case 4:
			n.PortDescription = printable(v)
		case 5:
			n.SystemName = printable(v)
		case 6:
			n.SystemDescription = printable(v)
		case 7:
			if l >= 4 {
				n.Capabilities, n.Enabled = capNames(binary.BigEndian.Uint16(v)), capNames(binary.BigEndian.Uint16(v[2:]))
			}
		case 8:
			if l >= 2 && int(v[0]) >= 2 && len(v) > int(v[0]) {
				if a, ok := netip.AddrFromSlice(v[2 : 1+int(v[0])]); ok {
					n.Management = append(n.Management, a.String())
				}
			}
		case 127:
			if l >= 6 && v[0] == 0x00 && v[1] == 0x80 && v[2] == 0xc2 && v[3] == 1 { // IEEE 802.1 port VLAN ID
				n.VLAN = int(binary.BigEndian.Uint16(v[4:]))
			}
			if l >= 6 && v[0] == 0x00 && v[1] == 0x12 && v[2] == 0x0f && v[3] == 4 { // IEEE 802.3 max frame size
				n.MaxFrame = int(binary.BigEndian.Uint16(v[4:]))
			}
		}
	}
	return n, n.ChassisID != "" || n.PortID != ""
}

var cdpCaps = []struct {
	bit  uint32
	name string
}{{0x01, "router"}, {0x02, "bridge"}, {0x04, "source-route-bridge"}, {0x08, "switch"}, {0x10, "host"}, {0x20, "igmp"}, {0x40, "repeater"}, {0x80, "phone"}, {0x100, "remote"}, {0x200, "cvta"}, {0x400, "two-port-mac-relay"}}

func cdpAddresses(v []byte) []string {
	out := []string{}
	if len(v) < 4 {
		return out
	}
	count := int(binary.BigEndian.Uint32(v))
	v = v[4:]
	for range min(count, 16) {
		if len(v) < 2 {
			break
		}
		pl := int(v[1])
		if len(v) < 2+pl+2 {
			break
		}
		al := int(binary.BigEndian.Uint16(v[2+pl:]))
		if len(v) < 4+pl+al {
			break
		}
		if a, ok := netip.AddrFromSlice(v[4+pl : 4+pl+al]); ok {
			out = append(out, a.String())
		}
		v = v[4+pl+al:]
	}
	return out
}

// ParseCDP decodes an 802.3 + SNAP frame carrying Cisco Discovery Protocol; ok is false otherwise.
func ParseCDP(frame []byte) (Neighbor, bool) {
	// dst(6) src(6) length(2) DSAP SSAP ctrl(3) OUI 00000c(3) PID 2000(2) | version ttl checksum(4)
	if len(frame) < 26 || !slices.Equal(frame[:6], cdpMAC) || frame[14] != 0xaa || frame[15] != 0xaa || frame[17] != 0 || frame[18] != 0 || frame[19] != 0x0c || binary.BigEndian.Uint16(frame[20:]) != 0x2000 {
		return Neighbor{}, false
	}
	n := Neighbor{Protocol: "CDP", SourceMAC: net.HardwareAddr(frame[6:12]).String(), TTL: int(frame[23]), Capabilities: []string{}, Enabled: []string{}, Management: []string{}}
	b := frame[26:]
	for len(b) >= 4 {
		typ, l := binary.BigEndian.Uint16(b), int(binary.BigEndian.Uint16(b[2:]))
		if l < 4 || len(b) < l {
			break
		}
		v := b[4:l]
		b = b[l:]
		switch typ {
		case 0x0001:
			n.ChassisID, n.SystemName = printable(v), printable(v)
		case 0x0002, 0x0016:
			for _, a := range cdpAddresses(v) {
				if !slices.Contains(n.Management, a) {
					n.Management = append(n.Management, a)
				}
			}
		case 0x0003:
			n.PortID = printable(v)
		case 0x0004:
			if len(v) >= 4 {
				bits := binary.BigEndian.Uint32(v)
				for _, c := range cdpCaps {
					if bits&c.bit != 0 {
						n.Capabilities = append(n.Capabilities, c.name)
					}
				}
				n.Enabled = n.Capabilities
			}
		case 0x0005:
			n.SystemDescription = printable(v)
		case 0x0006:
			n.Platform = printable(v)
		case 0x000a:
			if len(v) >= 2 {
				n.VLAN = int(binary.BigEndian.Uint16(v))
			}
		}
	}
	return n, n.ChassisID != "" || n.PortID != ""
}

// FrameConn reads raw Ethernet frames (injectable for tests).
type FrameConn interface {
	ReadFrom(b []byte) (int, net.Addr, error)
	SetReadDeadline(t time.Time) error
	Close() error
}

// discoveryFilter accepts only LLDP (ethertype 0x88cc, also VLAN-tagged) and CDP (dst 01:00:0c:cc:cc:cc).
func discoveryFilter() ([]bpf.RawInstruction, error) {
	prog, err := bpf.Assemble([]bpf.Instruction{
		bpf.LoadAbsolute{Off: 12, Size: 2},                             // 0 ethertype
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: etherLLDP, SkipTrue: 8},   // 1 → accept
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: etherVLAN, SkipFalse: 2},  // 2 → CDP check
		bpf.LoadAbsolute{Off: 16, Size: 2},                             // 3 ethertype after the VLAN tag
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: etherLLDP, SkipTrue: 5},   // 4 → accept
		bpf.LoadAbsolute{Off: 0, Size: 4},                              // 5 destination MAC 01:00:0c:cc:..
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0x01000ccc, SkipFalse: 2}, // 6 → reject
		bpf.LoadAbsolute{Off: 4, Size: 2},                              // 7 ..:cc:cc
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 0xcccc, SkipTrue: 1},      // 8 → accept
		bpf.RetConstant{Val: 0},                                        // 9 reject
		bpf.RetConstant{Val: 0xffff},                                   // 10 accept
	})
	if err != nil {
		return nil, fmt.Errorf("bpf: %w", err)
	}
	return prog, nil
}

// openDiscovery opens a raw packet socket (NET_RAW) that receives only LLDP and CDP frames, and
// joins their multicast groups (no promiscuous mode).
func openDiscovery(iface string) (FrameConn, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface: %w", err)
	}
	c, err := packet.Listen(ifi, packet.Raw, unix.ETH_P_ALL, nil)
	if err != nil {
		return nil, fmt.Errorf("raw socket: %w", err)
	}
	prog, err := discoveryFilter()
	if err == nil {
		if e := c.SetBPF(prog); e != nil {
			err = fmt.Errorf("attach filter: %w", e)
		}
	}
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	rc, err := c.SyscallConn()
	if err == nil {
		_ = rc.Control(func(fd uintptr) {
			for _, mac := range []net.HardwareAddr{lldpMAC, cdpMAC} {
				mreq := unix.PacketMreq{Ifindex: int32(ifi.Index), Type: unix.PACKET_MR_MULTICAST, Alen: 6} //nolint:gosec // interface index fits int32
				copy(mreq.Address[:], mac)
				_ = unix.SetsockoptPacketMreq(int(fd), unix.SOL_PACKET, unix.PACKET_ADD_MEMBERSHIP, &mreq) //nolint:gosec // fd fits int
			}
		})
	}
	return c, nil
}

// LLDPTool is LAN-07: LLDP / CDP neighbour discovery.
type LLDPTool struct {
	deps     tools.Deps
	Open     func(iface string) (FrameConn, error)
	ReadFile func(string) ([]byte, error)
}

func (*LLDPTool) ID() string                                { return "lan.lldp" }
func (*LLDPTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *LLDPTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Open == nil {
		t.Open = openDiscovery
	}
	if t.ReadFile == nil {
		t.ReadFile = os.ReadFile
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(80*time.Second)).Get("/lan/lldp/stream", t.handle)
}

// LLDPSummary is the done payload.
type LLDPSummary struct {
	Interface string     `json:"interface"`
	Seconds   int        `json:"seconds"`
	Neighbors []Neighbor `json:"neighbors"`
}

func (t *LLDPTool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		iface := q.String("interface", "", false, 32)
		seconds := q.Int("seconds", 35, 5, 65)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		if iface == "" {
			b, _ := t.ReadFile("/proc/net/route")
			routes := ParseDefaultRoutes(b)
			if len(routes) == 0 {
				return nil, httpapi.InvalidInput("interface", "no default route; name the interface to listen on")
			}
			iface = routes[0].Interface
		}
		conn, err := t.Open(iface)
		if err != nil {
			return nil, httpapi.InvalidInput("interface", "cannot listen on "+iface+" (unknown interface, or the api container lacks NET_RAW / host networking)")
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "interface": iface, "seconds": seconds}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			defer conn.Close()
			return LLDPSummary{Interface: iface, Seconds: seconds, Neighbors: listen(ctx, conn, time.Duration(seconds)*time.Second, emit)}, nil
		}}, nil
	})
}

func listen(ctx context.Context, conn FrameConn, dur time.Duration, emit stream.Emit) []Neighbor {
	var mu sync.Mutex
	seen := map[string]int{}
	out := []Neighbor{}
	deadline := time.Now().Add(dur)
	buf := make([]byte, 9216)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}
		nb, ok := ParseLLDP(buf[:n])
		if !ok {
			nb, ok = ParseCDP(buf[:n])
		}
		if !ok {
			continue
		}
		key := nb.Protocol + "|" + nb.SourceMAC + "|" + nb.PortID
		mu.Lock()
		if i, dup := seen[key]; dup {
			out[i] = nb // refreshed announcement
		} else if len(out) < maxNeighbor {
			seen[key] = len(out)
			out = append(out, nb)
			emit("neighbor", nb)
		}
		mu.Unlock()
	}
	return out
}
