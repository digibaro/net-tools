package middleware

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP resolves the real client address. X-Forwarded-For is honoured only when the direct peer
// is in trusted; the chain is then walked right-to-left, skipping trusted hops (docs/SECURITY.md §4).
func ClientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, trusted)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
		})
	}
}

// GetClientIP returns the client address stored by ClientIP (invalid Addr if absent).
func GetClientIP(ctx context.Context) netip.Addr {
	ip, _ := ctx.Value(clientIPKey).(netip.Addr)
	return ip
}

func resolveClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	ip := peer.Addr().Unmap()
	if !isTrusted(ip, trusted) {
		return ip
	}
	hops := r.Header.Values("X-Forwarded-For")
	parts := strings.Split(strings.Join(hops, ","), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err != nil {
			// Malformed entry: stop and use the last address we could trust.
			return ip
		}
		hop = hop.Unmap().WithZone("")
		ip = hop
		if !isTrusted(hop, trusted) {
			return hop
		}
	}
	return ip
}

func isTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
