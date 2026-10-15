package lan

import (
	"cmp"
	"context"
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
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/mdlayher/packet"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

var allDHCPv6 = netip.MustParseAddrPort("[ff02::1:2]:547")

// DHCPv6Link sends a SOLICIT and receives server replies (injectable for tests).
type DHCPv6Link interface {
	Send(b []byte) error
	// Receive returns one DHCPv6 payload sent from UDP port 547 and the server's address.
	Receive(b []byte) (int, netip.Addr, error)
	SetReadDeadline(t time.Time) error
	Close() error
	MAC() net.HardwareAddr
}

// dhcpv6Link sends from the link-local address (any port) and captures replies with an AF_PACKET
// socket: servers answer on client port 546, which needs CAP_NET_BIND_SERVICE to bind.
type dhcpv6Link struct {
	udp *net.UDPConn
	cap *packet.Conn
	ifi *net.Interface
}

func (l *dhcpv6Link) Send(b []byte) error {
	_, err := l.udp.WriteToUDP(b, &net.UDPAddr{IP: allDHCPv6.Addr().AsSlice(), Port: int(allDHCPv6.Port()), Zone: l.ifi.Name})
	return err //nolint:wrapcheck // counted, not shown
}

func (l *dhcpv6Link) Receive(b []byte) (int, netip.Addr, error) {
	buf := make([]byte, 2048)
	for {
		n, _, err := l.cap.ReadFrom(buf)
		if err != nil {
			return 0, netip.Addr{}, err //nolint:wrapcheck // deadline ends the loop
		}
		// Cooked IPv6 packet: next header at 6, source at 8..24, UDP header at 40, payload at 48.
		if n < 48 || buf[6] != 17 || binary.BigEndian.Uint16(buf[40:]) != 547 {
			continue
		}
		src, _ := netip.AddrFromSlice(buf[8:24])
		return copy(b, buf[48:n]), src, nil
	}
}

func (l *dhcpv6Link) SetReadDeadline(t time.Time) error {
	return l.cap.SetReadDeadline(t) //nolint:wrapcheck // passthrough
}
func (l *dhcpv6Link) MAC() net.HardwareAddr { return l.ifi.HardwareAddr }
func (l *dhcpv6Link) Close() error {
	_ = l.cap.Close()
	return l.udp.Close() //nolint:wrapcheck // passthrough
}

// dhcpv6Filter passes IPv6 UDP datagrams from port 547 (cooked packets, no extension headers).
func dhcpv6Filter() ([]bpf.RawInstruction, error) {
	prog, err := bpf.Assemble([]bpf.Instruction{
		bpf.LoadAbsolute{Off: 6, Size: 1},                       // 0 next header
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 17, SkipFalse: 3},  // 1 UDP?
		bpf.LoadAbsolute{Off: 40, Size: 2},                      // 2 UDP source port
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: 547, SkipFalse: 1}, // 3 DHCPv6 server?
		bpf.RetConstant{Val: 0xffff},                            // 4 accept
		bpf.RetConstant{Val: 0},                                 // 5 reject
	})
	if err != nil {
		return nil, fmt.Errorf("bpf: %w", err)
	}
	return prog, nil
}

func openDHCPv6(iface string) (DHCPv6Link, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface: %w", err)
	}
	addrs, _ := ifi.Addrs()
	var ll net.IP
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() == nil && n.IP.IsLinkLocalUnicast() {
			ll = n.IP
		}
	}
	if ll == nil {
		return nil, fmt.Errorf("%s has no IPv6 link-local address", iface)
	}
	udp, err := net.ListenUDP("udp6", &net.UDPAddr{IP: ll, Zone: iface})
	if err != nil {
		return nil, fmt.Errorf("udp socket: %w", err)
	}
	c, err := packet.Listen(ifi, packet.Datagram, unix.ETH_P_IPV6, nil)
	if err != nil {
		_ = udp.Close()
		return nil, fmt.Errorf("raw socket: %w", err)
	}
	prog, err := dhcpv6Filter()
	if err == nil {
		if e := c.SetBPF(prog); e != nil {
			err = fmt.Errorf("attach filter: %w", e)
		}
	}
	if err != nil {
		_ = c.Close()
		_ = udp.Close()
		return nil, err
	}
	return &dhcpv6Link{udp: udp, cap: c, ifi: ifi}, nil
}

// V6Lease is an address or prefix offered by a server.
type V6Lease struct {
	Value    string `json:"value"`
	ValidSec int64  `json:"validSeconds"`
}

// V6Offer is one ADVERTISE.
type V6Offer struct {
	Server     string    `json:"server"`
	DUID       string    `json:"duid"`
	Preference int       `json:"preference"`
	Addresses  []V6Lease `json:"addresses"`
	Prefixes   []V6Lease `json:"prefixes"`
	DNS        []string  `json:"dns"`
	Search     []string  `json:"search"`
	Status     string    `json:"status"`
}

// DHCPv6Result is the /lan/dhcpv6 response.
type DHCPv6Result struct {
	Interface string    `json:"interface"`
	Seconds   int       `json:"seconds"`
	Offers    []V6Offer `json:"offers"`
	Servers   int       `json:"servers"`
}

// OfferFromAdvertise extracts what a server offered.
func OfferFromAdvertise(src netip.Addr, m *dhcpv6.Message) V6Offer {
	o := V6Offer{Server: src.String(), Preference: int(m.Options.Preference()), Addresses: []V6Lease{}, Prefixes: []V6Lease{}, DNS: []string{}, Search: []string{}}
	if d := m.Options.ServerID(); d != nil {
		o.DUID = d.String()
	}
	for _, ia := range m.Options.IANA() {
		for _, a := range ia.Options.Addresses() {
			o.Addresses = append(o.Addresses, V6Lease{a.IPv6Addr.String(), int64(a.ValidLifetime / time.Second)})
		}
	}
	for _, pd := range m.Options.IAPD() {
		for _, p := range pd.Options.Prefixes() {
			if p.Prefix != nil {
				o.Prefixes = append(o.Prefixes, V6Lease{p.Prefix.String(), int64(p.ValidLifetime / time.Second)})
			}
		}
	}
	for _, d := range m.Options.DNS() {
		o.DNS = append(o.DNS, d.String())
	}
	if l := m.Options.DomainSearchList(); l != nil {
		o.Search = append(o.Search, l.Labels...)
	}
	if s := m.Options.Status(); s != nil && s.StatusCode != 0 {
		o.Status = strings.TrimSpace(s.StatusCode.String() + " " + s.StatusMessage)
	}
	return o
}

// DHCPv6Tool is LAN-09: DHCPv6 probe (SOLICIT only; nothing is leased).
type DHCPv6Tool struct {
	deps     tools.Deps
	Open     func(iface string) (DHCPv6Link, error)
	ReadFile func(string) ([]byte, error)
}

func (*DHCPv6Tool) ID() string                                { return "lan.dhcpv6" }
func (*DHCPv6Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *DHCPv6Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Open == nil {
		t.Open = openDHCPv6
	}
	if t.ReadFile == nil {
		t.ReadFile = os.ReadFile
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/lan/dhcpv6", t.handle)
}

func (t *DHCPv6Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	iface := q.String("interface", "", false, 32)
	seconds := q.Int("seconds", 4, 1, 10)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if iface == "" {
		b, _ := t.ReadFile("/proc/net/route")
		routes := ParseDefaultRoutes(b)
		if len(routes) == 0 {
			httpapi.Error(w, r, httpapi.InvalidInput("interface", "no default route; name the interface to probe"))
			return
		}
		iface = routes[0].Interface
	}
	link, err := t.Open(iface)
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("interface", "cannot probe DHCPv6 on "+iface+" (no link-local address, unknown interface, or no NET_RAW / host networking)"))
		return
	}
	defer link.Close()
	res, err := probeDHCPv6(r.Context(), link, time.Duration(seconds)*time.Second)
	if err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "the DHCPv6 probe failed", Err: err})
		return
	}
	res.Interface, res.Seconds = iface, seconds
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

func probeDHCPv6(ctx context.Context, link DHCPv6Link, dur time.Duration) (DHCPv6Result, error) {
	res := DHCPv6Result{Offers: []V6Offer{}}
	mac := link.MAC()
	var iaid [4]byte
	copy(iaid[:], mac[max(0, len(mac)-4):])
	sol, err := dhcpv6.NewSolicit(mac, dhcpv6.WithIAPD(iaid))
	if err != nil {
		return res, fmt.Errorf("solicit: %w", err)
	}
	if err := link.Send(sol.ToBytes()); err != nil {
		return res, fmt.Errorf("send: %w", err)
	}
	seen := map[string]bool{}
	duids := map[string]bool{}
	deadline := time.Now().Add(dur)
	nextSend := time.Now().Add(resendEvery)
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) && ctx.Err() == nil && len(res.Offers) < MaxOffers {
		if time.Now().After(nextSend) {
			_ = link.Send(sol.ToBytes())
			nextSend = time.Now().Add(resendEvery)
		}
		_ = link.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, src, err := link.Receive(buf)
		if err != nil {
			continue
		}
		d, err := dhcpv6.FromBytes(buf[:n])
		m, ok := d.(*dhcpv6.Message)
		if err != nil || !ok || m.TransactionID != sol.TransactionID || m.MessageType != dhcpv6.MessageTypeAdvertise {
			continue
		}
		o := OfferFromAdvertise(src, m)
		key := o.DUID + "|" + o.Server
		if seen[key] {
			continue
		}
		seen[key] = true
		duids[cmp.Or(o.DUID, o.Server)] = true
		res.Offers = append(res.Offers, o)
	}
	slices.SortFunc(res.Offers, func(a, b V6Offer) int { return b.Preference - a.Preference })
	res.Servers = len(duids)
	return res, nil
}
