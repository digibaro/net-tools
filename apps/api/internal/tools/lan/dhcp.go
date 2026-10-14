package lan

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// MaxOffers bounds what one probe collects.
const MaxOffers = 20

// DHCPOpener opens a raw DHCP client socket (UDP 68) on an interface and returns its MAC.
type DHCPOpener func(iface string) (net.PacketConn, net.HardwareAddr, error)

func openDHCP(iface string) (net.PacketConn, net.HardwareAddr, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, nil, fmt.Errorf("interface: %w", err)
	}
	c, err := nclient4.NewRawUDPConn(iface, dhcpv4.ClientPort) // AF_PACKET: needs NET_RAW only
	if err != nil {
		return nil, nil, fmt.Errorf("raw socket: %w", err)
	}
	return c, ifi.HardwareAddr, nil
}

// DHCPTool is LAN-05: rogue DHCP detection (DISCOVER only; no REQUEST, so no lease is taken).
type DHCPTool struct {
	deps     tools.Deps
	Open     DHCPOpener
	ReadFile func(string) ([]byte, error)
}

func (*DHCPTool) ID() string                                { return "lan.dhcp" }
func (*DHCPTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *DHCPTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Open == nil {
		t.Open = openDHCP
	}
	if t.ReadFile == nil {
		t.ReadFile = os.ReadFile
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/lan/dhcp", t.handle)
}

// Offer is one DHCPOFFER.
type Offer struct {
	Server       string   `json:"server"`
	From         string   `json:"from"`
	Relay        string   `json:"relay"`
	OfferedIP    string   `json:"offeredIp"`
	SubnetMask   string   `json:"subnetMask"`
	Routers      []string `json:"routers"`
	DNS          []string `json:"dns"`
	Domain       string   `json:"domain"`
	LeaseSeconds int      `json:"leaseSeconds"`
}

// DHCPResult is the response.
type DHCPResult struct {
	Interface string  `json:"interface"`
	MAC       string  `json:"mac"`
	Seconds   int     `json:"seconds"`
	Offers    []Offer `json:"offers"`
	Servers   int     `json:"servers"`
}

func ips(list []net.IP) []string {
	out := []string{}
	for _, ip := range list {
		out = append(out, ip.String())
	}
	return out
}

func (t *DHCPTool) handle(w http.ResponseWriter, r *http.Request) {
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
	conn, mac, err := t.Open(iface)
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("interface", "cannot open a raw socket on "+iface+" (unknown interface, or the api container lacks NET_RAW / host networking)"))
		return
	}
	defer conn.Close()
	res, err := probe(r.Context(), conn, mac, time.Duration(seconds)*time.Second)
	if err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "the DHCP probe failed", Err: err})
		return
	}
	res.Interface, res.Seconds = iface, seconds
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

func probe(ctx context.Context, conn net.PacketConn, mac net.HardwareAddr, dur time.Duration) (DHCPResult, error) {
	res := DHCPResult{MAC: mac.String(), Offers: []Offer{}}
	var xid dhcpv4.TransactionID
	_, _ = rand.Read(xid[:])
	disc, err := dhcpv4.NewDiscovery(mac, dhcpv4.WithTransactionID(xid), dhcpv4.WithBroadcast(true),
		dhcpv4.WithRequestedOptions(dhcpv4.OptionSubnetMask, dhcpv4.OptionRouter, dhcpv4.OptionDomainNameServer, dhcpv4.OptionDomainName))
	if err != nil {
		return res, fmt.Errorf("discover: %w", err)
	}
	dst := &net.UDPAddr{IP: net.IPv4bcast, Port: dhcpv4.ServerPort}
	if _, err := conn.WriteTo(disc.ToBytes(), dst); err != nil {
		return res, fmt.Errorf("send: %w", err)
	}
	seen := map[string]bool{}
	servers := map[string]bool{}
	deadline := time.Now().Add(dur)
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) && ctx.Err() == nil && len(res.Offers) < MaxOffers {
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return res, fmt.Errorf("read: %w", err)
		}
		m, err := dhcpv4.FromBytes(buf[:n])
		if err != nil || m.TransactionID != xid || m.MessageType() != dhcpv4.MessageTypeOffer || m.OpCode != dhcpv4.OpcodeBootReply {
			continue
		}
		o := Offer{OfferedIP: m.YourIPAddr.String(), Routers: ips(m.Router()), DNS: ips(m.DNS()), Domain: m.DomainName(), LeaseSeconds: int(m.IPAddressLeaseTime(0).Seconds())}
		if sid := m.ServerIdentifier(); sid != nil {
			o.Server = sid.String()
		}
		if ua, ok := from.(*net.UDPAddr); ok {
			o.From = ua.IP.String()
		}
		if gi := m.GatewayIPAddr; gi != nil && !gi.IsUnspecified() {
			o.Relay = gi.String()
		}
		if mask := m.SubnetMask(); mask != nil {
			o.SubnetMask = net.IP(mask).String()
		}
		key := o.Server + "|" + o.From + "|" + o.OfferedIP
		if seen[key] {
			continue
		}
		seen[key] = true
		servers[cmp.Or(o.Server, o.From)] = true
		res.Offers = append(res.Offers, o)
	}
	slices.SortFunc(res.Offers, func(a, b Offer) int { return strings.Compare(a.Server, b.Server) })
	res.Servers = len(servers)
	return res, nil
}
