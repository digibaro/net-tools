// Package lan implements the LAN-mode tools (LAN-01..06). They need host networking
// (docker-compose.lan.yml, NETTOOLS_LAN=true) and report requires_lan_mode otherwise.
package lan

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
)

// MagicPacket builds a Wake-on-LAN packet: 6×0xFF then the MAC 16 times.
func MagicPacket(mac net.HardwareAddr) []byte {
	p := make([]byte, 0, 102)
	for range 6 {
		p = append(p, 0xFF)
	}
	for range 16 {
		p = append(p, mac...)
	}
	return p
}

// Sender transmits a UDP datagram with SO_BROADCAST (injectable for tests).
type Sender func(ctx context.Context, dst netip.AddrPort, payload []byte) error

func udpBroadcast(ctx context.Context, dst netip.AddrPort, payload []byte) error {
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1) //nolint:gosec // fd fits int
		}); err != nil {
			return err //nolint:wrapcheck // socket setup
		}
		if serr != nil {
			return fmt.Errorf("SO_BROADCAST: %w", serr)
		}
		return nil
	}}
	pc, err := lc.ListenPacket(ctx, "udp4", ":0")
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}
	defer pc.Close()
	_ = pc.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := pc.WriteTo(payload, net.UDPAddrFromAddrPort(dst)); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}

// WOLTool is LAN-01.
type WOLTool struct {
	deps tools.Deps
	Send Sender
}

func (*WOLTool) ID() string                                { return "lan.wol" }
func (*WOLTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.RequireLAN(d) }

func (t *WOLTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Send == nil {
		t.Send = udpBroadcast
	}
	r.With(d.RateLimit(limits.Light), d.Timeout(5*time.Second)).Post("/lan/wol", t.handle)
}

// WOLRequest is the POST body.
type WOLRequest struct {
	MAC       string `json:"mac"`
	Broadcast string `json:"broadcast"`
	Port      int    `json:"port"`
}

// WOLResult is the response payload.
type WOLResult struct {
	MAC         string `json:"mac"`
	Destination string `json:"destination"`
	Bytes       int    `json:"bytes"`
}

var privateBroadcast = []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("169.254.0.0/16")}

// ValidBroadcast allows the limited broadcast or a directed broadcast inside private space only,
// so the tool cannot be used to send traffic onto the Internet.
func ValidBroadcast(s string) (netip.Addr, bool) {
	if s == "" {
		return netip.MustParseAddr("255.255.255.255"), true
	}
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil || !a.Is4() {
		return netip.Addr{}, false
	}
	if a == netip.MustParseAddr("255.255.255.255") {
		return a, true
	}
	for _, p := range privateBroadcast {
		if p.Contains(a) {
			return a, true
		}
	}
	return netip.Addr{}, false
}

func (t *WOLTool) handle(w http.ResponseWriter, r *http.Request) {
	var req WOLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("body", "invalid JSON"))
		return
	}
	mac, err := net.ParseMAC(strings.TrimSpace(req.MAC))
	if err != nil || len(mac) != 6 {
		httpapi.Error(w, r, httpapi.InvalidInput("mac", "enter a 48-bit MAC address such as 00:11:22:33:44:55"))
		return
	}
	dst, ok := ValidBroadcast(req.Broadcast)
	if !ok {
		httpapi.Error(w, r, httpapi.InvalidInput("broadcast", "use 255.255.255.255 or a broadcast address in a private network"))
		return
	}
	if req.Port == 0 {
		req.Port = 9
	}
	if req.Port != 7 && req.Port != 9 {
		httpapi.Error(w, r, httpapi.InvalidInput("port", "Wake-on-LAN uses UDP port 7 or 9"))
		return
	}
	pkt := MagicPacket(mac)
	to := netip.AddrPortFrom(dst, uint16(req.Port)) //nolint:gosec // 7 or 9
	if err := t.Send(r.Context(), to, pkt); err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInternal, Message: "the magic packet could not be sent", Err: err})
		return
	}
	httpapi.JSON(w, r, WOLResult{MAC: mac.String(), Destination: to.String(), Bytes: len(pkt)}, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
