// Package dnstest runs an in-process DNS server for tests.
package dnstest

import (
	"net"
	"net/netip"
	"strconv"
	"testing"

	"github.com/miekg/dns"
)

// Start runs a UDP+TCP DNS server on 127.0.0.1 and returns its address. handler builds replies.
func Start(t testing.TB, handler func(w dns.ResponseWriter, r *dns.Msg)) netip.AddrPort {
	t.Helper()
	var lc net.ListenConfig
	pc, err := lc.ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port
	ln, err := lc.Listen(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		_ = pc.Close()
		t.Skipf("cannot bind TCP on the same port: %v", err)
	}
	h := dns.HandlerFunc(handler)
	udp := &dns.Server{PacketConn: pc, Handler: h}
	tcp := &dns.Server{Listener: ln, Handler: h}
	go func() { _ = udp.ActivateAndServe() }()
	go func() { _ = tcp.ActivateAndServe() }()
	t.Cleanup(func() {
		_ = udp.Shutdown()
		_ = tcp.Shutdown()
	})
	return netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(port)) //nolint:gosec // ephemeral port
}
