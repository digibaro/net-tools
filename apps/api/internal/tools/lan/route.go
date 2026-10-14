package lan

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net/netip"
	"strings"
)

// Route is an IPv4 default route from /proc/net/route.
type Route struct {
	Interface string
	Gateway   netip.Addr
	Metric    int
}

// ParseDefaultRoutes reads /proc/net/route (little-endian hex) and returns the default routes,
// lowest metric first.
func ParseDefaultRoutes(b []byte) []Route {
	var out []Route
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Scan() // header
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 8 || f[1] != "00000000" || f[7] != "00000000" {
			continue
		}
		gw, err := hex.DecodeString(f[2])
		if err != nil || len(gw) != 4 {
			continue
		}
		var a [4]byte
		binary.BigEndian.PutUint32(a[:], binary.LittleEndian.Uint32(gw))
		metric := 0
		for _, c := range f[6] {
			if c < '0' || c > '9' {
				metric = 0
				break
			}
			metric = metric*10 + int(c-'0')
		}
		out = append(out, Route{Interface: f[0], Gateway: netip.AddrFrom4(a), Metric: metric})
	}
	for i := 1; i < len(out); i++ { // insertion sort by metric; tables are tiny
		for j := i; j > 0 && out[j].Metric < out[j-1].Metric; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// ParseARPTable reads /proc/net/arp into IP → MAC for complete entries.
func ParseARPTable(b []byte) map[netip.Addr]string {
	out := map[netip.Addr]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Scan()
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 4 || f[2] == "0x0" || f[3] == "00:00:00:00:00:00" {
			continue
		}
		if a, err := netip.ParseAddr(f[0]); err == nil {
			out[a] = f[3]
		}
	}
	return out
}
