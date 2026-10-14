package dnsx

import (
	"context"
	"net/netip"
	"slices"
	"strings"

	"github.com/miekg/dns"
)

// ZoneServer is one address of one of a zone's name servers (Addr is invalid and Error set when
// the name does not resolve).
type ZoneServer struct {
	Name  string
	Addr  netip.Addr
	Error string
}

// ZoneServers looks up the NS records of zone via s and resolves each name server, returning at
// most maxNS names and maxServers addresses. Callers must still apply the target policy to every
// address before contacting it.
func ZoneServers(ctx context.Context, s Server, zone string, maxNS, maxServers int) ([]string, []ZoneServer, error) {
	msg, _, err := Query(ctx, s, zone, dns.TypeNS)
	if err != nil {
		return nil, nil, err
	}
	names := []string{}
	for _, rr := range msg.Answer {
		if ns, ok := rr.(*dns.NS); ok && len(names) < maxNS {
			names = append(names, strings.ToLower(ns.Ns))
		}
	}
	slices.Sort(names)
	servers := []ZoneServer{}
	for _, ns := range names {
		addrs, err := Addresses(ctx, s, ns)
		if err != nil || len(addrs) == 0 {
			servers = append(servers, ZoneServer{Name: ns, Error: "nameserver name does not resolve"})
			continue
		}
		for _, a := range addrs {
			if len(servers) < maxServers {
				servers = append(servers, ZoneServer{Name: ns, Addr: a})
			}
		}
	}
	return names, servers, nil
}
