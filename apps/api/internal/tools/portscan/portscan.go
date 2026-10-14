// Package portscan implements CON-06: a bounded TCP connect scan of one host.
// Off unless PORTSCAN_ENABLED (default off in the public profile).
package portscan

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// Workers bounds simultaneous connects; ConnectTimeout bounds each one.
const (
	Workers        = 64
	ConnectTimeout = 1500 * time.Millisecond
)

// Presets are common port groups.
var Presets = map[string][]int{
	"top100": {7, 9, 13, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 88, 106, 110, 111, 113, 119, 135, 139, 143, 144, 179, 199, 389, 427, 443, 444, 445, 465, 513, 514, 515, 543, 544, 548, 554, 587, 631, 646, 873, 990, 993, 995, 1025, 1026, 1027, 1028, 1029, 1110, 1433, 1720, 1723, 1755, 1900, 2000, 2001, 2049, 2121, 2717, 3000, 3128, 3306, 3389, 3986, 4899, 5000, 5009, 5051, 5060, 5101, 5190, 5357, 5432, 5631, 5666, 5800, 5900, 6000, 6001, 6646, 7070, 8000, 8008, 8009, 8080, 8081, 8443, 8888, 9100, 9999, 10000, 32768, 49152, 49153, 49154, 49155, 49156, 49157},
	"web":    {80, 443, 8000, 8008, 8080, 8081, 8443, 8888, 3000, 5000, 9000, 9443},
	"mail":   {25, 110, 143, 465, 587, 993, 995, 2525, 4190},
	"db":     {1433, 1521, 3306, 5432, 5984, 6379, 7474, 8086, 9042, 9200, 11211, 27017},
	"remote": {22, 23, 3389, 5900, 5901, 5985, 5986},
}

// Services names well-known ports.
var Services = map[int]string{
	21: "ftp", 22: "ssh", 23: "telnet", 25: "smtp", 53: "dns", 80: "http", 110: "pop3", 111: "rpcbind", 135: "msrpc", 139: "netbios-ssn",
	143: "imap", 179: "bgp", 389: "ldap", 443: "https", 445: "smb", 465: "smtps", 514: "syslog", 515: "printer", 548: "afp", 554: "rtsp",
	587: "submission", 631: "ipp", 873: "rsync", 990: "ftps", 993: "imaps", 995: "pop3s", 1433: "mssql", 1521: "oracle", 1723: "pptp",
	1900: "upnp", 2049: "nfs", 2525: "smtp-alt", 3000: "http-dev", 3128: "squid", 3306: "mysql", 3389: "rdp", 4190: "sieve",
	5000: "http-alt", 5060: "sip", 5432: "postgresql", 5900: "vnc", 5901: "vnc-1", 5984: "couchdb", 5985: "winrm", 5986: "winrm-https",
	6379: "redis", 7474: "neo4j", 8000: "http-alt", 8008: "http-alt", 8080: "http-proxy", 8081: "http-alt", 8086: "influxdb",
	8443: "https-alt", 8888: "http-alt", 9000: "http-alt", 9042: "cassandra", 9100: "jetdirect", 9200: "elasticsearch",
	9443: "https-alt", 11211: "memcached", 27017: "mongodb",
}

// ParsePorts accepts a preset name or a list of ports and ranges ("22,80,8000-8100").
func ParsePorts(spec string, maxPorts int) ([]int, error) {
	spec = strings.TrimSpace(strings.ToLower(spec))
	if p, ok := Presets[spec]; ok {
		return slices.Clone(p), nil
	}
	seen := map[int]bool{}
	var out []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lo, hi, isRange := strings.Cut(part, "-")
		a, err1 := strconv.Atoi(strings.TrimSpace(lo))
		b := a
		var err2 error
		if isRange {
			b, err2 = strconv.Atoi(strings.TrimSpace(hi))
		}
		if err1 != nil || err2 != nil || a < 1 || b > 65535 || a > b {
			return nil, httpapi.InvalidInput("ports", "use a preset (top100, web, mail, db, remote) or ports like 22,80,8000-8100")
		}
		if b-a+1 > maxPorts {
			return nil, httpapi.InvalidInput("ports", "at most "+strconv.Itoa(maxPorts)+" ports per scan")
		}
		for p := a; p <= b; p++ {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
		if len(out) > maxPorts {
			return nil, httpapi.InvalidInput("ports", "at most "+strconv.Itoa(maxPorts)+" ports per scan")
		}
	}
	if len(out) == 0 {
		return nil, httpapi.InvalidInput("ports", "no ports given")
	}
	slices.Sort(out)
	return out, nil
}

// Dialer opens one TCP connection (injectable for tests).
type Dialer func(ctx context.Context, addr netip.AddrPort) error

func tcpDial(ctx context.Context, addr netip.AddrPort) error {
	d := net.Dialer{Timeout: ConnectTimeout}
	c, err := d.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return err //nolint:wrapcheck // classified by caller
	}
	return c.Close() //nolint:wrapcheck // irrelevant
}

// Tool is net.portscan.
type Tool struct {
	deps tools.Deps
	Dial Dialer
}

func (*Tool) ID() string { return "net.portscan" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	return tools.RequireFlag(d.Config.Features.Portscan)
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Dial == nil {
		t.Dial = tcpDial
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(d.Config.Limits.StreamTimeoutMax)).Get("/portscan/stream", t.handle)
}

// Port is the "port" event, sent for every open port.
type Port struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
}

// Progress is sent every 50 ports.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// Summary is the done payload.
type Summary struct {
	Address  string `json:"address"`
	Scanned  int    `json:"scanned"`
	Open     []Port `json:"open"`
	Closed   int    `json:"closed"`
	Filtered int    `json:"filtered"`
	Complete bool   `json:"complete"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		lim := t.deps.Config.Limits
		q := params.New(r.URL.Query())
		raw := q.String("target", "", true, 253)
		spec := q.String("ports", "top100", false, 400)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		ports, err := ParsePorts(spec, lim.PortscanMaxPorts)
		if err != nil {
			return nil, err
		}
		tgt, err := t.deps.Target.Resolve(ctx, raw, target.Opts{Field: "target"})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": raw, "address": tgt.IP.String()}, "ports": len(ports), "ratePps": lim.PortscanRatePPS}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return t.scan(ctx, tgt.IP, ports, lim.PortscanRatePPS, emit), nil
		}}, nil
	})
}

func (t *Tool) scan(ctx context.Context, ip netip.Addr, ports []int, pps int, emit stream.Emit) Summary {
	sum := Summary{Address: ip.String(), Open: []Port{}}
	tick := time.NewTicker(time.Second / time.Duration(max(pps, 1)))
	defer tick.Stop()
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, Workers)
	done := 0
loop:
	for _, p := range ports {
		select {
		case <-ctx.Done():
			break loop
		case <-tick.C: // pacing: at most pps connection attempts per second
		}
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			err := t.Dial(ctx, netip.AddrPortFrom(ip, uint16(p))) //nolint:gosec // 1..65535 validated
			mu.Lock()
			defer mu.Unlock()
			done++
			switch {
			case err == nil:
				port := Port{Port: p, Service: Services[p]}
				sum.Open = append(sum.Open, port)
				emit("port", port)
			case errors.Is(err, syscall.ECONNREFUSED):
				sum.Closed++
			default:
				sum.Filtered++ // timeout or unreachable: a firewall drops the probe
			}
			if done%50 == 0 {
				emit("progress", Progress{Done: done, Total: len(ports)})
			}
		})
	}
	wg.Wait()
	slices.SortFunc(sum.Open, func(a, b Port) int { return a.Port - b.Port })
	sum.Scanned = done
	sum.Complete = done == len(ports)
	return sum
}
