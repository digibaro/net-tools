package serverinfo

import (
	"bufio"
	"encoding/hex"
	"net/netip"
	"strconv"
	"strings"
)

// Route is one kernel route.
type Route struct {
	Family      int    `json:"family"`
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      int    `json:"metric"`
	Default     bool   `json:"default"`
}

func le32(s string) (netip.Addr, bool) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 4 {
		return netip.Addr{}, false
	}
	return netip.AddrFrom4([4]byte{b[3], b[2], b[1], b[0]}), true
}

// ParseRouteV4 parses /proc/net/route (little-endian hex addresses).
func ParseRouteV4(text string) []Route {
	var out []Route
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 8 || f[0] == "Iface" {
			continue
		}
		dst, ok1 := le32(f[1])
		gw, ok2 := le32(f[2])
		mask, ok3 := le32(f[7])
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		bits := 0
		for _, b := range mask.As4() {
			for i := 7; i >= 0; i-- {
				if b&(1<<i) != 0 {
					bits++
				}
			}
		}
		metric, _ := strconv.Atoi(f[6])
		r := Route{Family: 4, Destination: netip.PrefixFrom(dst, bits).String(), Interface: f[0], Metric: metric, Default: bits == 0}
		if gw != netip.IPv4Unspecified() {
			r.Gateway = gw.String()
		}
		out = append(out, r)
	}
	return out
}

func hex16(s string) (netip.Addr, bool) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 16 {
		return netip.Addr{}, false
	}
	return netip.AddrFrom16([16]byte(b)), true
}

// ParseRouteV6 parses /proc/net/ipv6_route.
func ParseRouteV6(text string) []Route {
	var out []Route
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 10 {
			continue
		}
		dst, ok1 := hex16(f[0])
		plen, err := strconv.ParseInt(f[1], 16, 32)
		gw, ok2 := hex16(f[4])
		if !ok1 || !ok2 || err != nil || f[9] == "lo" {
			continue
		}
		metric, _ := strconv.ParseInt(f[5], 16, 64)
		r := Route{Family: 6, Destination: netip.PrefixFrom(dst, int(plen)).String(), Interface: f[9], Metric: int(metric), Default: plen == 0}
		if gw != netip.IPv6Unspecified() {
			r.Gateway = gw.String()
		}
		out = append(out, r)
	}
	return out
}

// Resolv is the parsed resolv.conf.
type Resolv struct {
	Nameservers []string `json:"nameservers"`
	Search      []string `json:"search"`
	Options     []string `json:"options"`
}

// ParseResolvConf extracts nameserver, search/domain and options lines.
func ParseResolvConf(text string) Resolv {
	r := Resolv{Nameservers: []string{}, Search: []string{}, Options: []string{}}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		switch f[0] {
		case "nameserver":
			r.Nameservers = append(r.Nameservers, f[1])
		case "search", "domain":
			r.Search = append(r.Search, f[1:]...)
		case "options":
			r.Options = append(r.Options, f[1:]...)
		}
	}
	return r
}
