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
	return StartOn(t, netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), 0), handler)
}

// StartOn is Start on a chosen loopback address and port (0 = ephemeral).
func StartOn(t testing.TB, at netip.AddrPort, handler func(w dns.ResponseWriter, r *dns.Msg)) netip.AddrPort {
	t.Helper()
	var lc net.ListenConfig
	pc, err := lc.ListenPacket(t.Context(), "udp", at.String())
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port
	ln, err := lc.Listen(t.Context(), "tcp", net.JoinHostPort(at.Addr().String(), strconv.Itoa(port)))
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
	return netip.AddrPortFrom(at.Addr(), uint16(port)) //nolint:gosec // ephemeral port
}
